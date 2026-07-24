import type { MetadataRoute } from "next";

import { localContentSource } from "@/features/content";
import { absoluteUrl } from "@/lib/site";

const publicationDate = new Date("2026-07-24T00:00:00+07:00");

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const catalog = await localContentSource.listProducts();

  const staticRoutes = [
    { path: "/", priority: 1, changeFrequency: "weekly" as const },
    { path: "/tauco", priority: 0.9, changeFrequency: "monthly" as const },
    {
      path: "/tentang-kami",
      priority: 0.7,
      changeFrequency: "monthly" as const,
    },
    { path: "/produk", priority: 0.8, changeFrequency: "weekly" as const },
    { path: "/kontak", priority: 0.6, changeFrequency: "yearly" as const },
    {
      path: "/kebijakan-privasi",
      priority: 0.3,
      changeFrequency: "yearly" as const,
    },
  ];

  return [
    ...staticRoutes.map((route) => ({
      url: absoluteUrl(route.path),
      lastModified: publicationDate,
      changeFrequency: route.changeFrequency,
      priority: route.priority,
    })),
    ...catalog.products.map((product) => ({
      url: absoluteUrl(`/produk/${product.slug}`),
      lastModified: publicationDate,
      changeFrequency: "monthly" as const,
      priority: 0.8,
    })),
  ];
}
