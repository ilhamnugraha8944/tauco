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

async function send<T>(path: string, init: RequestInit, retry = true): Promise<T> {
  const method = init.method ?? "GET";
  const headers = new Headers(init.headers);
  const csrf = cookie("tauco_admin_csrf");

  if (init.body) {
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
      return send<T>(path, init, false);
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

  return (response.status === 204 ? undefined : await response.json()) as T;
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
};
