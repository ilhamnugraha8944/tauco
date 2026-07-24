import { z } from "zod";

import {
  contactEmailPattern,
  contactFieldLimits,
  contactPhonePattern,
  contactSubjectValues,
  type ContactMessage,
} from "./contact-model";

export const contactMessageSchema: z.ZodType<ContactMessage> = z
  .object({
    name: z
      .string()
      .trim()
      .min(contactFieldLimits.name.min, "Nama minimal 2 karakter.")
      .max(contactFieldLimits.name.max, "Nama maksimal 100 karakter."),
    email: z
      .string()
      .trim()
      .regex(contactEmailPattern, "Masukkan alamat email yang valid.")
      .max(contactFieldLimits.email.max, "Email maksimal 160 karakter."),
    phone: z
      .string()
      .trim()
      .max(
        contactFieldLimits.phone.max,
        "Nomor telepon maksimal 30 karakter.",
      )
      .refine(
        (value) => value === "" || contactPhonePattern.test(value),
        "Masukkan nomor telepon yang valid.",
      ),
    subject: z.enum(contactSubjectValues, {
      message: "Pilih topik pertanyaan.",
    }),
    message: z
      .string()
      .trim()
      .min(contactFieldLimits.message.min, "Pesan minimal 20 karakter.")
      .max(
        contactFieldLimits.message.max,
        "Pesan maksimal 2.000 karakter.",
      ),
    privacyConsent: z.literal(true, {
      message: "Persetujuan privasi wajib diberikan.",
    }),
    botField: z.string().max(0, "Pengiriman tidak dapat diproses."),
  })
  .strict();
