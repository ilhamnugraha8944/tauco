import { defineConfig, devices } from "@playwright/test";

const productionOrigin =
  process.env.PLAYWRIGHT_BASE_URL?.trim() ??
  "https://tauco-cap-badak.netlify.app";
const productionUrl = new URL(productionOrigin);
const outputDir =
  process.env.PLAYWRIGHT_OUTPUT_DIR ?? "test-results/production";
const reportFolder =
  process.env.PLAYWRIGHT_REPORT_DIR ?? "playwright-report/production";

if (
  productionUrl.origin !== "https://tauco-cap-badak.netlify.app" ||
  productionUrl.pathname !== "/" ||
  productionUrl.search ||
  productionUrl.hash ||
  productionUrl.hostname.includes("--")
) {
  throw new Error(
    "Production QA hanya boleh dijalankan terhadap https://tauco-cap-badak.netlify.app.",
  );
}

export default defineConfig({
  testDir: "./tests/e2e",
  outputDir,
  fullyParallel: true,
  forbidOnly: true,
  retries: 1,
  workers: 5,
  reporter: [
    ["list"],
    ["html", { outputFolder: reportFolder, open: "never" }],
  ],
  use: {
    baseURL: productionUrl.origin,
    screenshot: "only-on-failure",
    trace: "on-first-retry",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "production-edge",
      use: {
        ...devices["Desktop Chrome"],
        channel: "msedge",
      },
    },
  ],
});
