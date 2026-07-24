import { spawn } from "node:child_process";
import { once } from "node:events";
import { mkdir } from "node:fs/promises";
import { resolve } from "node:path";

import { chromium } from "@playwright/test";

const baseUrl = "http://localhost:3000";
const outputDirectory = resolve("artifacts/walkthrough");

const server = spawn(
  process.execPath,
  [
    resolve("node_modules/next/dist/bin/next"),
    "start",
    "-H",
    "localhost",
    "-p",
    "3000",
  ],
  {
    cwd: process.cwd(),
    env: {
      ...process.env,
      NEXT_PUBLIC_SITE_URL:
        process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000",
    },
    stdio: "inherit",
    windowsHide: true,
  },
);

async function waitForServer() {
  for (let attempt = 0; attempt < 80; attempt += 1) {
    if (server.exitCode !== null) {
      throw new Error("Server Next.js berhenti sebelum siap.");
    }

    try {
      const response = await fetch(baseUrl);

      if (response.ok) {
        return;
      }
    } catch {
      // Server masih memulai.
    }

    await new Promise((resolveDelay) => setTimeout(resolveDelay, 250));
  }

  throw new Error("Server local tidak siap untuk visual capture.");
}

async function stopServer() {
  if (server.exitCode !== null || server.signalCode !== null) {
    return;
  }

  server.kill("SIGTERM");

  await Promise.race([
    once(server, "exit"),
    new Promise((resolveDelay) => setTimeout(resolveDelay, 3000)),
  ]);

  if (server.exitCode === null && server.signalCode === null) {
    server.kill("SIGKILL");
  }
}

await mkdir(outputDirectory, { recursive: true });

let browser;

try {
  await waitForServer();
  browser = await chromium.launch();

  const captures = [
    {
      path: "/",
      fileName: "home-desktop-light.png",
      viewport: { width: 1440, height: 1000 },
      colorScheme: "light",
      fullPage: true,
    },
    {
      path: "/",
      fileName: "home-mobile-light.png",
      viewport: { width: 390, height: 844 },
      colorScheme: "light",
      fullPage: true,
    },
    {
      path: "/tauco",
      fileName: "tauco-desktop-dark.png",
      viewport: { width: 1440, height: 1000 },
      colorScheme: "dark",
      fullPage: true,
    },
    {
      path: "/tentang-kami",
      fileName: "about-desktop-light.png",
      viewport: { width: 1440, height: 1000 },
      colorScheme: "light",
      fullPage: true,
    },
    {
      path: "/produk/tauco-cap-badak",
      fileName: "product-tablet-light.png",
      viewport: { width: 768, height: 844 },
      colorScheme: "light",
      fullPage: true,
    },
    {
      path: "/kontak?topik=produk",
      fileName: "contact-desktop-light.png",
      viewport: { width: 1440, height: 1000 },
      colorScheme: "light",
      fullPage: true,
    },
    {
      path: "/kontak?topik=produk",
      fileName: "contact-mobile-dark.png",
      viewport: { width: 390, height: 844 },
      colorScheme: "dark",
      fullPage: true,
    },
  ];

  for (const capture of captures) {
    const context = await browser.newContext({
      viewport: capture.viewport,
      colorScheme: capture.colorScheme,
      reducedMotion: "reduce",
    });
    const page = await context.newPage();
    await page.goto(`${baseUrl}${capture.path}`, {
      waitUntil: "networkidle",
    });
    await page.screenshot({
      path: resolve(outputDirectory, capture.fileName),
      fullPage: capture.fullPage,
    });
    await context.close();
  }

  console.log(`Saved ${captures.length} captures to ${outputDirectory}.`);
} finally {
  await browser?.close();
  await stopServer();
}
