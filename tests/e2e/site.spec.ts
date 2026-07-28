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

function createAxeBuilder(page: Page) {
  return new AxeBuilder({ page }).exclude(
    'iframe[title="Netlify Drawer"]',
  );
}

async function fillValidContactForm(page: Page) {
  await page.getByLabel(/^Nama/).fill("Ilham Pratama");
  await page.getByLabel(/^Email/).fill("ilham@example.com");
  await page.getByLabel(/^Telepon/).fill("+62 812 3456 7890");
  await page.getByLabel("Topik").selectOption("Kerja sama dan distribusi");
  await page
    .getByRole("textbox", { name: /^Pesan/ })
    .fill("Saya ingin mengetahui informasi produk yang tersedia saat ini.");
  await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();
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

  test("privacy notice covers access, correction, deletion, and retention", async ({
    page,
  }) => {
    await page.goto("/kebijakan-privasi");

    await expect(
      page.getByRole("heading", {
        name: "Pihak yang dapat mengakses data",
      }),
    ).toBeVisible();
    await expect(
      page.getByText(/pengelola inbox yang ditunjuk/),
    ).toBeVisible();
    await expect(
      page.getByText(/Netlify dapat memproses data/),
    ).toBeVisible();
    await expect(
      page.getByText(/meminta akses, koreksi, atau penghapusan data/),
    ).toBeVisible();
    await expect(page.getByText(/disimpan paling lama 12 bulan/)).toBeVisible();
    await expect(page.locator("main")).not.toContainText(
      "penyimpanan lebih lama",
    );
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

  test("mobile menu opens and closes with keyboard while retaining focus", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");

    const menu = page.locator(".mobile-menu");
    const summary = menu.locator("summary");
    const firstLink = menu.getByRole("link", { name: "Beranda" });

    await summary.focus();
    await expect(summary).toBeFocused();
    await summary.press("Enter");
    await expect(menu).toHaveAttribute("open", "");

    await page.keyboard.press("Tab");
    await expect(firstLink).toBeFocused();
    await page.keyboard.press("Shift+Tab");
    await expect(summary).toBeFocused();

    await summary.press("Space");
    await expect(menu).not.toHaveAttribute("open", "");
    await expect(summary).toBeFocused();
  });

  test("all public copy and links remain available without JavaScript", async ({
    browser,
  }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();

    for (const route of primaryRoutes) {
      const response = await page.goto(route);

      expect(response?.status(), route).toBe(200);
      await expectSingleHeading(page);
      const mainText = (await page.locator("main").innerText()).trim();

      expect(mainText.length).toBeGreaterThan(0);
      await expect(page.locator('a[href^="/"]').first()).toBeVisible();
    }

    const notFoundResponse = await page.goto(
      "/produk/produk-tidak-dikenal",
    );
    expect(notFoundResponse?.status()).toBe(404);
    await expect(
      page.getByRole("heading", { name: "Halaman tidak ditemukan" }),
    ).toBeVisible();

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/");
    const menu = page.locator(".mobile-menu");
    const summary = menu.locator("summary");

    await summary.focus();
    await summary.press("Enter");
    await expect(menu).toHaveAttribute("open", "");
    await summary.press("Space");
    await expect(menu).not.toHaveAttribute("open", "");

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

    const form = page.locator("form.contact-form");

    await expect(form).not.toHaveAttribute(
      "novalidate",
      "",
    );
    await expect(form).toHaveAttribute("method", /post/i);
    await expect(form).toHaveAttribute("action", "/__forms.html");
    await expect(
      form.locator('input[name="form-name"]'),
    ).toHaveValue("kontak");
    await expect(form.locator('input[name="bot-field"]')).toHaveCount(1);
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

    await fillValidContactForm(page);
    await page.getByRole("button", { name: "Kirim pesan" }).click();

    await expect(
      page.getByText(
        "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
      ),
    ).toBeVisible();
    await expect(page.getByLabel(/^Nama/)).toHaveValue("");
    await expect(page.getByLabel(/^Email/)).toHaveValue("");
    await expect(page.getByLabel(/^Telepon/)).toHaveValue("");
    await expect(page.getByLabel("Topik")).toHaveValue("Pertanyaan umum");
    await expect(
      page.getByRole("textbox", { name: /^Pesan/ }),
    ).toHaveValue("");
    await expect(
      page.getByRole("checkbox", { name: /Saya menyetujui/ }),
    ).not.toBeChecked();
  });

  test("locks duplicate submissions while the request is pending", async ({
    page,
  }) => {
    let requestCount = 0;
    let releaseResponse: (() => void) | undefined;
    const responseGate = new Promise<void>((resolve) => {
      releaseResponse = resolve;
    });

    await page.route("**/__forms.html", async (route) => {
      requestCount += 1;
      await responseGate;
      await route.fulfill({ status: 200, body: "ok" });
    });
    await page.goto("/kontak");
    await fillValidContactForm(page);

    const form = page.locator("form.contact-form");
    const submitButton = page.getByRole("button", { name: "Kirim pesan" });

    await form.evaluate((element) => {
      if (element instanceof HTMLFormElement) {
        element.requestSubmit();
        element.requestSubmit();
      }
    });

    try {
      await expect(
        page.getByRole("button", { name: "Mengirim..." }),
      ).toBeDisabled();
      await expect(form).toHaveAttribute("aria-busy", "true");
      await expect(page.getByText("Sedang mengirim pesan.")).toBeVisible();
      await page.waitForTimeout(150);
      expect(requestCount).toBe(1);
    } finally {
      releaseResponse?.();
    }

    await expect(
      page.getByText(
        "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
      ),
    ).toBeVisible();
    await expect(submitButton).toBeEnabled();
    await expect(form).toHaveAttribute("aria-busy", "false");
  });

  test("shows a network error without losing the page", async ({ page }) => {
    await page.route("**/__forms.html", async (route) => {
      await route.abort("failed");
    });
    await page.goto("/kontak");

    await fillValidContactForm(page);
    await page.getByRole("button", { name: "Kirim pesan" }).click();

    await expect(
      page.getByText(
        "Pesan belum terkirim. Periksa koneksi Anda, lalu coba kembali.",
      ),
    ).toBeVisible();
    await expect(page.getByLabel(/^Nama/)).toHaveValue("Ilham Pratama");
    await expect(page.getByLabel(/^Email/)).toHaveValue(
      "ilham@example.com",
    );
    await expect(page.getByLabel(/^Telepon/)).toHaveValue(
      "+62 812 3456 7890",
    );
    await expect(page.getByLabel("Topik")).toHaveValue(
      "Kerja sama dan distribusi",
    );
    await expect(
      page.getByRole("textbox", { name: /^Pesan/ }),
    ).toHaveValue(
      "Saya ingin mengetahui informasi produk yang tersedia saat ini.",
    );
    await expect(
      page.getByRole("checkbox", { name: /Saya menyetujui/ }),
    ).toBeChecked();
    await expect(
      page.getByRole("button", { name: "Kirim pesan" }),
    ).toBeEnabled();
  });
});

test.describe("accessibility", () => {
  for (const route of ["/", "/tauco", "/produk", "/kontak"] as const) {
    test(`${route} has no serious axe violations`, async ({ page }) => {
      await page.goto(route);
      const results = await createAxeBuilder(page).analyze();

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
