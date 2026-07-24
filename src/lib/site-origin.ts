type OriginOptions = {
  requirePublicHttps?: boolean;
};

export function isNetlifyProductionEnvironment(
  environment: NodeJS.ProcessEnv,
): boolean {
  return (
    environment.NETLIFY === "true" &&
    environment.CONTEXT === "production"
  );
}

export function isSeoAuditEnvironment(
  environment: NodeJS.ProcessEnv,
): boolean {
  return environment.SEO_AUDIT_INDEXABLE === "true";
}

export function parseSiteOrigin(
  value: string,
  options: OriginOptions = {},
): URL {
  let url: URL;

  try {
    url = new URL(value);
  } catch {
    throw new Error("NEXT_PUBLIC_SITE_URL harus berupa URL absolut.");
  }

  if (!["http:", "https:"].includes(url.protocol)) {
    throw new Error("NEXT_PUBLIC_SITE_URL harus menggunakan HTTP atau HTTPS.");
  }

  if (
    url.username ||
    url.password ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  ) {
    throw new Error(
      "NEXT_PUBLIC_SITE_URL harus berupa origin tanpa path, query, hash, atau kredensial.",
    );
  }

  if (
    options.requirePublicHttps &&
    (url.protocol !== "https:" ||
      url.hostname === "localhost" ||
      url.hostname.endsWith(".example"))
  ) {
    throw new Error(
      "NEXT_PUBLIC_SITE_URL wajib berupa origin HTTPS publik untuk production.",
    );
  }

  return new URL(url.origin);
}
