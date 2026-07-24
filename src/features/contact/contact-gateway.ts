import type { ContactMessage } from "./contact-model";

export interface ContactGateway {
  submitContactMessage(message: ContactMessage): Promise<void>;
}
