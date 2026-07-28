import {
  expect,
  test,
  type TestInfo,
} from "@playwright/test";

const publicRoutes = [
  "/",
  "/tauco",
  "/tentang-kami",
  "/produk",
  "/produk/tauco-cap-badak",
  "/kontak",
  "/kebijakan-privasi",
] as const;

const productionOrigin =
  process.env.G4_PRODUCTION_ORIGIN?.trim() ||
  "https://tauco-cap-badak.netlify.app";

function requirePreviewOrigin(): URL {
  const configuredUrl = process.env.PLAYWRIGHT_BASE_URL?.trim();

  if (!configuredUrl) {
    throw new Error(
      "PLAYWRIGHT_BASE_URL wajib diisi dengan origin Deploy Preview.",
    );
  }

  const url = new URL(configuredUrl);

  if (
    url.protocol !== "https:" ||
    url.pathname !== "/" ||
    url.search ||
    url.hash ||
    !url.hostname.endsWith(".netlify.app") ||
    !url.hostname.includes("--")
  ) {
    throw new Error(
      "G4.3 harus dijalankan terhadap origin Deploy Preview Netlify.",
    );
  }

  return url;
}

function normalizedUrl(value: string): string {
  return new URL(value).toString();
}

test.beforeAll(() => {
  const previewOrigin = requirePreviewOrigin().origin;
  const configuredProductionOrigin = new URL(productionOrigin);

  if (
    configuredProductionOrigin.protocol !== "https:" ||
    configuredProductionOrigin.origin === previewOrigin
  ) {
    throw new Error(
      "G4_PRODUCTION_ORIGIN harus berupa origin HTTPS production, bukan preview.",
    );
  }
});

test.describe("G4.3 page-level preview isolation", () => {
  for (const route of publicRoutes) {
    test(`${route} remains noindex with production canonical`, async ({
      page,
    }) => {
      const previewUrl = requirePreviewOrigin();
      const response = await page.goto(route, {
        waitUntil: "domcontentloaded",
      });

      expect(response?.status(), route).toBe(200);
      expect(
        response?.headers()["x-robots-tag"],
        `${route} harus mempunyai X-Robots-Tag`,
      ).toMatch(/noindex/i);

      await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
        "content",
        /noindex/i,
      );
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
        "content",
        /nofollow/i,
      );

      const canonical = await page
        .locator('link[rel="canonical"]')
        .getAttribute("href");
      const openGraphUrl = await page
        .locator('meta[property="og:url"]')
        .getAttribute("content");
      const expectedUrl = new URL(route, productionOrigin).toString();

      expect(canonical).not.toBeNull();
      expect(openGraphUrl).not.toBeNull();
      expect(normalizedUrl(canonical ?? "")).toBe(expectedUrl);
      expect(normalizedUrl(openGraphUrl ?? "")).toBe(expectedUrl);
      expect(canonical).not.toContain(previewUrl.hostname);
      expect(openGraphUrl).not.toContain(previewUrl.hostname);

      const source = (await response?.text()) ?? "";
      const normalizedSource = source.toLowerCase();

      expect(normalizedSource).not.toContain("localhost");
      expect(normalizedSource).not.toContain(".example");
      expect(normalizedSource).not.toContain(".invalid");
      expect(normalizedSource).not.toContain(previewUrl.hostname);
    });
  }
});

test("robots.txt blocks all preview crawling", async ({
  page,
  request,
}, testInfo: TestInfo) => {
  const response = await request.get("/robots.txt");
  const body = await response.text();

  expect(response.status()).toBe(200);
  expect(response.headers()["x-robots-tag"]).toMatch(/noindex/i);
  expect(body).toMatch(/User-Agent:\s*\*/i);
  expect(body).toMatch(/Disallow:\s*\/(?:\s|$)/i);
  expect(body).not.toMatch(/^Allow:\s*\//im);

  await page.goto("/robots.txt", { waitUntil: "domcontentloaded" });
  await testInfo.attach("preview-robots-txt", {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
});

test("sitemap contains only the seven production canonical URLs", async ({
  request,
}, testInfo: TestInfo) => {
  const response = await request.get("/sitemap.xml");
  const body = await response.text();
  const urls = Array.from(
    body.matchAll(/<loc>([^<]+)<\/loc>/g),
    (match) => match[1],
  );
  const expectedUrls = publicRoutes.map((route) =>
    new URL(route, productionOrigin).toString(),
  );
  const previewHostname = requirePreviewOrigin().hostname;

  expect(response.status()).toBe(200);
  expect(response.headers()["x-robots-tag"]).toMatch(/noindex/i);
  expect(new Set(urls).size).toBe(urls.length);
  expect(urls.sort()).toEqual(expectedUrls.sort());
  expect(body).not.toContain(previewHostname);
  expect(body.toLowerCase()).not.toContain("localhost");
  expect(body.toLowerCase()).not.toContain(".example");
  expect(body.toLowerCase()).not.toContain(".invalid");

  await testInfo.attach("preview-sitemap-evidence", {
    body: Buffer.from(
      JSON.stringify(
        {
          productionOrigin,
          previewOrigin: requirePreviewOrigin().origin,
          urls,
          xRobotsTag: response.headers()["x-robots-tag"],
        },
        null,
        2,
      ),
    ),
    contentType: "application/json",
  });
});
