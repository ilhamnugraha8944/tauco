export const contactSubjectValues = [
  "Informasi produk",
  "Kerja sama dan distribusi",
  "Pertanyaan umum",
] as const;

export type ContactSubject = (typeof contactSubjectValues)[number];

export const contactFieldLimits = {
  name: {
    min: 2,
    max: 100,
  },
  email: {
    max: 160,
  },
  phone: {
    min: 7,
    max: 30,
  },
  message: {
    min: 20,
    max: 2000,
  },
} as const;

export const contactEmailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
export const contactPhonePattern = /^[+0-9()\s-]{7,30}$/;

export type ContactMessageInput = {
  name: string;
  email: string;
  phone: string;
  subject: string;
  message: string;
  privacyConsent: boolean;
  botField: string;
};

export type ContactMessage = Omit<ContactMessageInput, "subject"> & {
  subject: ContactSubject;
  privacyConsent: true;
};

export type ContactFieldErrors = Partial<
  Record<keyof ContactMessageInput, string>
>;

export function isContactSubject(value: string): value is ContactSubject {
  return contactSubjectValues.some((subject) => subject === value);
}
