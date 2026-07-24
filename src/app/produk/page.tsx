import type { Metadata } from "next";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { ButtonLink } from "@/components/button-link";
import { Container } from "@/components/container";
import { ProvisionalImage } from "@/components/provisional-image";
import { TextLink } from "@/components/text-link";
import { localContentSource } from "@/features/content";
import { createMetadata } from "@/lib/seo";

export async function generateMetadata(): Promise<Metadata> {
  const content = await localContentSource.listProducts();
  return createMetadata(content.metadata);
}

export default async function ProductsPage() {
  const content = await localContentSource.listProducts();

  return (
    <>
      <section className="catalog-header">
        <Container>
          <Breadcrumbs
            items={[
              { label: "Beranda", href: "/" },
              { label: "Produk", href: "/produk" },
            ]}
          />
          <div className="catalog-heading">
            <h1>{content.heading}</h1>
            <p>{content.description}</p>
          </div>
        </Container>
      </section>

      <section className="catalog-list" aria-label="Katalog produk">
        <Container>
          {content.products.length ? (
            <div className="catalog-products">
              {content.products.map((product) => (
                <article className="catalog-product" key={product.slug}>
                  <a
                    href={`/produk/${product.slug}`}
                    aria-label={`Lihat detail ${product.name}`}
                    className="catalog-image-link"
                  >
                    <ProvisionalImage
                      src={product.image.src}
                      alt={product.image.alt}
                      sizes="(max-width: 767px) 100vw, 58vw"
                      className="catalog-image"
                      imageClassName="aspect-[4/3]"
                    />
                  </a>
                  <div className="catalog-product-copy">
                    <p className="category-label">{product.category}</p>
                    <h2>
                      <a href={`/produk/${product.slug}`}>
                        {product.name}
                      </a>
                    </h2>
                    <p>{product.summary}</p>
                    <TextLink href={`/produk/${product.slug}`}>
                      Lihat detail produk
                    </TextLink>
                  </div>
                </article>
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <h2>Katalog sedang disiapkan</h2>
              <p>
                Informasi hanya akan ditampilkan setelah berhasil
                diverifikasi.
              </p>
            </div>
          )}

          <div className="catalog-contact">
            <p>Perlu informasi ukuran, harga, atau ketersediaan terbaru?</p>
            <ButtonLink href={content.contactLink.href}>
              {content.contactLink.label}
            </ButtonLink>
          </div>
        </Container>
      </section>
    </>
  );
}
