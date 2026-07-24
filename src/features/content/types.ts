import type { z } from "zod";

import type {
  aboutContentSchema,
  homeContentSchema,
  imageAssetSchema,
  internalLinkSchema,
  moneySchema,
  productCatalogContentSchema,
  productResearchEvidenceSchema,
  productSchema,
  seoMetadataSchema,
  sourceReferenceSchema,
  taucoGuideContentSchema,
  textSectionSchema,
} from "./schemas";

export type AboutContent = z.infer<typeof aboutContentSchema>;
export type HomeContent = z.infer<typeof homeContentSchema>;
export type ImageAsset = z.infer<typeof imageAssetSchema>;
export type InternalLink = z.infer<typeof internalLinkSchema>;
export type Money = z.infer<typeof moneySchema>;
export type Product = z.infer<typeof productSchema>;
export type ProductCatalogContent = z.infer<
  typeof productCatalogContentSchema
>;
export type ProductResearchEvidence = z.infer<
  typeof productResearchEvidenceSchema
>;
export type SeoMetadata = z.infer<typeof seoMetadataSchema>;
export type SourceReference = z.infer<typeof sourceReferenceSchema>;
export type TaucoGuideContent = z.infer<typeof taucoGuideContentSchema>;
export type TextSection = z.infer<typeof textSectionSchema>;
