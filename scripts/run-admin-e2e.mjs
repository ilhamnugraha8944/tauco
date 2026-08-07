import { spawn, spawnSync } from "node:child_process";
import { once } from "node:events";
import { resolve } from "node:path";

const environment = {
  ...process.env,
  NEXT_PUBLIC_SITE_URL: "http://localhost:3100",
  ADMIN_CMS_ENABLED: "true",
  ADMIN_API_ORIGIN: "http://127.0.0.1:18081",
  ADMIN_BFF_SHARED_SECRET: "tauco-admin-e2e-bff-secret-32-bytes-minimum",
};

function run(args, overrides = {}) {
  return spawn(process.execPath, args, {
    cwd: process.cwd(), env: { ...environment, ...overrides }, stdio: "inherit", windowsHide: true,
  });
}

async function ready(url, processHandle) {
  for (let attempt = 0; attempt < 120; attempt += 1) {
    if (processHandle.exitCode !== null) throw new Error(`Process berhenti sebelum ${url} siap.`);
    try { if ((await fetch(url)).status < 500) return; } catch {}
    await new Promise((resolveDelay) => setTimeout(resolveDelay, 250));
  }
  throw new Error(`${url} belum siap.`);
}

async function stop(processHandle) {
  if (!processHandle || processHandle.exitCode !== null) return;
  processHandle.kill("SIGTERM");
  await Promise.race([once(processHandle, "exit"), new Promise((resolveDelay) => setTimeout(resolveDelay, 3000))]);
  if (processHandle.exitCode === null) processHandle.kill("SIGKILL");
}

const build = spawnSync(process.execPath, [resolve("node_modules/next/dist/bin/next"), "build", "--webpack"], {
  cwd: process.cwd(), env: environment, stdio: "inherit", windowsHide: true,
});
if (build.status !== 0) process.exit(build.status ?? 1);

let fixture;
let next;
let remoteNext;
try {
  fixture = run([resolve("scripts/admin-api-fixture.mjs")]);
  next = run([resolve("node_modules/next/dist/bin/next"), "start", "-H", "localhost", "-p", "3100"]);
  remoteNext = run(
    [resolve("node_modules/next/dist/bin/next"), "start", "-H", "localhost", "-p", "3101"],
    { ADMIN_API_ORIGIN: "http://127.0.0.2:18081" },
  );
  await Promise.all([
    ready("http://127.0.0.1:18081", fixture),
    ready("http://localhost:3100/admin/login", next),
    ready("http://localhost:3101/admin/login", remoteNext),
  ]);
  const playwright = run([resolve("node_modules/@playwright/test/cli.js"), "test", "--config=playwright.admin.config.ts"]);
  const [code] = await once(playwright, "exit");
  process.exitCode = typeof code === "number" ? code : 1;
} finally {
  await stop(remoteNext);
  await stop(next);
  await stop(fixture);
}
