import { spawn } from "node:child_process";
import { once } from "node:events";
import { resolve } from "node:path";

const externalBaseUrl = process.env.PLAYWRIGHT_BASE_URL;
const baseUrl = externalBaseUrl ?? "http://localhost:3000";
const childEnvironment = {
  ...process.env,
  NEXT_PUBLIC_SITE_URL:
    process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000",
};

function startProcess(args, environment) {
  return spawn(process.execPath, args, {
    cwd: process.cwd(),
    env: environment,
    stdio: "inherit",
    windowsHide: true,
  });
}

async function waitForServer(url, server) {
  for (let attempt = 0; attempt < 80; attempt += 1) {
    if (server.exitCode !== null) {
      throw new Error(
        `Next.js berhenti sebelum siap dengan exit code ${server.exitCode}.`,
      );
    }

    try {
      const response = await fetch(url);

      if (response.ok) {
        return;
      }
    } catch {
      // Server masih memulai.
    }

    await new Promise((resolveDelay) => setTimeout(resolveDelay, 250));
  }

  throw new Error(`Server local tidak siap pada ${url}.`);
}

async function stopServer(server) {
  if (
    !server ||
    server.exitCode !== null ||
    server.signalCode !== null
  ) {
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

let server;

try {
  if (!externalBaseUrl) {
    server = startProcess(
      [
        resolve("node_modules/next/dist/bin/next"),
        "start",
        "-H",
        "localhost",
        "-p",
        "3000",
      ],
      childEnvironment,
    );
    await waitForServer(baseUrl, server);
  }

  const playwright = startProcess(
    [
      resolve("node_modules/@playwright/test/cli.js"),
      "test",
      ...process.argv.slice(2),
    ],
    {
      ...childEnvironment,
      PLAYWRIGHT_BASE_URL: baseUrl,
    },
  );
  const [exitCode] = await once(playwright, "exit");
  process.exitCode = typeof exitCode === "number" ? exitCode : 1;
} finally {
  await stopServer(server);
}
