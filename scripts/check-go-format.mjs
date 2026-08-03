import { readdirSync, readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const backend = path.join(root, "backend");

function goFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    return entry.isDirectory() ? goFiles(target) : entry.name.endsWith(".go") ? [target] : [];
  });
}

const files = goFiles(backend);
const result = spawnSync("gofmt", ["-l", ...files], {
  cwd: root,
  encoding: "utf8",
  maxBuffer: 10 * 1024 * 1024,
  windowsHide: true,
});
if (result.error || result.status !== 0) {
  process.stderr.write(result.stderr || result.error?.message || "gofmt failed\n");
  process.exit(1);
}
if (result.stdout.trim()) {
  process.stderr.write(`Go files are not formatted:\n${result.stdout}`);
  process.exit(1);
}
// Keep an explicit read so an empty or inaccessible tree cannot pass silently.
if (files.length === 0 || files.some((file) => readFileSync(file).length === 0)) {
  process.stderr.write("Go source tree is empty or contains an empty file.\n");
  process.exit(1);
}
process.stdout.write(`Go format check passed (${files.length} files).\n`);
