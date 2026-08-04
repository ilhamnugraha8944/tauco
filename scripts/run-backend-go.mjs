import { mkdirSync } from "node:fs";
import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const cache = path.join(root, "backend", ".go-build-cache");
mkdirSync(cache, { recursive: true });
const result = spawnSync("go", ["-C", "backend", ...process.argv.slice(2)], {
  cwd: root,
  encoding: "utf8",
  env: { ...process.env, GOCACHE: cache },
  maxBuffer: 50 * 1024 * 1024,
  windowsHide: true,
});
if (result.stdout) process.stdout.write(result.stdout);
if (result.stderr) process.stderr.write(result.stderr);
if (result.error) process.stderr.write(`${result.error.message}\n`);
process.exit(result.status ?? 1);
