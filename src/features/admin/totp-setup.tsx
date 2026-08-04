"use client";

import { Check, Copy } from "@phosphor-icons/react";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";

import { adminAPI, AdminAPIError } from "@/features/admin/admin-api";

type Setup = { manualKey: string; otpauthUri: string; expiresAt: string };

export function TOTPSetup() {
  const router = useRouter();
  const [setup, setSetup] = useState<Setup>();
  const [codes, setCodes] = useState<string[]>([]);
  const [pending, setPending] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");

  async function createSetup() {
    setPending(true);
    setError("");
    try {
      const response = await adminAPI.setupTOTP();
      setSetup(response.data);
    } catch (cause) {
      if (cause instanceof AdminAPIError && cause.status === 401) {
        router.replace("/admin/login");
        return;
      }
      setError(cause instanceof AdminAPIError ? cause.message : "Kunci autentikator belum dapat dibuat.");
    } finally {
      setPending(false);
    }
  }

  async function enable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError("");
    const value = String(new FormData(event.currentTarget).get("totpCode") ?? "");
    try {
      const response = await adminAPI.enableTOTP(value);
      setCodes(response.data.codes);
    } catch (cause) {
      setError(cause instanceof AdminAPIError ? cause.message : "Kode belum dapat diverifikasi.");
    } finally {
      setPending(false);
    }
  }

  async function copyCodes() {
    await navigator.clipboard.writeText(codes.join("\n"));
    setCopied(true);
  }

  if (codes.length > 0) {
    return (
      <div className="admin-auth-content">
        <div>
          <h1>Simpan recovery code</h1>
          <p>Setiap kode hanya dapat digunakan satu kali. Simpan di tempat yang aman.</p>
        </div>
        <div className="admin-recovery-list" aria-label="Recovery codes">
          {codes.map((code) => <code key={code}>{code}</code>)}
        </div>
        <button type="button" className="admin-secondary-button" onClick={copyCodes}>
          {copied ? <Check size={19} aria-hidden="true" /> : <Copy size={19} aria-hidden="true" />}
          {copied ? "Sudah disalin" : "Salin semua"}
        </button>
        <button type="button" className="admin-primary-button" onClick={() => router.replace("/admin/content")}>Buka CMS</button>
      </div>
    );
  }

  return (
    <div className="admin-auth-content">
      <div>
        <h1>Aktifkan autentikator</h1>
        <p>Hubungkan aplikasi TOTP sebelum membuka area pengelolaan konten.</p>
      </div>

      {!setup ? (
        <button type="button" className="admin-primary-button" onClick={createSetup} disabled={pending}>
          {pending ? "Membuat kunci..." : "Buat kunci autentikator"}
        </button>
      ) : (
        <>
          <div className="admin-setup-key">
            <span>Kunci manual</span>
            <code>{setup.manualKey}</code>
            <a href={setup.otpauthUri}>Buka aplikasi autentikator</a>
          </div>
          <form className="admin-form" onSubmit={enable} aria-busy={pending}>
            <div className="admin-field">
              <label htmlFor="setup-totp-code">Kode 6 digit</label>
              <input id="setup-totp-code" name="totpCode" type="text" inputMode="numeric" autoComplete="one-time-code" pattern="[0-9]{6}" minLength={6} maxLength={6} required />
            </div>
            <button className="admin-primary-button" type="submit" disabled={pending}>{pending ? "Memverifikasi..." : "Aktifkan TOTP"}</button>
          </form>
        </>
      )}

      {error ? <p className="admin-form-error" role="alert">{error}</p> : null}
    </div>
  );
}
