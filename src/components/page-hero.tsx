import { Breadcrumbs, type BreadcrumbItem } from "@/components/breadcrumbs";
import { Container } from "@/components/container";
import { ProvisionalImage } from "@/components/provisional-image";
import type { ImageAsset } from "@/features/content";

type PageHeroProps = {
  eyebrow: string;
  title: string;
  description: string;
  image: ImageAsset;
  breadcrumbs: BreadcrumbItem[];
  adminPreview?: boolean;
};

export function PageHero({
  eyebrow,
  title,
  description,
  image,
  breadcrumbs,
  adminPreview = false,
}: PageHeroProps) {
  return (
    <section className="page-hero">
      <Container>
        <Breadcrumbs items={breadcrumbs} />
        <div className="page-hero-grid">
          <div className="page-hero-copy">
            <p className="eyebrow">{eyebrow}</p>
            <h1>{title}</h1>
            <p className="hero-description">{description}</p>
          </div>
          <ProvisionalImage
            src={image.src}
            alt={image.alt}
            sizes="(max-width: 639px) calc(100vw - 3rem), (max-width: 1023px) calc(100vw - 4rem), 46vw"
            className="page-hero-image"
            imageClassName="aspect-[4/3]"
            preload
            adminPreview={adminPreview}
          />
        </div>
      </Container>
    </section>
  );
}
