import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import process, { loadEnvFile } from "node:process";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const backendEnvPath = path.join(repositoryRoot, "backend", ".env");
const goCachePath = path.join(repositoryRoot, "backend", ".go-build-cache");

if (existsSync(backendEnvPath)) {
  loadEnvFile(backendEnvPath);
}

function fail(message) {
  process.stderr.write(`${message}\n`);
  process.exit(2);
}

function requireMigrationURL() {
  if (!process.env.MIGRATION_DATABASE_URL?.trim()) {
    fail(
      "MIGRATION_DATABASE_URL belum tersedia. Salin backend/.env.example " +
        "menjadi backend/.env atau set environment variable secara eksplisit.",
    );
  }
}

function requireRuntimeConfiguration() {
  const missing = [
    "DATABASE_URL",
    "CURSOR_HMAC_SECRET",
    "CONTACT_HMAC_SECRET",
    "RATE_LIMIT_HMAC_SECRET",
    "METRICS_BEARER_TOKEN",
    "REDIS_URL",
    "CORS_ALLOWED_ORIGINS",
  ].filter(
    (name) => !process.env[name]?.trim(),
  );
  if (missing.length > 0) {
    fail(
      `${missing.join(", ")} belum tersedia. Salin backend/.env.example ` +
        "menjadi backend/.env atau set environment variable secara eksplisit.",
    );
  }
}

function run(command, args, environment = process.env) {
  mkdirSync(goCachePath, { recursive: true });
  const result = spawnSync(command, args, {
    cwd: repositoryRoot,
    encoding: "utf8",
    env: { ...environment, GOCACHE: goCachePath },
    maxBuffer: 50 * 1024 * 1024,
    windowsHide: true,
  });
  if (result.stdout) {
    process.stdout.write(result.stdout);
  }
  if (result.stderr) {
    process.stderr.write(result.stderr);
  }
  if (result.error) {
    process.stderr.write(`${result.error.message}\n`);
  }
  process.exit(result.status ?? 1);
}

const [operation, ...operationArguments] = process.argv.slice(2);

switch (operation) {
  case "api": {
    requireRuntimeConfiguration();
    if (operationArguments.length !== 0) {
      fail("api tidak menerima argument tambahan.");
    }
    run("go", ["-C", "backend", "run", "./cmd/api"]);
    break;
  }
  case "worker": {
    requireRuntimeConfiguration();
    if (operationArguments.length > 1 || (operationArguments.length === 1 && operationArguments[0] !== "--check")) {
      fail("worker hanya menerima argument opsional --check.");
    }
    for (const name of ["SMTP_HOST", "SMTP_PORT", "SMTP_FROM", "SMTP_TO", "MEDIA_LOCAL_ROOT"]) {
      if (!process.env[name]?.trim()) {
        fail(`${name} wajib tersedia untuk worker.`);
      }
    }
    run("go", ["-C", "backend", "run", "./cmd/worker", ...operationArguments]);
    break;
  }
  case "media-import": {
    requireRuntimeConfiguration();
    if (!process.env.MEDIA_LOCAL_ROOT?.trim()) {
      fail("MEDIA_LOCAL_ROOT wajib tersedia untuk media import.");
    }
    run("go", ["-C", "backend", "run", "./cmd/media-import", ...operationArguments]);
    break;
  }
  case "ops": {
    requireRuntimeConfiguration();
    run("go", ["-C", "backend", "run", "./cmd/ops", ...operationArguments]);
    break;
  }
  case "loadcheck": {
    requireRuntimeConfiguration();
    if (operationArguments.length !== 0) fail("loadcheck tidak menerima argument.");
    run("go", ["-C", "backend", "run", "./cmd/loadcheck"]);
    break;
  }
  case "migrate": {
    requireMigrationURL();
    if (operationArguments.length === 0) {
      fail("Perintah migration wajib diisi.");
    }
    run("go", [
      "-C",
      "backend",
      "run",
      "./cmd/migrate",
      ...operationArguments,
    ]);
    break;
  }
  case "seed-phase1a": {
    requireMigrationURL();
    if (operationArguments.length !== 0) {
      fail("seed-phase1a tidak menerima argument tambahan.");
    }
    run("go", [
      "-C",
      "backend",
      "run",
      "./cmd/seed",
      "--content-dir",
      "../content",
    ]);
    break;
  }
  case "integration": {
    const integrationURL =
      process.env.MIGRATION_TEST_DATABASE_URL ??
      "postgres://tauco_app:tauco-local-postgres-only@" +
        "127.0.0.1:5432/tauco?sslmode=disable";
    let integrationHost;
    try {
      integrationHost = new URL(integrationURL).hostname.toLowerCase();
    } catch {
      fail("MIGRATION_TEST_DATABASE_URL bukan URL PostgreSQL yang valid.");
    }
    const isLoopback =
      integrationHost === "127.0.0.1" ||
      integrationHost === "localhost" ||
      integrationHost === "::1";
    if (
      !isLoopback &&
      process.env.MIGRATION_TEST_ALLOW_REMOTE !== "true"
    ) {
      fail(
        "Integration database non-loopback ditolak. Gunakan database " +
          "disposable terisolasi dan set MIGRATION_TEST_ALLOW_REMOTE=true " +
          "secara eksplisit bila benar-benar diperlukan.",
      );
    }
    const environment = {
      ...process.env,
      MIGRATION_TEST_DATABASE_URL: integrationURL,
    };
    run(
      "node",
      ["scripts/run-go-tests.mjs", "-count=1"],
      environment,
    );
    break;
  }
  default:
    fail(
      "usage: run-backend-db.mjs " +
        "<api|worker|media-import args...|ops args...|loadcheck|migrate args...|seed-phase1a|integration>",
    );
}
