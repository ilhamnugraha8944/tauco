import type { Metadata } from "next";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { ContactForm } from "@/components/contact-form";
import { Container } from "@/components/container";
import { createMetadata, staticPageMetadata } from "@/lib/seo";

export const metadata: Metadata = createMetadata(
  staticPageMetadata.contact,
);

export default function ContactPage() {
  return (
    <section className="contact-page">
      <Container>
        <Breadcrumbs
          items={[
            { label: "Beranda", href: "/" },
            { label: "Kontak", href: "/kontak" },
          ]}
        />
        <div className="contact-page-grid">
          <div className="contact-copy">
            <p className="eyebrow">Kontak</p>
            <h1>Apa yang ingin Anda tanyakan?</h1>
            <p>
              Gunakan formulir ini untuk menanyakan informasi produk atau
              menyampaikan peluang kerja sama.
            </p>
          </div>
          <ContactForm />
        </div>
      </Container>
    </section>
  );
}
