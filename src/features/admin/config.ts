export function isAdminCMSEnabled(): boolean {
  if (process.env.ADMIN_CMS_ENABLED !== "true") {
    return false;
  }

  if (isRemoteDeployment() && process.env.ADMIN_REMOTE_ENABLED !== "true") {
    return false;
  }

  return true;
}

export function getAdminAPIOrigin(): URL {
  const raw = process.env.ADMIN_API_ORIGIN?.trim() ?? "";

  if (!raw) {
    throw new Error("ADMIN_API_ORIGIN wajib diisi ketika Admin CMS aktif.");
  }

  const url = new URL(raw);

  if (
    !["http:", "https:"].includes(url.protocol) ||
    url.username ||
    url.password ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  ) {
    throw new Error("ADMIN_API_ORIGIN wajib berupa origin HTTP(S) tanpa path.");
  }

  if (isRemoteDeployment() && url.protocol !== "https:") {
    throw new Error("ADMIN_API_ORIGIN wajib memakai HTTPS pada remote deployment.");
  }

  return new URL(url.origin);
}

function isRemoteDeployment(): boolean {
  return (
    process.env.APP_ENV === "staging" ||
    process.env.APP_ENV === "production" ||
    ["production", "deploy-preview", "branch-deploy"].includes(process.env.CONTEXT ?? "")
  );
}

export function getAdminBFFSecret(): string {
  const secret = process.env.ADMIN_BFF_SHARED_SECRET ?? "";

  if (secret !== secret.trim() || new TextEncoder().encode(secret).byteLength < 32) {
    throw new Error("ADMIN_BFF_SHARED_SECRET wajib berisi minimal 32 byte.");
  }

  if (isRemoteDeployment() && /change-me|example|local-/iu.test(secret)) {
    throw new Error("ADMIN_BFF_SHARED_SECRET remote deployment tidak boleh memakai nilai contoh.");
  }

  return secret;
}

export function isLocalAdminAPIOrigin(): boolean {
  if (isRemoteDeployment()) {
    return false;
  }

  return ["localhost", "127.0.0.1", "[::1]"].includes(getAdminAPIOrigin().hostname);
}
