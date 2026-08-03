import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const result = spawnSync("docker", [
  "run", "--rm",
	"-v", `${root}:/repo`,
  "-v", "tauco_go_mod:/go/pkg/mod",
  "-v", "tauco_go_build:/root/.cache/go-build",
	"-w", "/repo/backend",
  "golang:1.26.5-bookworm",
  "go", "test", "-race", "./...",
], { cwd: root, encoding: "utf8", maxBuffer: 50 * 1024 * 1024, windowsHide: true });
if (result.stdout) process.stdout.write(result.stdout);
if (result.stderr) process.stderr.write(result.stderr);
if (result.error) process.stderr.write(`${result.error.message}\n`);
process.exit(result.status ?? 1);
