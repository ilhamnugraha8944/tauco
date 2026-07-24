import type { Metadata } from "next";

import { ButtonLink } from "@/components/button-link";
import { Container } from "@/components/container";
import { JsonLd } from "@/components/json-ld";
import { PageHero } from "@/components/page-hero";
import { SourceList } from "@/components/source-list";
import { localContentSource } from "@/features/content";
import { absoluteUrl, siteConfig } from "@/lib/site";
import { createMetadata } from "@/lib/seo";

export async function generateMetadata(): Promise<Metadata> {
  const content = await localContentSource.getTaucoGuide();
  return createMetadata(content.metadata);
}

export default async function TaucoGuidePage() {
  const content = await localContentSource.getTaucoGuide();
  const publishedDate = new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone: "Asia/Jakarta",
  }).format(new Date(`${content.publishedAt}T00:00:00+07:00`));

  return (
    <>
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "Article",
          headline: content.metadata.title,
          description: content.metadata.description,
          image: absoluteUrl(content.metadata.openGraphImage.src),
          datePublished: content.publishedAt,
          dateModified: content.updatedAt,
          inLanguage: siteConfig.language,
          mainEntityOfPage: absoluteUrl("/tauco"),
          author: {
            "@type": "Organization",
            name: siteConfig.name,
          },
          publisher: {
            "@type": "Organization",
            name: siteConfig.name,
          },
          citation: content.sources.map((source) => source.url),
        }}
      />
      <PageHero
        eyebrow={content.hero.eyebrow}
        title={content.hero.title}
        description={content.hero.description}
        image={content.hero.image}
        breadcrumbs={[
          { label: "Beranda", href: "/" },
          { label: "Mengenal Tauco", href: "/tauco" },
        ]}
      />

      <Container className="article-layout">
        <aside className="article-toc">
          <p>Isi panduan</p>
          <nav aria-label="Daftar isi panduan tauco">
            <ol>
              {content.sections.map((section) => (
                <li key={section.id}>
                  <a href={`#${section.id}`}>{section.heading}</a>
                </li>
              ))}
            </ol>
          </nav>
        </aside>

        <article className="article-content">
          <p className="article-meta">
            Diterbitkan oleh {siteConfig.name} pada{" "}
            <time dateTime={content.publishedAt}>{publishedDate}</time>
          </p>
          {content.sections.map((section) => (
            <section id={section.id} key={section.id}>
              <h2>{section.heading}</h2>
              {section.paragraphs.map((paragraph) => (
                <p key={paragraph}>{paragraph}</p>
              ))}
            </section>
          ))}

          <div className="related-actions">
            {content.relatedLinks.map((link) => (
              <ButtonLink key={link.href} href={link.href} variant="secondary">
                {link.label}
              </ButtonLink>
            ))}
          </div>

          <SourceList sources={content.sources} />
        </article>
      </Container>
    </>
  );
}
