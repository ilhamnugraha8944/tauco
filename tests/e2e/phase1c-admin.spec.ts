import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
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
  const directPut = page.waitForResponse((response) =>
    response.url().includes("127.0.0.1:18081/__fixture/media-uploads/") &&
    response.request().method() === "PUT",
  );
  await page.getByRole("button", { name: "Upload gambar" }).click();
  expect((await directPut).status()).toBe(204);
  await expect(page.getByText("Upload diterima. Varian WebP diproses oleh worker.")).toBeVisible();
  await expect(page.locator(".admin-media-card")).toHaveCount(1, { timeout: 10_000 });
  await expect(page.locator(".admin-media-card img")).toBeVisible();
  await expect(page.getByRole("button", { name: "Upload gambar" })).toBeEnabled();

  let uploadEvidence = await (await page.request.get("http://127.0.0.1:18081/__fixture/media-upload-evidence")).json();
  expect(uploadEvidence).toMatchObject({ directPutCount: 1, legacyPostCount: 0 });

  await page.request.get("http://127.0.0.1:18081/__fixture/media-upload-evidence?enabled=false");
  await page.getByLabel("File gambar").setInputFiles(resolve("public/images/tauco-dish-provisional.webp"));
  await page.getByLabel("Alt text").fill("Fallback upload lokal untuk pengujian");
  const legacyUpload = page.waitForResponse((response) =>
    response.url().endsWith("/admin-api/media") && response.request().method() === "POST",
  );
  await page.getByRole("button", { name: "Upload gambar" }).click();
  expect((await legacyUpload).status()).toBe(202);
  uploadEvidence = await (await page.request.get("http://127.0.0.1:18081/__fixture/media-upload-evidence")).json();
  expect(uploadEvidence).toMatchObject({ directPutCount: 1, legacyPostCount: 1 });

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

  await page.getByRole("link", { name: "Produk" }).click();
  await expect(page.getByRole("heading", { name: "Pengelolaan produk" })).toBeVisible();
  await page.getByLabel("Slug").fill("produk-baru");
  await page.getByLabel("SKU opsional").fill("BARU-001");
  await page.getByLabel("Urutan").fill("20");
  await page.getByRole("button", { name: "Buat produk" }).click();
  await expect(page.getByText("Produk baru dibuat. Lengkapi draft sebelum publish.")).toBeVisible();
  await expect(page.locator(".admin-product-list article")).toHaveCount(2);

  await page.locator(".admin-product-list article").filter({ hasText: "tauco-cap-badak" }).getByRole("link", { name: "Kelola" }).click();
  await expect(page.getByRole("heading", { name: "Editor produk" })).toBeVisible();
  await expect(page.getByLabel("Slug").first()).toHaveAttribute("readonly", "");
  await page.getByLabel("Ringkasan", { exact: true }).fill("Produk tauco Cianjur untuk bumbu masakan rumahan.");
  await page.getByLabel("Fact-check dikonfirmasi").check();
  await page.getByRole("button", { name: "Save Draft" }).click();
  await expect(page.getByText(/Draft revision \d+ tersimpan/)).toBeVisible();

  const productPreviewPromise = page.waitForEvent("popup");
  await page.getByRole("link", { name: "Preview" }).click();
  const productPreview = await productPreviewPromise;
  await expect(productPreview.getByRole("heading", { name: "Tauco Cap Badak" })).toBeVisible();
  await productPreview.close();

  await page.getByLabel("Fact-check dikonfirmasi").check();
  await page.getByRole("button", { name: "Publish", exact: true }).click();
  await expect(page.getByText("Produk dipublikasikan pada API lokal.")).toBeVisible();
  await page.getByRole("button", { name: "Unpublish", exact: true }).click();
  await expect(page.getByText("Produk di-unpublish dari API lokal.")).toBeVisible();
  await page.getByRole("button", { name: "Archive", exact: true }).click();
  await expect(page.getByText("Produk diarsipkan.")).toBeVisible();
  await page.getByRole("button", { name: "Unarchive", exact: true }).click();
  await expect(page.getByText("Produk dipulihkan dari arsip.")).toBeVisible();

  const productAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(productAccessibility.violations).toEqual([]);

  await page.getByRole("link", { name: "Inbox" }).click();
  await expect(page.getByRole("heading", { name: "Inbox kontak" })).toBeVisible();
  await expect(page.locator(".admin-status-chip")).toHaveText("unread");
  await page.getByRole("link", { name: "Buka pesan" }).click();
  await expect(page.getByRole("heading", { name: "Pertanyaan umum" })).toBeVisible();
  await expect(page.getByText("unread", { exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByText("unread", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Tandai read" }).click();
  await expect(page.getByText("Status diubah menjadi read.")).toBeVisible();

  const inboxAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(inboxAccessibility.violations).toEqual([]);

  await page.getByRole("link", { name: "Aktivitas" }).click();
  await expect(page.getByRole("heading", { name: "Aktivitas CMS" })).toBeVisible();
  await expect(page.getByText("contact.status_changed")).toBeVisible();
  await page.getByLabel("Event type").fill("contact.status_changed");
  await page.getByLabel("Entity type").fill("contact_message");
  await page.getByRole("button", { name: "Terapkan filter" }).click();
  await expect(page.locator(".admin-activity-list li")).toHaveCount(1);
  await expect(page.locator(".admin-activity-list")).not.toContainText("visitor@example.test");

  const activityAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(activityAccessibility.violations).toEqual([]);

  const shellAccessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(shellAccessibility.violations).toEqual([]);

  await page.getByRole("button", { name: "Keluar" }).click();
  await expect(page).toHaveURL(/\/admin\/login$/);
});

test("BFF rejects unknown paths and unsupported methods", async ({ request }) => {
  const intentId = "019cf000-0000-7000-8000-000000000906";
  expect((await request.get("/admin-api/not-allowed")).status()).toBe(404);
  const response = await request.get("/admin-api/auth/login");
  expect(response.status()).toBe(405);
  expect(response.headers().allow).toBe("POST");
  expect((await request.post("/admin-api/pages/tauco-guide/drafts")).status()).toBe(404);
  expect((await request.post("/admin-api/contact-messages/019cf000-0000-7000-8000-000000000950/status")).status()).toBe(405);
  expect((await request.put(`/admin-api/media/upload-intents/${intentId}`)).status()).toBe(405);
  expect((await request.put(`/admin-api/media/upload-intents/${intentId}/content`)).status()).toBe(404);
});

test("remote-origin BFF allows direct PUT and rejects legacy multipart", async ({ request }) => {
  const origin = "http://localhost:3101";
  const login = await request.post("/admin-api/auth/login", {
    headers: { Origin: origin },
    data: { email: "owner@example.test", password: "correct-password-for-c4" },
  });
  expect(login.status()).toBe(200);

  const csrf = (await request.storageState()).cookies.find((item) => item.name === "tauco_admin_csrf")?.value;
  expect(csrf).toBeTruthy();
  const mutationHeaders = { Origin: origin, "X-CSRF-Token": csrf ?? "" };
  const enabled = await request.post("/admin-api/auth/totp/enable", {
    headers: mutationHeaders,
    data: { totpCode: "123456" },
  });
  expect(enabled.status()).toBe(200);

  const source = readFileSync(resolve("public/images/tauco-dish-provisional.webp"));
  const created = await request.post("/admin-api/media/upload-intents", {
    headers: mutationHeaders,
    data: {
      mimeType: "image/webp",
      bytes: source.byteLength,
      sha256: createHash("sha256").update(source).digest("hex"),
      altText: "Direct upload melalui BFF remote",
      decorative: false,
    },
  });
  expect(created.status()).toBe(201);
  const createdBody = (await created.json()) as {
    data: { intent: { id: string }; upload: { url: string; headers: Record<string, string> } };
  };
  const upload = createdBody.data.upload;

  const directPut = await request.put(upload.url, {
    headers: { ...upload.headers, Origin: origin },
    data: source,
  });
  expect(directPut.status()).toBe(204);

  const finalized = await request.post(
    `/admin-api/media/upload-intents/${createdBody.data.intent.id}/finalize`,
    { headers: mutationHeaders },
  );
  expect(finalized.status()).toBe(202);

  const legacy = await request.post("/admin-api/media", {
    headers: mutationHeaders,
    multipart: {
      altText: "Multipart harus ditolak pada origin remote",
      decorative: "false",
      file: { name: "tauco.webp", mimeType: "image/webp", buffer: source },
    },
  });
  expect(legacy.status()).toBe(404);
  expect(legacy.url()).toBe("http://localhost:3101/admin-api/media");

  const evidence = await (await request.get("http://127.0.0.1:18081/__fixture/media-upload-evidence")).json();
  expect(evidence).toMatchObject({ directPutCount: 1, legacyPostCount: 0 });
});

test("admin auth remains readable in dark mode", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
  await page.goto("/admin/login");
  const accessibility = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(accessibility.violations).toEqual([]);
});
