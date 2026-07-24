import AxeBuilder from "@axe-core/playwright";
import {
  expect,
  test,
  type Page,
  type TestInfo,
} from "@playwright/test";

const primaryRoutes = [
  "/",
  "/tauco",
  "/tentang-kami",
  "/produk",
  "/produk/tauco-cap-badak",
  "/kontak",
  "/kebijakan-privasi",
] as const;

function desktopOnly(testInfo: TestInfo) {
  test.skip(
    testInfo.project.name !== "chromium-desktop",
    "Cukup dijalankan pada satu project browser.",
  );
}

async function readStructuredData(page: Page) {
  const scripts = await page
    .locator('script[type="application/ld+json"]')
    .allTextContents();

  return scripts.flatMap((script) => {
    const parsed = JSON.parse(script);
    return Array.isArray(parsed) ? parsed : [parsed];
  });
}

test.describe("SEO and platform contracts", () => {
  test("every internal link from every public route resolves", async ({
    page,
    request,
  }, testInfo) => {
    desktopOnly(testInfo);
    const hrefs = new Set<string>();

    for (const route of primaryRoutes) {
      await page.goto(route);
      const routeHrefs = await page
        .locator('a[href^="/"]')
        .evaluateAll((links) =>
          links
            .map((link) => link.getAttribute("href"))
            .filter((href): href is string => Boolean(href)),
        );

      routeHrefs.forEach((href) => hrefs.add(href));
    }

    for (const href of hrefs) {
      const response = await request.get(href);
      expect(response.status(), `${href} should resolve`).toBeLessThan(400);
    }
  });

  test("every interior route exposes BreadcrumbList JSON-LD", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    for (const route of primaryRoutes.slice(1)) {
      await page.goto(route);
      const structuredData = await readStructuredData(page);

      expect(
        structuredData.some(
          (entry) => entry["@type"] === "BreadcrumbList",
        ),
        `${route} harus memiliki BreadcrumbList`,
      ).toBe(true);
    }
  });

  test("article dates, citations, and visible publication date agree", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);
    await page.goto("/tauco");

    const structuredData = await readStructuredData(page);
    const article = structuredData.find(
      (entry) => entry["@type"] === "Article",
    );
    const visibleDate = page.locator("time");

    expect(article).toBeTruthy();
    expect(article.datePublished).toBe(
      await visibleDate.getAttribute("datetime"),
    );
    expect(article.dateModified).toBe("2026-07-24");
    expect(article.citation).toHaveLength(3);
    await expect(visibleDate).toContainText("24 Juli 2026");
  });

  test("product schema omits unverified image and commerce fields", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);
    await page.goto("/produk/tauco-cap-badak");

    const structuredData = await readStructuredData(page);
    const product = structuredData.find(
      (entry) => entry["@type"] === "Product",
    );

    expect(product).toBeTruthy();
    expect(product).not.toHaveProperty("image");
    expect(product).not.toHaveProperty("offers");
    expect(product).not.toHaveProperty("review");
    expect(product).not.toHaveProperty("aggregateRating");
  });

  test("sitemap contains the exact unique public URL set", async ({
    request,
  }, testInfo) => {
    desktopOnly(testInfo);
    const response = await request.get("/sitemap.xml");
    const body = await response.text();
    const urls = Array.from(
      body.matchAll(/<loc>([^<]+)<\/loc>/g),
      (match) => match[1],
    );
    const expected = primaryRoutes.map(
      (route) =>
        `http://localhost:3000${route}`,
    );

    expect(new Set(urls).size).toBe(urls.length);
    expect(urls.sort()).toEqual(expected.sort());
  });

  test("security headers are present and framework signature is hidden", async ({
    request,
  }, testInfo) => {
    desktopOnly(testInfo);
    const response = await request.get("/");

    expect(response.headers()["x-content-type-options"]).toBe("nosniff");
    expect(response.headers()["x-frame-options"]).toBe("SAMEORIGIN");
    expect(response.headers()["referrer-policy"]).toBe(
      "strict-origin-when-cross-origin",
    );
    expect(response.headers()["permissions-policy"]).toContain(
      "camera=()",
    );
    expect(response.headers()["x-powered-by"]).toBeUndefined();
  });
});

test.describe("WCAG and responsive pre-flight", () => {
  for (const route of primaryRoutes) {
    test(`${route} has no WCAG A or AA axe violations`, async ({
      page,
    }) => {
      await page.goto(route);
      const results = await new AxeBuilder({ page })
        .withTags([
          "wcag2a",
          "wcag2aa",
          "wcag21a",
          "wcag21aa",
          "wcag22aa",
        ])
        .analyze();

      expect(results.violations).toEqual([]);
    });
  }

  test("public pages do not overflow at narrow and tablet widths", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    for (const viewport of [
      { width: 320, height: 568 },
      { width: 768, height: 844 },
    ]) {
      await page.setViewportSize(viewport);

      for (const route of primaryRoutes) {
        await page.goto(route);
        const dimensions = await page.evaluate(() => ({
          clientWidth: document.documentElement.clientWidth,
          scrollWidth: document.documentElement.scrollWidth,
        }));

        expect(dimensions.scrollWidth, `${route} pada ${viewport.width}px`).toBe(
          dimensions.clientWidth,
        );
      }
    }
  });

  test("long hero and product hierarchy stay above the fold", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    for (const width of [390, 1440]) {
      await page.setViewportSize({ width, height: 844 });
      await page.goto("/tentang-kami");
      const lineMetrics = await page.locator("h1").evaluate((heading) => {
        const styles = getComputedStyle(heading);
        return {
          height: heading.getBoundingClientRect().height,
          lineHeight: Number.parseFloat(styles.lineHeight),
        };
      });

      expect(
        lineMetrics.height / lineMetrics.lineHeight,
        `Judul Tentang Kami pada ${width}px`,
      ).toBeLessThanOrEqual(2.1);
    }

    await page.setViewportSize({ width: 768, height: 844 });
    await page.goto("/produk/tauco-cap-badak");
    const headingBox = await page.locator("h1").boundingBox();
    const imageBox = await page.locator(".product-detail-image").boundingBox();

    expect(headingBox).not.toBeNull();
    expect(imageBox).not.toBeNull();
    expect(headingBox?.y).toBeLessThan(500);
    expect(imageBox?.y).toBeGreaterThan(headingBox?.y ?? 0);
  });

  test("visible copy does not contain em dash or en dash", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    for (const route of primaryRoutes) {
      await page.goto(route);
      const visibleText = await page.locator("body").innerText();
      expect(visibleText, route).not.toMatch(/[—–]/);
    }
  });
});
