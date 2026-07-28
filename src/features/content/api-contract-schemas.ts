import { z } from "zod";

import {
  createApiPaginatedSuccessEnvelopeSchema,
  createApiSuccessEnvelopeSchema,
} from "../api-contract";
import {
  aboutContentSchema,
  homeContentSchema,
  productCatalogContentSchema,
  productDetailSchema,
  productSummarySchema,
  taucoGuideContentSchema,
} from "./schemas";

export const homeApiResponseSchema =
  createApiSuccessEnvelopeSchema(homeContentSchema);
export const aboutApiResponseSchema =
  createApiSuccessEnvelopeSchema(aboutContentSchema);
export const taucoGuideApiResponseSchema =
  createApiSuccessEnvelopeSchema(taucoGuideContentSchema);
export const apiProductCatalogContentSchema = productCatalogContentSchema
  .extend({
    products: z.array(productSummarySchema).max(50),
  })
  .strict();
export const productsApiResponseSchema =
  createApiPaginatedSuccessEnvelopeSchema(apiProductCatalogContentSchema);
export const productDetailApiResponseSchema =
  createApiSuccessEnvelopeSchema(productDetailSchema);
