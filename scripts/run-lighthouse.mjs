import { spawn } from "node:child_process";
import { once } from "node:events";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { relative, resolve } from "node:path";

import * as chromeLauncher from "chrome-launcher";
import lighthouse from "lighthouse";

const baseUrl = "http://localhost:3000";
const outputDirectory = resolve(".lighthouseci");
const lighthouseConfig = JSON.parse(
  await readFile(resolve("lighthouserc.json"), "utf8"),
);
const auditEnvironment = {
  ...process.env,
  NEXT_PUBLIC_SITE_URL: baseUrl,
  SEO_AUDIT_INDEXABLE: "true",
};

function startProcess(args, environment = auditEnvironment) {
  return spawn(process.execPath, args, {
    cwd: process.cwd(),
    env: environment,
    stdio: "inherit",
    windowsHide: true,
  });
}

async function runProcess(args) {
  const processHandle = startProcess(args);
  const [exitCode] = await once(processHandle, "exit");

  if (exitCode !== 0) {
    throw new Error(
      `Proses ${args.join(" ")} berhenti dengan exit code ${exitCode}.`,
    );
  }
}

async function waitForServer(server) {
  for (let attempt = 0; attempt < 120; attempt += 1) {
    if (server.exitCode !== null) {
      throw new Error("Server audit berhenti sebelum siap.");
    }

    try {
      const response = await fetch(baseUrl);

      if (response.ok) {
        return;
      }
    } catch {
      // Server audit masih memulai.
    }

    await new Promise((resolveDelay) => setTimeout(resolveDelay, 250));
  }

  throw new Error(`Server audit tidak siap pada ${baseUrl}.`);
}

async function stopServer(server) {
  if (!server || server.exitCode !== null || server.signalCode !== null) {
    return;
  }

  server.kill("SIGTERM");

  await Promise.race([
    once(server, "exit"),
    new Promise((resolveDelay) => setTimeout(resolveDelay, 3000)),
  ]);

  if (server.exitCode === null && server.signalCode === null) {
    server.kill("SIGKILL");
    await once(server, "exit");
  }
}

function getNumberOfRuns() {
  const override = process.argv
    .slice(2)
    .find((argument) => argument.startsWith("--collect.numberOfRuns="));
  const configuredRuns = lighthouseConfig.ci.collect.numberOfRuns;
  const value = override ? Number(override.split("=")[1]) : configuredRuns;

  if (!Number.isInteger(value) || value < 1) {
    throw new Error("Jumlah Lighthouse run harus berupa bilangan bulat positif.");
  }

  return value;
}

function median(values) {
  const sortedValues = [...values].sort((left, right) => left - right);
  const middle = Math.floor(sortedValues.length / 2);

  if (sortedValues.length % 2 === 0) {
    return (sortedValues[middle - 1] + sortedValues[middle]) / 2;
  }

  return sortedValues[middle];
}

function aggregate(values, method) {
  if (method === "median") {
    return median(values);
  }

  if (method === "max") {
    return Math.max(...values);
  }

  if (method === "min") {
    return Math.min(...values);
  }

  return Math.min(...values);
}

function reportSlug(url) {
  const pathname = new URL(url).pathname;

  if (pathname === "/") {
    return "home";
  }

  return pathname.replace(/^\/|\/$/g, "").replaceAll("/", "-");
}

function checkAssertions(resultsByUrl) {
  const failures = [];
  const warnings = [];
  const assertions = lighthouseConfig.ci.assert.assertions;

  for (const [url, reports] of resultsByUrl) {
    for (const [auditName, [level, options]] of Object.entries(assertions)) {
      const categoryPrefix = "categories:";
      const isCategory = auditName.startsWith(categoryPrefix);
      const values = reports.map((report) => {
        if (isCategory) {
          const categoryName = auditName.slice(categoryPrefix.length);
          return report.categories[categoryName].score;
        }

        return report.audits[auditName].numericValue;
      });
      const actualValue = aggregate(values, options.aggregationMethod);
      const hasMinimum = typeof options.minScore === "number";
      const expectedValue = hasMinimum
        ? options.minScore
        : options.maxNumericValue;
      const passed = hasMinimum
        ? actualValue >= expectedValue
        : actualValue <= expectedValue;

      if (passed) {
        continue;
      }

      const comparison = hasMinimum ? ">=" : "<=";
      const message = `${url} ${auditName}: ${actualValue.toFixed(3)} harus ${comparison} ${expectedValue}.`;

      if (level === "warn") {
        warnings.push(message);
      } else {
        failures.push(message);
      }
    }
  }

  for (const warning of warnings) {
    console.warn(`Peringatan Lighthouse: ${warning}`);
  }

  if (failures.length > 0) {
    throw new Error(`Lighthouse gate gagal:\n${failures.join("\n")}`);
  }
}

async function runLighthouseAudits() {
  const urls = lighthouseConfig.ci.collect.url;
  const numberOfRuns = getNumberOfRuns();
  const configuredFlags = lighthouseConfig.ci.collect.settings.chromeFlags
    .split(/\s+/)
    .filter(Boolean);
  const manifest = [];
  const resultsByUrl = new Map();

  await mkdir(outputDirectory, { recursive: true });

  for (const url of urls) {
    const reports = [];
    console.log(`Menjalankan Lighthouse ${numberOfRuns}x untuk ${url}`);

    for (let run = 1; run <= numberOfRuns; run += 1) {
      const chrome = await chromeLauncher.launch({
        chromeFlags: configuredFlags,
      });

      try {
        const result = await lighthouse(url, {
          logLevel: "error",
          onlyCategories: [
            "performance",
            "accessibility",
            "best-practices",
            "seo",
          ],
          output: ["json", "html"],
          port: chrome.port,
        });

        if (!result) {
          throw new Error(`Lighthouse tidak menghasilkan report untuk ${url}.`);
        }

        const [jsonReport, htmlReport] = result.report;
        const slug = reportSlug(url);
        const jsonPath = resolve(
          outputDirectory,
          `${slug}-run-${run}.report.json`,
        );
        const htmlPath = resolve(
          outputDirectory,
          `${slug}-run-${run}.report.html`,
        );

        await Promise.all([
          writeFile(jsonPath, jsonReport, "utf8"),
          writeFile(htmlPath, htmlReport, "utf8"),
        ]);

        reports.push(result.lhr);
        manifest.push({
          url,
          jsonPath: relative(process.cwd(), jsonPath),
          htmlPath: relative(process.cwd(), htmlPath),
          isRepresentativeRun: false,
          summary: Object.fromEntries(
            Object.entries(result.lhr.categories).map(([name, category]) => [
              name,
              category.score,
            ]),
          ),
        });
        console.log(`  Run ${run}/${numberOfRuns} selesai.`);
      } finally {
        await chrome.kill();
      }
    }

    resultsByUrl.set(url, reports);
  }

  checkAssertions(resultsByUrl);
  await writeFile(
    resolve(outputDirectory, "manifest.json"),
    JSON.stringify(manifest, null, 2),
    "utf8",
  );

  console.log(
    `Lighthouse gate lulus. ${manifest.length} report tersimpan di ${outputDirectory}.`,
  );
}

let server;

try {
  await runProcess([
    resolve("node_modules/next/dist/bin/next"),
    "build",
    "--webpack",
  ]);

  server = startProcess([
    resolve("node_modules/next/dist/bin/next"),
    "start",
    "-H",
    "localhost",
    "-p",
    "3000",
  ]);
  await waitForServer(server);

  await runLighthouseAudits();
} finally {
  await stopServer(server);
}
