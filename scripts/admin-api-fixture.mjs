import { createServer } from "node:http";

const port = Number(process.env.ADMIN_FIXTURE_PORT ?? "18081");
const csrf = "c4-fixture-csrf-token";
const user = {
  id: "019cf000-0000-7000-8000-000000000901",
  email: "owner@example.test",
  status: "active",
  mfaEnabled: true,
  roles: ["super_admin"],
  permissions: ["account.manage", "media.read", "media.write", "content.read", "content.write", "content.publish"],
};
const media = [];
const mediaId = "019cf000-0000-7000-8000-000000000905";

function json(response, status, data, headers = {}) {
  response.writeHead(status, {
    "Content-Type": "application/json",
    "Cache-Control": "no-store",
    "X-Request-ID": "c4-fixture-request",
    ...headers,
  });
  response.end(JSON.stringify(data));
}

function problem(response, status, code) {
  json(response, status, { status, code, detail: "Fixture menolak permintaan." });
}

async function body(request) {
  const chunks = [];
  for await (const chunk of request) chunks.push(chunk);
  return chunks.length ? JSON.parse(Buffer.concat(chunks).toString("utf8")) : {};
}

const server = createServer(async (request, response) => {
  const path = new URL(request.url ?? "/", `http://127.0.0.1:${port}`).pathname;
  const cookie = request.headers.cookie ?? "";

  if (request.method === "POST" && !request.headers.origin) {
    problem(response, 403, "ORIGIN_MISSING");
    return;
  }

  if (path === "/api/v1/admin/auth/login" && request.method === "POST") {
    const input = await body(request);
    if (input.email !== user.email || input.password !== "correct-password-for-c4") {
      problem(response, 401, "AUTHENTICATION_FAILED");
      return;
    }
    user.mfaEnabled = false;
    json(response, 200, { data: { status: "mfa_setup_required", expiresAt: new Date(Date.now() + 600_000).toISOString(), user }, meta: { apiVersion: "v1", requestId: "c4-fixture-request" } }, {
      "Set-Cookie": [
        "tauco_admin_access=password-session; Path=/; HttpOnly; SameSite=Strict",
        "tauco_admin_refresh=refresh-session; Path=/; HttpOnly; SameSite=Strict",
        `tauco_admin_csrf=${csrf}; Path=/; SameSite=Strict`,
      ],
    });
    return;
  }

  const csrfValid = request.headers["x-csrf-token"] === csrf && cookie.includes(`tauco_admin_csrf=${csrf}`);
  if (request.method === "POST" && !csrfValid) {
    problem(response, 403, "CSRF_REJECTED");
    return;
  }

  if (path === "/api/v1/admin/auth/totp/setup" && request.method === "POST" && cookie.includes("tauco_admin_access=password-session")) {
    json(response, 200, { data: { manualKey: "JBSWY3DPEHPK3PXP", otpauthUri: "otpauth://totp/Tauco%20Cap%20Badak", expiresAt: new Date(Date.now() + 600_000).toISOString() }, meta: { apiVersion: "v1", requestId: "c4-fixture-request" } });
    return;
  }

  if (path === "/api/v1/admin/auth/totp/enable" && request.method === "POST") {
    const input = await body(request);
    if (!/^\d{6}$/.test(input.totpCode ?? "")) {
      problem(response, 401, "AUTHENTICATION_FAILED");
      return;
    }
    user.mfaEnabled = true;
    json(response, 200, { data: { codes: Array.from({ length: 10 }, (_, index) => `C4AA-C4BB-${"ABCDEFGHJK"[index]}222`) }, meta: { apiVersion: "v1", requestId: "c4-fixture-request" } }, {
      "Set-Cookie": "tauco_admin_access=mfa-session; Path=/; HttpOnly; SameSite=Strict",
    });
    return;
  }

  if (path === "/api/v1/admin/auth/me" && request.method === "GET" && cookie.includes("tauco_admin_access=")) {
    json(response, 200, { data: user, meta: { apiVersion: "v1", requestId: "c4-fixture-request" } });
    return;
  }

  if (path === "/api/v1/admin/auth/recovery-codes/regenerate" && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    json(response, 200, { data: { codes: Array.from({ length: 10 }, (_, index) => `NEWA-NEWB-${"ABCDEFGHJK"[index]}222`) }, meta: { apiVersion: "v1", requestId: "c4-fixture-request" } });
    return;
  }

  if (path === "/api/v1/admin/media" && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    json(response, 200, { data: media, meta: { apiVersion: "v1", requestId: "c5-fixture-request", page: { hasMore: false, limit: 50, nextCursor: null } } });
    return;
  }

  if (path === "/api/v1/admin/media" && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    if (!request.headers["content-type"]?.startsWith("multipart/form-data") || chunks.length === 0) {
      problem(response, 422, "VALIDATION_FAILED");
      return;
    }
    const item = { id: mediaId, status: "ready", mimeType: "image/png", width: 900, height: 1200, bytes: 8192, altText: "Tumis tahu dan sayuran dengan bumbu tauco", decorative: false, variants: [{ width: 320, height: 427, bytes: 2048, url: `/api/v1/media/${mediaId}/variants/320.webp` }], createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() };
    media.splice(0, media.length, item);
    json(response, 202, { data: item, meta: { apiVersion: "v1", requestId: "c5-fixture-request" } });
    return;
  }

  if (path === `/api/v1/admin/media/${mediaId}` && request.method === "GET" && media[0]) {
    json(response, 200, { data: media[0], meta: { apiVersion: "v1", requestId: "c5-fixture-request" } });
    return;
  }

  if (path === `/api/v1/media/${mediaId}/display.webp` && request.method === "GET" && media[0]) {
    const image = Buffer.from("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEALmk0mk0iIiIiIgBoSygABc6zbAAA", "base64");
    response.writeHead(200, { "Content-Type": "image/webp", "Content-Length": image.length, "Cache-Control": "public, max-age=0" });
    response.end(image);
    return;
  }

  if (path === "/api/v1/admin/auth/logout" && request.method === "POST") {
    response.writeHead(204, {
      "Cache-Control": "no-store",
      "Set-Cookie": [
        "tauco_admin_access=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict",
        "tauco_admin_refresh=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict",
        "tauco_admin_csrf=; Path=/; Max-Age=0; SameSite=Strict",
      ],
    });
    response.end();
    return;
  }

  problem(response, 401, "UNAUTHORIZED");
});

server.listen(port, "127.0.0.1", () => process.stdout.write(`Admin fixture listening on ${port}\n`));
