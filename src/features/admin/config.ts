export function isAdminCMSEnabled(): boolean {
  return process.env.ADMIN_CMS_ENABLED === "true";
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

  return new URL(url.origin);
}
