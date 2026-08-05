import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { JsonLd } from "@/components/json-ld";
import { localContentSource } from "@/features/content";
import { ProductPresentation } from "@/features/content/product-presentation";
import { createMetadata } from "@/lib/seo";

type ProductPageProps = {
  params: Promise<{ slug: string }>;
};

export const dynamicParams = false;

export async function generateStaticParams() {
  const catalog = await localContentSource.listProducts();

  return catalog.products.map((product) => ({
    slug: product.slug,
  }));
}

export async function generateMetadata({
  params,
}: ProductPageProps): Promise<Metadata> {
  const { slug } = await params;
  const product = await localContentSource.getProductBySlug(slug);

  if (!product) {
    return {};
  }

  return createMetadata(product.metadata);
}

export default async function ProductDetailPage({ params }: ProductPageProps) {
  const { slug } = await params;
  const product = await localContentSource.getProductBySlug(slug);

  if (!product) {
    notFound();
  }

  return (
    <>
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "Product",
          name: product.name,
          description: product.summary,
          category: product.category,
          brand: {
            "@type": "Brand",
            name: "Tauco Cap Badak",
          },
        }}
      />
      <ProductPresentation product={product} />
    </>
  );
}
