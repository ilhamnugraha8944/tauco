import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  testMatch: "phase1c-admin.spec.ts",
  outputDir: "test-results/admin",
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:3100",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium-desktop", grepInvert: /remote-origin BFF/, use: { ...devices["Desktop Chrome"] } },
    { name: "mobile-chrome", grepInvert: /remote-origin BFF/, use: { ...devices["Pixel 7"] } },
    {
      name: "remote-bff",
      grep: /remote-origin BFF/,
      use: { ...devices["Desktop Chrome"], baseURL: "http://localhost:3101" },
    },
  ],
});
