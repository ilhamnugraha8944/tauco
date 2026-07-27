"use client";

import {
  type FormEvent,
  useEffect,
  useRef,
  useState,
} from "react";

import {
  contactFieldLimits,
  contactPhonePattern,
  contactSubjectValues,
  type ContactFieldErrors,
  type ContactMessage,
  type ContactMessageInput,
} from "@/features/contact/contact-model";
import { validateContactMessage } from "@/features/contact/contact-validation";
import { contactGateway } from "@/features/contact/netlify-contact-gateway";

type SubmissionState =
  | { status: "idle"; errors: ContactFieldErrors }
  | { status: "pending"; errors: ContactFieldErrors }
  | { status: "success"; errors: ContactFieldErrors }
  | { status: "validation-error"; errors: ContactFieldErrors }
  | {
      status: "network-error";
      errors: ContactFieldErrors;
      message: string;
    };

const initialState: SubmissionState = {
  status: "idle",
  errors: {},
};

const focusableFieldOrder = [
  "name",
  "email",
  "phone",
  "subject",
  "message",
  "privacyConsent",
] as const;

function getPayload(form: HTMLFormElement): ContactMessageInput {
  const data = new FormData(form);

  return {
    name: String(data.get("name") ?? ""),
    email: String(data.get("email") ?? ""),
    phone: String(data.get("phone") ?? ""),
    subject: String(data.get("subject") ?? ""),
    message: String(data.get("message") ?? ""),
    privacyConsent: data.get("privacyConsent") === "yes",
    botField: String(data.get("bot-field") ?? ""),
  };
}

type FieldErrorProps = {
  id: string;
  message?: string;
};

function FieldError({ id, message }: FieldErrorProps) {
  if (!message) {
    return null;
  }

  return (
    <p className="field-error" id={id}>
      {message}
    </p>
  );
}

export function ContactForm({
  defaultSubject = "Pertanyaan umum",
}: {
  defaultSubject?: ContactMessage["subject"];
}) {
  const formRef = useRef<HTMLFormElement>(null);
  const submissionLockRef = useRef(false);
  const [state, setState] = useState<SubmissionState>(initialState);
  const isPending = state.status === "pending";

  useEffect(() => {
    if (formRef.current) {
      formRef.current.noValidate = true;
    }

    const requestedTopic = new URLSearchParams(window.location.search).get(
      "topik",
    );

    if (requestedTopic === "produk") {
      const subjectField = formRef.current?.elements.namedItem("subject");

      if (subjectField instanceof HTMLSelectElement) {
        subjectField.value = "Informasi produk";
      }
    }
  }, []);

  useEffect(() => {
    if (state.status !== "validation-error") {
      return;
    }

    const firstInvalidField = focusableFieldOrder.find(
      (fieldName) => state.errors[fieldName],
    );
    const field = firstInvalidField
      ? formRef.current?.elements.namedItem(firstInvalidField)
      : null;

    if (
      field instanceof HTMLInputElement ||
      field instanceof HTMLSelectElement ||
      field instanceof HTMLTextAreaElement
    ) {
      field.focus();
    }
  }, [state]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (submissionLockRef.current) {
      return;
    }

    const form = event.currentTarget;
    const result = validateContactMessage(getPayload(form));

    if (!result.success) {
      setState({
        status: "validation-error",
        errors: result.errors,
      });
      return;
    }

    submissionLockRef.current = true;
    setState({ status: "pending", errors: {} });

    try {
      await contactGateway.submitContactMessage(result.data);
      formRef.current?.reset();
      setState({ status: "success", errors: {} });
    } catch {
      setState({
        status: "network-error",
        errors: {},
        message:
          "Pesan belum terkirim. Periksa koneksi Anda, lalu coba kembali.",
      });
    } finally {
      submissionLockRef.current = false;
    }
  }

  return (
    <form
      ref={formRef}
      name="kontak"
      method="POST"
      action="/__forms.html"
      data-netlify="true"
      data-netlify-honeypot="bot-field"
      onSubmit={handleSubmit}
      aria-busy={isPending}
      className="contact-form"
      aria-describedby="required-fields-note"
    >
      <input type="hidden" name="form-name" value="kontak" />

      <p id="required-fields-note" className="field-optional">
        Kolom bertanda <span aria-hidden="true">*</span> wajib diisi.
      </p>

      <p className="honeypot" aria-hidden="true">
        <label htmlFor="bot-field">
          Jangan isi bidang ini
          <input
            id="bot-field"
            name="bot-field"
            tabIndex={-1}
            autoComplete="off"
          />
        </label>
      </p>

      <div className="form-grid">
        <div className="field-group">
          <label htmlFor="name">
            Nama <span aria-hidden="true">*</span>
            <span className="sr-only"> (wajib)</span>
          </label>
          <input
            id="name"
            name="name"
            type="text"
            autoComplete="name"
            required
            minLength={contactFieldLimits.name.min}
            maxLength={contactFieldLimits.name.max}
            aria-invalid={Boolean(state.errors.name)}
            aria-describedby={state.errors.name ? "name-error" : undefined}
          />
          <FieldError id="name-error" message={state.errors.name} />
        </div>

        <div className="field-group">
          <label htmlFor="email">
            Email <span aria-hidden="true">*</span>
            <span className="sr-only"> (wajib)</span>
          </label>
          <input
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            required
            maxLength={contactFieldLimits.email.max}
            aria-invalid={Boolean(state.errors.email)}
            aria-describedby={state.errors.email ? "email-error" : undefined}
          />
          <FieldError id="email-error" message={state.errors.email} />
        </div>

        <div className="field-group">
          <label htmlFor="phone">
            Telepon <span className="field-optional">(opsional)</span>
          </label>
          <input
            id="phone"
            name="phone"
            type="tel"
            autoComplete="tel"
            inputMode="tel"
            minLength={contactFieldLimits.phone.min}
            maxLength={contactFieldLimits.phone.max}
            pattern={contactPhonePattern.source}
            aria-invalid={Boolean(state.errors.phone)}
            aria-describedby={state.errors.phone ? "phone-error" : undefined}
          />
          <FieldError id="phone-error" message={state.errors.phone} />
        </div>

        <div className="field-group">
          <label htmlFor="subject">
            Topik <span aria-hidden="true">*</span>
            <span className="sr-only"> (wajib)</span>
          </label>
          <select
            id="subject"
            name="subject"
            defaultValue={defaultSubject}
            required
            aria-invalid={Boolean(state.errors.subject)}
            aria-describedby={
              state.errors.subject ? "subject-error" : undefined
            }
          >
            {contactSubjectValues.map((subject) => (
              <option key={subject} value={subject}>
                {subject}
              </option>
            ))}
          </select>
          <FieldError id="subject-error" message={state.errors.subject} />
        </div>
      </div>

      <div className="field-group">
        <label htmlFor="message">
          Pesan <span aria-hidden="true">*</span>
          <span className="sr-only"> (wajib)</span>
        </label>
        <textarea
          id="message"
          name="message"
          rows={7}
          required
          minLength={contactFieldLimits.message.min}
          maxLength={contactFieldLimits.message.max}
          aria-invalid={Boolean(state.errors.message)}
          aria-describedby={state.errors.message ? "message-error" : undefined}
        />
        <FieldError id="message-error" message={state.errors.message} />
      </div>

      <div className="consent-group">
        <input
          id="privacyConsent"
          name="privacyConsent"
          type="checkbox"
          value="yes"
          required
          aria-invalid={Boolean(state.errors.privacyConsent)}
          aria-describedby={
            state.errors.privacyConsent ? "privacy-error" : undefined
          }
        />
        <label htmlFor="privacyConsent">
          Saya menyetujui penggunaan data untuk menjawab pesan ini sesuai{" "}
          <a href="/kebijakan-privasi">Kebijakan Privasi</a>.
          <span aria-hidden="true"> *</span>
          <span className="sr-only"> (wajib)</span>
        </label>
      </div>
      <FieldError
        id="privacy-error"
        message={state.errors.privacyConsent}
      />

      {state.status === "validation-error" ? (
        <p className="field-error" role="alert">
          Periksa kembali kolom yang ditandai sebelum mengirim pesan.
        </p>
      ) : null}

      <div className="form-actions">
        <button type="submit" disabled={isPending} className="button-link">
          {isPending ? "Mengirim..." : "Kirim pesan"}
        </button>
        <div className="form-status" aria-live="polite" aria-atomic="true">
          {state.status === "pending" ? (
            <p>Sedang mengirim pesan.</p>
          ) : null}
          {state.status === "success" ? (
            <p className="success-message">
              Pesan berhasil dikirim. Terima kasih telah menghubungi kami.
            </p>
          ) : null}
          {state.status === "network-error" ? (
            <p className="field-error" role="alert">
              {state.message}
            </p>
          ) : null}
        </div>
      </div>
    </form>
  );
}
