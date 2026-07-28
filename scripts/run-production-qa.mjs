import { spawnSync } from "node:child_process";
import { resolve } from "node:path";

const productionOrigin = "https://tauco-cap-badak.netlify.app";
const playwrightCli = resolve(
  "node_modules/@playwright/test/cli.js",
);
const result = spawnSync(
  process.execPath,
  [playwrightCli, "test", ...process.argv.slice(2)],
  {
    cwd: process.cwd(),
    env: {
      ...process.env,
      PLAYWRIGHT_BASE_URL: productionOrigin,
    },
    stdio: "inherit",
    windowsHide: true,
  },
);

if (result.error) {
  throw result.error;
}

process.exitCode =
  typeof result.status === "number" ? result.status : 1;
