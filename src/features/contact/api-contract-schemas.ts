import { z } from "zod";

import { createApiSuccessEnvelopeSchema } from "../api-contract";
import {
  contactEmailPattern,
  contactFieldLimits,
  contactPhonePattern,
  contactSubjectValues,
  type ContactMessage,
} from "./contact-model";

const hasCanonicalWhitespace = (value: string) => value === value.trim();

// This schema validates the JSON wire representation after form input has
// already been normalized by contactMessageSchema.
export const contactApiRequestSchema: z.ZodType<ContactMessage> = z
  .object({
    name: z
      .string()
      .min(contactFieldLimits.name.min)
      .max(contactFieldLimits.name.max)
      .refine(hasCanonicalWhitespace, "Nama harus sudah dinormalisasi."),
    email: z
      .string()
      .regex(contactEmailPattern)
      .max(contactFieldLimits.email.max)
      .refine(hasCanonicalWhitespace, "Email harus sudah dinormalisasi."),
    phone: z
      .string()
      .max(contactFieldLimits.phone.max)
      .refine(
        (value) => value === "" || contactPhonePattern.test(value),
        "Nomor telepon tidak valid.",
      )
      .refine(
        hasCanonicalWhitespace,
        "Nomor telepon harus sudah dinormalisasi.",
      ),
    subject: z.enum(contactSubjectValues),
    message: z
      .string()
      .min(contactFieldLimits.message.min)
      .max(contactFieldLimits.message.max)
      .refine(hasCanonicalWhitespace, "Pesan harus sudah dinormalisasi."),
    privacyConsent: z.literal(true),
    botField: z.string().max(0),
  })
  .strict();

export const contactApiResultSchema = z
  .object({
    status: z.literal("received"),
  })
  .strict();

export const contactApiResponseSchema = createApiSuccessEnvelopeSchema(
  contactApiResultSchema,
);
