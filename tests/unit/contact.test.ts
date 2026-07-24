import { afterEach, describe, expect, it, vi } from "vitest";

import {
  contactMessageSchema,
  NetlifyContactGateway,
  type ContactMessage,
  validateContactMessage,
} from "@/features/contact";

const validMessage: ContactMessage = {
  name: "Ilham Pratama",
  email: "ilham@example.com",
  phone: "+62 812 3456 7890",
  subject: "Informasi produk",
  message: "Saya ingin mengetahui informasi produk yang tersedia saat ini.",
  privacyConsent: true,
  botField: "",
};

describe("contactMessageSchema", () => {
  it("accepts a complete valid message", () => {
    expect(contactMessageSchema.safeParse(validMessage).success).toBe(true);
  });

  it("requires privacy consent", () => {
    const result = contactMessageSchema.safeParse({
      ...validMessage,
      privacyConsent: false,
    });

    expect(result.success).toBe(false);
  });

  it("rejects a filled honeypot", () => {
    const result = contactMessageSchema.safeParse({
      ...validMessage,
      botField: "spam",
    });

    expect(result.success).toBe(false);
  });

  it("rejects a short message", () => {
    const result = contactMessageSchema.safeParse({
      ...validMessage,
      message: "Terlalu singkat",
    });

    expect(result.success).toBe(false);
  });
});

describe("validateContactMessage", () => {
  it("normalizes the browser payload consistently", () => {
    const result = validateContactMessage({
      ...validMessage,
      name: "  Ilham Pratama  ",
      email: "  ilham@example.com  ",
      message:
        "  Saya ingin mengetahui informasi produk yang tersedia saat ini.  ",
    });

    expect(result).toEqual({
      success: true,
      data: validMessage,
    });
  });

  it("returns field-level errors that match the server contract", () => {
    const invalidMessage = {
      ...validMessage,
      name: "I",
      email: "alamat-tidak-valid",
      phone: "abc",
      subject: "Topik tidak dikenal",
      message: "Terlalu singkat",
      privacyConsent: false,
      botField: "spam",
    };
    const browserResult = validateContactMessage(invalidMessage);
    const schemaResult = contactMessageSchema.safeParse(invalidMessage);

    expect(browserResult).toEqual({
      success: false,
      errors: {
        name: "Nama minimal 2 karakter.",
        email: "Masukkan alamat email yang valid.",
        phone: "Masukkan nomor telepon yang valid.",
        subject: "Pilih topik pertanyaan.",
        message: "Pesan minimal 20 karakter.",
        privacyConsent: "Persetujuan privasi wajib diberikan.",
        botField: "Pengiriman tidak dapat diproses.",
      },
    });
    expect(schemaResult.success).toBe(false);
  });
});

describe("NetlifyContactGateway", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts a URL-encoded payload to the Netlify form blueprint", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", fetchMock);

    const gateway = new NetlifyContactGateway();
    await gateway.submitContactMessage(validMessage);

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, options] = fetchMock.mock.calls[0] as [
      string,
      RequestInit,
    ];
    const body = new URLSearchParams(String(options.body));

    expect(url).toBe("/__forms.html");
    expect(options.method).toBe("POST");
    expect(options.headers).toEqual({
      "Content-Type": "application/x-www-form-urlencoded",
    });
    expect(body.get("form-name")).toBe("kontak");
    expect(body.get("subject")).toBe("Informasi produk");
    expect(body.get("privacyConsent")).toBe("yes");
    expect(body.get("bot-field")).toBe("");
  });

  it("throws when the form endpoint rejects the request", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false }));

    const gateway = new NetlifyContactGateway();

    await expect(gateway.submitContactMessage(validMessage)).rejects.toThrow(
      "Netlify Forms tidak menerima pesan.",
    );
  });
});
