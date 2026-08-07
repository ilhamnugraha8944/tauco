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
  const required = [
    "DATABASE_URL",
    "CURSOR_HMAC_SECRET",
    "RATE_LIMIT_HMAC_SECRET",
    "METRICS_BEARER_TOKEN",
    "REDIS_URL",
    "CORS_ALLOWED_ORIGINS",
  ];
  if (process.env.CONTACT_API_ENABLED === "true") required.push("CONTACT_HMAC_SECRET");
  if (process.env.ADMIN_REMOTE_ENABLED === "true") {
    required.push("ADMIN_DATABASE_URL", "ADMIN_ALLOWED_ORIGINS", "ADMIN_BFF_SHARED_SECRET");
  }
  const missing = required.filter(
    (name) => !process.env[name]?.trim(),
  );
  if (missing.length > 0) {
    fail(
      `${missing.join(", ")} belum tersedia. Salin backend/.env.example ` +
        "menjadi backend/.env atau set environment variable secara eksplisit.",
    );
  }
}

function run(command, args, environment = process.env, interactive = false) {
  mkdirSync(goCachePath, { recursive: true });
  const result = spawnSync(command, args, {
    cwd: repositoryRoot,
    env: { ...environment, GOCACHE: goCachePath },
    ...(interactive
      ? { stdio: "inherit" }
      : { encoding: "utf8", maxBuffer: 50 * 1024 * 1024 }),
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
  case "admin": {
    if (
      !["keygen", "keygen-ed25519"].includes(operationArguments[0]) &&
      !process.env.ADMIN_DATABASE_URL?.trim()
    ) {
      fail("ADMIN_DATABASE_URL wajib tersedia untuk admin CLI.");
    }
    run("go", ["-C", "backend", "run", "./cmd/admin", ...operationArguments], process.env, true);
    break;
  }
  case "worker": {
    requireRuntimeConfiguration();
    const workerModes = new Set(["--check", "--once", "--cleanup-media"]);
    if (operationArguments.length > 1 || (operationArguments.length === 1 && !workerModes.has(operationArguments[0]))) {
      fail("worker hanya menerima salah satu: --check, --once, atau --cleanup-media.");
    }
    const workerRequired = [];
    if (process.env.CONTACT_API_ENABLED === "true") {
      workerRequired.push("SMTP_HOST", "SMTP_PORT", "SMTP_FROM", "SMTP_TO");
    }
    if ((process.env.MEDIA_STORAGE_DRIVER ?? "local") === "s3") {
      workerRequired.push("MEDIA_S3_ENDPOINT", "MEDIA_S3_REGION", "MEDIA_S3_BUCKET", "MEDIA_S3_ACCESS_KEY_ID", "MEDIA_S3_SECRET_ACCESS_KEY");
    } else {
      workerRequired.push("MEDIA_LOCAL_ROOT");
    }
    for (const name of workerRequired) {
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
  case "provision": {
    if (!process.env.PROVISION_DATABASE_URL?.trim()) fail("PROVISION_DATABASE_URL wajib tersedia hanya selama provisioning.");
    if (operationArguments.length !== 0) fail("provision tidak menerima argument tambahan.");
    run("go", ["-C", "backend", "run", "./cmd/provision"], process.env, true);
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
        "<api|admin args...|worker [--check|--once|--cleanup-media]|media-import args...|ops args...|loadcheck|migrate args...|seed-phase1a|integration>",
    );
}
