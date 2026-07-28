export type { ContactGateway } from "./contact-gateway";
export {
  contactApiRequestSchema,
  contactApiResponseSchema,
  contactApiResultSchema,
} from "./api-contract-schemas";
export { contactMessageSchema } from "./contact-schema";
export {
  contactEmailPattern,
  contactFieldLimits,
  contactPhonePattern,
  contactSubjectValues,
  isContactSubject,
} from "./contact-model";
export type {
  ContactFieldErrors,
  ContactMessage,
  ContactMessageInput,
  ContactSubject,
} from "./contact-model";
export { validateContactMessage } from "./contact-validation";
export type {
  ContactValidationResult,
} from "./contact-validation";
export {
  contactGateway,
  NetlifyContactGateway,
} from "./netlify-contact-gateway";
