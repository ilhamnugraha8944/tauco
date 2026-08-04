import type { Metadata } from "next";

import { JsonLd } from "@/components/json-ld";
import { localContentSource } from "@/features/content";
import { HomePresentation } from "@/features/content/home-presentation";
import { absoluteUrl, siteConfig } from "@/lib/site";
import { createMetadata } from "@/lib/seo";

export async function generateMetadata(): Promise<Metadata> {
  const content = await localContentSource.getHome();
  return createMetadata(content.metadata);
}

export default async function HomePage() {
  const [content, catalog] = await Promise.all([
    localContentSource.getHome(),
    localContentSource.listProducts(),
  ]);

  return (
    <>
      <JsonLd
        data={[
          {
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: siteConfig.name,
            url: absoluteUrl("/"),
            inLanguage: siteConfig.language,
          },
          {
            "@context": "https://schema.org",
            "@type": "Organization",
            name: siteConfig.name,
            url: absoluteUrl("/"),
          },
        ]}
      />

      <HomePresentation content={content} catalog={catalog} />
    </>
  );
}
