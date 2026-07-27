import { describe, expect, it } from "vitest";

import aboutJson from "../../content/about.json";
import homeJson from "../../content/home.json";
import productsJson from "../../content/products.json";
import taucoGuideJson from "../../content/tauco-guide.json";
import {
  contentBundleSchema,
  LocalContentSource,
} from "../../src/features/content";

const baseBundleInput = {
  home: homeJson,
  about: aboutJson,
  taucoGuide: taucoGuideJson,
  productCatalog: productsJson,
};

describe("LocalContentSource", () => {
  const source = new LocalContentSource();

  it("implements async access to every Phase 1A content area", async () => {
    const [home, about, guide, catalog] = await Promise.all([
      source.getHome(),
      source.getAbout(),
      source.getTaucoGuide(),
      source.listProducts(),
    ]);

    expect(home.metadata.canonicalPath).toBe("/");
    expect(about.metadata.canonicalPath).toBe("/tentang-kami");
    expect(guide.metadata.canonicalPath).toBe("/tauco");
    expect(catalog.metadata.canonicalPath).toBe("/produk");
    expect(catalog.products).toHaveLength(1);
  });

  it("keeps the tauco definition direct and the guide hero concise", async () => {
    const guide = await source.getTaucoGuide();
    const definition = guide.sections.find(
      (section) => section.id === "pengertian-tauco",
    );
    const definitionWordCount =
      definition?.paragraphs[0].trim().split(/\s+/).length ?? 0;
    const heroWordCount = guide.hero.description.trim().split(/\s+/).length;

    expect(definitionWordCount).toBeGreaterThanOrEqual(40);
    expect(definitionWordCount).toBeLessThanOrEqual(60);
    expect(heroWordCount).toBeLessThanOrEqual(20);
  });

  it("finds a product by its stable slug", async () => {
    const product = await source.getProductBySlug("tauco-cap-badak");

    expect(product).not.toBeNull();
    expect(product?.name).toBe("Tauco Cap Badak");
    expect(product?.metadata.canonicalPath).toBe(
      "/produk/tauco-cap-badak",
    );
    expect(product?.researchEvidence.facts).toContainEqual({
      label: "Kategori sampel",
      value: "Semipadat",
    });
    expect(product).not.toHaveProperty("status");
  });

  it("returns null for unknown or malformed slugs", async () => {
    await expect(source.getProductBySlug("produk-tidak-ada")).resolves.toBeNull();
    await expect(source.getProductBySlug("Tauco Cap Badak")).resolves.toBeNull();
  });

  it("keeps every product slug unique", async () => {
    const catalog = await source.listProducts();
    const slugs = catalog.products.map((product) => product.slug);

    expect(new Set(slugs).size).toBe(slugs.length);
  });

  it("only references existing featured products", async () => {
    const [home, catalog] = await Promise.all([
      source.getHome(),
      source.listProducts(),
    ]);
    const availableSlugs = new Set(
      catalog.products.map((product) => product.slug),
    );

    expect(
      home.featuredProductSlugs.every((slug) => availableSlugs.has(slug)),
    ).toBe(true);
  });

  it("returns summaries from listProducts without detail-only fields", async () => {
    const catalog = await source.listProducts();
    const product = catalog.products[0];

    expect(product).toMatchObject({
      slug: "tauco-cap-badak",
      name: "Tauco Cap Badak",
    });
    expect(product).not.toHaveProperty("status");
    expect(product).not.toHaveProperty("description");
    expect(product).not.toHaveProperty("researchEvidence");
    expect(product).not.toHaveProperty("purchaseNote");
    expect(product).not.toHaveProperty("metadata");
  });

  it("filters draft products from summaries and detail lookup", async () => {
    const currentProduct = productsJson.products[0];
    const draftSlug = "tauco-uji-draft";
    const bundle = contentBundleSchema.parse({
      ...baseBundleInput,
      productCatalog: {
        ...productsJson,
        products: [
          currentProduct,
          {
            ...currentProduct,
            slug: draftSlug,
            status: "draft",
            name: "Tauco Uji Draft",
            metadata: {
              ...currentProduct.metadata,
              title: "Tauco Uji Draft | Produk Belum Terbit",
              canonicalPath: `/produk/${draftSlug}`,
            },
          },
        ],
      },
    });
    const sourceWithDraft = new LocalContentSource(bundle);
    const catalog = await sourceWithDraft.listProducts();

    expect(catalog.products.map((product) => product.slug)).toEqual([
      "tauco-cap-badak",
    ]);
    await expect(
      sourceWithDraft.getProductBySlug(draftSlug),
    ).resolves.toBeNull();
  });

  it("supports an empty published catalog", async () => {
    const bundle = contentBundleSchema.parse({
      ...baseBundleInput,
      home: {
        ...homeJson,
        featuredProductSlugs: [],
      },
      taucoGuide: {
        ...taucoGuideJson,
        relatedLinks: taucoGuideJson.relatedLinks.filter(
          (link) => !link.href.startsWith("/produk/"),
        ),
      },
      productCatalog: {
        ...productsJson,
        products: [],
      },
    });
    const emptySource = new LocalContentSource(bundle);

    await expect(emptySource.listProducts()).resolves.toMatchObject({
      products: [],
    });
    await expect(
      emptySource.getProductBySlug("tauco-cap-badak"),
    ).resolves.toBeNull();
  });
});
