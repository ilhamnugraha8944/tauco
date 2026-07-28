import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const backendRoot = path.join(repositoryRoot, "backend");
const configPath = path.join(backendRoot, "openapi", "oapi-codegen.yaml");
const generatedPath = path.join(
  backendRoot,
  "internal",
  "delivery",
  "api",
  "api.gen.go",
);
const temporaryDirectory = mkdtempSync(
  path.join(tmpdir(), "tauco-openapi-check-"),
);

try {
  const temporaryGeneratedPath = path.join(
    temporaryDirectory,
    "api.gen.go",
  );
  const temporaryConfigPath = path.join(
    temporaryDirectory,
    "oapi-codegen.yaml",
  );
  const sourceConfig = readFileSync(configPath, "utf8");
  const outputPattern = /^output:\s*.+$/mu;
  if (!outputPattern.test(sourceConfig)) {
    throw new Error("Konfigurasi oapi-codegen tidak memiliki output.");
  }

  const temporaryConfig = sourceConfig.replace(
    outputPattern,
    `output: ${JSON.stringify(temporaryGeneratedPath.replaceAll("\\", "/"))}`,
  );
  writeFileSync(temporaryConfigPath, temporaryConfig, "utf8");

  const generation = spawnSync(
    "go",
    [
      "-C",
      "backend",
      "tool",
      "oapi-codegen",
      "--config",
      temporaryConfigPath,
      "openapi/openapi.yaml",
    ],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
      maxBuffer: 50 * 1024 * 1024,
      windowsHide: true,
    },
  );
  if (generation.stdout) {
    process.stdout.write(generation.stdout);
  }
  if (generation.stderr) {
    process.stderr.write(generation.stderr);
  }
  if (generation.error || generation.status !== 0) {
    throw new Error(
      generation.error?.message ?? "Regenerasi OpenAPI gagal.",
    );
  }

  const normalize = (value) => value.replaceAll("\r\n", "\n");
  const committed = normalize(readFileSync(generatedPath, "utf8"));
  const regenerated = normalize(
    readFileSync(temporaryGeneratedPath, "utf8"),
  );
  if (committed !== regenerated) {
    throw new Error(
      "Generated OpenAPI drift terdeteksi. Jalankan " +
        "`npm.cmd run backend:generate` dan review hasilnya.",
    );
  }

  process.stdout.write("OpenAPI generated code is reproducible.\n");
} catch (error) {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
} finally {
  rmSync(temporaryDirectory, { recursive: true, force: true });
}
