"use client";

import { Eye, EyeSlash } from "@phosphor-icons/react";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";

import { adminAPI, AdminAPIError } from "@/features/admin/admin-api";

export function LoginForm() {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [useRecovery, setUseRecovery] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError("");

    const form = new FormData(event.currentTarget);
    const totpCode = String(form.get("totpCode") ?? "").trim();
    const recoveryCode = String(form.get("recoveryCode") ?? "").trim();

    try {
      const response = await adminAPI.login({
        email: String(form.get("email") ?? "").trim(),
        password: String(form.get("password") ?? ""),
        ...(totpCode ? { totpCode } : {}),
        ...(recoveryCode ? { recoveryCode } : {}),
      });

      router.replace(
        response.data.status === "mfa_setup_required"
          ? "/admin/setup-totp"
          : "/admin/content",
      );
      router.refresh();
    } catch (cause) {
      setError(
        cause instanceof AdminAPIError
          ? cause.message
          : "Backend admin belum dapat dihubungi.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="admin-auth-content">
      <div>
        <h1>Masuk ke CMS</h1>
        <p>Gunakan akun admin dan faktor kedua jika sudah diaktifkan.</p>
      </div>

      <form className="admin-form" onSubmit={submit} aria-busy={pending}>
        <div className="admin-field">
          <label htmlFor="admin-email">Email</label>
          <input id="admin-email" name="email" type="email" autoComplete="username" required maxLength={254} />
        </div>

        <div className="admin-field">
          <label htmlFor="admin-password">Password</label>
          <div className="admin-password-field">
            <input
              id="admin-password"
              name="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              required
              minLength={12}
              maxLength={128}
            />
            <button
              type="button"
              className="admin-icon-button"
              onClick={() => setShowPassword((value) => !value)}
              aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
            >
              {showPassword ? <EyeSlash size={20} aria-hidden="true" /> : <Eye size={20} aria-hidden="true" />}
            </button>
          </div>
        </div>

        {useRecovery ? (
          <div className="admin-field">
            <label htmlFor="admin-recovery">Recovery code</label>
            <input id="admin-recovery" name="recoveryCode" type="text" autoComplete="one-time-code" pattern="[A-Za-z0-9]{4}-[A-Za-z0-9]{4}-[A-Za-z0-9]{4}" placeholder="XXXX-XXXX-XXXX" />
          </div>
        ) : (
          <div className="admin-field">
            <label htmlFor="admin-totp">Kode autentikator <span>(jika sudah aktif)</span></label>
            <input id="admin-totp" name="totpCode" type="text" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" maxLength={6} placeholder="6 digit" />
          </div>
        )}

        <button type="button" className="admin-text-button" onClick={() => setUseRecovery((value) => !value)}>
          {useRecovery ? "Gunakan kode autentikator" : "Gunakan recovery code"}
        </button>

        {error ? <p className="admin-form-error" role="alert">{error}</p> : null}

        <button className="admin-primary-button" type="submit" disabled={pending}>
          {pending ? "Memeriksa akun..." : "Masuk"}
        </button>
      </form>
    </div>
  );
}
