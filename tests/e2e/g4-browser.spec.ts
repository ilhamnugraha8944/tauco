import {
  expect,
  test,
  type Page,
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

async function expectImagesLoaded(page: Page, route: string) {
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
          message: `Image ${index + 1} pada ${route} harus termuat`,
          timeout: 10_000,
        },
      )
      .toBe(true);
  }
}

test("all public routes render across the browser matrix", async ({
  browser,
  page,
}, testInfo: TestInfo) => {
  const results: Array<{
    imageCount: number;
    route: string;
    status: number | null;
  }> = [];

  for (const route of publicRoutes) {
    const response = await page.goto(route, {
      waitUntil: "domcontentloaded",
    });

    expect(response?.status(), `${testInfo.project.name}: ${route}`).toBe(
      200,
    );
    await expect(page.locator("h1")).toHaveCount(1);
    await expect(page.locator("h1")).toBeVisible();
    await expectImagesLoaded(page, route);
    results.push({
      imageCount: await page.locator("img").count(),
      route,
      status: response?.status() ?? null,
    });
  }

  await page.goto("/", { waitUntil: "domcontentloaded" });
  await testInfo.attach(`${testInfo.project.name}-homepage`, {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
  await testInfo.attach(`${testInfo.project.name}-matrix`, {
    body: Buffer.from(
      JSON.stringify(
        {
          browserVersion: browser.version(),
          project: testInfo.project.name,
          results,
        },
        null,
        2,
      ),
    ),
    contentType: "application/json",
  });
});

test("navigation matches the project viewport", async ({
  page,
}, testInfo) => {
  await page.goto("/", { waitUntil: "domcontentloaded" });

  const desktopNavigation = page.getByRole("navigation", {
    name: "Navigasi utama",
  });
  const mobileMenu = page.locator(".mobile-menu");

  if (testInfo.project.name === "android-chrome-emulation") {
    await expect(desktopNavigation).toBeHidden();
    await expect(mobileMenu.locator("summary")).toBeVisible();
    await mobileMenu.locator("summary").click();
    await expect(mobileMenu).toHaveAttribute("open", "");
    await mobileMenu
      .getByRole("link", { name: "Produk", exact: true })
      .click();
    await expect(page).toHaveURL(/\/produk$/);
    await expect(
      page.getByRole("heading", { name: "Produk Tauco Cap Badak" }),
    ).toBeVisible();
    return;
  }

  await expect(desktopNavigation).toBeVisible();
  await expect(mobileMenu).toBeHidden();
  await desktopNavigation
    .getByRole("link", { name: "Produk", exact: true })
    .click();
  await expect(page).toHaveURL(/\/produk$/);
});

test("required viewport widths remain usable", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "edge-desktop",
    "Viewport matrix cukup dijalankan sekali pada browser desktop aktual.",
  );

  for (const viewport of [
    { height: 700, width: 320 },
    { height: 844, width: 768 },
    { height: 768, width: 1024 },
    { height: 900, width: 1440 },
  ]) {
    await page.setViewportSize(viewport);

    for (const route of publicRoutes) {
      await page.goto(route, { waitUntil: "domcontentloaded" });
      const dimensions = await page.evaluate(() => ({
        clientWidth: document.documentElement.clientWidth,
        scrollWidth: document.documentElement.scrollWidth,
      }));

      expect(
        dimensions.scrollWidth,
        `${route} pada ${viewport.width}px`,
      ).toBe(dimensions.clientWidth);
      await expect(page.locator("h1")).toBeVisible();
    }
  }
});

test("200 percent zoom reflow proxy remains usable", async ({
  page,
}, testInfo) => {
  test.skip(
    testInfo.project.name !== "edge-desktop",
    "Reflow proxy cukup dijalankan sekali pada browser desktop aktual.",
  );

  await page.setViewportSize({
    height: 900,
    width: 640,
  });

  for (const route of publicRoutes) {
    await page.goto(route, { waitUntil: "domcontentloaded" });
    const dimensions = await page.evaluate(() => ({
      clientWidth: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
    }));

    expect(
      dimensions.scrollWidth,
      `${route} pada effective viewport 640px`,
    ).toBe(dimensions.clientWidth);
    await expect(page.locator("h1")).toBeVisible();
    await expect(page.locator("main")).toBeVisible();
  }
});
