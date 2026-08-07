import { createServer } from "node:http";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const port = Number(process.env.ADMIN_FIXTURE_PORT ?? "18081");
const bffSecret = process.env.ADMIN_BFF_SHARED_SECRET ?? "";
const uploadOrigins = new Set(["http://localhost:3100", "http://localhost:3101"]);
const csrf = "c4-fixture-csrf-token";
const user = {
  id: "019cf000-0000-7000-8000-000000000901",
  email: "owner@example.test",
  status: "active",
  mfaEnabled: true,
  roles: ["super_admin"],
  permissions: ["account.manage", "media.read", "media.write", "content.read", "content.write", "content.publish", "product.read", "product.write", "product.publish", "inbox.read", "inbox.write", "activity.read"],
};
const media = [];
const mediaId = "019cf000-0000-7000-8000-000000000905";
const uploadIntentId = "019cf000-0000-7000-8000-000000000906";
let uploadIntent = null;
let uploadReceived = false;
let uploadPolls = 0;
let directUploadEnabled = true;
let directPutCount = 0;
let legacyPostCount = 0;
let revisionSequence = 10;
function initialPageState() { return Object.fromEntries(["home", "about"].map((key, index) => {
  const content = JSON.parse(readFileSync(resolve(`content/${key}.json`), "utf8"));
  const pageId = `019cf000-0000-7000-8000-0000000009${20 + index}`;
  const revision = { id: `019cf000-0000-7000-8000-0000000009${30 + index}`, ownerId: pageId, revisionNumber: 1, status: "published", schemaVersion: 1, content, createdAt: new Date().toISOString(), publishedAt: new Date().toISOString() };
  return [key, { id: pageId, key, latestRevision: revision, publishedRevisionId: revision.id, revisions: [revision] }];
})); }
let pageState = initialPageState();
function initialProductState() {
  const content = JSON.parse(readFileSync(resolve("content/products.json"), "utf8")).products[0];
  delete content.status;
  const productId = "019cf000-0000-7000-8000-000000000940";
  const revision = { id: "019cf000-0000-7000-8000-000000000941", ownerId: productId, revisionNumber: 1, status: "published", schemaVersion: 1, content, createdAt: new Date().toISOString(), publishedAt: new Date().toISOString() };
  return [{ id: productId, slug: content.slug, sku: "TAUCO-001", sortOrder: 0, publishedRevisionId: revision.id, archivedAt: null, updatedAt: revision.createdAt, revisions: [revision] }];
}
let productState = initialProductState();
function initialInboxState() { const now = new Date().toISOString(); return [{ id:"019cf000-0000-7000-8000-000000000950", name:"Pengunjung Test", email:"visitor@example.test", phone:null, subject:"Pertanyaan umum", message:"Pesan fixture untuk menguji inbox lokal tanpa data pelanggan production.", status:"unread", createdAt:now, updatedAt:now }]; }
function initialActivityState() { return [{ id:"019cf000-0000-7000-8000-000000000951", eventType:"contact.received", entityType:"contact_message", entityId:"019cf000-0000-7000-8000-000000000950", actorType:"visitor", actorId:null, requestId:"c8-fixture-request", createdAt:new Date().toISOString() }]; }
let inboxState = initialInboxState();
let activityState = initialActivityState();

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

function fixtureMedia(intent) {
  return {
    id: mediaId,
    status: "ready",
    mimeType: "image/png",
    width: 900,
    height: 1200,
    bytes: intent?.bytes ?? 8192,
    altText: intent?.altText ?? "Tumis tahu dan sayuran dengan bumbu tauco",
    decorative: intent?.decorative ?? false,
    variants: [{ width: 320, height: 427, bytes: 2048, url: `/api/v1/media/${mediaId}/variants/320.webp` }],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
}

const server = createServer(async (request, response) => {
  const requestURL = new URL(request.url ?? "/", `http://127.0.0.1:${port}`);
  const path = requestURL.pathname;
  const cookie = request.headers.cookie ?? "";

  if (path === "/__fixture/media-upload-evidence" && request.method === "GET") {
    if (requestURL.searchParams.has("enabled")) {
      directUploadEnabled = requestURL.searchParams.get("enabled") === "true";
    }
    json(response, 200, { directUploadEnabled, directPutCount, legacyPostCount });
    return;
  }

  if (path === `/__fixture/media-uploads/${uploadIntentId}` && request.method === "OPTIONS") {
    const origin = request.headers.origin ?? "";
    response.writeHead(204, {
      ...(uploadOrigins.has(origin) ? { "Access-Control-Allow-Origin": origin } : {}),
      "Access-Control-Allow-Methods": "PUT",
      "Access-Control-Allow-Headers": "content-type,x-fixture-upload-token",
      "Access-Control-Max-Age": "60",
    });
    response.end();
    return;
  }

  if (path === `/__fixture/media-uploads/${uploadIntentId}` && request.method === "PUT") {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    const uploaded = Buffer.concat(chunks);
    const origin = request.headers.origin ?? "";
    const valid = uploadIntent &&
      uploadOrigins.has(origin) &&
      request.headers["x-fixture-upload-token"] === "direct-put-only" &&
      request.headers["content-type"] === uploadIntent.mimeType &&
      uploaded.length === uploadIntent.bytes;
    if (!valid) {
      response.writeHead(422, uploadOrigins.has(origin) ? { "Access-Control-Allow-Origin": origin } : {});
      response.end();
      return;
    }
    uploadReceived = true;
    directPutCount += 1;
    response.writeHead(204, { "Access-Control-Allow-Origin": origin });
    response.end();
    return;
  }

  if (path.startsWith("/api/v1/admin/") && request.headers["x-tauco-admin-bff-secret"] !== bffSecret) {
    problem(response, 404, "ROUTE_NOT_FOUND");
    return;
  }

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
    uploadIntent = null;
    uploadReceived = false;
    uploadPolls = 0;
    directUploadEnabled = true;
    directPutCount = 0;
    legacyPostCount = 0;
    revisionSequence = 10;
    pageState = initialPageState();
    productState = initialProductState();
    inboxState = initialInboxState();
    activityState = initialActivityState();
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
  if (["POST", "PATCH"].includes(request.method ?? "") && !csrfValid) {
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

  if (path === "/api/v1/admin/media/upload-intents" && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    if (!directUploadEnabled) {
      problem(response, 404, "ROUTE_NOT_FOUND");
      return;
    }
    const input = await body(request);
    if (!["image/jpeg", "image/png", "image/webp"].includes(input.mimeType) ||
      !Number.isInteger(input.bytes) || input.bytes < 1 || input.bytes > 10 * 1024 * 1024 ||
      !/^[0-9a-f]{64}$/.test(input.sha256 ?? "")) {
      problem(response, 422, "VALIDATION_FAILED");
      return;
    }
    const now = new Date().toISOString();
    const expiresAt = new Date(Date.now() + 10 * 60_000).toISOString();
    uploadIntent = {
      id: uploadIntentId,
      status: "pending",
      mimeType: input.mimeType,
      bytes: input.bytes,
      sha256: input.sha256,
      altText: input.altText,
      decorative: input.decorative,
      expiresAt,
      createdAt: now,
      updatedAt: now,
    };
    uploadReceived = false;
    uploadPolls = 0;
    json(response, 201, {
      data: {
        intent: uploadIntent,
        upload: {
          url: `http://127.0.0.1:${port}/__fixture/media-uploads/${uploadIntentId}`,
          method: "PUT",
          headers: { "Content-Type": input.mimeType, "X-Fixture-Upload-Token": "direct-put-only" },
          expiresAt,
        },
      },
      meta: { apiVersion: "v1", requestId: "d2-fixture-request" },
    });
    return;
  }

  const uploadIntentMatch = /^\/api\/v1\/admin\/media\/upload-intents\/([0-9a-f-]{36})$/.exec(path);
  if (uploadIntentMatch && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    if (!uploadIntent || uploadIntentMatch[1] !== uploadIntent.id) {
      problem(response, 404, "MEDIA_UPLOAD_INTENT_NOT_FOUND");
      return;
    }
    uploadPolls += 1;
    uploadIntent = { ...uploadIntent, status: uploadPolls === 1 ? "queued" : "completed", updatedAt: new Date().toISOString() };
    if (uploadIntent.status === "completed") {
      uploadIntent.mediaAssetId = mediaId;
      media.splice(0, media.length, fixtureMedia(uploadIntent));
    }
    json(response, 200, { data: uploadIntent, meta: { apiVersion: "v1", requestId: "d2-fixture-request" } });
    return;
  }

  const finalizeUploadMatch = /^\/api\/v1\/admin\/media\/upload-intents\/([0-9a-f-]{36})\/finalize$/.exec(path);
  if (finalizeUploadMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    if (!uploadIntent || finalizeUploadMatch[1] !== uploadIntent.id) {
      problem(response, 404, "MEDIA_UPLOAD_INTENT_NOT_FOUND");
      return;
    }
    if (!uploadReceived) {
      problem(response, 409, "MEDIA_UPLOAD_MISSING");
      return;
    }
    uploadIntent = { ...uploadIntent, status: "queued", updatedAt: new Date().toISOString() };
    json(response, 202, { data: uploadIntent, meta: { apiVersion: "v1", requestId: "d2-fixture-request" } });
    return;
  }

  if (path === "/api/v1/admin/media" && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const chunks = [];
    for await (const chunk of request) chunks.push(chunk);
    if (!request.headers["content-type"]?.startsWith("multipart/form-data") || chunks.length === 0) {
      problem(response, 422, "VALIDATION_FAILED");
      return;
    }
    legacyPostCount += 1;
    const item = fixtureMedia();
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

  if (path === "/api/v1/admin/products" && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    json(response, 200, { data: productState.map(productResponse), meta: { apiVersion: "v1", requestId: "c7-fixture-request", page: { hasMore: false, limit: 50, nextCursor: null } } });
    return;
  }

  if (path === "/api/v1/admin/products" && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const input = await body(request);
    const id = `019cf000-0000-7000-8000-${String(++revisionSequence).padStart(12, "0")}`;
    const product = { id, slug: input.slug, sku: input.sku, sortOrder: input.sortOrder, publishedRevisionId: null, archivedAt: null, updatedAt: new Date().toISOString(), revisions: [] };
    productState.push(product);
    json(response, 201, { data: productResponse(product), meta: { apiVersion: "v1", requestId: "c7-fixture-request" } }, { ETag: `"revision-${id}"` });
    return;
  }

  const productMatch = /^\/api\/v1\/admin\/products\/([0-9a-f-]{36})$/.exec(path);
  if (productMatch && cookie.includes("tauco_admin_access=mfa-session")) {
    const product = productState.find((item) => item.id === productMatch[1]);
    if (!product) { problem(response, 404, "PRODUCT_NOT_FOUND"); return; }
    const latest = product.revisions[0];
    const etag = `"revision-${latest?.id ?? product.id}"`;
    if (request.method === "GET") {
      json(response, 200, { data: productResponse(product), meta: { apiVersion: "v1", requestId: "c7-fixture-request" } }, { ETag: etag }); return;
    }
    if (request.method === "PATCH") {
      if (request.headers["if-match"] !== etag) { problem(response, 412, "PRECONDITION_FAILED"); return; }
      const input = await body(request);
      if (product.revisions.some((item) => item.status === "published") && input.slug && input.slug !== product.slug) { problem(response, 409, "PRODUCT_CONFLICT"); return; }
      Object.assign(product, input, { updatedAt: new Date().toISOString() });
      json(response, 200, { data: productResponse(product), meta: { apiVersion: "v1", requestId: "c7-fixture-request" } }, { ETag: etag }); return;
    }
  }

  const productDraftMatch = /^\/api\/v1\/admin\/products\/([0-9a-f-]{36})\/drafts$/.exec(path);
  if (productDraftMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const product = productState.find((item) => item.id === productDraftMatch[1]);
    if (!product) { problem(response, 404, "PRODUCT_NOT_FOUND"); return; }
    const latest = product.revisions[0];
    if (request.headers["if-match"] !== `"revision-${latest?.id ?? product.id}"`) { problem(response, 412, "PRECONDITION_FAILED"); return; }
    const input = await body(request);
    const revision = newProductRevision(product, input.content, "draft");
    product.revisions.unshift(revision); product.updatedAt = revision.createdAt;
    json(response, 201, { data: revision, meta: { apiVersion: "v1", requestId: "c7-fixture-request" } }, { ETag: `"revision-${revision.id}"` }); return;
  }

  const productRevisionMatch = /^\/api\/v1\/admin\/products\/([0-9a-f-]{36})\/revisions\/([0-9a-f-]{36})$/.exec(path);
  if (productRevisionMatch && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    const product = productState.find((item) => item.id === productRevisionMatch[1]);
    const revision = product?.revisions.find((item) => item.id === productRevisionMatch[2]);
    if (!revision) { problem(response, 404, "PRODUCT_REVISION_NOT_FOUND"); return; }
    json(response, 200, { data: revision, meta: { apiVersion: "v1", requestId: "c7-fixture-request" } }, { ETag: `"revision-${revision.id}"` }); return;
  }

  const productPublishMatch = /^\/api\/v1\/admin\/products\/([0-9a-f-]{36})\/revisions\/([0-9a-f-]{36})\/publish$/.exec(path);
  if (productPublishMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const product = productState.find((item) => item.id === productPublishMatch[1]);
    const latest = product?.revisions[0]; const source = product?.revisions.find((item) => item.id === productPublishMatch[2]);
    if (!product || !source) { problem(response, 404, "PRODUCT_REVISION_NOT_FOUND"); return; }
    if (request.headers["if-match"] !== `"revision-${latest.id}"`) { problem(response, 412, "PRECONDITION_FAILED"); return; }
    const revision = newProductRevision(product, source.content, "published");
    product.revisions.unshift(revision); product.publishedRevisionId = revision.id; product.updatedAt = revision.createdAt;
    json(response, 200, { data: revision, meta: { apiVersion: "v1", requestId: "c7-fixture-request" } }, { ETag: `"revision-${revision.id}"` }); return;
  }

  const productStateMatch = /^\/api\/v1\/admin\/products\/([0-9a-f-]{36})\/(unpublish|archive|unarchive)$/.exec(path);
  if (productStateMatch && request.method === "POST" && cookie.includes("tauco_admin_access=mfa-session")) {
    const product = productState.find((item) => item.id === productStateMatch[1]);
    const latest = product?.revisions[0];
    if (!product || !latest) { problem(response, 404, "PRODUCT_NOT_FOUND"); return; }
    if (request.headers["if-match"] !== `"revision-${latest.id}"`) { problem(response, 412, "PRECONDITION_FAILED"); return; }
    if (productStateMatch[2] === "unpublish") product.publishedRevisionId = null;
    if (productStateMatch[2] === "archive") product.archivedAt = new Date().toISOString();
    if (productStateMatch[2] === "unarchive") product.archivedAt = null;
    response.writeHead(204, { "Cache-Control": "no-store" }); response.end(); return;
  }

  if (path === "/api/v1/admin/contact-messages" && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    const status = requestURL.searchParams.get("status");
    const items = status ? inboxState.filter((item) => item.status === status) : inboxState;
    json(response, 200, { data:items, meta:{ apiVersion:"v1", requestId:"c8-fixture-request", page:{ hasMore:false, limit:20, nextCursor:null } } }); return;
  }

  const inboxMatch = /^\/api\/v1\/admin\/contact-messages\/([0-9a-f-]{36})$/.exec(path);
  if (inboxMatch && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    const item = inboxState.find((message) => message.id === inboxMatch[1]);
    if (!item) { problem(response,404,"CONTACT_NOT_FOUND"); return; }
    json(response,200,{data:item,meta:{apiVersion:"v1",requestId:"c8-fixture-request"}},{ETag:messageETag(item)}); return;
  }

  const inboxStatusMatch = /^\/api\/v1\/admin\/contact-messages\/([0-9a-f-]{36})\/status$/.exec(path);
  if (inboxStatusMatch && request.method === "PATCH" && cookie.includes("tauco_admin_access=mfa-session")) {
    const item = inboxState.find((message) => message.id === inboxStatusMatch[1]);
    if (!item) { problem(response,404,"CONTACT_NOT_FOUND"); return; }
    if (request.headers["if-match"] !== messageETag(item)) { problem(response,412,"PRECONDITION_FAILED"); return; }
    const input = await body(request); const previous = item.status;
    item.status = input.status; item.updatedAt = new Date(Date.now()+10).toISOString();
    activityState.unshift({id:`019cf000-0000-7000-8000-${String(++revisionSequence).padStart(12,"0")}`,eventType:"contact.status_changed",entityType:"contact_message",entityId:item.id,actorType:"admin",actorId:user.id,requestId:"c8-fixture-request",createdAt:item.updatedAt});
    json(response,200,{data:item,meta:{apiVersion:"v1",requestId:"c8-fixture-request"}},{ETag:messageETag(item),"X-Fixture-Previous-Status":previous}); return;
  }

  if (path === "/api/v1/admin/activity-logs" && request.method === "GET" && cookie.includes("tauco_admin_access=mfa-session")) {
    const eventType=requestURL.searchParams.get("eventType"),entityType=requestURL.searchParams.get("entityType");
    const items=activityState.filter((item)=>(!eventType||item.eventType===eventType)&&(!entityType||item.entityType===entityType));
    json(response,200,{data:items,meta:{apiVersion:"v1",requestId:"c8-fixture-request",page:{hasMore:false,limit:20,nextCursor:null}}}); return;
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

server.listen(port, "0.0.0.0", () => process.stdout.write(`Admin fixture listening on ${port}\n`));

function productResponse(product) {
  return { ...product, revisions: product.revisions.map(revisionSummary) };
}

function newProductRevision(product, content, status) {
  revisionSequence += 1;
  const now = new Date().toISOString();
  return { id: `019cf000-0000-7000-8000-${String(revisionSequence).padStart(12, "0")}`, ownerId: product.id, revisionNumber: (product.revisions[0]?.revisionNumber ?? 0) + 1, status, schemaVersion: 1, content, createdAt: now, ...(status === "published" ? { publishedAt: now } : {}) };
}

function messageETag(message){return `"message-${message.id}-${Date.parse(message.updatedAt)*1000}"`;}
