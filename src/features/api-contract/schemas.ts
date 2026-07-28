import { z } from "zod";

const requestIdPattern = /^[A-Za-z0-9._:-]+$/;
const cursorPattern = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;
const problemInstancePattern =
  /^\/(?:$|(?:[A-Za-z0-9._~!$&'()*+,;=:@-]|%[0-9A-Fa-f]{2})(?:[A-Za-z0-9._~!$&'()*+,;=:@/-]|%[0-9A-Fa-f]{2})*)$/;
const problemTypePattern =
  /^urn:tauco-cap-badak:problem:[a-z0-9]+(?:-[a-z0-9]+)*$/;
const stableErrorCodePattern = /^[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$/;

export const apiRequestIdSchema = z
  .string()
  .min(1)
  .max(128)
  .regex(requestIdPattern, "Request ID memuat karakter yang tidak aman.");

export const apiResponseMetaSchema = z
  .object({
    requestId: apiRequestIdSchema,
    apiVersion: z.literal("v1"),
  })
  .strict();

export const apiPageSchema = z
  .object({
    nextCursor: z
      .string()
      .min(1)
      .max(1024)
      .regex(
        cursorPattern,
        "Cursor harus berisi payload dan signature base64url.",
      )
      .nullable(),
    hasMore: z.boolean(),
    limit: z.number().int().min(1).max(50),
  })
  .strict()
  .superRefine((page, context) => {
    if (page.hasMore && page.nextCursor === null) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: "nextCursor wajib tersedia ketika hasMore bernilai true.",
        path: ["nextCursor"],
      });
    }

    if (!page.hasMore && page.nextCursor !== null) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: "nextCursor harus null ketika hasMore bernilai false.",
        path: ["nextCursor"],
      });
    }
  });

export const apiPaginatedResponseMetaSchema = apiResponseMetaSchema
  .extend({
    page: apiPageSchema,
  })
  .strict();

export function createApiSuccessEnvelopeSchema<
  DataSchema extends z.ZodType,
>(dataSchema: DataSchema) {
  return z
    .object({
      data: dataSchema,
      meta: apiResponseMetaSchema,
    })
    .strict();
}

export function createApiPaginatedSuccessEnvelopeSchema<
  DataSchema extends z.ZodType,
>(dataSchema: DataSchema) {
  return z
    .object({
      data: dataSchema,
      meta: apiPaginatedResponseMetaSchema,
    })
    .strict();
}

export const apiProblemFieldErrorSchema = z
  .object({
    field: z.string().trim().min(1).max(120),
    code: z
      .string()
      .min(3)
      .max(80)
      .regex(stableErrorCodePattern, "Kode error field harus stabil."),
    message: z.string().trim().min(3).max(240),
  })
  .strict();

export const apiProblemDetailsSchema = z
  .object({
    type: z
      .string()
      .max(160)
      .regex(problemTypePattern, "Problem type harus berupa URN yang dikenal."),
    title: z.string().trim().min(3).max(120),
    status: z.number().int().min(400).max(599),
    detail: z.string().trim().min(3).max(500),
    instance: z.string().min(1).max(2048).regex(problemInstancePattern),
    code: z
      .string()
      .min(3)
      .max(80)
      .regex(stableErrorCodePattern, "Problem code harus stabil."),
    requestId: apiRequestIdSchema,
    errors: z.array(apiProblemFieldErrorSchema).min(1).max(32).optional(),
  })
  .strict();

export type ApiResponseMeta = z.infer<typeof apiResponseMetaSchema>;
export type ApiPage = z.infer<typeof apiPageSchema>;
export type ApiPaginatedResponseMeta = z.infer<
  typeof apiPaginatedResponseMetaSchema
>;
export type ApiProblemFieldError = z.infer<
  typeof apiProblemFieldErrorSchema
>;
export type ApiProblemDetails = z.infer<typeof apiProblemDetailsSchema>;
