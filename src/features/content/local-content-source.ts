import aboutJson from "../../../content/about.json";
import homeJson from "../../../content/home.json";
import productsJson from "../../../content/products.json";
import taucoGuideJson from "../../../content/tauco-guide.json";

import type { ContentSource } from "./content-source";
import {
  contentBundleSchema,
  productCatalogContentSchema,
  productDetailSchema,
  productSummarySchema,
  slugSchema,
} from "./schemas";
import type {
  AboutContent,
  ContentBundle,
  HomeContent,
  ProductCatalogContent,
  ProductDetail,
  ProductRecord,
  ProductSummary,
  TaucoGuideContent,
} from "./types";

const defaultContentBundle = contentBundleSchema.parse({
  home: homeJson,
  about: aboutJson,
  taucoGuide: taucoGuideJson,
  productCatalog: productsJson,
});

function toProductSummary(product: ProductRecord): ProductSummary {
  return productSummarySchema.parse({
    slug: product.slug,
    name: product.name,
    category: product.category,
    summary: product.summary,
    image: product.image,
    facts: product.facts,
  });
}

function toProductDetail(product: ProductRecord): ProductDetail {
  return productDetailSchema.parse({
    slug: product.slug,
    name: product.name,
    category: product.category,
    summary: product.summary,
    image: product.image,
    facts: product.facts,
    metadata: product.metadata,
    description: product.description,
    usageSuggestions: product.usageSuggestions,
    priceEstimate: product.priceEstimate,
    purchaseNote: product.purchaseNote,
    contactLink: product.contactLink,
    researchEvidence: product.researchEvidence,
  });
}

export class LocalContentSource implements ContentSource {
  private readonly contentBundle: ContentBundle;
  private readonly productCatalog: ProductCatalogContent;
  private readonly productBySlug: Map<string, ProductDetail>;

  constructor(contentBundle: ContentBundle = defaultContentBundle) {
    this.contentBundle = contentBundle;

    const publishedProducts = contentBundle.productCatalog.products.filter(
      (product) => product.status === "published",
    );

    this.productCatalog = productCatalogContentSchema.parse({
      metadata: contentBundle.productCatalog.metadata,
      heading: contentBundle.productCatalog.heading,
      description: contentBundle.productCatalog.description,
      contactLink: contentBundle.productCatalog.contactLink,
      products: publishedProducts.map(toProductSummary),
    });

    this.productBySlug = new Map(
      publishedProducts.map((product) => [
        product.slug,
        toProductDetail(product),
      ]),
    );
  }

  async getHome(): Promise<HomeContent> {
    return this.contentBundle.home;
  }

  async getAbout(): Promise<AboutContent> {
    return this.contentBundle.about;
  }

  async getTaucoGuide(): Promise<TaucoGuideContent> {
    return this.contentBundle.taucoGuide;
  }

  async listProducts(): Promise<ProductCatalogContent> {
    return this.productCatalog;
  }

  async getProductBySlug(slug: string): Promise<ProductDetail | null> {
    if (!slugSchema.safeParse(slug).success) {
      return null;
    }

    return this.productBySlug.get(slug) ?? null;
  }
}

export const localContentSource: ContentSource = new LocalContentSource();
