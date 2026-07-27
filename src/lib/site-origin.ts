type OriginOptions = {
  requirePublicHttps?: boolean;
};

type ResolveSiteEnvironmentOptions = {
  configuredSiteUrl?: string;
  localSiteUrl?: string;
};

export type SiteEnvironmentKind =
  | "local"
  | "seo-audit"
  | "netlify-production"
  | "netlify-non-production";

export type ResolvedSiteEnvironment = {
  kind: SiteEnvironmentKind;
  siteUrl: URL;
  netlifyContext?: string;
  isNetlify: boolean;
  isNetlifyProduction: boolean;
  isSeoAudit: boolean;
  isIndexable: boolean;
  requiresPublicHttps: boolean;
};

export const DEFAULT_LOCAL_SITE_URL = "http://localhost:3000";

const reservedPublicHostnameSuffixes = [
  ".example",
  ".internal",
  ".invalid",
  ".local",
  ".localhost",
  ".test",
] as const;
const publicHostnamePattern =
  /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

function normalizeHostname(hostname: string): string {
  return hostname
    .toLowerCase()
    .replace(/^\[/, "")
    .replace(/\]$/, "")
    .replace(/\.$/, "");
}

function isIpLiteral(hostname: string): boolean {
  const normalizedHostname = normalizeHostname(hostname);

  if (normalizedHostname.includes(":")) {
    return true;
  }

  const octets = normalizedHostname.split(".");

  return (
    octets.length === 4 &&
    octets.every(
      (octet) =>
        /^\d{1,3}$/.test(octet) &&
        Number(octet) >= 0 &&
        Number(octet) <= 255,
    )
  );
}

function isLoopbackHostname(hostname: string): boolean {
  const normalizedHostname = normalizeHostname(hostname);

  if (
    normalizedHostname === "localhost" ||
    normalizedHostname === "::1"
  ) {
    return true;
  }

  const octets = normalizedHostname.split(".");

  return (
    octets.length === 4 &&
    octets.every((octet) => /^\d{1,3}$/.test(octet)) &&
    Number(octets[0]) === 127
  );
}

function isPublicHostname(hostname: string): boolean {
  const normalizedHostname = normalizeHostname(hostname);

  if (
    !normalizedHostname.includes(".") ||
    isIpLiteral(normalizedHostname) ||
    !publicHostnamePattern.test(normalizedHostname)
  ) {
    return false;
  }

  return !reservedPublicHostnameSuffixes.some(
    (suffix) =>
      normalizedHostname === suffix.slice(1) ||
      normalizedHostname.endsWith(suffix),
  );
}

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
  return (
    environment.NETLIFY !== "true" &&
    environment.SEO_AUDIT_INDEXABLE === "true"
  );
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
    (url.protocol !== "https:" || !isPublicHostname(url.hostname))
  ) {
    throw new Error(
      "NEXT_PUBLIC_SITE_URL wajib berupa origin HTTPS dengan hostname publik.",
    );
  }

  return new URL(url.origin);
}

export function resolveSiteEnvironment(
  environment: NodeJS.ProcessEnv,
  options: ResolveSiteEnvironmentOptions = {},
): ResolvedSiteEnvironment {
  const isNetlify = environment.NETLIFY === "true";
  const netlifyContext = isNetlify
    ? environment.CONTEXT?.trim() || "unknown"
    : undefined;
  const isNetlifyProduction =
    isNetlifyProductionEnvironment(environment);
  const requiresPublicHttps = isNetlify;
  const configuredSiteUrl = (
    options.configuredSiteUrl ??
    environment.NEXT_PUBLIC_SITE_URL ??
    ""
  ).trim();

  if (requiresPublicHttps && !configuredSiteUrl) {
    throw new Error(
      "NEXT_PUBLIC_SITE_URL wajib diisi untuk setiap deploy Netlify.",
    );
  }

  const siteUrl = parseSiteOrigin(
    configuredSiteUrl ||
      options.localSiteUrl ||
      DEFAULT_LOCAL_SITE_URL,
    { requirePublicHttps: requiresPublicHttps },
  );
  const isSeoAudit =
    isSeoAuditEnvironment(environment) &&
    isLoopbackHostname(siteUrl.hostname);
  const isIndexable = isNetlifyProduction || isSeoAudit;

  let kind: SiteEnvironmentKind = "local";

  if (isNetlifyProduction) {
    kind = "netlify-production";
  } else if (isNetlify) {
    kind = "netlify-non-production";
  } else if (isSeoAudit) {
    kind = "seo-audit";
  }

  return {
    kind,
    siteUrl,
    netlifyContext,
    isNetlify,
    isNetlifyProduction,
    isSeoAudit,
    isIndexable,
    requiresPublicHttps,
  };
}
