import { createServer } from "node:http";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

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
let revisionSequence = 10;
function initialPageState() { return Object.fromEntries(["home", "about"].map((key, index) => {
  const content = JSON.parse(readFileSync(resolve(`content/${key}.json`), "utf8"));
  const pageId = `019cf000-0000-7000-8000-0000000009${20 + index}`;
  const revision = { id: `019cf000-0000-7000-8000-0000000009${30 + index}`, ownerId: pageId, revisionNumber: 1, status: "published", schemaVersion: 1, content, createdAt: new Date().toISOString(), publishedAt: new Date().toISOString() };
  return [key, { id: pageId, key, latestRevision: revision, publishedRevisionId: revision.id, revisions: [revision] }];
})); }
let pageState = initialPageState();

function revisionSummary(revision) {
  return { id: revision.id, revisionNumber: revision.revisionNumber, status: revision.status, createdBy: revision.createdBy, createdAt: revision.createdAt, publishedAt: revision.publishedAt };
}

function pageResponse(state) {
  return { ...state, revisions: state.revisions.map(revisionSummary), updatedAt: state.latestRevision.createdAt };
}

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
    media.length = 0;
    revisionSequence = 10;
    pageState = initialPageState();
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

  const pageMatch = /^\/api\/v1\/admin\/pages\/(home|about)$/.exec(path);
  if (pageMatch && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    const state = pageState[pageMatch[1]];
    json(response, 200, { data: pageResponse(state), meta: { apiVersion: "v1", requestId: "c6-fixture-request" } }, { ETag: `"revision-${state.latestRevision.id}"` });
    return;
  }

  const draftMatch = /^\/api\/v1\/admin\/pages\/(home|about)\/drafts$/.exec(path);
  if (draftMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const state = pageState[draftMatch[1]];
    if (request.headers["if-match"] !== `"revision-${state.latestRevision.id}"`) { problem(response, 412, "PRECONDITION_FAILED"); return; }
    const input = await body(request);
    revisionSequence += 1;
    const revision = { id: `019cf000-0000-7000-8000-${String(revisionSequence).padStart(12, "0")}`, ownerId: state.id, revisionNumber: state.latestRevision.revisionNumber + 1, status: "draft", schemaVersion: 1, content: input.content, createdAt: new Date().toISOString() };
    state.latestRevision = revision;
    state.revisions.unshift(revision);
    json(response, 201, { data: revision, meta: { apiVersion: "v1", requestId: "c6-fixture-request" } }, { ETag: `"revision-${revision.id}"` });
    return;
  }

  const revisionMatch = /^\/api\/v1\/admin\/pages\/(home|about)\/revisions\/([0-9a-f-]{36})$/.exec(path);
  if (revisionMatch && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    const revision = pageState[revisionMatch[1]].revisions.find((item) => item.id === revisionMatch[2]);
    if (!revision) { problem(response, 404, "CONTENT_NOT_FOUND"); return; }
    json(response, 200, { data: revision, meta: { apiVersion: "v1", requestId: "c6-fixture-request" } }, { ETag: `"revision-${revision.id}"` });
    return;
  }

  const publishMatch = /^\/api\/v1\/admin\/pages\/(home|about)\/revisions\/([0-9a-f-]{36})\/publish$/.exec(path);
  if (publishMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const state = pageState[publishMatch[1]];
    if (request.headers["if-match"] !== `"revision-${state.latestRevision.id}"`) { problem(response, 412, "PRECONDITION_FAILED"); return; }
    const source = state.revisions.find((item) => item.id === publishMatch[2]);
    if (!source) { problem(response, 404, "CONTENT_NOT_FOUND"); return; }
    revisionSequence += 1;
    const revision = { ...source, id: `019cf000-0000-7000-8000-${String(revisionSequence).padStart(12, "0")}`, revisionNumber: state.latestRevision.revisionNumber + 1, status: "published", createdAt: new Date().toISOString(), publishedAt: new Date().toISOString() };
    state.latestRevision = revision; state.publishedRevisionId = revision.id; state.revisions.unshift(revision);
    json(response, 200, { data: revision, meta: { apiVersion: "v1", requestId: "c6-fixture-request" } }, { ETag: `"revision-${revision.id}"` });
    return;
  }

  const unpublishMatch = /^\/api\/v1\/admin\/pages\/(home|about)\/unpublish$/.exec(path);
  if (unpublishMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const state = pageState[unpublishMatch[1]];
    if (request.headers["if-match"] !== `"revision-${state.latestRevision.id}"`) { problem(response, 412, "PRECONDITION_FAILED"); return; }
    state.publishedRevisionId = null;
    response.writeHead(204, { "Cache-Control": "no-store" }); response.end(); return;
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
