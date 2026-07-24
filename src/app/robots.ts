import type { MetadataRoute } from "next";

import {
  absoluteUrl,
  isIndexableProduction,
} from "@/lib/site";

export default function robots(): MetadataRoute.Robots {
  if (!isIndexableProduction) {
    return {
      rules: {
        userAgent: "*",
        disallow: "/",
      },
    };
  }

  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/__forms.html"],
    },
    sitemap: absoluteUrl("/sitemap.xml"),
    host: absoluteUrl("/"),
  };
}
