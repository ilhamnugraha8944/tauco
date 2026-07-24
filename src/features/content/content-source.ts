import type {
  AboutContent,
  HomeContent,
  Product,
  ProductCatalogContent,
  TaucoGuideContent,
} from "./types";

export interface ContentSource {
  getHome(): Promise<HomeContent>;
  getAbout(): Promise<AboutContent>;
  getTaucoGuide(): Promise<TaucoGuideContent>;
  listProducts(): Promise<ProductCatalogContent>;
  getProductBySlug(slug: string): Promise<Product | null>;
}
