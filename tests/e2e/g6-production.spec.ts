import AxeBuilder from "@axe-core/playwright";
import {
  expect,
  test,
  type APIResponse,
  type Page,
} from "@playwright/test";

const productionOrigin = "https://tauco-cap-badak.netlify.app";
const publicRoutes = [
  "/",
  "/tauco",
  "/tentang-kami",
  "/produk",
  "/produk/tauco-cap-badak",
  "/kontak",
  "/kebijakan-privasi",
] as const;
const interiorRoutes = publicRoutes.filter((route) => route !== "/");

function exactUrl(pathname: string) {
  return new URL(pathname, productionOrigin).toString();
}

function header(response: APIResponse, name: string) {
  return response.headers()[name.toLowerCase()] ?? "";
}

async function readJsonLd(page: Page) {
  const entries = await page
    .locator('script[type="application/ld+json"]')
    .evaluateAll((scripts) =>
      scripts.map((script) => JSON.parse(script.textContent ?? "")),
    );

  return entries.flatMap((entry) =>
    Array.isArray(entry) ? entry : [entry],
  );
}

test.describe("G6.2 production route smoke test", () => {
  for (const route of publicRoutes) {
    test(`${route} returns 200 with one h1`, async ({ page, request }) => {
      const response = await request.get(route);

      expect(response.status()).toBe(200);
      await page.goto(route, { waitUntil: "domcontentloaded" });
      await expect(page.locator("h1")).toHaveCount(1);
      await expect(page.locator("h1")).toBeVisible();
    });
  }

  test("unknown product slug returns a true 404", async ({
    page,
    request,
  }) => {
    const response = await request.get("/produk/slug-tidak-ada");

    expect(response.status()).toBe(404);
    const navigation = await page.goto("/produk/slug-tidak-ada");
    expect(navigation?.status()).toBe(404);
    await expect(page.locator("h1")).toHaveCount(1);
  });
});

test.describe("G6.3 production SEO verification", () => {
  for (const route of publicRoutes) {
    test(`${route} is indexable with exact canonical and Open Graph`, async ({
      page,
      request,
    }) => {
      const response = await request.get(route);
      const source = await response.text();
      const expectedUrl =
        route === "/" ? productionOrigin : exactUrl(route);

      expect(response.status()).toBe(200);
      expect(header(response, "x-robots-tag").toLowerCase()).not.toContain(
        "noindex",
      );
      expect(source.toLowerCase()).not.toContain("localhost");
      expect(source).not.toContain("deploy-preview-");
      expect(source).not.toContain("--tauco-cap-badak.netlify.app");

      await page.goto(route, { waitUntil: "domcontentloaded" });
      await expect(page.locator('meta[name="robots"]')).not.toHaveAttribute(
        "content",
        /noindex|nofollow/i,
      );
      await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
        "href",
        expectedUrl,
      );
      await expect(page.locator('meta[property="og:url"]')).toHaveAttribute(
        "content",
        expectedUrl,
      );

      const openGraphImage = page.locator('meta[property="og:image"]');
      if ((await openGraphImage.count()) > 0) {
        await expect(openGraphImage).toHaveAttribute(
          "content",
          new RegExp(`^${productionOrigin.replaceAll(".", "\\.")}/`),
        );
      }

      expect(source).toMatch(/<h1(?:\s|>)/i);
      expect(source).toMatch(/href="\/(?:[^"]*)"/i);
    });
  }

  test("robots allows crawling, protects the form blueprint, and declares sitemap", async ({
    request,
  }) => {
    const response = await request.get("/robots.txt");
    const body = await response.text();

    expect(response.status()).toBe(200);
    expect(body).toMatch(/^User-agent:\s*\*/im);
    expect(body).toMatch(/^Allow:\s*\/\s*$/im);
    expect(body).toMatch(/^Disallow:\s*\/__forms\.html\s*$/im);
    expect(body).toContain(
      `Sitemap: ${productionOrigin}/sitemap.xml`,
    );
    expect(body.toLowerCase()).not.toContain("localhost");
    expect(body).not.toContain("deploy-preview-");
  });

  test("sitemap contains exactly the seven production URLs", async ({
    request,
  }) => {
    const response = await request.get("/sitemap.xml");
    const body = await response.text();
    const locations = [
      ...body.matchAll(/<loc>([^<]+)<\/loc>/g),
    ].map((match) => match[1]);
    const expectedLocations = publicRoutes.map(exactUrl);

    expect(response.status()).toBe(200);
    expect(new Set(locations).size).toBe(locations.length);
    expect(locations.sort()).toEqual(expectedLocations.sort());
    expect(body.toLowerCase()).not.toContain("localhost");
    expect(body).not.toContain("deploy-preview-");
  });

  test("homepage exposes valid WebSite and Organization JSON-LD", async ({
    page,
  }) => {
    await page.goto("/", { waitUntil: "domcontentloaded" });
    const structuredData = await readJsonLd(page);
    const types = structuredData.map((entry) => entry["@type"]);

    expect(types).toContain("WebSite");
    expect(types).toContain("Organization");
  });

  for (const route of interiorRoutes) {
    test(`${route} exposes BreadcrumbList JSON-LD`, async ({ page }) => {
      await page.goto(route, { waitUntil: "domcontentloaded" });
      const structuredData = await readJsonLd(page);

      expect(
        structuredData.some((entry) => entry["@type"] === "BreadcrumbList"),
      ).toBe(true);
    });
  }

  test("product schema is factual and omits unverified commerce fields", async ({
    page,
  }) => {
    await page.goto("/produk/tauco-cap-badak", {
      waitUntil: "domcontentloaded",
    });
    const structuredData = await readJsonLd(page);
    const product = structuredData.find(
      (entry) => entry["@type"] === "Product",
    );

    expect(product).toBeTruthy();
    expect(product.name).toBe("Tauco Cap Badak");
    expect(product).not.toHaveProperty("offers");
    expect(product).not.toHaveProperty("aggregateRating");
    expect(product).not.toHaveProperty("review");
    expect(product).not.toHaveProperty("sku");
  });
});

test.describe("G6.4 production quality", () => {
  test("HTTPS and security headers are present", async ({ request }) => {
    const response = await request.get("/");

    expect(response.url()).toMatch(/^https:/);
    expect(header(response, "strict-transport-security")).toContain(
      "max-age=",
    );
    expect(header(response, "x-content-type-options")).toBe("nosniff");
    expect(header(response, "x-frame-options")).toBe("SAMEORIGIN");
    expect(header(response, "referrer-policy")).toBe(
      "strict-origin-when-cross-origin",
    );
    expect(header(response, "permissions-policy")).toContain("camera=()");
  });

  test("all routes are free of mixed content, broken images, and blocking console errors", async ({
    page,
  }) => {
    const errors: string[] = [];

    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      const location = message.location().url;
      if (
        message.type() === "error" &&
        !location.includes("app.netlify.com")
      ) {
        errors.push(message.text());
      }
    });

    for (const route of publicRoutes) {
      await page.goto(route, { waitUntil: "networkidle" });
      const mixedResources = page.locator(
        'img[src^="http:"],script[src^="http:"],link[rel="stylesheet"][href^="http:"]',
      );

      await expect(mixedResources).toHaveCount(0);

      for (const image of await page.locator("img").all()) {
        await expect(image).toBeVisible();
        expect(
          await image.evaluate(
            (element) =>
              element instanceof HTMLImageElement &&
              element.complete &&
              element.naturalWidth > 0,
          ),
        ).toBe(true);
      }
    }

    expect(errors).toEqual([]);
  });

  test("contact form opens with its production Netlify contract", async ({
    page,
  }) => {
    await page.goto("/kontak", { waitUntil: "domcontentloaded" });
    const form = page.locator("form.contact-form");

    await expect(form).toBeVisible();
    await expect(form).toHaveAttribute("name", "kontak");
    await expect(form).toHaveAttribute("action", "/__forms.html");
    await expect(form).toHaveAttribute("data-netlify", "true");
    await expect(form.locator('input[name="bot-field"]')).toHaveCount(1);
  });

  test("mobile dark mode, keyboard focus, and WCAG AA remain usable", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.emulateMedia({ colorScheme: "dark" });
    await page.goto("/", { waitUntil: "domcontentloaded" });

    const background = await page.locator("body").evaluate((element) =>
      getComputedStyle(element).backgroundColor,
    );
    expect(background).toBe("rgb(17, 22, 19)");

    const menu = page.locator("details.mobile-menu");
    await menu.locator("summary").focus();
    await page.keyboard.press("Enter");
    await expect(menu).toHaveAttribute("open", "");
    await page.keyboard.press("Tab");
    await expect(menu.locator("a").first()).toBeFocused();

    const focusOutline = await menu.locator("a").first().evaluate((element) =>
      getComputedStyle(element).outlineStyle,
    );
    expect(focusOutline).not.toBe("none");

    const accessibility = await new AxeBuilder({ page })
      .exclude('iframe[title="Netlify Drawer"]')
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();

    expect(accessibility.violations).toEqual([]);
  });
});
