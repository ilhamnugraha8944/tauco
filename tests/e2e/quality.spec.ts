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

function createAxeBuilder(page: Page) {
  return new AxeBuilder({ page }).exclude(
    'iframe[title="Netlify Drawer"]',
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

async function fillValidContactForm(page: Page) {
  await page.getByLabel(/^Nama/).fill("Ilham Pratama");
  await page.getByLabel(/^Email/).fill("ilham@example.com");
  await page
    .getByRole("textbox", { name: /^Pesan/ })
    .fill("Saya ingin mengetahui informasi produk yang tersedia saat ini.");
  await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();
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
      const results = await createAxeBuilder(page)
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

  test("public pages do not overflow at required viewport widths", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    for (const viewport of [
      { width: 320, height: 568 },
      { width: 390, height: 844 },
      { width: 768, height: 844 },
      { width: 1024, height: 768 },
      { width: 1440, height: 900 },
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

  test("responsive boundaries preserve navigation and hero hierarchy", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    await page.setViewportSize({ width: 1023, height: 768 });
    await page.goto("/");
    await expect(page.locator(".mobile-menu")).toBeVisible();
    await expect(page.locator(".site-header > div > nav")).toBeHidden();

    await page.setViewportSize({ width: 1024, height: 768 });
    await expect(page.locator(".mobile-menu")).toBeHidden();
    const desktopNavigation = page.locator(".site-header > div > nav");
    await expect(desktopNavigation).toBeVisible();
    const navigationMetrics = await desktopNavigation.evaluate(
      (navigation) => {
        const links = Array.from(
          navigation.querySelectorAll<HTMLAnchorElement>("a"),
        );
        const tops = links.map((link) => link.getBoundingClientRect().top);

        return {
          headerHeight:
            document.querySelector("header")?.getBoundingClientRect().height ??
            0,
          topDifference: Math.max(...tops) - Math.min(...tops),
        };
      },
    );

    expect(navigationMetrics.headerHeight).toBeLessThanOrEqual(80);
    expect(navigationMetrics.topDifference).toBeLessThanOrEqual(1);

    for (const viewport of [
      { width: 390, height: 844 },
      { width: 1024, height: 768 },
      { width: 1440, height: 900 },
    ]) {
      await page.setViewportSize(viewport);
      await page.goto("/");
      const lineMetrics = await page.locator("h1").evaluate((heading) => {
        const styles = getComputedStyle(heading);
        return {
          height: heading.getBoundingClientRect().height,
          lineHeight: Number.parseFloat(styles.lineHeight),
        };
      });

      expect(
        lineMetrics.height / lineMetrics.lineHeight,
        `Judul homepage pada ${viewport.width}px`,
      ).toBeLessThanOrEqual(2.1);

      const heroDescriptionWordCount = (
        await page.locator(".hero-description").innerText()
      )
        .trim()
        .split(/\s+/).length;
      const heroButton = page.locator(".hero-actions .button-link");
      const heroButtonBox = await heroButton.boundingBox();
      const buttonDimensions = await heroButton.evaluate((button) => ({
        clientWidth: button.clientWidth,
        scrollWidth: button.scrollWidth,
      }));

      expect(heroDescriptionWordCount).toBeLessThanOrEqual(20);
      expect(heroButtonBox).not.toBeNull();
      expect(
        (heroButtonBox?.y ?? 0) + (heroButtonBox?.height ?? 0),
        `CTA homepage pada ${viewport.width}px`,
      ).toBeLessThanOrEqual(viewport.height);
      expect(buttonDimensions.scrollWidth).toBe(
        buttonDimensions.clientWidth,
      );
    }

    await page.setViewportSize({ width: 768, height: 844 });
    await page.goto("/produk/tauco-cap-badak");
    const headingBox = await page.locator("h1").boundingBox();
    const contactButtonBox = await page
      .getByRole("link", { name: "Tanyakan produk" })
      .boundingBox();
    const imageBox = await page.locator(".product-detail-image").boundingBox();

    expect(headingBox).not.toBeNull();
    expect(contactButtonBox).not.toBeNull();
    expect(imageBox).not.toBeNull();
    expect(headingBox?.y).toBeLessThan(500);
    expect(contactButtonBox?.y).toBeLessThan(imageBox?.y ?? 0);
    expect(imageBox?.y).toBeGreaterThan(headingBox?.y ?? 0);
  });

  test("visible and accessible copy does not contain em dash or en dash", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);

    for (const route of primaryRoutes) {
      await page.goto(route);
      const copy = await page.evaluate(() => {
        const attributeCopy = Array.from(
          document.querySelectorAll<HTMLElement>(
            "[alt], [title], [aria-label], figcaption, button",
          ),
        ).flatMap((element) => [
          element.getAttribute("alt") ?? "",
          element.getAttribute("title") ?? "",
          element.getAttribute("aria-label") ?? "",
          element.innerText,
        ]);

        return [document.body.innerText, ...attributeCopy].join("\n");
      });

      expect(copy, route).not.toMatch(/[\u2013\u2014]/);
    }
  });

  test("dark mode keeps every public route axe-clean", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);
    await page.emulateMedia({ colorScheme: "dark" });

    for (const route of primaryRoutes) {
      await page.goto(route);
      const results = await createAxeBuilder(page)
        .withTags([
          "wcag2a",
          "wcag2aa",
          "wcag21a",
          "wcag21aa",
          "wcag22aa",
        ])
        .analyze();

      expect(results.violations, `${route} pada dark mode`).toEqual([]);
    }
  });

  test("dynamic contact states remain axe-clean", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);
    const analyzeCurrentState = async (state: string) => {
      const results = await createAxeBuilder(page)
        .withTags([
          "wcag2a",
          "wcag2aa",
          "wcag21a",
          "wcag21aa",
          "wcag22aa",
        ])
        .analyze();

      expect(results.violations, `Form state ${state}`).toEqual([]);
    };

    await page.goto("/kontak");
    await page.getByRole("button", { name: "Kirim pesan" }).click();
    await expect(page.getByLabel(/^Nama/)).toBeFocused();
    await analyzeCurrentState("validation-error");

    await page.reload();
    let releaseResponse: (() => void) | undefined;
    const responseGate = new Promise<void>((resolve) => {
      releaseResponse = resolve;
    });
    await page.route("**/__forms.html", async (route) => {
      await responseGate;
      await route.fulfill({ status: 200, body: "ok" });
    });
    await fillValidContactForm(page);
    await page.getByRole("button", { name: "Kirim pesan" }).click();
    await expect(
      page.getByRole("button", { name: "Mengirim..." }),
    ).toBeDisabled();

    try {
      await analyzeCurrentState("pending");
    } finally {
      releaseResponse?.();
    }

    await expect(
      page.getByText(
        "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
      ),
    ).toBeVisible();
    await analyzeCurrentState("success");

    await page.unroute("**/__forms.html");
    await page.reload();
    await page.route("**/__forms.html", async (route) => {
      await route.abort("failed");
    });
    await fillValidContactForm(page);
    await page.getByRole("button", { name: "Kirim pesan" }).click();
    await expect(
      page.getByText(
        "Pesan belum terkirim. Periksa koneksi Anda, lalu coba kembali.",
      ),
    ).toBeVisible();
    await analyzeCurrentState("network-error");
  });

  test("focus remains visible and reduced motion shortens transitions", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");

    const menuSummary = page.locator(".mobile-menu summary");
    await menuSummary.focus();
    const outlineWidth = await menuSummary.evaluate((summary) =>
      Number.parseFloat(getComputedStyle(summary).outlineWidth),
    );
    expect(outlineWidth).toBeGreaterThanOrEqual(2);

    await page.emulateMedia({ reducedMotion: "reduce" });
    const transitionDurations = await page
      .locator(".button-link")
      .first()
      .evaluate((button) =>
        getComputedStyle(button)
          .transitionDuration.split(",")
          .map((value) =>
            value.trim().endsWith("ms")
              ? Number.parseFloat(value) / 1000
              : Number.parseFloat(value),
          ),
      );

    expect(Math.max(...transitionDurations)).toBeLessThanOrEqual(0.001);
  });

  test("keyboard-only walkthrough reaches every visible control", async ({
    page,
  }, testInfo) => {
    desktopOnly(testInfo);
    await page.setViewportSize({ width: 1440, height: 900 });

    for (const route of primaryRoutes) {
      await page.goto(route);
      const expectedIds = await page
        .locator(
          [
            "a[href]",
            "button:not([disabled])",
            'input:not([type="hidden"]):not([tabindex="-1"]):not([disabled])',
            "select:not([disabled])",
            "textarea:not([disabled])",
            "summary",
            '[tabindex]:not([tabindex="-1"])',
          ].join(","),
        )
        .evaluateAll((elements) => {
          const focusableElements = Array.from(
            new Set(elements),
          ).filter((element) => {
            if (!(element instanceof HTMLElement)) {
              return false;
            }

            const styles = getComputedStyle(element);

            return (
              styles.display !== "none" &&
              styles.visibility !== "hidden" &&
              element.getClientRects().length > 0
            );
          });

          return focusableElements.map((element, index) => {
            const id = `${index}`;
            element.dataset.keyboardAuditId = id;
            return id;
          });
        });
      const reachedIds = new Set<string>();

      for (let index = 0; index < expectedIds.length; index += 1) {
        await page.keyboard.press("Tab");
        const focusState = await page.evaluate(() => {
          const activeElement = document.activeElement;

          if (!(activeElement instanceof HTMLElement)) {
            return null;
          }

          const styles = getComputedStyle(activeElement);

          return {
            id: activeElement.dataset.keyboardAuditId ?? "",
            outlineStyle: styles.outlineStyle,
            outlineWidth: Number.parseFloat(styles.outlineWidth),
          };
        });

        expect(focusState, `${route} langkah ${index + 1}`).not.toBeNull();
        expect(focusState?.id, `${route} langkah ${index + 1}`).not.toBe("");
        expect(focusState?.outlineStyle).not.toBe("none");
        expect(focusState?.outlineWidth).toBeGreaterThanOrEqual(2);
        reachedIds.add(focusState?.id ?? "");
      }

      expect(
        [...reachedIds].sort(),
        `Seluruh control pada ${route} harus tercapai dengan Tab`,
      ).toEqual([...expectedIds].sort());
    }
  });
});
