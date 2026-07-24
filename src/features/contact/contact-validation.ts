import {
  contactEmailPattern,
  contactFieldLimits,
  contactPhonePattern,
  isContactSubject,
  type ContactFieldErrors,
  type ContactMessage,
  type ContactMessageInput,
} from "./contact-model";

export type ContactValidationResult =
  | {
      success: true;
      data: ContactMessage;
    }
  | {
      success: false;
      errors: ContactFieldErrors;
    };

export function validateContactMessage(
  input: ContactMessageInput,
): ContactValidationResult {
  const normalized = {
    ...input,
    name: input.name.trim(),
    email: input.email.trim(),
    phone: input.phone.trim(),
    message: input.message.trim(),
  };
  const errors: ContactFieldErrors = {};

  if (normalized.name.length < contactFieldLimits.name.min) {
    errors.name = "Nama minimal 2 karakter.";
  } else if (normalized.name.length > contactFieldLimits.name.max) {
    errors.name = "Nama maksimal 100 karakter.";
  }

  if (
    !contactEmailPattern.test(normalized.email) ||
    normalized.email.length > contactFieldLimits.email.max
  ) {
    errors.email =
      normalized.email.length > contactFieldLimits.email.max
        ? "Email maksimal 160 karakter."
        : "Masukkan alamat email yang valid.";
  }

  if (normalized.phone.length > contactFieldLimits.phone.max) {
    errors.phone = "Nomor telepon maksimal 30 karakter.";
  } else if (
    normalized.phone !== "" &&
    !contactPhonePattern.test(normalized.phone)
  ) {
    errors.phone = "Masukkan nomor telepon yang valid.";
  }

  const subject = isContactSubject(normalized.subject)
    ? normalized.subject
    : undefined;

  if (!subject) {
    errors.subject = "Pilih topik pertanyaan.";
  }

  if (normalized.message.length < contactFieldLimits.message.min) {
    errors.message = "Pesan minimal 20 karakter.";
  } else if (
    normalized.message.length > contactFieldLimits.message.max
  ) {
    errors.message = "Pesan maksimal 2.000 karakter.";
  }

  if (!normalized.privacyConsent) {
    errors.privacyConsent = "Persetujuan privasi wajib diberikan.";
  }

  if (normalized.botField !== "") {
    errors.botField = "Pengiriman tidak dapat diproses.";
  }

  if (Object.keys(errors).length > 0 || !subject) {
    return {
      success: false,
      errors,
    };
  }

  return {
    success: true,
    data: {
      ...normalized,
      subject,
      privacyConsent: true,
    },
  };
}
