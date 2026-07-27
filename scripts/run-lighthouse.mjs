import { spawn } from "node:child_process";
import { once } from "node:events";
import {
  mkdir,
  mkdtemp,
  readFile,
  rename,
  rm,
  stat,
  writeFile,
} from "node:fs/promises";
import { createServer } from "node:net";
import { dirname, relative, resolve } from "node:path";

import * as chromeLauncher from "chrome-launcher";
import lighthouse from "lighthouse";

const outputDirectory = resolve(".lighthouseci");
const auditBuildDirectory = resolve(".next-lighthouse");
const manifestPath = resolve(outputDirectory, "manifest.json");
const statusPath = resolve(outputDirectory, "run-status.json");
const nextEnvironmentPath = resolve("next-env.d.ts");
const lighthouseConfig = JSON.parse(
  await readFile(resolve("lighthouserc.json"), "utf8"),
);
const activeProcesses = new Set();

let auditEnvironment;

function startProcess(args, environment = auditEnvironment) {
  if (!environment) {
    throw new Error("Environment audit belum disiapkan.");
  }

  const processHandle = spawn(process.execPath, args, {
    cwd: process.cwd(),
    env: environment,
    stdio: "inherit",
    windowsHide: true,
  });

  activeProcesses.add(processHandle);
  processHandle.once("exit", () => activeProcesses.delete(processHandle));
  processHandle.once("error", () => activeProcesses.delete(processHandle));

  return processHandle;
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

async function stopProcess(processHandle) {
  if (
    !processHandle ||
    processHandle.exitCode !== null ||
    processHandle.signalCode !== null
  ) {
    return;
  }

  processHandle.kill("SIGTERM");

  await Promise.race([
    once(processHandle, "exit"),
    new Promise((resolveDelay) => setTimeout(resolveDelay, 3000)),
  ]);

  if (
    processHandle.exitCode === null &&
    processHandle.signalCode === null
  ) {
    processHandle.kill("SIGKILL");
    await once(processHandle, "exit");
  }
}

async function stopActiveProcesses() {
  await Promise.allSettled(
    [...activeProcesses].map((processHandle) =>
      stopProcess(processHandle),
    ),
  );
}

async function findAvailablePort() {
  return new Promise((resolvePort, rejectPort) => {
    const probe = createServer();

    probe.unref();
    probe.once("error", rejectPort);
    probe.listen(0, "127.0.0.1", () => {
      const address = probe.address();

      if (!address || typeof address === "string") {
        probe.close();
        rejectPort(new Error("Port audit local tidak dapat ditentukan."));
        return;
      }

      const { port } = address;

      probe.close((error) => {
        if (error) {
          rejectPort(error);
          return;
        }

        resolvePort(port);
      });
    });
  });
}

async function waitForServer(server, baseUrl) {
  for (let attempt = 0; attempt < 120; attempt += 1) {
    if (server.exitCode !== null) {
      throw new Error("Server audit berhenti sebelum siap.");
    }

    try {
      const response = await fetch(baseUrl, { cache: "no-store" });

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

async function verifyAuditServer(baseUrl) {
  const [homepageResponse, robotsResponse] = await Promise.all([
    fetch(baseUrl, { cache: "no-store" }),
    fetch(new URL("/robots.txt", baseUrl), { cache: "no-store" }),
  ]);

  if (!homepageResponse.ok || !robotsResponse.ok) {
    throw new Error("Endpoint verifikasi audit tidak merespons dengan sukses.");
  }

  const [homepage, robots] = await Promise.all([
    homepageResponse.text(),
    robotsResponse.text(),
  ]);
  const canonicalValue = homepage.match(
    /<link rel="canonical" href="([^"]+)"/,
  )?.[1];
  const robotsValue = homepage.match(
    /<meta name="robots" content="([^"]+)"/,
  )?.[1];

  if (!canonicalValue) {
    throw new Error("Canonical tidak ditemukan pada audit build.");
  }

  const canonical = new URL(canonicalValue);
  const auditOrigin = new URL(baseUrl);

  if (
    canonical.origin !== auditOrigin.origin ||
    canonical.pathname !== "/"
  ) {
    throw new Error("Canonical audit tidak memakai origin server aktif.");
  }

  if (
    !robotsValue?.includes("index") ||
    !robotsValue.includes("follow") ||
    robotsValue.includes("noindex") ||
    robotsValue.includes("nofollow")
  ) {
    throw new Error("Audit build tidak berada pada mode indexable.");
  }

  if (
    !robots.includes("Allow: /") ||
    !robots.includes(
      `Sitemap: ${new URL("/sitemap.xml", baseUrl).toString()}`,
    )
  ) {
    throw new Error("robots.txt audit tidak sesuai dengan origin aktif.");
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

function getAuditUrls(baseUrl) {
  return lighthouseConfig.ci.collect.url.map((configuredUrl) => {
    const pathname = new URL(configuredUrl).pathname;

    return new URL(pathname, `${baseUrl}/`).toString();
  });
}

async function runLighthouseAudits(
  baseUrl,
  runDirectory,
  numberOfRuns,
) {
  const urls = getAuditUrls(baseUrl);
  const configuredFlags = lighthouseConfig.ci.collect.settings.chromeFlags
    .split(/\s+/)
    .filter(Boolean);
  const manifest = [];
  const resultsByUrl = new Map();

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
        const jsonFileName = `${slug}-run-${run}.report.json`;
        const htmlFileName = `${slug}-run-${run}.report.html`;

        await Promise.all([
          writeFile(
            resolve(runDirectory, jsonFileName),
            jsonReport,
            "utf8",
          ),
          writeFile(
            resolve(runDirectory, htmlFileName),
            htmlReport,
            "utf8",
          ),
        ]);

        reports.push(result.lhr);
        manifest.push({
          url,
          jsonPath: relative(
            process.cwd(),
            resolve(outputDirectory, jsonFileName),
          ),
          htmlPath: relative(
            process.cwd(),
            resolve(outputDirectory, htmlFileName),
          ),
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
        try {
          await chrome.kill();
        } catch {
          // Chrome sudah berhenti.
        }
      }
    }

    resultsByUrl.set(url, reports);
  }

  checkAssertions(resultsByUrl);

  return manifest;
}

async function pathExists(pathname) {
  try {
    await stat(pathname);
    return true;
  } catch (error) {
    if (error?.code === "ENOENT") {
      return false;
    }

    throw error;
  }
}

async function writeJson(pathname, value) {
  await mkdir(dirname(pathname), { recursive: true });
  const temporaryPath = `${pathname}.${process.pid}.tmp`;

  await writeFile(
    temporaryPath,
    `${JSON.stringify(value, null, 2)}\n`,
    "utf8",
  );
  await rm(pathname, { force: true });
  await rename(temporaryPath, pathname);
}

async function captureFile(pathname) {
  try {
    return {
      exists: true,
      content: await readFile(pathname),
    };
  } catch (error) {
    if (error?.code === "ENOENT") {
      return {
        exists: false,
        content: undefined,
      };
    }

    throw error;
  }
}

async function restoreFile(pathname, snapshot) {
  if (!snapshot.exists) {
    await rm(pathname, { force: true });
    return;
  }

  const current = await captureFile(pathname);

  if (
    !current.exists ||
    !current.content.equals(snapshot.content)
  ) {
    await writeFile(pathname, snapshot.content);
  }
}

async function promoteRunDirectory(runDirectory, runId) {
  const backupDirectory = resolve(
    auditBuildDirectory,
    `lighthouse-previous-${runId}`,
  );
  const hadPreviousOutput = await pathExists(outputDirectory);

  if (hadPreviousOutput) {
    await rename(outputDirectory, backupDirectory);
  }

  try {
    await rename(runDirectory, outputDirectory);
  } catch (error) {
    if (hadPreviousOutput && (await pathExists(backupDirectory))) {
      await rename(backupDirectory, outputDirectory);
    }

    throw error;
  }

  if (hadPreviousOutput) {
    await rm(backupDirectory, { recursive: true, force: true });
  }
}

const startedAt = new Date().toISOString();
const runId = startedAt.replace(/[:.]/g, "-");
const numberOfRuns = getNumberOfRuns();
const nextEnvironmentSnapshot = await captureFile(nextEnvironmentPath);

let server;
let runDirectory;
let baseUrl;

try {
  const port = await findAvailablePort();
  baseUrl = `http://127.0.0.1:${port}`;
  auditEnvironment = {
    ...process.env,
    NETLIFY: "false",
    CONTEXT: "dev",
    NEXT_PUBLIC_SITE_URL: baseUrl,
    SEO_AUDIT_INDEXABLE: "true",
  };

  await mkdir(outputDirectory, { recursive: true });
  await rm(manifestPath, { force: true });
  await writeJson(statusPath, {
    status: "running",
    runId,
    startedAt,
    baseUrl,
    numberOfRuns,
  });

  await runProcess([
    resolve("node_modules/next/dist/bin/next"),
    "build",
    "--webpack",
  ]);

  await mkdir(auditBuildDirectory, { recursive: true });
  runDirectory = await mkdtemp(
    resolve(auditBuildDirectory, "lighthouse-run-"),
  );

  server = startProcess([
    resolve("node_modules/next/dist/bin/next"),
    "start",
    "-H",
    "127.0.0.1",
    "-p",
    String(port),
  ]);
  await waitForServer(server, baseUrl);
  await verifyAuditServer(baseUrl);

  const manifest = await runLighthouseAudits(
    baseUrl,
    runDirectory,
    numberOfRuns,
  );
  const buildId = (
    await readFile(resolve(auditBuildDirectory, "BUILD_ID"), "utf8")
  ).trim();
  const completedAt = new Date().toISOString();

  await writeJson(resolve(runDirectory, "manifest.json"), manifest);
  await writeJson(resolve(runDirectory, "run-status.json"), {
    status: "passed",
    runId,
    startedAt,
    completedAt,
    baseUrl,
    buildId,
    numberOfRuns,
    reportCount: manifest.length,
  });
  await promoteRunDirectory(runDirectory, runId);
  runDirectory = undefined;

  console.log(
    `Lighthouse gate lulus. ${manifest.length} report tersimpan di ${outputDirectory}.`,
  );
} catch (error) {
  await mkdir(outputDirectory, { recursive: true });
  await rm(manifestPath, { force: true });
  await writeJson(statusPath, {
    status: "failed",
    runId,
    startedAt,
    completedAt: new Date().toISOString(),
    baseUrl,
    numberOfRuns,
    error: error instanceof Error ? error.message : String(error),
  });
  throw error;
} finally {
  await stopProcess(server);
  await stopActiveProcesses();

  if (runDirectory) {
    await rm(runDirectory, { recursive: true, force: true });
  }

  await restoreFile(nextEnvironmentPath, nextEnvironmentSnapshot);
}
