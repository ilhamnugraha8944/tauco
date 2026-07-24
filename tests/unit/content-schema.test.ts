import { describe, expect, it } from "vitest";

import aboutJson from "../../content/about.json";
import homeJson from "../../content/home.json";
import productsJson from "../../content/products.json";
import taucoGuideJson from "../../content/tauco-guide.json";
import {
  contentBundleSchema,
  imageAssetSchema,
  internalLinkSchema,
  productCatalogContentSchema,
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
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "Ilustrasi tauco semipadat dalam mangkuk dengan kedelai",
      }).success,
    ).toBe(true);
    expect(
      imageAssetSchema.safeParse({
        src: "/images/tauco-fermentation-provisional.png",
        alt: "Foto produk",
      }).success,
    ).toBe(false);
    expect(
      imageAssetSchema.safeParse({
        src: "https://example.com/product.webp",
        alt: "Produk Tauco Cap Badak dari Cianjur",
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
        },
      }).success,
    ).toBe(false);
  });

  it("rejects duplicate product slugs", () => {
    const firstProduct = productsJson.products[0];
    const catalogWithDuplicate = {
      ...productsJson,
      products: [firstProduct, { ...firstProduct }],
    };

    expect(
      productCatalogContentSchema.safeParse(catalogWithDuplicate).success,
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
});
