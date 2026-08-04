import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { resolve } from "node:path";

test("admin login, TOTP setup, shell, account, and logout", async ({ page }) => {
  const response = await page.goto("/admin/login");
  expect(response?.headers()["cache-control"]).toContain("no-store");
  expect(response?.headers()["x-robots-tag"]).toContain("noindex");
  await expect(page.locator('meta[name="robots"]').first()).toHaveAttribute("content", /noindex/);
  await expect(page.getByRole("heading", { name: "Masuk ke CMS" })).toBeVisible();
  await expect(page.locator(".site-header")).toBeHidden();

  const loginAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(loginAccessibility.violations).toEqual([]);

  await page.getByLabel("Email").fill("owner@example.test");
  await page.getByLabel("Password", { exact: true }).fill("correct-password-for-c4");
  await page.getByRole("button", { name: "Masuk", exact: true }).click();
  await expect(page).toHaveURL(/\/admin\/setup-totp$/);

  await page.goto("/admin/content");
  await expect(page).toHaveURL(/\/admin\/setup-totp$/);

  await page.getByRole("button", { name: "Buat kunci autentikator" }).click();
  await expect(page.getByText("JBSWY3DPEHPK3PXP")).toBeVisible();
  await page.getByLabel("Kode 6 digit").fill("123456");
  await page.getByRole("button", { name: "Aktifkan TOTP" }).click();
  await expect(page.locator(".admin-recovery-list code")).toHaveCount(10);
  await page.getByRole("button", { name: "Buka CMS" }).click();

  await expect(page).toHaveURL(/\/admin\/content$/);
  await expect(page.getByRole("heading", { name: "Pengelolaan konten" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Navigasi CMS" })).toBeVisible();
  await expect(page.locator("body")).not.toHaveCSS("overflow-x", "scroll");

  await page.getByRole("link", { name: "Akun" }).click();
  await expect(page.locator(".admin-account-summary").getByText("owner@example.test")).toBeVisible();
  await page.getByLabel("Kode autentikator").fill("654321");
  await page.getByRole("button", { name: "Buat ulang kode" }).click();
  await expect(page.locator(".admin-recovery-list code")).toHaveCount(10);

  await page.getByRole("link", { name: "Media" }).click();
  await expect(page.getByRole("heading", { name: "Pustaka media" })).toBeVisible();
  await page.getByLabel("File gambar").setInputFiles(resolve("public/images/tauco-dish-provisional.webp"));
  await page.getByLabel("Alt text").fill("Tumis tahu dan sayuran dengan bumbu tauco");
  await page.getByRole("button", { name: "Upload gambar" }).click();
  await expect(page.getByText("Upload diterima. Varian WebP diproses oleh worker.")).toBeVisible();
  await expect(page.locator(".admin-media-card")).toHaveCount(1);
  await expect(page.locator(".admin-media-card img")).toBeVisible();

  const mediaAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(mediaAccessibility.violations).toEqual([]);

  await page.getByRole("link", { name: "Konten", exact: true }).click();
  await page.getByRole("link", { name: /Homepage/ }).click();
  await expect(page.getByRole("heading", { name: "Editor terstruktur" })).toBeVisible();
  await page.getByRole("group", { name: "Hero", exact: true }).getByLabel("Judul").fill("Tauco Cap Badak Cianjur");
  await page.getByLabel("Fact-check dikonfirmasi").check();
  await page.getByRole("button", { name: "Save Draft" }).click();
  await expect(page.getByText(/Draft revision \d+ tersimpan/)).toBeVisible();
  await expect(page.locator(".admin-revision-panel li")).toHaveCount(2);

  const previewPromise = page.waitForEvent("popup");
  await page.getByRole("link", { name: "Preview" }).click();
  const preview = await previewPromise;
  await expect(preview.getByRole("heading", { name: "Tauco Cap Badak Cianjur" })).toBeVisible();
  await preview.close();

  await page.getByLabel("Fact-check dikonfirmasi").check();
  await page.getByRole("button", { name: "Publish", exact: true }).click();
  await expect(page.getByText(/dipublikasikan pada API lokal/)).toBeVisible();
  await expect(page.locator(".admin-revision-panel li")).toHaveCount(3);
  await page.getByRole("button", { name: "Unpublish", exact: true }).click();
  await expect(page.getByText("Halaman di-unpublish dari API lokal.")).toBeVisible();

  const editorAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(editorAccessibility.violations).toEqual([]);

  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
  const editorDarkAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(editorDarkAccessibility.violations).toEqual([]);
  await page.emulateMedia({ colorScheme: "light", reducedMotion: "no-preference" });

  await page.getByRole("link", { name: "Konten", exact: true }).click();
  await page.getByRole("link", { name: /Tentang Kami/ }).click();
  await page.getByRole("group", { name: "Bagian narasi" }).getByRole("group", { name: "Item 1" }).getByLabel("Heading").fill("Profil Tauco Cap Badak");
  await page.getByLabel("Fact-check dikonfirmasi").check();
  await page.getByRole("button", { name: "Save Draft" }).click();
  await expect(page.getByText(/Draft revision \d+ tersimpan/)).toBeVisible();

  const shellAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(shellAccessibility.violations).toEqual([]);

  await page.getByRole("button", { name: "Keluar" }).click();
  await expect(page).toHaveURL(/\/admin\/login$/);
});

test("BFF rejects unknown paths and unsupported methods", async ({ request }) => {
  expect((await request.get("/admin-api/not-allowed")).status()).toBe(404);
  const response = await request.get("/admin-api/auth/login");
  expect(response.status()).toBe(405);
  expect(response.headers().allow).toBe("POST");
  expect((await request.post("/admin-api/pages/tauco-guide/drafts")).status()).toBe(404);
});

test("admin auth remains readable in dark mode", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
  await page.goto("/admin/login");
  const accessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(accessibility.violations).toEqual([]);
});
