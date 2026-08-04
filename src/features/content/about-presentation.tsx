import { ButtonLink } from "@/components/button-link";
import { Container } from "@/components/container";
import { PageHero } from "@/components/page-hero";
import { SourceList } from "@/components/source-list";
import type { AboutContent } from "@/features/content/types";

export function AboutPresentation({ content, adminPreview = false }: { content: AboutContent; adminPreview?: boolean }) {
  return <><PageHero eyebrow={content.hero.eyebrow} title={content.hero.title} description={content.hero.description} image={content.hero.image} breadcrumbs={[{label:"Beranda",href:"/"},{label:"Tentang Kami",href:"/tentang-kami"}]} adminPreview={adminPreview}/><Container className="narrative-layout"><article className="narrative-content">{content.sections.map((section)=><section id={section.id} key={section.id}><h2>{section.heading}</h2>{section.paragraphs.map((paragraph)=><p key={paragraph}>{paragraph}</p>)}</section>)}<div className="related-actions">{content.relatedLinks.map((link)=><ButtonLink key={link.href} href={link.href} variant="secondary">{link.label}</ButtonLink>)}</div><SourceList sources={content.sources}/></article></Container></>;
}
