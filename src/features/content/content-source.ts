import type {
  AboutContent,
  HomeContent,
  ProductCatalogContent,
  ProductDetail,
  TaucoGuideContent,
} from "./types";

export interface ContentSource {
  getHome(): Promise<HomeContent>;
  getAbout(): Promise<AboutContent>;
  getTaucoGuide(): Promise<TaucoGuideContent>;
  listProducts(): Promise<ProductCatalogContent>;
  getProductBySlug(slug: string): Promise<ProductDetail | null>;
}
