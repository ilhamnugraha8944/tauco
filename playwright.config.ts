import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3000";
const outputDir = process.env.PLAYWRIGHT_OUTPUT_DIR ?? "test-results";
const reportFolder =
  process.env.PLAYWRIGHT_REPORT_DIR ?? "playwright-report";

export default defineConfig({
  testDir: "./tests/e2e",
  testIgnore: ["**/g4-*.spec.ts", "**/g6-*.spec.ts", "**/g7-*.spec.ts"],
  outputDir,
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ["list"],
    ["html", { outputFolder: reportFolder, open: "never" }],
  ],
  use: {
    baseURL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium-desktop",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "mobile-chrome",
      use: { ...devices["Pixel 7"] },
    },
  ],
});
