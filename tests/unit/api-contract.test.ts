import { describe, expect, it } from "vitest";

import aboutFixture from "../../contracts/fixtures/about.success.json";
import contactMessageFixture from "../../contracts/fixtures/contact-message.request.json";
import contactMessageSuccessFixture from "../../contracts/fixtures/contact-message.success.json";
import homeFixture from "../../contracts/fixtures/home.success.json";
import productDetailFixture from "../../contracts/fixtures/product-detail.success.json";
import productsListFixture from "../../contracts/fixtures/products-list.success.json";
import taucoGuideFixture from "../../contracts/fixtures/tauco-guide.success.json";
import validationProblemFixture from "../../contracts/fixtures/validation.problem.json";
import {
  aboutApiResponseSchema,
  aboutContentSchema,
  apiProblemDetailsSchema,
  homeApiResponseSchema,
  homeContentSchema,
  LocalContentSource,
  productCatalogContentSchema,
  productDetailApiResponseSchema,
  productDetailSchema,
  productsApiResponseSchema,
  taucoGuideApiResponseSchema,
  taucoGuideContentSchema,
} from "../../src/features/content";
import {
  contactApiRequestSchema,
  contactApiResponseSchema,
  contactMessageSchema,
} from "../../src/features/contact";

describe("API v1 frontend parity contract", () => {
  it("parses every committed success fixture", () => {
    expect(homeApiResponseSchema.safeParse(homeFixture).success).toBe(true);
    expect(aboutApiResponseSchema.safeParse(aboutFixture).success).toBe(true);
    expect(
      taucoGuideApiResponseSchema.safeParse(taucoGuideFixture).success,
    ).toBe(true);
    expect(productsApiResponseSchema.safeParse(productsListFixture).success).toBe(
      true,
    );
    expect(
      productDetailApiResponseSchema.safeParse(productDetailFixture).success,
    ).toBe(true);
  });

  it("keeps fixture data compatible with every Phase 1A content contract", () => {
    expect(homeContentSchema.safeParse(homeFixture.data).success).toBe(true);
    expect(aboutContentSchema.safeParse(aboutFixture.data).success).toBe(true);
    expect(
      taucoGuideContentSchema.safeParse(taucoGuideFixture.data).success,
    ).toBe(true);
    expect(
      productCatalogContentSchema.safeParse(productsListFixture.data).success,
    ).toBe(true);
    expect(
      productDetailSchema.safeParse(productDetailFixture.data).success,
    ).toBe(true);
  });

  it("matches the current LocalContentSource output without content drift", async () => {
    const source = new LocalContentSource();
    const [home, about, taucoGuide, products, productDetail] =
      await Promise.all([
        source.getHome(),
        source.getAbout(),
        source.getTaucoGuide(),
        source.listProducts(),
        source.getProductBySlug("tauco-cap-badak"),
      ]);

    expect(homeFixture.data).toEqual(home);
    expect(aboutFixture.data).toEqual(about);
    expect(taucoGuideFixture.data).toEqual(taucoGuide);
    expect(productsListFixture.data).toEqual(products);
    expect(productDetailFixture.data).toEqual(productDetail);
  });

  it("keeps product list entries limited to summary fields", () => {
    const product = productsListFixture.data.products[0];

    expect(product).not.toHaveProperty("status");
    expect(product).not.toHaveProperty("metadata");
    expect(product).not.toHaveProperty("description");
    expect(product).not.toHaveProperty("usageSuggestions");
    expect(product).not.toHaveProperty("priceEstimate");
    expect(product).not.toHaveProperty("purchaseNote");
    expect(product).not.toHaveProperty("contactLink");
    expect(product).not.toHaveProperty("researchEvidence");
  });

  it("keeps product detail free from repository publication status", () => {
    expect(productDetailFixture.data).not.toHaveProperty("status");
    expect(productDetailSchema.safeParse(productDetailFixture.data).success).toBe(
      true,
    );
  });

  it("parses the committed RFC 7807 validation problem", () => {
    expect(
      apiProblemDetailsSchema.safeParse(validationProblemFixture).success,
    ).toBe(true);
  });

  it("keeps the shared contact request fixture compatible with the form contract", () => {
    expect(contactMessageSchema.safeParse(contactMessageFixture).success).toBe(
      true,
    );
    expect(
      contactApiRequestSchema.safeParse(contactMessageFixture).success,
    ).toBe(true);
    expect(
      contactApiResponseSchema.safeParse(contactMessageSuccessFixture).success,
    ).toBe(true);
  });

  it("normalizes form input before enforcing the canonical API payload", () => {
    const paddedInput = {
      ...contactMessageFixture,
      name: `  ${contactMessageFixture.name}  `,
      email: ` ${contactMessageFixture.email} `,
      message: `  ${contactMessageFixture.message}  `,
    };

    const normalized = contactMessageSchema.parse(paddedInput);
    expect(normalized.name).toBe(contactMessageFixture.name);
    expect(normalized.email).toBe(contactMessageFixture.email);
    expect(normalized.message).toBe(contactMessageFixture.message);
    expect(contactApiRequestSchema.safeParse(paddedInput).success).toBe(false);
    expect(contactApiRequestSchema.safeParse(normalized).success).toBe(true);
  });

  it("rejects malformed or extended success metadata", () => {
    const unsupportedVersion = {
      ...homeFixture,
      meta: {
        ...homeFixture.meta,
        apiVersion: "v2",
      },
    };
    const unknownMetaField = {
      ...homeFixture,
      meta: {
        ...homeFixture.meta,
        internalTrace: "not-public",
      },
    };
    const unknownEnvelopeField = {
      ...homeFixture,
      debug: true,
    };
    const duplicateFeaturedProduct = {
      ...homeFixture,
      data: {
        ...homeFixture.data,
        featuredProductSlugs: [
          homeFixture.data.featuredProductSlugs[0],
          homeFixture.data.featuredProductSlugs[0],
        ],
      },
    };

    expect(homeApiResponseSchema.safeParse(unsupportedVersion).success).toBe(
      false,
    );
    expect(homeApiResponseSchema.safeParse(unknownMetaField).success).toBe(
      false,
    );
    expect(homeApiResponseSchema.safeParse(unknownEnvelopeField).success).toBe(
      false,
    );
    expect(
      homeApiResponseSchema.safeParse(duplicateFeaturedProduct).success,
    ).toBe(false);
  });

  it("rejects malformed, inconsistent, or extended page metadata", () => {
    const signedCursorPage = {
      ...productsListFixture,
      meta: {
        ...productsListFixture.meta,
        page: {
          nextCursor: "eyJ2ZXJzaW9uIjoxfQ.signature_base64url",
          hasMore: true,
          limit: 20,
        },
      },
    };
    const invalidLimit = {
      ...productsListFixture,
      meta: {
        ...productsListFixture.meta,
        page: {
          ...productsListFixture.meta.page,
          limit: 51,
        },
      },
    };
    const missingNextCursor = {
      ...productsListFixture,
      meta: {
        ...productsListFixture.meta,
        page: {
          ...productsListFixture.meta.page,
          hasMore: true,
        },
      },
    };
    const unexpectedNextCursor = {
      ...productsListFixture,
      meta: {
        ...productsListFixture.meta,
        page: {
          ...productsListFixture.meta.page,
          nextCursor: "opaque_cursor",
        },
      },
    };
    const unknownPageField = {
      ...productsListFixture,
      meta: {
        ...productsListFixture.meta,
        page: {
          ...productsListFixture.meta.page,
          offset: 0,
        },
      },
    };
    const oversizedPage = {
      ...productsListFixture,
      data: {
        ...productsListFixture.data,
        products: Array.from(
          { length: 51 },
          () => productsListFixture.data.products[0],
        ),
      },
    };

    expect(productsApiResponseSchema.safeParse(signedCursorPage).success).toBe(
      true,
    );
    expect(productsApiResponseSchema.safeParse(invalidLimit).success).toBe(
      false,
    );
    expect(productsApiResponseSchema.safeParse(missingNextCursor).success).toBe(
      false,
    );
    expect(
      productsApiResponseSchema.safeParse(unexpectedNextCursor).success,
    ).toBe(false);
    expect(productsApiResponseSchema.safeParse(unknownPageField).success).toBe(
      false,
    );
    expect(productsApiResponseSchema.safeParse(oversizedPage).success).toBe(
      false,
    );
  });

  it("rejects malformed or extended RFC 7807 problems", () => {
    const successStatus = {
      ...validationProblemFixture,
      status: 200,
    };
    const emptyErrors = {
      ...validationProblemFixture,
      errors: [],
    };
    const unsafeRequestId = {
      ...validationProblemFixture,
      requestId: "visitor@example.com",
    };
    const unknownField = {
      ...validationProblemFixture,
      stack: "must never be public",
    };

    expect(apiProblemDetailsSchema.safeParse(successStatus).success).toBe(false);
    expect(apiProblemDetailsSchema.safeParse(emptyErrors).success).toBe(false);
    expect(apiProblemDetailsSchema.safeParse(unsafeRequestId).success).toBe(
      false,
    );
    expect(apiProblemDetailsSchema.safeParse(unknownField).success).toBe(false);
  });

  it("accepts bounded escaped problem paths and rejects unsafe references", () => {
    for (const instance of [
      "/API/V1/Missing",
      "/api/v1/missing/",
      "/API/V1/Missing%20Product/",
    ]) {
      expect(
        apiProblemDetailsSchema.safeParse({
          ...validationProblemFixture,
          instance,
        }).success,
      ).toBe(true);
    }

    for (const instance of [
      "relative/path",
      "//external.example/path",
      "/api/v1/raw path",
      `/${"a".repeat(2048)}`,
    ]) {
      expect(
        apiProblemDetailsSchema.safeParse({
          ...validationProblemFixture,
          instance,
        }).success,
      ).toBe(false);
    }
  });
});
