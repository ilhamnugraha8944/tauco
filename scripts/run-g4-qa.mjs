import { spawnSync } from "node:child_process";
import { resolve } from "node:path";

const previewOrigin = process.env.PLAYWRIGHT_BASE_URL?.trim();

if (!previewOrigin) {
  throw new Error(
    "PLAYWRIGHT_BASE_URL wajib diisi dengan origin Deploy Preview.",
  );
}

const previewUrl = new URL(previewOrigin);

if (
  previewUrl.protocol !== "https:" ||
  previewUrl.pathname !== "/" ||
  previewUrl.search ||
  previewUrl.hash ||
  !previewUrl.hostname.endsWith(".netlify.app") ||
  !previewUrl.hostname.includes("--")
) {
  throw new Error(
    "qa:g4 hanya boleh dijalankan terhadap origin Deploy Preview Netlify.",
  );
}

const playwrightCli = resolve(
  "node_modules/@playwright/test/cli.js",
);
const accessibilityGrep = [
  "has no WCAG A or AA axe violations",
  "public pages do not overflow",
  "responsive boundaries preserve navigation and hero hierarchy",
  "dark mode keeps every public route axe-clean",
  "dynamic contact states remain axe-clean",
  "focus remains visible and reduced motion shortens transitions",
  "keyboard-only walkthrough reaches every visible control",
  "mobile menu opens and closes with keyboard while retaining focus",
  "shows inline validation errors and focuses the first invalid field",
].join("|");
const suites = [
  {
    args: [
      "--config=playwright.g4.config.ts",
      "tests/e2e/g4-functional.spec.ts",
      "--project=chromium-desktop",
    ],
    name: "functional",
  },
  {
    args: [
      "--config=playwright.g4.config.ts",
      "tests/e2e/g4-seo.spec.ts",
      "--project=chromium-desktop",
    ],
    name: "seo-isolation",
  },
  {
    args: [
      "--config=playwright.g4.config.ts",
      "tests/e2e/g4-form.spec.ts",
      "--project=chromium-desktop",
    ],
    name: "form",
  },
  {
    args: [
      "--config=playwright.g4.config.ts",
      "tests/e2e/quality.spec.ts",
      "tests/e2e/site.spec.ts",
      "--project=chromium-desktop",
      "--grep",
      accessibilityGrep,
    ],
    name: "accessibility",
  },
  {
    args: [
      "--config=playwright.g4-browsers.config.ts",
      "tests/e2e/g4-browser.spec.ts",
    ],
    name: "browser-matrix",
  },
];

for (const suite of suites) {
  console.log(`\nG4 suite: ${suite.name}`);

  const result = spawnSync(
    process.execPath,
    [playwrightCli, "test", ...suite.args],
    {
      cwd: process.cwd(),
      env: {
        ...process.env,
        PLAYWRIGHT_OUTPUT_DIR: resolve(
          "test-results",
          "g4",
          suite.name,
        ),
        PLAYWRIGHT_REPORT_DIR: resolve(
          "playwright-report",
          "g4",
          suite.name,
        ),
      },
      stdio: "inherit",
      windowsHide: true,
    },
  );

  if (result.error) {
    throw result.error;
  }

  if (result.status !== 0) {
    throw new Error(
      `G4 suite ${suite.name} gagal dengan exit code ${result.status}.`,
    );
  }
}

console.log(
  `\nSeluruh automated G4 suite lulus untuk ${previewUrl.origin}.`,
);
