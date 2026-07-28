import {
  expect,
  test,
  type Page,
  type TestInfo,
} from "@playwright/test";

const publicRoutes = [
  { path: "/", heading: "Tauco Cap Badak" },
  { path: "/tauco", heading: "Apa itu tauco?" },
  {
    path: "/tentang-kami",
    heading: "Tentang Tauco Cap Badak",
  },
  { path: "/produk", heading: "Produk Tauco Cap Badak" },
  {
    path: "/produk/tauco-cap-badak",
    heading: "Tauco Cap Badak",
  },
  {
    path: "/kontak",
    heading: "Apa yang ingin Anda tanyakan?",
  },
  {
    path: "/kebijakan-privasi",
    heading: "Kebijakan Privasi",
  },
] as const;

const primaryNavigation = [
  { href: "/", label: "Beranda" },
  { href: "/tauco", label: "Mengenal Tauco" },
  { href: "/tentang-kami", label: "Tentang Kami" },
  { href: "/produk", label: "Produk" },
  { href: "/kontak", label: "Kontak" },
] as const;

function requireDeployPreviewUrl(): URL {
  const configuredUrl = process.env.PLAYWRIGHT_BASE_URL?.trim();

  if (!configuredUrl) {
    throw new Error(
      [
        "PLAYWRIGHT_BASE_URL wajib diisi dengan origin Deploy Preview.",
        "Contoh:",
        '$env:PLAYWRIGHT_BASE_URL="https://deploy-preview-1--tauco-cap-badak.netlify.app"',
      ].join("\n"),
    );
  }

  const url = new URL(configuredUrl);

  if (
    url.protocol !== "https:" ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  ) {
    throw new Error(
      "PLAYWRIGHT_BASE_URL wajib berupa origin HTTPS tanpa path, query, atau hash.",
    );
  }

  if (
    !url.hostname.endsWith(".netlify.app") ||
    !url.hostname.includes("--")
  ) {
    throw new Error(
      "G4.2 harus dijalankan terhadap Deploy Preview atau Branch Deploy Netlify, bukan production/local.",
    );
  }

  return url;
}

async function attachScreenshot(
  page: Page,
  testInfo: TestInfo,
  name: string,
) {
  await testInfo.attach(name, {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
}

test.beforeAll(() => {
  requireDeployPreviewUrl();
});

test.describe("G4.2 public route contracts", () => {
  for (const route of publicRoutes) {
    test(`${route.path} returns 200 with one visible h1`, async ({
      page,
    }) => {
      const response = await page.goto(route.path, {
        waitUntil: "domcontentloaded",
      });

      expect(response?.status(), route.path).toBe(200);
      await expect(page.locator("h1")).toHaveCount(1);
      await expect(page.locator("h1")).toBeVisible();
      await expect(page.locator("h1")).toContainText(route.heading);
    });
  }

  test("unknown product slug returns a true 404", async ({ page }) => {
    const response = await page.goto(
      "/produk/produk-yang-tidak-dikenal",
      { waitUntil: "domcontentloaded" },
    );

    expect(response?.status()).toBe(404);
    await expect(
      page.getByRole("heading", {
        name: "Halaman tidak ditemukan",
      }),
    ).toBeVisible();
  });
});

test("every internal link from every public route resolves", async ({
  page,
  request,
}) => {
  const targetOrigin = requireDeployPreviewUrl().origin;
  const targets = new Set<string>();

  for (const route of publicRoutes) {
    await page.goto(route.path, { waitUntil: "domcontentloaded" });

    const routeLinks = await page
      .locator("a[href]")
      .evaluateAll((links) =>
        links
          .map((link) => link.getAttribute("href"))
          .filter((href): href is string => Boolean(href)),
      );
    const brokenSamePageFragments = await page
      .locator('a[href*="#"]')
      .evaluateAll((links) =>
        links
          .map((link) => link.getAttribute("href"))
          .filter((href): href is string => {
            if (!href) {
              return false;
            }

            const target = new URL(href, window.location.href);

            if (
              target.origin !== window.location.origin ||
              target.pathname !== window.location.pathname ||
              !target.hash
            ) {
              return false;
            }

            return !document.getElementById(
              decodeURIComponent(target.hash.slice(1)),
            );
          }),
      );

    expect(
      brokenSamePageFragments,
      `Fragment link rusak pada ${route.path}`,
    ).toEqual([]);

    for (const href of routeLinks) {
      const target = new URL(href, targetOrigin);

      if (
        target.origin === targetOrigin &&
        ["http:", "https:"].includes(target.protocol)
      ) {
        targets.add(`${target.pathname}${target.search}`);
      }
    }
  }

  expect(targets.size).toBeGreaterThan(0);

  for (const target of targets) {
    const response = await request.get(target);

    expect(
      response.status(),
      `Internal link ${target} harus merespons di bawah 400`,
    ).toBeLessThan(400);
  }
});

test("mobile navigation opens and navigates", async ({
  page,
}, testInfo) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/", { waitUntil: "domcontentloaded" });

  const desktopNavigation = page.getByRole("navigation", {
    name: "Navigasi utama",
  });
  const mobileMenu = page.locator(".mobile-menu");
  const summary = mobileMenu.locator("summary");

  await expect(desktopNavigation).toBeHidden();
  await expect(summary).toBeVisible();
  await summary.click();
  await expect(mobileMenu).toHaveAttribute("open", "");

  for (const item of primaryNavigation) {
    const link = mobileMenu.getByRole("link", {
      name: item.label,
      exact: true,
    });

    await expect(link).toBeVisible();
    await expect(link).toHaveAttribute("href", item.href);
  }

  await attachScreenshot(page, testInfo, "mobile-navigation-open");
  await mobileMenu
    .getByRole("link", { name: "Mengenal Tauco", exact: true })
    .click();
  await expect(page).toHaveURL(/\/tauco$/);
  await expect(
    page.getByRole("heading", { name: "Apa itu tauco?" }),
  ).toBeVisible();
});

test("dark mode follows the system color preference", async ({
  page,
}, testInfo) => {
  const readTheme = () =>
    page.evaluate(() => {
      const rootStyles = getComputedStyle(
        document.documentElement,
      );
      const bodyStyles = getComputedStyle(document.body);

      return {
        prefersDark: window.matchMedia(
          "(prefers-color-scheme: dark)",
        ).matches,
        backgroundToken: rootStyles
          .getPropertyValue("--background")
          .trim(),
        textToken: rootStyles.getPropertyValue("--text").trim(),
        renderedBackground: bodyStyles.backgroundColor,
        renderedText: bodyStyles.color,
      };
    });

  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/", { waitUntil: "domcontentloaded" });
  const lightTheme = await readTheme();

  expect(lightTheme.prefersDark).toBe(false);
  expect(lightTheme.backgroundToken).toBe("#f2f4f1");
  expect(lightTheme.textToken).toBe("#17211d");
  await attachScreenshot(page, testInfo, "system-theme-light");

  await page.emulateMedia({ colorScheme: "dark" });
  await expect
    .poll(async () => (await readTheme()).prefersDark)
    .toBe(true);
  const darkTheme = await readTheme();

  expect(darkTheme.backgroundToken).toBe("#111613");
  expect(darkTheme.textToken).toBe("#eef2ef");
  expect(darkTheme.renderedBackground).not.toBe(
    lightTheme.renderedBackground,
  );
  expect(darkTheme.renderedText).not.toBe(lightTheme.renderedText);
  await attachScreenshot(page, testInfo, "system-theme-dark");
});

test("all images load and cumulative layout shift stays within target", async ({
  page,
}, testInfo) => {
  const metrics: Array<{
    cls: number;
    imageCount: number;
    route: string;
  }> = [];

  await page.addInitScript(() => {
    type LayoutShiftEntry = PerformanceEntry & {
      hadRecentInput: boolean;
      value: number;
    };
    type LayoutShiftWindow = Window & {
      __g4Cls?: number;
      __g4ClsObserver?: PerformanceObserver;
      __g4ClsSupported?: boolean;
    };

    const trackedWindow = window as LayoutShiftWindow;
    trackedWindow.__g4Cls = 0;
    trackedWindow.__g4ClsSupported =
      "PerformanceObserver" in window &&
      PerformanceObserver.supportedEntryTypes.includes(
        "layout-shift",
      );

    if (!trackedWindow.__g4ClsSupported) {
      return;
    }

    trackedWindow.__g4ClsObserver = new PerformanceObserver(
      (list) => {
        for (const entry of list.getEntries()) {
          const layoutShift = entry as LayoutShiftEntry;

          if (!layoutShift.hadRecentInput) {
            trackedWindow.__g4Cls =
              (trackedWindow.__g4Cls ?? 0) + layoutShift.value;
          }
        }
      },
    );
    trackedWindow.__g4ClsObserver.observe({
      type: "layout-shift",
      buffered: true,
    });
  });

  for (const route of publicRoutes) {
    await page.goto(route.path, { waitUntil: "load" });
    await page.evaluate(async () => {
      await document.fonts.ready;
    });

    const images = page.locator("img");
    const imageCount = await images.count();

    for (let index = 0; index < imageCount; index += 1) {
      const image = images.nth(index);
      await image.scrollIntoViewIfNeeded();
      await expect
        .poll(
          () =>
            image.evaluate(
              (element) =>
                element instanceof HTMLImageElement &&
                element.complete &&
                element.naturalWidth > 0,
            ),
          {
            message: `Image ${index + 1} pada ${route.path} harus termuat`,
            timeout: 10_000,
          },
        )
        .toBe(true);

      const dimensions = await image.evaluate((element) => {
        if (!(element instanceof HTMLImageElement)) {
          throw new Error("Locator img tidak menghasilkan HTMLImageElement.");
        }

        return {
          naturalHeight: element.naturalHeight,
          naturalWidth: element.naturalWidth,
          renderedHeight: element.getBoundingClientRect().height,
          renderedWidth: element.getBoundingClientRect().width,
        };
      });

      expect(dimensions.naturalHeight).toBeGreaterThan(0);
      expect(dimensions.naturalWidth).toBeGreaterThan(0);
      expect(dimensions.renderedHeight).toBeGreaterThan(0);
      expect(dimensions.renderedWidth).toBeGreaterThan(0);
    }

    await page.waitForTimeout(500);
    const layoutShift = await page.evaluate(() => {
      const trackedWindow = window as Window & {
        __g4Cls?: number;
        __g4ClsSupported?: boolean;
      };

      return {
        supported: trackedWindow.__g4ClsSupported,
        value: trackedWindow.__g4Cls ?? 0,
      };
    });

    expect(
      layoutShift.supported,
      `Layout Instability API harus tersedia untuk ${route.path}`,
    ).toBe(true);
    expect(
      layoutShift.value,
      `CLS ${route.path} harus <= 0.1`,
    ).toBeLessThanOrEqual(0.1);
    metrics.push({
      cls: layoutShift.value,
      imageCount,
      route: route.path,
    });
  }

  await testInfo.attach("image-and-cls-metrics", {
    body: Buffer.from(JSON.stringify(metrics, null, 2)),
    contentType: "application/json",
  });
});

test("primary copy and links exist in the initial HTML", async ({
  request,
}) => {
  for (const route of publicRoutes) {
    const response = await request.get(route.path);
    const html = await response.text();

    expect(response.status(), route.path).toBe(200);
    expect(html, `${route.path} harus memiliki main`).toMatch(
      /<main(?:\s|>)/i,
    );
    expect(html, `${route.path} harus memiliki h1`).toMatch(
      /<h1(?:\s|>)/i,
    );
    expect(html, `${route.path} harus memuat copy utama`).toContain(
      route.heading,
    );
    expect(html, `${route.path} harus memuat internal link`).toMatch(
      /<a[^>]+href=["']\//i,
    );
  }
});
