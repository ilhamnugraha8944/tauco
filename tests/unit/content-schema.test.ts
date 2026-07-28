import { describe, expect, it } from "vitest";

import aboutJson from "../../content/about.json";
import homeJson from "../../content/home.json";
import productsJson from "../../content/products.json";
import taucoGuideJson from "../../content/tauco-guide.json";
import {
  contentBundleSchema,
  imageAssetSchema,
  internalLinkSchema,
  productCatalogDocumentSchema,
  seoMetadataSchema,
  slugSchema,
} from "../../src/features/content";

const validBundle = {
  home: homeJson,
  about: aboutJson,
  taucoGuide: taucoGuideJson,
  productCatalog: productsJson,
};

describe("content schemas", () => {
  it("validates the complete local content bundle", () => {
    expect(contentBundleSchema.safeParse(validBundle).success).toBe(true);
  });

  it("rejects malformed slugs", () => {
    expect(slugSchema.safeParse("tauco-cap-badak").success).toBe(true);
    expect(slugSchema.safeParse("Tauco Cap Badak").success).toBe(false);
    expect(slugSchema.safeParse("tauco_cap_badak").success).toBe(false);
  });

  it("accepts safe internal query parameters and rejects malformed links", () => {
    expect(
      internalLinkSchema.safeParse({
        label: "Lihat produk",
        href: "/produk",
      }).success,
    ).toBe(true);
    expect(
      internalLinkSchema.safeParse({
        label: "Situs lain",
        href: "https://example.com",
      }).success,
    ).toBe(false);
    expect(
      internalLinkSchema.safeParse({
        label: "Tanyakan produk",
        href: "/kontak?topik=produk",
      }).success,
    ).toBe(true);
    expect(
      internalLinkSchema.safeParse({
        label: "Produk dengan fragment",
        href: "/produk#baru",
      }).success,
    ).toBe(false);
  });

  it("requires descriptive alt text and an internal image path", () => {
    const informativeImage = imageAssetSchema.safeParse({
      src: "/images/tauco-fermentation-provisional.png",
      alt: "Ilustrasi tauco semipadat dalam mangkuk dengan kedelai",
      decorative: false,
    });

    expect(informativeImage.success).toBe(true);

    if (informativeImage.success) {
      expect(informativeImage.data.decorative).toBe(false);
    }

    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "Foto produk",
        decorative: false,
      }).success,
    ).toBe(false);
    expect(
      imageAssetSchema.safeParse({
        src: "https://example.com/product.webp",
        alt: "Produk Tauco Cap Badak dari Cianjur",
        decorative: false,
      }).success,
    ).toBe(false);

    for (const genericAlt of [
      "foto tauco",
      "FOTO TAUCO",
      "Gambar Produk",
      "IMAGE PRODUCT",
      "photo product",
    ]) {
      expect(
        imageAssetSchema.safeParse({
          src: "/images/tauco-fermentation-provisional.png",
          alt: genericAlt,
          decorative: false,
        }).success,
      ).toBe(false);
    }

    const paddedAlt = imageAssetSchema.parse({
      src: "/images/tauco-fermentation-provisional.png",
      alt: "  Tauco semipadat dalam mangkuk  ",
      decorative: false,
    });
    expect(paddedAlt.alt).toBe("Tauco semipadat dalam mangkuk");
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "        ",
        decorative: false,
      }).success,
    ).toBe(false);
  });

  it("requires an empty alt only for explicitly decorative images", () => {
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "Ilustrasi tauco semipadat dalam mangkuk dengan kedelai",
      }).success,
    ).toBe(false);
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "",
        decorative: true,
      }).success,
    ).toBe(true);
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "Tauco dalam mangkuk",
        decorative: true,
      }).success,
    ).toBe(false);
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "",
        decorative: false,
      }).success,
    ).toBe(false);
  });

  it("rejects incomplete SEO metadata", () => {
    expect(
      seoMetadataSchema.safeParse({
        title: "Produk Tauco Cap Badak",
        description: "Deskripsi ini terlalu pendek.",
        canonicalPath: "/produk",
        openGraphImage: {
          src: "/images/tauco-fermentation-provisional.png",
          alt: "Ilustrasi tauco semipadat dalam mangkuk dengan kedelai",
          decorative: false,
        },
      }).success,
    ).toBe(false);
  });

  it("requires an informative Open Graph image", () => {
    expect(
      seoMetadataSchema.safeParse({
        ...homeJson.metadata,
        openGraphImage: {
          src: "/images/tauco-hero-provisional.webp",
          alt: "",
          decorative: true,
        },
      }).success,
    ).toBe(false);
  });

  it("requires an explicit product publication status", () => {
    const productWithoutStatus = structuredClone(productsJson.products[0]) as {
      status?: string;
    };
    delete productWithoutStatus.status;

    expect(
      productCatalogDocumentSchema.safeParse({
        ...productsJson,
        products: [productWithoutStatus],
      }).success,
    ).toBe(false);
  });

  it("accepts an authored catalog without product records", () => {
    expect(
      productCatalogDocumentSchema.safeParse({
        ...productsJson,
        products: [],
      }).success,
    ).toBe(true);
  });

  it("rejects duplicate product slugs across publication statuses", () => {
    const firstProduct = productsJson.products[0];
    const catalogWithDuplicate = {
      ...productsJson,
      products: [
        firstProduct,
        {
          ...firstProduct,
          status: "draft",
        },
      ],
    };

    expect(
      productCatalogDocumentSchema.safeParse(catalogWithDuplicate).success,
    ).toBe(false);
  });

  it("rejects featured products that do not exist", () => {
    const bundleWithUnknownFeaturedProduct = {
      ...validBundle,
      home: {
        ...homeJson,
        featuredProductSlugs: ["produk-tidak-ada"],
      },
    };

    expect(
      contentBundleSchema.safeParse(bundleWithUnknownFeaturedProduct).success,
    ).toBe(false);
  });

  it("rejects a featured product that is still a draft", () => {
    const bundleWithDraftFeaturedProduct = {
      ...validBundle,
      productCatalog: {
        ...productsJson,
        products: productsJson.products.map((product) => ({
          ...product,
          status: "draft",
        })),
      },
    };

    expect(
      contentBundleSchema.safeParse(bundleWithDraftFeaturedProduct).success,
    ).toBe(false);
  });
});
