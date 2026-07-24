import type { ContactGateway } from "./contact-gateway";
import type { ContactMessage } from "./contact-model";

function encodeForm(message: ContactMessage): string {
  const params = new URLSearchParams({
    "form-name": "kontak",
    name: message.name,
    email: message.email,
    phone: message.phone,
    subject: message.subject,
    message: message.message,
    privacyConsent: message.privacyConsent ? "yes" : "",
    "bot-field": message.botField,
  });

  return params.toString();
}

export class NetlifyContactGateway implements ContactGateway {
  async submitContactMessage(message: ContactMessage): Promise<void> {
    const response = await fetch("/__forms.html", {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: encodeForm(message),
    });

    if (!response.ok) {
      throw new Error("Netlify Forms tidak menerima pesan.");
    }
  }
}

export const contactGateway: ContactGateway = new NetlifyContactGateway();
