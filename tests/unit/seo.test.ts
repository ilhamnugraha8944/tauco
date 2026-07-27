import { describe, expect, it } from "vitest";

import {
  localContentSource,
  seoMetadataSchema,
  type SeoMetadata,
} from "@/features/content";
import { absoluteUrl } from "@/lib/site";
import { parseSiteOrigin } from "@/lib/site-origin";
import { createMetadata, staticPageMetadata } from "@/lib/seo";

async function getPublicRouteMetadata(): Promise<
  Array<{ route: string; metadata: SeoMetadata }>
> {
  const [home, about, taucoGuide, productCatalog] =
    await Promise.all([
      localContentSource.getHome(),
      localContentSource.getAbout(),
      localContentSource.getTaucoGuide(),
      localContentSource.listProducts(),
    ]);
  const productMetadata = await Promise.all(
    productCatalog.products.map(async (product) => {
      const detail = await localContentSource.getProductBySlug(
        product.slug,
      );

      if (!detail) {
        throw new Error(
          `Detail produk published tidak ditemukan: ${product.slug}`,
        );
      }

      return {
        route: `/produk/${product.slug}`,
        metadata: detail.metadata,
      };
    }),
  );

  return [
    { route: "/", metadata: home.metadata },
    { route: "/tauco", metadata: taucoGuide.metadata },
    { route: "/tentang-kami", metadata: about.metadata },
    { route: "/produk", metadata: productCatalog.metadata },
    ...productMetadata,
    { route: "/kontak", metadata: staticPageMetadata.contact },
    {
      route: "/kebijakan-privasi",
      metadata: staticPageMetadata.privacy,
    },
  ];
}

describe("SEO helpers", () => {
  it("creates absolute URLs from one configured origin", () => {
    expect(absoluteUrl("/tauco")).toBe("http://localhost:3000/tauco");
  });

  it.each([
    ["http://localhost:3000", "http://localhost:3000/"],
    ["https://tauco-cap-badak.example", "https://tauco-cap-badak.example/"],
    ["https://subdomain.example:8443", "https://subdomain.example:8443/"],
  ])("accepts an origin-only site URL: %s", (value, expected) => {
    expect(parseSiteOrigin(value).toString()).toBe(expected);
  });

  it.each([
    ["credentials", "https://user:password@example.com"],
    ["path", "https://example.com/tauco"],
    ["query", "https://example.com?preview=true"],
    ["hash", "https://example.com#tauco"],
  ])("rejects a site URL containing %s", (_case, value) => {
    expect(() => parseSiteOrigin(value)).toThrow(
      "NEXT_PUBLIC_SITE_URL harus berupa origin tanpa path, query, hash, atau kredensial.",
    );
  });

  it("maps content metadata without changing the canonical path", async () => {
    const home = await localContentSource.getHome();
    const metadata = createMetadata(home.metadata);

    expect(metadata.title).toBe(home.metadata.title);
    expect(metadata.description).toBe(home.metadata.description);
    expect(metadata.alternates).toEqual({
      canonical: "/",
    });
  });

  it("keeps titles, descriptions, and canonicals unique across public routes", async () => {
    const entries = await getPublicRouteMetadata();

    expect(entries.map(({ route }) => route)).toEqual([
      "/",
      "/tauco",
      "/tentang-kami",
      "/produk",
      "/produk/tauco-cap-badak",
      "/kontak",
      "/kebijakan-privasi",
    ]);

    for (const key of [
      "title",
      "description",
      "canonicalPath",
    ] as const) {
      const values = entries.map(({ metadata }) => metadata[key]);
      expect(new Set(values).size, `${key} harus unik`).toBe(
        values.length,
      );
    }

    entries.forEach(({ route, metadata }) => {
      expect(metadata.canonicalPath).toBe(route);
      expect(seoMetadataSchema.safeParse(metadata).success).toBe(true);
    });
  });

  it("validates static contact and privacy metadata with the shared schema", () => {
    expect(
      seoMetadataSchema.safeParse(staticPageMetadata.contact).success,
    ).toBe(true);
    expect(
      seoMetadataSchema.safeParse(staticPageMetadata.privacy).success,
    ).toBe(true);
    expect(staticPageMetadata.contact.canonicalPath).toBe("/kontak");
    expect(staticPageMetadata.privacy.canonicalPath).toBe(
      "/kebijakan-privasi",
    );
  });

  it("marks every public route as noindex and nofollow locally", async () => {
    const entries = await getPublicRouteMetadata();

    entries.forEach(({ route, metadata: contentMetadata }) => {
      const metadata = createMetadata(contentMetadata);

      expect(metadata.robots, route).toMatchObject({
        index: false,
        follow: false,
        nocache: true,
      });
    });
  });
});
