import { describe, expect, it } from "vitest";

import {
  parseSiteOrigin,
  resolveSiteEnvironment,
} from "@/lib/site-origin";

function environment(
  values: Record<string, string | undefined> = {},
): NodeJS.ProcessEnv {
  return {
    NODE_ENV: "test",
    ...values,
  };
}

describe("public site origin guard", () => {
  it.each([
    "https://tauco-cap-badak.netlify.app",
    "https://www.taucocapbadak.id",
    "https://catalog.taucocapbadak.id:8443",
  ])("accepts a public HTTPS origin: %s", (value) => {
    expect(
      parseSiteOrigin(value, { requirePublicHttps: true }).origin,
    ).toBe(value);
  });

  it.each([
    "http://tauco-cap-badak.netlify.app",
    "https://localhost",
    "https://shop.localhost",
    "https://127.0.0.1",
    "https://192.168.1.10",
    "https://[::1]",
    "https://tauco.local",
    "https://tauco.internal",
    "https://tauco.test",
    "https://tauco.invalid",
    "https://tauco.example",
    "https://-tauco.example.com",
    "https://tauco_.example.com",
    "https://intranet",
  ])("rejects a non-public production origin: %s", (value) => {
    expect(() =>
      parseSiteOrigin(value, { requirePublicHttps: true }),
    ).toThrow(
      "NEXT_PUBLIC_SITE_URL wajib berupa origin HTTPS dengan hostname publik.",
    );
  });
});

describe("site environment matrix", () => {
  it("keeps ordinary local development non-indexable", () => {
    const result = resolveSiteEnvironment(environment());

    expect(result).toMatchObject({
      kind: "local",
      isNetlify: false,
      isNetlifyProduction: false,
      isSeoAudit: false,
      isIndexable: false,
      requiresPublicHttps: false,
    });
    expect(result.siteUrl.origin).toBe("http://localhost:3000");
  });

  it("enables indexability only for a local loopback SEO audit", () => {
    const result = resolveSiteEnvironment(
      environment({
        SEO_AUDIT_INDEXABLE: "true",
        NEXT_PUBLIC_SITE_URL: "http://127.0.0.1:43127",
      }),
    );

    expect(result).toMatchObject({
      kind: "seo-audit",
      isNetlify: false,
      isSeoAudit: true,
      isIndexable: true,
    });
    expect(result.siteUrl.origin).toBe("http://127.0.0.1:43127");
  });

  it("does not let the audit flag index a public non-Netlify build", () => {
    const result = resolveSiteEnvironment(
      environment({
        SEO_AUDIT_INDEXABLE: "true",
        NEXT_PUBLIC_SITE_URL: "https://tauco-cap-badak.netlify.app",
      }),
    );

    expect(result).toMatchObject({
      kind: "local",
      isSeoAudit: false,
      isIndexable: false,
    });
  });

  it("makes only Netlify production indexable", () => {
    const result = resolveSiteEnvironment(
      environment({
        NETLIFY: "true",
        CONTEXT: "production",
        NEXT_PUBLIC_SITE_URL: "https://tauco-cap-badak.netlify.app",
      }),
    );

    expect(result).toMatchObject({
      kind: "netlify-production",
      netlifyContext: "production",
      isNetlify: true,
      isNetlifyProduction: true,
      isSeoAudit: false,
      isIndexable: true,
      requiresPublicHttps: true,
    });
  });

  it.each(["deploy-preview", "branch-deploy", "dev", "unknown"])(
    "keeps Netlify %s non-indexable",
    (context) => {
      const result = resolveSiteEnvironment(
        environment({
          NETLIFY: "true",
          CONTEXT: context,
          NEXT_PUBLIC_SITE_URL:
            "https://tauco-cap-badak.netlify.app",
        }),
      );

      expect(result).toMatchObject({
        kind: "netlify-non-production",
        netlifyContext: context,
        isNetlify: true,
        isNetlifyProduction: false,
        isSeoAudit: false,
        isIndexable: false,
        requiresPublicHttps: true,
      });
    },
  );

  it("does not let the audit flag open a Netlify preview", () => {
    const result = resolveSiteEnvironment(
      environment({
        NETLIFY: "true",
        CONTEXT: "deploy-preview",
        SEO_AUDIT_INDEXABLE: "true",
        NEXT_PUBLIC_SITE_URL: "https://tauco-cap-badak.netlify.app",
      }),
    );

    expect(result).toMatchObject({
      kind: "netlify-non-production",
      isSeoAudit: false,
      isIndexable: false,
    });
  });

  it.each(["production", "deploy-preview", "branch-deploy"])(
    "requires an explicit public origin for Netlify %s",
    (context) => {
      expect(() =>
        resolveSiteEnvironment(
          environment({
            NETLIFY: "true",
            CONTEXT: context,
          }),
        ),
      ).toThrow(
        "NEXT_PUBLIC_SITE_URL wajib diisi untuk setiap deploy Netlify.",
      );
    },
  );

  it.each(["production", "deploy-preview", "branch-deploy"])(
    "rejects a local origin for Netlify %s",
    (context) => {
      expect(() =>
        resolveSiteEnvironment(
          environment({
            NETLIFY: "true",
            CONTEXT: context,
            NEXT_PUBLIC_SITE_URL: "http://localhost:3000",
          }),
        ),
      ).toThrow(
        "NEXT_PUBLIC_SITE_URL wajib berupa origin HTTPS dengan hostname publik.",
      );
    },
  );
});
