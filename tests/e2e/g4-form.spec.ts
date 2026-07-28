import {
  expect,
  test,
  type Page,
  type TestInfo,
} from "@playwright/test";

function requirePreviewOrigin(): string {
  const configuredUrl = process.env.PLAYWRIGHT_BASE_URL?.trim();

  if (!configuredUrl) {
    throw new Error(
      "PLAYWRIGHT_BASE_URL wajib diisi dengan origin Deploy Preview.",
    );
  }

  return new URL(configuredUrl).origin;
}

async function fillValidForm(page: Page, message: string) {
  await page.getByLabel(/^Nama/).fill("QA Phase 1A");
  await page.getByLabel(/^Email/).fill("qa-phase1a@example.com");
  await page.getByLabel("Topik").selectOption("Pertanyaan umum");
  await page
    .getByRole("textbox", { name: /^Pesan/ })
    .fill(message);
  await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();
}

test.beforeAll(() => {
  requirePreviewOrigin();
});

test("contact form exposes the complete Netlify field contract", async ({
  page,
}) => {
  await page.goto("/kontak", { waitUntil: "domcontentloaded" });

  const form = page.locator("form.contact-form");

  await expect(form).toHaveAttribute("name", "kontak");
  await expect(form).toHaveAttribute("method", /post/i);
  await expect(form).toHaveAttribute("action", "/__forms.html");
  await expect(form).toHaveAttribute("data-netlify", "true");
  await expect(form).toHaveAttribute(
    "data-netlify-honeypot",
    "bot-field",
  );
  await expect(form.locator('input[name="form-name"]')).toHaveValue(
    "kontak",
  );
  await expect(form.locator('input[name="bot-field"]')).toHaveCount(1);
  await expect(form.locator('input[name="bot-field"]')).toHaveValue("");
  await expect(form.locator('input[type="file"]')).toHaveCount(0);

  await expect(page.getByLabel(/^Nama/)).toHaveAttribute("required", "");
  await expect(page.getByLabel(/^Email/)).toHaveAttribute("required", "");
  await expect(page.getByLabel(/^Telepon/)).not.toHaveAttribute(
    "required",
    "",
  );
  await expect(page.getByLabel("Topik")).toHaveAttribute("required", "");
  await expect(
    page.getByRole("textbox", { name: /^Pesan/ }),
  ).toHaveAttribute("required", "");
  await expect(
    page.getByRole("checkbox", { name: /Saya menyetujui/ }),
  ).toHaveAttribute("required", "");
});

test("native validation rejects invalid email and short messages without JavaScript", async ({
  browser,
}) => {
  const context = await browser.newContext({
    baseURL: requirePreviewOrigin(),
    javaScriptEnabled: false,
  });
  const page = await context.newPage();

  try {
    await page.goto("/kontak", { waitUntil: "domcontentloaded" });
    const form = page.locator("form.contact-form");

    await expect(form).not.toHaveAttribute("novalidate", "");
    await page.getByLabel(/^Nama/).fill("QA Phase 1A");
    await page.getByLabel(/^Email/).fill("bukan-email");
    await page
      .getByRole("textbox", { name: /^Pesan/ })
      .fill("Terlalu pendek");
    await page
      .getByRole("checkbox", { name: /Saya menyetujui/ })
      .check();

    expect(
      await form.evaluate((element) =>
        element instanceof HTMLFormElement
          ? element.checkValidity()
          : true,
      ),
    ).toBe(false);
    expect(
      await page
        .getByLabel(/^Email/)
        .evaluate((element) =>
          element instanceof HTMLInputElement
            ? element.validity.typeMismatch
            : false,
        ),
    ).toBe(true);
    expect(
      await page
        .getByRole("textbox", { name: /^Pesan/ })
        .evaluate((element) =>
          element instanceof HTMLTextAreaElement
            ? element.validity.tooShort
            : false,
        ),
    ).toBe(true);
  } finally {
    await context.close();
  }
});

test("client validation focuses the first invalid field", async ({
  page,
}) => {
  await page.goto("/kontak", { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "Kirim pesan" }).click();

  await expect(page.getByLabel(/^Nama/)).toBeFocused();
  await expect(
    page.getByRole("alert").filter({
      hasText:
        "Periksa kembali kolom yang ditandai sebelum mengirim pesan.",
    }),
  ).toBeVisible();

  await page.getByLabel(/^Nama/).fill("QA Phase 1A");
  await page.getByLabel(/^Email/).fill("bukan-email");
  await page
    .getByRole("textbox", { name: /^Pesan/ })
    .fill("Terlalu pendek");
  await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();
  await page.getByRole("button", { name: "Kirim pesan" }).click();

  await expect(page.getByLabel(/^Email/)).toBeFocused();
  await expect(
    page.getByText("Masukkan alamat email yang valid."),
  ).toBeVisible();
  await expect(
    page.getByText("Pesan minimal 20 karakter."),
  ).toBeVisible();
});

test("success feedback appears after an accepted request", async ({
  page,
}, testInfo) => {
  await page.route("**/__forms.html", async (route) => {
    await route.fulfill({ body: "ok", status: 200 });
  });
  await page.goto("/kontak", { waitUntil: "domcontentloaded" });
  await fillValidForm(
    page,
    "Pengujian state sukses Phase 1A, bukan pertanyaan pelanggan.",
  );
  await page.getByRole("button", { name: "Kirim pesan" }).click();

  await expect(
    page.getByText(
      "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
    ),
  ).toBeVisible();
  await expect(page.getByLabel(/^Nama/)).toHaveValue("");
  await expect(page.getByLabel(/^Email/)).toHaveValue("");
  await testInfo.attach("contact-success-state", {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
});

test("duplicate submissions are locked while the request is pending", async ({
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
    await route.fulfill({ body: "ok", status: 200 });
  });
  await page.goto("/kontak", { waitUntil: "domcontentloaded" });
  await fillValidForm(
    page,
    "Pengujian duplicate lock Phase 1A, bukan pertanyaan pelanggan.",
  );

  const form = page.locator("form.contact-form");

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
    await page.waitForTimeout(200);
    expect(requestCount).toBe(1);
  } finally {
    releaseResponse?.();
  }

  await expect(
    page.getByText(
      "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
    ),
  ).toBeVisible();
});

test("network errors retain input and restore the submit control", async ({
  page,
}) => {
  await page.route("**/__forms.html", async (route) => {
    await route.abort("failed");
  });
  await page.goto("/kontak", { waitUntil: "domcontentloaded" });
  const message =
    "Pengujian network error Phase 1A, bukan pertanyaan pelanggan.";

  await fillValidForm(page, message);
  await page.getByRole("button", { name: "Kirim pesan" }).click();

  await expect(
    page.getByText(
      "Pesan belum terkirim. Periksa koneksi Anda, lalu coba kembali.",
    ),
  ).toBeVisible();
  await expect(page.getByLabel(/^Nama/)).toHaveValue("QA Phase 1A");
  await expect(page.getByLabel(/^Email/)).toHaveValue(
    "qa-phase1a@example.com",
  );
  await expect(
    page.getByRole("textbox", { name: /^Pesan/ }),
  ).toHaveValue(message);
  await expect(
    page.getByRole("checkbox", { name: /Saya menyetujui/ }),
  ).toBeChecked();
  await expect(
    page.getByRole("button", { name: "Kirim pesan" }),
  ).toBeEnabled();
});

test("live Netlify Forms accepts one synthetic submission", async ({
  page,
}, testInfo: TestInfo) => {
  test.skip(
    process.env.G4_LIVE_FORM_SUBMISSION !== "true",
    "Aktifkan hanya sekali dengan G4_LIVE_FORM_SUBMISSION=true.",
  );

  const runId =
    process.env.G4_SUBMISSION_RUN_ID?.trim() ||
    new Date().toISOString();
  const message =
    `Pengujian deployment Phase 1A ${runId}, bukan pertanyaan pelanggan.`;

  await page.goto("/kontak", { waitUntil: "domcontentloaded" });
  await fillValidForm(page, message);

  const responsePromise = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      response.url().endsWith("/__forms.html"),
  );
  await page.getByRole("button", { name: "Kirim pesan" }).click();
  const response = await responsePromise;
  const requestId =
    response.headers()["x-nf-request-id"] ??
    response.headers()["x-request-id"] ??
    "tidak tersedia";

  expect(response.status()).toBe(200);
  await expect(
    page.getByText(
      "Pesan berhasil dikirim. Terima kasih telah menghubungi kami.",
    ),
  ).toBeVisible();

  const evidence = {
    formName: "kontak",
    requestId,
    runId,
    status: response.status(),
    submittedAt: new Date().toISOString(),
    target: requirePreviewOrigin(),
  };

  console.log(`G4 live form evidence: ${JSON.stringify(evidence)}`);
  await testInfo.attach("live-netlify-form-evidence", {
    body: Buffer.from(JSON.stringify(evidence, null, 2)),
    contentType: "application/json",
  });
  await testInfo.attach("live-contact-success", {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
});
