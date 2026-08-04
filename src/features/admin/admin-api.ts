export type AdminUser = {
  id: string;
  email: string;
  status: "active" | "disabled";
  mfaEnabled: boolean;
  roles: string[];
  permissions: string[];
};

type Problem = { detail?: string; code?: string };

export class AdminAPIError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: string,
  ) {
    super(message);
  }
}

function cookie(name: string): string {
  const prefix = `${encodeURIComponent(name)}=`;
  const match = document.cookie
    .split(";")
    .map((value) => value.trim())
    .find((value) => value.startsWith(prefix));

  return match ? decodeURIComponent(match.slice(prefix.length)) : "";
}

async function sendWithResponse<T>(path: string, init: RequestInit, retry = true): Promise<{ body: T; headers: Headers }> {
  const method = init.method ?? "GET";
  const headers = new Headers(init.headers);
  const csrf = cookie("tauco_admin_csrf");

  if (typeof init.body === "string") {
    headers.set("Content-Type", "application/json");
  }

  if (method !== "GET" && csrf) {
    headers.set("X-CSRF-Token", csrf);
  }

  const response = await fetch(`/admin-api/${path}`, {
    ...init,
    headers,
    cache: "no-store",
    credentials: "same-origin",
  });

  if (response.status === 401 && retry && path !== "auth/login" && path !== "auth/refresh") {
    try {
      await send("auth/refresh", { method: "POST" }, false);
      return sendWithResponse<T>(path, init, false);
    } catch {
      // Gunakan error asli agar UI tetap memberi pesan yang konsisten.
    }
  }

  if (!response.ok) {
    const value = (await response.json().catch(() => ({}))) as Problem;
    throw new AdminAPIError(
      value.detail ?? "Permintaan admin tidak dapat diproses.",
      response.status,
      value.code ?? "ADMIN_REQUEST_FAILED",
    );
  }

  return { body: (response.status === 204 ? undefined : await response.json()) as T, headers: response.headers };
}

async function send<T>(path: string, init: RequestInit, retry = true): Promise<T> {
  return (await sendWithResponse<T>(path, init, retry)).body;
}

export const adminAPI = {
  login(input: { email: string; password: string; totpCode?: string; recoveryCode?: string }) {
    return send<{ data: { status: "authenticated" | "mfa_setup_required"; user: AdminUser } }>(
      "auth/login",
      { method: "POST", body: JSON.stringify(input) },
      false,
    );
  },
  me() {
    return send<{ data: AdminUser }>("auth/me", { method: "GET" });
  },
  setupTOTP() {
    return send<{ data: { manualKey: string; otpauthUri: string; expiresAt: string } }>("auth/totp/setup", { method: "POST" });
  },
  enableTOTP(totpCode: string) {
    return send<{ data: { codes: string[] } }>("auth/totp/enable", { method: "POST", body: JSON.stringify({ totpCode }) });
  },
  regenerateRecoveryCodes(totpCode: string) {
    return send<{ data: { codes: string[] } }>("auth/recovery-codes/regenerate", { method: "POST", body: JSON.stringify({ totpCode }) });
  },
  logout() {
    return send<void>("auth/logout", { method: "POST" }, false);
  },
  listMedia() {
    return send<{ data: AdminMedia[] }>("media?limit=50", { method: "GET" });
  },
  getMedia(id: string) {
    return send<{ data: AdminMedia }>(`media/${id}`, { method: "GET" });
  },
  uploadMedia(input: { file: File; altText: string; decorative: boolean }) {
    const body = new FormData();
    body.set("file", input.file);
    body.set("altText", input.altText);
    body.set("decorative", String(input.decorative));
    return send<{ data: AdminMedia }>("media", { method: "POST", body });
  },
  retryMedia(id: string) {
    return send<{ data: AdminMedia }>(`media/${id}/retry`, { method: "POST" });
  },
  async getPage(key: AdminPageKey) {
    const response = await sendWithResponse<{ data: AdminPage }>(`pages/${key}`, { method: "GET" });
    return { ...response.body, etag: response.headers.get("etag") ?? "" };
  },
  async getPageRevision(key: AdminPageKey, id: string) {
    const response = await sendWithResponse<{ data: AdminRevision }>(`pages/${key}/revisions/${id}`, { method: "GET" });
    return { ...response.body, etag: response.headers.get("etag") ?? "" };
  },
  async savePageDraft(key: AdminPageKey, etag: string, baseRevisionId: string, content: EditableContent) {
    const response = await sendWithResponse<{ data: AdminRevision }>(`pages/${key}/drafts`, { method: "POST", headers: { "If-Match": etag }, body: JSON.stringify({ baseRevisionId, content }) });
    return { ...response.body, etag: response.headers.get("etag") ?? "" };
  },
  async publishPage(key: AdminPageKey, revisionId: string, etag: string) {
    const response = await sendWithResponse<{ data: AdminRevision }>(`pages/${key}/revisions/${revisionId}/publish`, { method: "POST", headers: { "If-Match": etag } });
    return { ...response.body, etag: response.headers.get("etag") ?? "" };
  },
  unpublishPage(key: AdminPageKey, etag: string) {
    return send<void>(`pages/${key}/unpublish`, { method: "POST", headers: { "If-Match": etag } });
  },
};

export type AdminMedia = {
  id: string;
  status: "processing" | "ready" | "failed";
  mimeType: string;
  width: number;
  height: number;
  bytes: number;
  altText: string;
  decorative: boolean;
  lastErrorCode?: string;
  variants: Array<{ width: number; height: number; bytes: number; url: string }>;
  createdAt: string;
  updatedAt: string;
};

export type AdminPageKey = "home" | "about";
export type EditableContent = Record<string, unknown>;
export type AdminRevision = {
  id: string;
  ownerId: string;
  revisionNumber: number;
  status: "draft" | "published" | "archived";
  schemaVersion: number;
  content: EditableContent;
  createdBy?: string;
  createdAt: string;
  publishedAt?: string;
};
export type AdminPage = {
  id: string;
  key: AdminPageKey;
  latestRevision: AdminRevision;
  publishedRevisionId?: string | null;
  revisions: Array<Omit<AdminRevision, "ownerId" | "schemaVersion" | "content">>;
  updatedAt: string;
};
