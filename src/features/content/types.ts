import type { z } from "zod";

import type {
  aboutContentSchema,
  contentBundleSchema,
  decorativeImageAssetSchema,
  homeContentSchema,
  imageAssetSchema,
  informativeImageAssetSchema,
  internalLinkSchema,
  moneySchema,
  productCatalogContentSchema,
  productCatalogDocumentSchema,
  productDetailSchema,
  productRecordSchema,
  productResearchEvidenceSchema,
  productStatusSchema,
  productSummarySchema,
  seoMetadataSchema,
  sourceReferenceSchema,
  taucoGuideContentSchema,
  textSectionSchema,
} from "./schemas";

export type AboutContent = z.infer<typeof aboutContentSchema>;
export type ContentBundle = z.infer<typeof contentBundleSchema>;
export type DecorativeImageAsset = z.infer<
  typeof decorativeImageAssetSchema
>;
export type HomeContent = z.infer<typeof homeContentSchema>;
export type ImageAsset = z.infer<typeof imageAssetSchema>;
export type InformativeImageAsset = z.infer<
  typeof informativeImageAssetSchema
>;
export type InternalLink = z.infer<typeof internalLinkSchema>;
export type Money = z.infer<typeof moneySchema>;
export type ProductCatalogContent = z.infer<
  typeof productCatalogContentSchema
>;
export type ProductCatalogDocument = z.infer<
  typeof productCatalogDocumentSchema
>;
export type ProductDetail = z.infer<typeof productDetailSchema>;
export type ProductRecord = z.infer<typeof productRecordSchema>;
export type ProductResearchEvidence = z.infer<
  typeof productResearchEvidenceSchema
>;
export type ProductStatus = z.infer<typeof productStatusSchema>;
export type ProductSummary = z.infer<typeof productSummarySchema>;
export type Product = ProductDetail;
export type SeoMetadata = z.infer<typeof seoMetadataSchema>;
export type SourceReference = z.infer<typeof sourceReferenceSchema>;
export type TaucoGuideContent = z.infer<typeof taucoGuideContentSchema>;
export type TextSection = z.infer<typeof textSectionSchema>;
