import {
  isNetlifyProductionEnvironment,
  isSeoAuditEnvironment,
  parseSiteOrigin,
} from "./site-origin";

const LOCAL_SITE_URL = "http://localhost:3000";

const configuredSiteUrl = process.env.NEXT_PUBLIC_SITE_URL?.trim();
const isNetlifyProduction =
  isNetlifyProductionEnvironment(process.env);
const isSeoAudit = isSeoAuditEnvironment(process.env);

if (isNetlifyProduction && !configuredSiteUrl) {
  throw new Error(
    "NEXT_PUBLIC_SITE_URL wajib diisi sebelum production deploy agar canonical tidak salah.",
  );
}

export const siteUrl = parseSiteOrigin(
  configuredSiteUrl || LOCAL_SITE_URL,
  { requirePublicHttps: isNetlifyProduction },
);
export const isIndexableProduction =
  isNetlifyProduction || isSeoAudit;

export const siteConfig = {
  name: "Tauco Cap Badak",
  fullName: "Tauco Cap Badak Cianjur",
  description:
    "Informasi produk Tauco Cap Badak dari Cianjur dan panduan mengenal tauco.",
  locale: "id_ID",
  language: "id-ID",
} as const;

export function absoluteUrl(pathname = "/"): string {
  return new URL(pathname, siteUrl).toString();
}
