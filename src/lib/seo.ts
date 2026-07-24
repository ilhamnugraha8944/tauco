import type { Metadata } from "next";

import {
  seoMetadataSchema,
  type SeoMetadata,
} from "@/features/content";

import {
  absoluteUrl,
  isIndexableProduction,
  siteConfig,
  siteUrl,
} from "./site";

export function createMetadata(content: SeoMetadata): Metadata {
  return {
    metadataBase: siteUrl,
    title: content.title,
    description: content.description,
    alternates: {
      canonical: content.canonicalPath,
    },
    openGraph: {
      type: "website",
      locale: siteConfig.locale,
      siteName: siteConfig.name,
      title: content.title,
      description: content.description,
      url: absoluteUrl(content.canonicalPath),
      images: [
        {
          url: absoluteUrl(content.openGraphImage.src),
          alt: content.openGraphImage.alt,
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title: content.title,
      description: content.description,
      images: [absoluteUrl(content.openGraphImage.src)],
    },
    robots: isIndexableProduction
      ? {
          index: true,
          follow: true,
          googleBot: {
            index: true,
            follow: true,
            "max-image-preview": "large",
            "max-snippet": -1,
            "max-video-preview": -1,
          },
        }
      : {
          index: false,
          follow: false,
          nocache: true,
        },
  };
}

export const staticPageMetadata = {
  contact: seoMetadataSchema.parse({
    title: "Kontak Tauco Cap Badak | Informasi Produk",
    description:
      "Kirim pertanyaan mengenai Tauco Cap Badak, informasi produk, atau peluang kerja sama melalui formulir kontak kami.",
    canonicalPath: "/kontak",
    openGraphImage: {
      src: "/images/tauco-hero-provisional.webp",
      alt: "Semangkuk tauco dengan kedelai dan tumisan sayuran",
    },
  }),
  privacy: seoMetadataSchema.parse({
    title: "Kebijakan Privasi | Tauco Cap Badak",
    description:
      "Pelajari jenis data yang dikumpulkan melalui formulir kontak Tauco Cap Badak, tujuan penggunaan, dan masa penyimpanannya.",
    canonicalPath: "/kebijakan-privasi",
    openGraphImage: {
      src: "/images/tauco-hero-provisional.webp",
      alt: "Semangkuk tauco dengan kedelai dan tumisan sayuran",
    },
  }),
} as const;
