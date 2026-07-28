import { mkdir, rm, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

import { chromium } from "@playwright/test";
import * as chromeLauncher from "chrome-launcher";
import lighthouse from "lighthouse";

const productionOrigin = "https://tauco-cap-badak.netlify.app";
const routes = [
  "/",
  "/tauco",
  "/produk/tauco-cap-badak",
  "/kontak",
];
const numberOfRuns = 3;
const outputDirectory = resolve(".lighthouseci-production");

function median(values) {
  const sorted = [...values].sort((left, right) => left - right);
  return sorted[Math.floor(sorted.length / 2)];
}

function slug(pathname) {
  return pathname === "/"
    ? "home"
    : pathname.replace(/^\/|\/$/g, "").replaceAll("/", "-");
}

await rm(outputDirectory, { recursive: true, force: true });
await mkdir(outputDirectory, { recursive: true });

const manifest = [];
const failures = [];

for (const pathname of routes) {
  const url = new URL(pathname, productionOrigin).toString();
  const reports = [];

  console.log(`Lighthouse production ${numberOfRuns}x: ${url}`);

  for (let run = 1; run <= numberOfRuns; run += 1) {
    const chrome = await chromeLauncher.launch({
      chromePath: chromium.executablePath(),
      chromeFlags: [
        "--headless",
        "--no-sandbox",
        "--disable-gpu",
      ],
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
      const reportName = `${slug(pathname)}-run-${run}`;
      const performance = result.lhr.categories.performance.score;
      const accessibility = result.lhr.categories.accessibility.score;
      const bestPractices = result.lhr.categories["best-practices"].score;
      const seo = result.lhr.categories.seo.score;
      const lcp =
        result.lhr.audits["largest-contentful-paint"].numericValue;
      const cls =
        result.lhr.audits["cumulative-layout-shift"].numericValue;

      await Promise.all([
        writeFile(
          resolve(outputDirectory, `${reportName}.report.json`),
          jsonReport,
          "utf8",
        ),
        writeFile(
          resolve(outputDirectory, `${reportName}.report.html`),
          htmlReport,
          "utf8",
        ),
      ]);

      reports.push({
        accessibility,
        bestPractices,
        cls,
        lcp,
        performance,
        seo,
      });
      manifest.push({
        path: pathname,
        run,
        url,
        performance,
        accessibility,
        bestPractices,
        seo,
        lcp,
        cls,
      });
      console.log(
        `  Run ${run}: Performance ${Math.round(
          performance * 100,
        )}, A11y ${Math.round(accessibility * 100)}, SEO ${Math.round(
          seo * 100,
        )}, LCP ${Math.round(lcp)} ms, CLS ${cls.toFixed(3)}.`,
      );
    } finally {
      try {
        await chrome.kill();
      } catch {
        // Chrome sudah berhenti.
      }
    }
  }

  const medians = {
    performance: median(reports.map((report) => report.performance)),
    accessibility: median(
      reports.map((report) => report.accessibility),
    ),
    seo: median(reports.map((report) => report.seo)),
    lcp: median(reports.map((report) => report.lcp)),
    cls: median(reports.map((report) => report.cls)),
  };

  if (medians.performance < 0.9) {
    failures.push(
      `${pathname} performance ${medians.performance} harus >= 0.9`,
    );
  }
  if (medians.accessibility < 0.9) {
    failures.push(
      `${pathname} accessibility ${medians.accessibility} harus >= 0.9`,
    );
  }
  if (medians.seo < 0.95) {
    failures.push(`${pathname} SEO ${medians.seo} harus >= 0.95`);
  }
  if (medians.lcp > 2500) {
    failures.push(`${pathname} LCP ${medians.lcp} ms harus <= 2500 ms`);
  }
  if (medians.cls > 0.1) {
    failures.push(`${pathname} CLS ${medians.cls} harus <= 0.1`);
  }
}

await writeFile(
  resolve(outputDirectory, "manifest.json"),
  `${JSON.stringify(manifest, null, 2)}\n`,
  "utf8",
);

if (failures.length > 0) {
  throw new Error(
    `Lighthouse production gate gagal:\n${failures.join("\n")}`,
  );
}

console.log(
  `Lighthouse production lulus; ${manifest.length} report tersimpan di ${outputDirectory}.`,
);
