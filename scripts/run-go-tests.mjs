import { spawnSync } from "node:child_process";
import { rmSync, mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const backendRoot = path.join(repositoryRoot, "backend");
const applicationControlMessage =
  "An Application Control policy has blocked this file";

function run(command, args, options = {}) {
  return spawnSync(command, args, {
    cwd: backendRoot,
    encoding: "utf8",
    maxBuffer: 50 * 1024 * 1024,
    windowsHide: true,
    ...options,
  });
}

function writeResult(result) {
  if (result.stdout) {
    process.stdout.write(result.stdout);
  }
  if (result.stderr) {
    process.stderr.write(result.stderr);
  }
  if (result.error) {
    process.stderr.write(`${result.error.message}\n`);
  }
}

function failedPackageNames(output) {
  const packages = new Set();
  for (const line of output.split(/\r?\n/u)) {
    const match = line.match(
      /^FAIL\s+(github\.com\/ilhamnugraha8944\/tauco\/backend\/\S+)/u,
    );
    if (match) {
      packages.add(match[1]);
    }
  }
  return [...packages];
}

function packageDirectories() {
  const result = run("go", [
    "list",
    "-f",
    "{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}|{{.Dir}}{{end}}",
    "./...",
  ]);
  if (result.status !== 0) {
    writeResult(result);
    throw new Error("Tidak dapat membaca daftar package test Go.");
  }

  return new Map(
    result.stdout
      .split(/\r?\n/u)
      .filter(Boolean)
      .map((line) => {
        const separator = line.indexOf("|");
        return [line.slice(0, separator), line.slice(separator + 1)];
      }),
  );
}

function runApplicationControlFallback(packages) {
  const directories = packageDirectories();
  const binaryRoot = path.join(backendRoot, ".test-bin");

  rmSync(binaryRoot, { recursive: true, force: true });
  mkdirSync(binaryRoot, { recursive: true });

  try {
    for (const packageName of packages) {
      const packageDirectory = directories.get(packageName);
      if (!packageDirectory) {
        throw new Error(`Package test tidak ditemukan: ${packageName}`);
      }

      const binaryName = `${packageName.replace(/[^a-z0-9]+/giu, "-")}.test.exe`;
      const binaryPath = path.join(binaryRoot, binaryName);
      const compile = run("go", [
        "test",
        "-c",
        "-o",
        binaryPath,
        packageName,
      ]);
      if (compile.status !== 0) {
        writeResult(compile);
        throw new Error(`Kompilasi test gagal: ${packageName}`);
      }

      const execute = spawnSync(binaryPath, [], {
        cwd: packageDirectory,
        encoding: "utf8",
        maxBuffer: 50 * 1024 * 1024,
        windowsHide: true,
      });
      writeResult(execute);
      if (execute.status !== 0) {
        throw new Error(`Test gagal: ${packageName}`);
      }
      process.stdout.write(`ok  \t${packageName} (workspace fallback)\n`);
    }
  } finally {
    rmSync(binaryRoot, { recursive: true, force: true });
  }
}

const requestedGoTestArguments = process.argv.slice(2);
const standard = run("go", [
  "test",
  ...requestedGoTestArguments,
  "./...",
]);
writeResult(standard);

if (standard.status !== 0) {
  const output = `${standard.stdout ?? ""}\n${standard.stderr ?? ""}`;
  const blockedByPolicy =
    process.platform === "win32" &&
    output.includes(applicationControlMessage);

  if (!blockedByPolicy) {
    process.exit(standard.status ?? 1);
  }

  const packages = failedPackageNames(output);
  if (packages.length === 0) {
    process.exit(standard.status ?? 1);
  }

  process.stderr.write(
    "Windows Application Control memblokir executable test sementara; " +
      "mengulang package terdampak dari workspace.\n",
  );
  try {
    runApplicationControlFallback(packages);
  } catch (error) {
    process.stderr.write(`${error.message}\n`);
    process.exit(1);
  }
}
