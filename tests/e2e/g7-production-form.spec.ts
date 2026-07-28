import { expect, test, type TestInfo } from "@playwright/test";

test("one live production form submission is accepted", async ({
  page,
}, testInfo: TestInfo) => {
  test.skip(
    process.env.G7_LIVE_FORM_SUBMISSION !== "true",
    "Aktifkan hanya sekali dengan G7_LIVE_FORM_SUBMISSION=true.",
  );

  const runId =
    process.env.G7_SUBMISSION_RUN_ID?.trim() ??
    new Date().toISOString();
  const message =
    `Pengujian production Phase 1A ${runId}, bukan pertanyaan pelanggan.`;

  await page.goto("/kontak", { waitUntil: "domcontentloaded" });
  await page.getByLabel(/^Nama/).fill("QA Production Phase 1A");
  await page.getByLabel(/^Email/).fill("qa-phase1a@example.com");
  await page.getByLabel("Topik").selectOption("Pertanyaan umum");
  await page.getByRole("textbox", { name: /^Pesan/ }).fill(message);
  await page.getByRole("checkbox", { name: /Saya menyetujui/ }).check();

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
    target: "https://tauco-cap-badak.netlify.app",
  };

  console.log(`G7 live form evidence: ${JSON.stringify(evidence)}`);
  await testInfo.attach("g7-live-form-evidence", {
    body: Buffer.from(JSON.stringify(evidence, null, 2)),
    contentType: "application/json",
  });
  await testInfo.attach("g7-live-form-success", {
    body: await page.screenshot({ fullPage: true }),
    contentType: "image/png",
  });
});
