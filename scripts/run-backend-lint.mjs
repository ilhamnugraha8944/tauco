import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const result = spawnSync("docker", [
  "run", "--rm", "-v", `${root}:/repo`, "-w", "/repo/backend",
  "golangci/golangci-lint:v2.11.4", "golangci-lint", "run", "./...",
], { cwd: root, encoding: "utf8", maxBuffer: 50 * 1024 * 1024, windowsHide: true });
if (result.stdout) process.stdout.write(result.stdout);
if (result.stderr) process.stderr.write(result.stderr);
if (result.error) process.stderr.write(`${result.error.message}\n`);
process.exit(result.status ?? 1);
