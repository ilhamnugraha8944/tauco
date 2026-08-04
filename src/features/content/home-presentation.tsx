import { ButtonLink } from "@/components/button-link";
import { Container } from "@/components/container";
import { ProvisionalImage } from "@/components/provisional-image";
import { TextLink } from "@/components/text-link";
import type { HomeContent, ProductCatalogContent } from "@/features/content/types";

export function HomePresentation({ content, catalog, adminPreview = false }: { content: HomeContent; catalog: ProductCatalogContent; adminPreview?: boolean }) {
  const featuredProduct = catalog.products.find((product) => content.featuredProductSlugs.includes(product.slug));
  return <>
    <section className="home-hero"><Container className="home-hero-grid">
      <div className="home-hero-copy"><p className="eyebrow">{content.hero.eyebrow}</p><h1>{content.hero.title}</h1><p className="hero-description">{content.hero.description}</p><div className="hero-actions">{content.hero.actions.map((action,index)=><ButtonLink key={action.href} href={action.href} variant={index===0?"primary":"secondary"}>{action.label}</ButtonLink>)}</div></div>
      <ProvisionalImage src={content.hero.image.src} alt={content.hero.image.alt} sizes="(max-width: 639px) calc(100vw - 3rem), (max-width: 1023px) calc(100vw - 4rem), 56vw" className="hero-figure" imageClassName="aspect-[3/2]" preload adminPreview={adminPreview}/>
    </Container></section>
    <section className="intro-section" id={content.introduction.id}><Container className="intro-grid"><h2>{content.introduction.heading}</h2><div className="prose-large">{content.introduction.paragraphs.map((paragraph)=><p key={paragraph}>{paragraph}</p>)}</div></Container></section>
    {featuredProduct?<section className="featured-product"><Container className="featured-product-grid"><div className="featured-product-copy"><p className="eyebrow">Produk utama</p><h2>{featuredProduct.name}</h2><p>{featuredProduct.summary}</p><dl className="compact-facts">{featuredProduct.facts.map((fact)=><div key={fact.label}><dt>{fact.label}</dt><dd>{fact.value}</dd></div>)}</dl><TextLink href={`/produk/${featuredProduct.slug}`}>Lihat detail produk</TextLink></div></Container></section>:<section className="empty-state"><Container><h2>Informasi produk sedang disiapkan</h2><p>Silakan hubungi kami untuk mendapatkan keterangan terbaru.</p><TextLink href="/kontak">Kontak</TextLink></Container></section>}
    <section className="guide-preview"><Container><div className="guide-preview-inner"><h2>{content.guidePreview.heading}</h2><p>{content.guidePreview.description}</p><TextLink href={content.guidePreview.link.href}>{content.guidePreview.link.label}</TextLink></div></Container></section>
    <section className="about-preview"><Container className="about-preview-grid"><div><h2>{content.aboutPreview.heading}</h2><p>{content.aboutPreview.description}</p><TextLink href={content.aboutPreview.link.href}>{content.aboutPreview.link.label}</TextLink></div></Container></section>
    <section className="contact-cta"><Container><div className="contact-cta-inner"><h2>Butuh informasi produk?</h2><p>Sampaikan pertanyaan mengenai produk atau peluang kerja sama melalui formulir kontak.</p><ButtonLink href="/kontak" variant="primary">Hubungi kami</ButtonLink></div></Container></section>
  </>;
}
