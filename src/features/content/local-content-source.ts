import aboutJson from "../../../content/about.json";
import homeJson from "../../../content/home.json";
import productsJson from "../../../content/products.json";
import taucoGuideJson from "../../../content/tauco-guide.json";

import type { ContentSource } from "./content-source";
import { contentBundleSchema, slugSchema } from "./schemas";
import type {
  AboutContent,
  HomeContent,
  Product,
  ProductCatalogContent,
  TaucoGuideContent,
} from "./types";

const contentBundle = contentBundleSchema.parse({
  home: homeJson,
  about: aboutJson,
  taucoGuide: taucoGuideJson,
  productCatalog: productsJson,
});

export class LocalContentSource implements ContentSource {
  private readonly productBySlug = new Map(
    contentBundle.productCatalog.products.map((product) => [
      product.slug,
      product,
    ]),
  );

  async getHome(): Promise<HomeContent> {
    return contentBundle.home;
  }

  async getAbout(): Promise<AboutContent> {
    return contentBundle.about;
  }

  async getTaucoGuide(): Promise<TaucoGuideContent> {
    return contentBundle.taucoGuide;
  }

  async listProducts(): Promise<ProductCatalogContent> {
    return contentBundle.productCatalog;
  }

  async getProductBySlug(slug: string): Promise<Product | null> {
    if (!slugSchema.safeParse(slug).success) {
      return null;
    }

    return this.productBySlug.get(slug) ?? null;
  }
}

export const localContentSource: ContentSource = new LocalContentSource();
