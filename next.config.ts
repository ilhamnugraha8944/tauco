import type { NextConfig } from "next";

import {
  isNetlifyProductionEnvironment,
  isSeoAuditEnvironment,
  parseSiteOrigin,
} from "./src/lib/site-origin";

const isNetlifyProduction =
  isNetlifyProductionEnvironment(process.env);
const isSeoAudit = isSeoAuditEnvironment(process.env);
const siteUrl = process.env.NEXT_PUBLIC_SITE_URL;

if (isNetlifyProduction) {
  try {
    parseSiteOrigin(siteUrl ?? "", { requirePublicHttps: true });
  } catch {
    throw new Error(
      "NEXT_PUBLIC_SITE_URL wajib berupa origin HTTPS publik untuk deploy production Netlify.",
    );
  }
}

const securityHeaders = [
  {
    key: "X-Content-Type-Options",
    value: "nosniff",
  },
  {
    key: "X-Frame-Options",
    value: "SAMEORIGIN",
  },
  {
    key: "Referrer-Policy",
    value: "strict-origin-when-cross-origin",
  },
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=()",
  },
];

const nextConfig: NextConfig = {
  distDir: isSeoAudit ? ".next-lighthouse" : ".next",
  poweredByHeader: false,
  reactStrictMode: true,
  images: {
    formats: ["image/avif", "image/webp"],
    qualities: [50, 75],
  },
  experimental: {
    inlineCss: true,
    optimizePackageImports: ["@phosphor-icons/react"],
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;
