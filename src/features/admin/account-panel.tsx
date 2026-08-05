"use client";

import { Check, Copy } from "@phosphor-icons/react";
import { type FormEvent, useEffect, useState } from "react";

import { adminAPI, AdminAPIError, type AdminUser } from "@/features/admin/admin-api";

export function AccountPanel() {
  const [user, setUser] = useState<AdminUser>();
  const [codes, setCodes] = useState<string[]>([]);
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    adminAPI.me().then((response) => { if (active) setUser(response.data); });
    return () => { active = false; };
  }, []);

  async function regenerate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError("");
    const totpCode = String(new FormData(event.currentTarget).get("totpCode") ?? "");
    try {
      const response = await adminAPI.regenerateRecoveryCodes(totpCode);
      setCodes(response.data.codes);
      event.currentTarget.reset();
    } catch (cause) {
      setError(cause instanceof AdminAPIError ? cause.message : "Recovery code belum dapat dibuat ulang.");
    } finally {
      setPending(false);
    }
  }

  async function copyCodes() {
    await navigator.clipboard.writeText(codes.join("\n"));
    setCopied(true);
  }

  return (
    <div className="admin-page-stack">
      <header className="admin-page-header">
        <h1>Akun admin</h1>
        <p>Periksa identitas, role, dan recovery code akun ini.</p>
      </header>

      <section className="admin-account-summary" aria-busy={!user}>
        <div><span>Email</span><strong>{user?.email ?? "Memuat..."}</strong></div>
        <div><span>Role</span><strong>{user?.roles.join(", ") ?? "Memuat..."}</strong></div>
        <div><span>TOTP</span><strong>{user?.mfaEnabled ? "Aktif" : "Belum aktif"}</strong></div>
      </section>

      <section className="admin-account-action">
        <div>
          <h2>Buat ulang recovery code</h2>
          <p>Kode lama yang belum digunakan akan langsung dicabut.</p>
        </div>
        <form className="admin-form" onSubmit={regenerate} aria-busy={pending}>
          <div className="admin-field">
            <label htmlFor="account-totp-code">Kode autentikator</label>
            <input id="account-totp-code" name="totpCode" type="text" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" minLength={6} maxLength={6} required />
          </div>
          {error ? <p className="admin-form-error" role="alert">{error}</p> : null}
          <button className="admin-primary-button" type="submit" disabled={pending}>{pending ? "Membuat ulang..." : "Buat ulang kode"}</button>
        </form>

        {codes.length > 0 ? (
          <div className="admin-recovery-result" aria-live="polite">
            <div className="admin-recovery-list">{codes.map((code) => <code key={code}>{code}</code>)}</div>
            <button type="button" className="admin-secondary-button" onClick={copyCodes}>
              {copied ? <Check size={19} aria-hidden="true" /> : <Copy size={19} aria-hidden="true" />}
              {copied ? "Sudah disalin" : "Salin semua"}
            </button>
          </div>
        ) : null}
      </section>
    </div>
  );
}
