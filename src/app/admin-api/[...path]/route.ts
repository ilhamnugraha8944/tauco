import { notFound } from "next/navigation";
import type { NextRequest } from "next/server";

import { getAdminAPIOrigin, isAdminCMSEnabled } from "@/features/admin/config";

export const dynamic = "force-dynamic";

const exactRoutes = new Map<string, ReadonlySet<string>>([
  ["auth/login", new Set(["POST"])],
  ["auth/totp/setup", new Set(["POST"])],
  ["auth/totp/enable", new Set(["POST"])],
  ["auth/refresh", new Set(["POST"])],
  ["auth/logout", new Set(["POST"])],
  ["auth/me", new Set(["GET"])],
  ["auth/recovery-codes/regenerate", new Set(["POST"])],
]);

const dynamicRoutes = [
  { match: /^media$/, methods: new Set(["GET", "POST"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^media\/[0-9a-f-]{36}$/, methods: new Set(["GET"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^media\/[0-9a-f-]{36}\/retry$/, methods: new Set(["POST"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^public-media\/[0-9a-f-]{36}\/display\.webp$/, methods: new Set(["GET"]), target: (path: string) => `/api/v1/${path.replace("public-media", "media")}` },
  { match: /^public-media\/[0-9a-f-]{36}\/variants\/(320|640|1280)\.webp$/, methods: new Set(["GET"]), target: (path: string) => `/api/v1/${path.replace("public-media", "media")}` },
  { match: /^pages\/(home|about)$/, methods: new Set(["GET"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^pages\/(home|about)\/drafts$/, methods: new Set(["POST"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^pages\/(home|about)\/revisions\/[0-9a-f-]{36}$/, methods: new Set(["GET"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^pages\/(home|about)\/revisions\/[0-9a-f-]{36}\/publish$/, methods: new Set(["POST"]), target: (path: string) => `/api/v1/admin/${path}` },
  { match: /^pages\/(home|about)\/unpublish$/, methods: new Set(["POST"]), target: (path: string) => `/api/v1/admin/${path}` },
] as const;

const forwardedRequestHeaders = [
  "content-type",
  "content-length",
  "cookie",
  "if-match",
  "origin",
  "referer",
  "sec-fetch-site",
  "user-agent",
  "x-csrf-token",
  "x-request-id",
] as const;

const forwardedResponseHeaders = [
  "cache-control",
  "content-length",
  "content-type",
  "etag",
  "retry-after",
  "x-request-id",
] as const;

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

function problem(status: number, code: string, detail: string): Response {
  return Response.json(
    {
      type: `urn:tauco-cap-badak:problem:${code.toLowerCase()}`,
      title: status === 502 ? "Layanan admin tidak tersedia" : "Permintaan tidak valid",
      status,
      detail,
      instance: "/admin-api",
      code,
      requestId: "",
    },
    {
      status,
      headers: {
        "Cache-Control": "no-store",
        "Content-Type": "application/problem+json",
      },
    },
  );
}

async function proxy(request: NextRequest, context: RouteContext): Promise<Response> {
  if (!isAdminCMSEnabled()) {
    notFound();
  }

  const path = (await context.params).path.join("/");
  const dynamicRoute = dynamicRoutes.find((route) => route.match.test(path));
  const methods = exactRoutes.get(path) ?? dynamicRoute?.methods;

  if (!methods) {
    notFound();
  }

  if (!methods.has(request.method)) {
    return new Response(null, {
      status: 405,
      headers: { Allow: [...methods].join(", "), "Cache-Control": "no-store" },
    });
  }

  const payloadLimit = path === "media" && request.method === "POST" ? 10 * 1024 * 1024 + 64 * 1024 : 64 * 1024;
  const contentLength = Number(request.headers.get("content-length") ?? "0");

  if (Number.isFinite(contentLength) && contentLength > payloadLimit) {
    return problem(413, "PAYLOAD_TOO_LARGE", "Payload admin melebihi batas yang diizinkan.");
  }

  const body = request.method === "GET" ? undefined : await request.arrayBuffer();

  if (body && body.byteLength > payloadLimit) {
    return problem(413, "PAYLOAD_TOO_LARGE", "Payload admin melebihi batas yang diizinkan.");
  }

  const headers = new Headers();

  for (const name of forwardedRequestHeaders) {
    const value = request.headers.get(name);

    if (value) {
      headers.set(name, value);
    }
  }

  const targetPath = dynamicRoute?.target(path) ?? `/api/v1/admin/${path}`;
  const target = new URL(targetPath, getAdminAPIOrigin());
  target.search = request.nextUrl.search;

  try {
    const upstream = await fetch(target, {
      method: request.method,
      headers,
      body,
      cache: "no-store",
      redirect: "manual",
      signal: AbortSignal.timeout(10_000),
    });
    const responseHeaders = new Headers();

    for (const name of forwardedResponseHeaders) {
      const value = upstream.headers.get(name);

      if (value) {
        responseHeaders.set(name, value);
      }
    }

    for (const cookie of upstream.headers.getSetCookie()) {
      responseHeaders.append("Set-Cookie", cookie);
    }

    responseHeaders.set("Cache-Control", "no-store");

    return new Response(upstream.status === 204 ? null : upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch {
    return problem(502, "ADMIN_UPSTREAM_UNAVAILABLE", "Backend admin belum dapat dihubungi.");
  }
}

export const GET = proxy;
export const POST = proxy;
export const PATCH = proxy;
export const PUT = proxy;
export const DELETE = proxy;
