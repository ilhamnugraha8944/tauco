import {
  resolveSiteEnvironment,
} from "./site-origin";

export const siteEnvironment = resolveSiteEnvironment(process.env);
export const siteUrl = siteEnvironment.siteUrl;
export const isIndexableProduction = siteEnvironment.isIndexable;

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
