"use client";

import { ArrowClockwise, ImageSquare, UploadSimple } from "@phosphor-icons/react";
import Image from "next/image";
import { useCallback, useEffect, useState } from "react";

import { adminAPI, type AdminMedia } from "@/features/admin/admin-api";

const maxBytes = 10 * 1024 * 1024;

export function MediaManager() {
  const [items, setItems] = useState<AdminMedia[]>([]);
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [decorative, setDecorative] = useState(false);

  const load = useCallback(async () => {
    try {
      const response = await adminAPI.listMedia();
      setItems(response.data);
      setMessage("");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Media tidak dapat dimuat.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let active = true;
    adminAPI.listMedia().then(
      (response) => { if (active) { setItems(response.data); setLoading(false); } },
      (error) => { if (active) { setMessage(error instanceof Error ? error.message : "Media tidak dapat dimuat."); setLoading(false); } },
    );
    return () => { active = false; };
  }, []);
  useEffect(() => {
    if (!items.some((item) => item.status === "processing")) return;
    const timer = window.setInterval(() => { void load(); }, 2000);
    return () => window.clearInterval(timer);
  }, [items, load]);

  async function upload(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const file = data.get("file");
    const altText = String(data.get("altText") ?? "");
    if (!(file instanceof File) || file.size === 0) { setMessage("Pilih gambar terlebih dahulu."); return; }
    if (file.size > maxBytes) { setMessage("Ukuran gambar maksimal 10 MiB."); return; }
    if (!decorative && !altText.trim()) { setMessage("Alt text wajib untuk gambar informatif."); return; }
    setPending(true);
    try {
      const response = await adminAPI.uploadMedia({ file, altText: decorative ? "" : altText.trim(), decorative });
      setItems((current) => [response.data, ...current.filter((item) => item.id !== response.data.id)]);
      setMessage("Upload diterima. Varian WebP diproses oleh worker.");
      form.reset();
      setDecorative(false);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Upload gagal.");
    } finally { setPending(false); }
  }

  async function retry(id: string) {
    setPending(true);
    try {
      const response = await adminAPI.retryMedia(id);
      setItems((current) => current.map((item) => item.id === id ? response.data : item));
      setMessage("Pemrosesan ulang dijadwalkan.");
    } catch (error) { setMessage(error instanceof Error ? error.message : "Retry gagal."); }
    finally { setPending(false); }
  }

  return (
    <div className="admin-page-stack admin-media-page">
      <header className="admin-page-header">
        <p className="admin-kicker">Gate C5</p>
        <h1>Pustaka media</h1>
        <p>Upload satu sumber, lalu worker membuat varian WebP tanpa menahan respons browser.</p>
      </header>

      <form className="admin-media-upload" onSubmit={upload}>
        <div>
          <h2>Tambah gambar</h2>
          <p>JPEG, PNG, atau WebP statis. Maksimal 10 MiB.</p>
        </div>
        <label className="admin-field">
          <span>File gambar</span>
          <input name="file" type="file" accept="image/jpeg,image/png,image/webp" required />
        </label>
        <label className="admin-check-field">
          <input name="decorative" type="checkbox" checked={decorative} onChange={(event) => setDecorative(event.target.checked)} />
          <span>Gambar hanya dekoratif</span>
        </label>
        <label className="admin-field">
          <span>Alt text</span>
          <input name="altText" type="text" maxLength={300} disabled={decorative} required={!decorative} placeholder="Jelaskan isi dan fungsi gambar" />
        </label>
        <button className="admin-primary-button" disabled={pending} type="submit">
          <UploadSimple size={19} aria-hidden="true" /> {pending ? "Mengirim..." : "Upload gambar"}
        </button>
      </form>

      <p className="admin-status-message" aria-live="polite">{message}</p>
      <section className="admin-media-library" aria-labelledby="media-library-title">
        <div className="admin-section-heading">
          <div><h2 id="media-library-title">Media terbaru</h2><p>Status diperbarui otomatis saat worker berjalan.</p></div>
          <button className="admin-secondary-button" type="button" onClick={() => void load()}><ArrowClockwise size={18} aria-hidden="true" /> Muat ulang</button>
        </div>
        {loading ? <p>Memuat media...</p> : items.length === 0 ? (
          <div className="admin-empty-state"><ImageSquare size={36} aria-hidden="true" /><h3>Belum ada media</h3><p>Upload pertama akan muncul di sini.</p></div>
        ) : (
          <ul className="admin-media-grid">
            {items.map((item) => (
              <li key={item.id} className="admin-media-card">
                <div className="admin-media-preview">
                  {item.status === "ready" ? <Image unoptimized width={item.width} height={item.height} src={`/admin-api/public-media/${item.id}/display.webp`} alt={item.decorative ? "" : item.altText} /> : <ImageSquare size={34} aria-hidden="true" />}
                </div>
                <div className="admin-media-meta">
                  <span className={`admin-status admin-status-${item.status}`}>{item.status}</span>
                  <strong>{item.decorative ? "Gambar dekoratif" : item.altText}</strong>
                  <small>{item.width} × {item.height} · {(item.bytes / 1024).toFixed(1)} KiB</small>
                  {item.lastErrorCode ? <code>{item.lastErrorCode}</code> : null}
                  {item.status === "failed" ? <button className="admin-text-button" type="button" disabled={pending} onClick={() => void retry(item.id)}>Proses ulang</button> : null}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
