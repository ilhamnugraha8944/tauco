import { z } from "zod";

const slugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const internalPathPattern =
  /^\/(?:[a-z0-9]+(?:-[a-z0-9]+)*(?:\/[a-z0-9]+(?:-[a-z0-9]+)*)*)?$/;
const internalHrefPattern =
  /^\/(?:[a-z0-9]+(?:-[a-z0-9]+)*(?:\/[a-z0-9]+(?:-[a-z0-9]+)*)*)?(?:\?[a-z0-9-]+=[a-z0-9-]+(?:&[a-z0-9-]+=[a-z0-9-]+)*)?$/;
const internalImagePattern =
  /^(?:\/images\/[a-z0-9][a-z0-9/-]*\.(?:avif|jpe?g|png|svg|webp)|\/api\/v1\/media\/[0-9a-f-]{36}\/(?:display|variants\/(?:320|640|1280))\.webp)$/;

const shortTextSchema = z.string().trim().min(2).max(120);
const paragraphSchema = z.string().trim().min(20).max(600);
const isoDateSchema = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "Tanggal harus menggunakan format YYYY-MM-DD.")
  .refine((value) => !Number.isNaN(Date.parse(`${value}T00:00:00Z`)), {
    message: "Tanggal harus valid.",
  });

export const slugSchema = z
  .string()
  .min(1)
  .max(80)
  .regex(
    slugPattern,
    "Slug harus menggunakan huruf kecil, angka, dan tanda hubung.",
  );

export const internalPathSchema = z
  .string()
  .regex(
    internalPathPattern,
    "Tautan internal harus berupa path absolut tanpa query, fragment, atau trailing slash.",
  );

export const internalHrefSchema = z
  .string()
  .regex(
    internalHrefPattern,
    "Tautan internal harus berupa path absolut dengan query sederhana dan tanpa fragment.",
  );

export const internalLinkSchema = z
  .object({
    label: z.string().trim().min(2).max(80),
    href: internalHrefSchema,
  })
  .strict();

const imageSourceSchema = z
  .string()
  .regex(
    internalImagePattern,
    "Gambar harus menggunakan path /images dan format yang didukung.",
  );

export const informativeImageAssetSchema = z
  .object({
    src: imageSourceSchema,
    alt: z
      .string()
      .trim()
      .min(8)
      .max(160)
      .refine(
        (value) =>
          !/^(?:foto|gambar|image|photo)(?:\s+(?:produk|tauco|product))?$/i.test(
            value,
          ),
        "Alt text harus menjelaskan isi gambar, bukan label generik.",
      ),
    decorative: z.literal(false),
  })
  .strict();

export const decorativeImageAssetSchema = z
  .object({
    src: imageSourceSchema,
    alt: z.literal(""),
    decorative: z.literal(true),
  })
  .strict();

export const imageAssetSchema = z.union([
  informativeImageAssetSchema,
  decorativeImageAssetSchema,
]);

export const seoMetadataSchema = z
  .object({
    title: z.string().trim().min(8).max(80),
    description: z.string().trim().min(50).max(180),
    canonicalPath: internalPathSchema,
    openGraphImage: informativeImageAssetSchema,
  })
  .strict();

export const textSectionSchema = z
  .object({
    id: slugSchema,
    heading: shortTextSchema,
    paragraphs: z.array(paragraphSchema).min(1).max(6),
  })
  .strict();

export const sourceReferenceSchema = z
  .object({
    label: z.string().trim().min(3).max(140),
    publisher: z.string().trim().min(3).max(140),
    url: z
      .string()
      .url()
      .refine((value) => value.startsWith("https://"), {
        message: "Sumber eksternal harus menggunakan HTTPS.",
      }),
  })
  .strict();

const pageHeroSchema = z
  .object({
    eyebrow: shortTextSchema,
    title: z.string().trim().min(4).max(100),
    description: z.string().trim().min(40).max(240),
    image: imageAssetSchema,
  })
  .strict();

export const homeContentSchema = z
  .object({
    metadata: seoMetadataSchema,
    hero: pageHeroSchema.extend({
      actions: z.array(internalLinkSchema).min(1).max(2),
    }),
    introduction: textSectionSchema,
    featuredProductSlugs: z
      .array(slugSchema)
      .max(6)
      .refine((slugs) => new Set(slugs).size === slugs.length, {
        message: "Slug produk unggulan tidak boleh duplikat.",
      }),
    guidePreview: z
      .object({
        heading: shortTextSchema,
        description: z.string().trim().min(40).max(240),
        link: internalLinkSchema,
      })
      .strict(),
    aboutPreview: z
      .object({
        heading: shortTextSchema,
        description: z.string().trim().min(40).max(240),
        link: internalLinkSchema,
      })
      .strict(),
  })
  .strict();

export const aboutContentSchema = z
  .object({
    metadata: seoMetadataSchema,
    hero: pageHeroSchema,
    sections: z.array(textSectionSchema).min(2).max(8),
    relatedLinks: z.array(internalLinkSchema).min(1).max(4),
    sources: z.array(sourceReferenceSchema).min(1).max(8),
  })
  .strict();

export const taucoGuideContentSchema = z
  .object({
    metadata: seoMetadataSchema,
    publishedAt: isoDateSchema,
    updatedAt: isoDateSchema,
    hero: pageHeroSchema,
    sections: z.array(textSectionSchema).min(4).max(12),
    relatedLinks: z.array(internalLinkSchema).min(1).max(4),
    sources: z.array(sourceReferenceSchema).min(1).max(8),
  })
  .strict()
  .superRefine((guide, context) => {
    if (guide.updatedAt < guide.publishedAt) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: "Tanggal pembaruan tidak boleh mendahului tanggal terbit.",
        path: ["updatedAt"],
      });
    }
  });

export const moneySchema = z
  .object({
    currency: z.literal("IDR"),
    amount: z.number().int().nonnegative(),
    qualifier: z.enum(["estimated", "fixed"]),
  })
  .strict();

const productFactSchema = z
  .object({
    label: z.string().trim().min(2).max(60),
    value: z.string().trim().min(2).max(160),
  })
  .strict();

export const productResearchEvidenceSchema = z
  .object({
    heading: shortTextSchema,
    summary: z.string().trim().min(40).max(300),
    facts: z.array(productFactSchema).min(1).max(8),
    scopeNote: z.string().trim().min(60).max(400),
    source: sourceReferenceSchema,
  })
  .strict();

export const productStatusSchema = z.enum(["draft", "published"]);

export const productSummarySchema = z
  .object({
    slug: slugSchema,
    name: z.string().trim().min(3).max(120),
    category: shortTextSchema,
    summary: z.string().trim().min(40).max(220),
    image: informativeImageAssetSchema,
    facts: z.array(productFactSchema).min(1).max(12),
  })
  .strict();

export const productDetailSchema = productSummarySchema
  .extend({
    metadata: seoMetadataSchema,
    description: z.array(paragraphSchema).min(1).max(6),
    usageSuggestions: z
      .array(z.string().trim().min(20).max(220))
      .min(1)
      .max(8),
    priceEstimate: moneySchema.nullable(),
    purchaseNote: z.string().trim().min(20).max(220),
    contactLink: internalLinkSchema,
    researchEvidence: productResearchEvidenceSchema,
  })
  .strict();

export const productRecordSchema = productDetailSchema
  .extend({
    status: productStatusSchema,
  })
  .strict();

export const productSchema = productDetailSchema;

const productCatalogBaseSchema = z
  .object({
    metadata: seoMetadataSchema,
    heading: z.string().trim().min(4).max(120),
    description: z.string().trim().min(40).max(240),
    contactLink: internalLinkSchema,
  })
  .strict();

export const productCatalogContentSchema = productCatalogBaseSchema
  .extend({
    products: z.array(productSummarySchema).max(1000),
  })
  .strict();

export const productCatalogDocumentSchema = productCatalogBaseSchema
  .extend({
    products: z.array(productRecordSchema).max(1000),
  })
  .strict()
  .superRefine((catalog, context) => {
    const seenSlugs = new Set<string>();

    catalog.products.forEach((product, index) => {
      if (seenSlugs.has(product.slug)) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Slug produk duplikat: ${product.slug}`,
          path: ["products", index, "slug"],
        });
      }

      seenSlugs.add(product.slug);

      const expectedCanonical = `/produk/${product.slug}`;
      if (product.metadata.canonicalPath !== expectedCanonical) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Canonical produk harus ${expectedCanonical}.`,
          path: ["products", index, "metadata", "canonicalPath"],
        });
      }
    });
  });

export const contentBundleSchema = z
  .object({
    home: homeContentSchema,
    about: aboutContentSchema,
    taucoGuide: taucoGuideContentSchema,
    productCatalog: productCatalogDocumentSchema,
  })
  .strict()
  .superRefine((bundle, context) => {
    const expectedCanonicals = [
      {
        actual: bundle.home.metadata.canonicalPath,
        expected: "/",
        path: ["home", "metadata", "canonicalPath"],
      },
      {
        actual: bundle.about.metadata.canonicalPath,
        expected: "/tentang-kami",
        path: ["about", "metadata", "canonicalPath"],
      },
      {
        actual: bundle.taucoGuide.metadata.canonicalPath,
        expected: "/tauco",
        path: ["taucoGuide", "metadata", "canonicalPath"],
      },
      {
        actual: bundle.productCatalog.metadata.canonicalPath,
        expected: "/produk",
        path: ["productCatalog", "metadata", "canonicalPath"],
      },
    ] as const;

    expectedCanonicals.forEach(({ actual, expected, path }) => {
      if (actual !== expected) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Canonical halaman harus ${expected}.`,
          path: [...path],
        });
      }
    });

    const publishedProductSlugs = new Set(
      bundle.productCatalog.products
        .filter((product) => product.status === "published")
        .map((product) => product.slug),
    );

    bundle.home.featuredProductSlugs.forEach((slug, index) => {
      if (!publishedProductSlugs.has(slug)) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Produk unggulan tidak ditemukan atau belum published: ${slug}`,
          path: ["home", "featuredProductSlugs", index],
        });
      }
    });

    const validPaths = new Set([
      "/",
      "/kontak",
      "/kebijakan-privasi",
      "/produk",
      "/tauco",
      "/tentang-kami",
      ...bundle.productCatalog.products
        .filter((product) => product.status === "published")
        .map((product) => `/produk/${product.slug}`),
    ]);

    const links = [
      ...bundle.home.hero.actions.map((link, index) => ({
        link,
        path: ["home", "hero", "actions", index, "href"],
      })),
      {
        link: bundle.home.guidePreview.link,
        path: ["home", "guidePreview", "link", "href"],
      },
      {
        link: bundle.home.aboutPreview.link,
        path: ["home", "aboutPreview", "link", "href"],
      },
      ...bundle.about.relatedLinks.map((link, index) => ({
        link,
        path: ["about", "relatedLinks", index, "href"],
      })),
      ...bundle.taucoGuide.relatedLinks.map((link, index) => ({
        link,
        path: ["taucoGuide", "relatedLinks", index, "href"],
      })),
      {
        link: bundle.productCatalog.contactLink,
        path: ["productCatalog", "contactLink", "href"],
      },
      ...bundle.productCatalog.products.map((product, index) => ({
        link: product.contactLink,
        path: ["productCatalog", "products", index, "contactLink", "href"],
      })),
    ];

    links.forEach(({ link, path }) => {
      const pathname = link.href.split("?")[0];

      if (!validPaths.has(pathname)) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Tautan mengarah ke route yang tidak dikenal: ${link.href}`,
          path,
        });
      }
    });
  });
