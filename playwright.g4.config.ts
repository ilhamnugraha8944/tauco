import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL;
const outputDir = process.env.PLAYWRIGHT_OUTPUT_DIR ?? "test-results";
const reportFolder =
  process.env.PLAYWRIGHT_REPORT_DIR ?? "playwright-report";

if (!baseURL) {
  throw new Error(
    "PLAYWRIGHT_BASE_URL wajib diisi untuk preview QA G4.",
  );
}

const previewUrl = new URL(baseURL);

if (
  previewUrl.protocol !== "https:" ||
  previewUrl.pathname !== "/" ||
  previewUrl.search ||
  previewUrl.hash ||
  !previewUrl.hostname.endsWith(".netlify.app") ||
  !previewUrl.hostname.includes("--")
) {
  throw new Error(
    "Preview QA G4 hanya boleh dijalankan terhadap origin Deploy Preview Netlify.",
  );
}

export default defineConfig({
  testDir: "./tests/e2e",
  outputDir,
  fullyParallel: true,
  forbidOnly: true,
  retries: 1,
  workers: 6,
  reporter: [
    ["list"],
    ["html", { outputFolder: reportFolder, open: "never" }],
  ],
  use: {
    baseURL,
    screenshot: "only-on-failure",
    trace: "on-first-retry",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium-desktop",
      use: {
        ...devices["Desktop Chrome"],
      },
    },
  ],
});
