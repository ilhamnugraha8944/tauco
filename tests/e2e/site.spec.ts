import AxeBuilder from "@axe-core/playwright";
import {
  expect,
  test,
  type Page,
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

async function expectSingleHeading(page: Page) {
  await expect(page.locator("h1")).toHaveCount(1);
}

test.describe("public pages", () => {
  for (const route of primaryRoutes) {
    test(`${route} renders with one heading and local noindex`, async ({
      page,
    }) => {
      const response = await page.goto(route);

      expect(response?.status()).toBe(200);
      await expectSingleHeading(page);
      await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
        "content",
        /noindex/,
      );
      await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
        "href",
        new RegExp(`http://localhost:3000${route === "/" ? "/?$" : route}`),
      );
    });
  }

  test("unknown product slug returns a true 404", async ({ page }) => {
    const response = await page.goto("/produk/produk-tidak-dikenal");

    expect(response?.status()).toBe(404);
    await expect(
      page.getByRole("heading", { name: "Halaman tidak ditemukan" }),
    ).toBeVisible();
  });

  test("homepage includes valid WebSite and Organization JSON-LD", async ({
    page,
  }) => {
    await page.goto("/");
    const scripts = await page
      .locator('script[type="application/ld+json"]')
      .allTextContents();
    const values = scripts.flatMap((script) => {
      const parsed = JSON.parse(script);
      return Array.isArray(parsed) ? parsed : [parsed];
    });

    expect(values.some((value) => value["@type"] === "WebSite")).toBe(true);
    expect(values.some((value) => value["@type"] === "Organization")).toBe(
      true,
    );
  });

  test("product JSON-LD does not invent commerce fields", async ({ page }) => {
    await page.goto("/produk/tauco-cap-badak");
    const structuredData = await page
      .locator('script[type="application/ld+json"]')
      .allTextContents();
    const product = structuredData
      .map((value) => JSON.parse(value))
      .find((value) => value["@type"] === "Product");

    expect(product).toBeTruthy();
    expect(product).not.toHaveProperty("offers");
    expect(product).not.toHaveProperty("review");
    expect(product).not.toHaveProperty("aggregateRating");
  });

  test("all internal navigation targets respond successfully", async ({
    page,
    request,
  }) => {
    await page.goto("/");
    const hrefs = await page.locator('a[href^="/"]').evaluateAll((links) =>
      Array.from(
        new Set(
          links
            .map((link) => link.getAttribute("href"))
            .filter((href): href is string => Boolean(href)),
        ),
      ),
    );

    for (const href of hrefs) {
      const response = await request.get(href);
      expect(response.status(), `${href} should resolve`).toBeLessThan(400);
    }
  });
});

test.describe("navigation and progressive enhancement", () => {
  test("mobile menu exposes every primary destination", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");
    const menu = page.locator(".mobile-menu");

    await menu.locator("summary").click();
    await expect(menu).toHaveAttribute("open", "");
    await expect(
      menu.getByRole("link", { name: "Mengenal Tauco" }),
    ).toBeVisible();
    await expect(menu.getByRole("link", { name: "Kontak" })).toBeVisible();
  });

  test("core copy and links remain available without JavaScript", async ({
    browser,
  }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();

    await page.goto("/");
    await expect(
      page.getByRole("heading", {
        name: "Tauco Cap Badak",
        level: 1,
        exact: true,
      }),
    ).toBeVisible();
    await expect(page.getByRole("link", { name: "Lihat produk" })).toBeVisible();

    await context.close();
  });
});

test.describe("contact form", () => {
  test("shows inline validation errors and focuses the first invalid field", async ({
    page,
  }) => {
    await page.goto("/kontak");
    await page.getByRole("button", { name: "Kirim pesan" }).click();

    await expect(page.getByLabel(/^Nama/)).toBeFocused();
    await expect(
      page
        .getByRole("alert")
        .filter({
          hasText:
            "Periksa kembali kolom yang ditandai sebelum mengirim pesan.",
        }),
    ).toBeVisible();
    await expect(page.getByText("Nama minimal 2 karakter.")).toBeVisible();
    await expect(
      page.getByText("Masukkan alamat email yang valid."),
    ).toBeVisible();
    await expect(
      page.getByText("Persetujuan privasi wajib diberikan."),
    ).toBeVisible();
  });

  test("publishes native constraints when JavaScript is unavailable", async ({
    browser,
  }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();

    await page.goto("/kontak");

    await expect(page.locator("form.contact-form")).not.toHaveAttribute(
      "novalidate",
      "",
    );
    await expect(page.getByLabel(/^Nama/)).toHaveAttribute("required", "");
    await expect(page.getByLabel(/^Nama/)).toHaveAttribute("minlength", "2");
    await expect(page.getByLabel(/^Nama/)).toHaveAttribute(
      "maxlength",
      "100",
    );
    await expect(page.getByLabel(/^Email/)).toHaveAttribute("required", "");
    await expect(
      page.getByRole("textbox", { name: /^Pesan/ }),
    ).toHaveAttribute("minlength", "20");
    await expect(
      page.getByRole("textbox", { name: /^Pesan/ }),
    ).toHaveAttribute("maxlength", "2000");
    await expect(
      page.getByRole("checkbox", { name: /Saya menyetujui/ }),
    ).toHaveAttribute("required", "");

    await context.close();
  });

  test("prefills the product topic from the query contract", async ({
    page,
  }) => {
    await page.goto("/kontak?topik=produk");

    await expect(page.getByLabel("Topik")).toHaveValue("Informasi produk");
  });

  test("shows a success state after the gateway accepts the message", async ({
    page,
  }) => {
    await page.route("**/__forms.html", async (route) => {
      await route.fulfill({ status: 200, body: "ok" });
    });
    await page.goto("/kontak");

    await page.getByLabel(/^Nama/).fill("Ilham Pratama");
    await page.getByLabel(/^Email/).fill("ilham@example.com");
    await page
      .getByRole("textbox", { name: /^Pesan/ })
      .fill("Saya ingin mengetahui informasi produk yang tersedia saat ini.");
    await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();
    await page.getByRole("button", { name: "Kirim pesan" }).click();

    await expect(
      page.getByText(
        "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
      ),
    ).toBeVisible();
  });

  test("shows a network error without losing the page", async ({ page }) => {
    await page.route("**/__forms.html", async (route) => {
      await route.abort("failed");
    });
    await page.goto("/kontak");

    await page.getByLabel(/^Nama/).fill("Ilham Pratama");
    await page.getByLabel(/^Email/).fill("ilham@example.com");
    await page
      .getByRole("textbox", { name: /^Pesan/ })
      .fill("Saya ingin mengetahui informasi produk yang tersedia saat ini.");
    await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();
    await page.getByRole("button", { name: "Kirim pesan" }).click();

    await expect(
      page.getByText(
        "Pesan belum terkirim. Periksa koneksi Anda, lalu coba kembali.",
      ),
    ).toBeVisible();
  });
});

test.describe("accessibility", () => {
  for (const route of ["/", "/tauco", "/produk", "/kontak"] as const) {
    test(`${route} has no serious axe violations`, async ({ page }) => {
      await page.goto(route);
      const results = await new AxeBuilder({ page }).analyze();

      expect(
        results.violations.filter((violation) =>
          ["serious", "critical"].includes(violation.impact ?? ""),
        ),
      ).toEqual([]);
    });
  }
});

test("robots and sitemap are generated locally", async ({ request }) => {
  const [robots, sitemap] = await Promise.all([
    request.get("/robots.txt"),
    request.get("/sitemap.xml"),
  ]);

  expect(await robots.text()).toContain("Disallow: /");
  const sitemapBody = await sitemap.text();
  expect(sitemapBody).toContain("http://localhost:3000/tauco");
  expect(sitemapBody).toContain(
    "http://localhost:3000/produk/tauco-cap-badak",
  );
});
