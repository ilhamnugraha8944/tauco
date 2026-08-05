"use client";

import Image from "next/image";
import { useEffect, useState } from "react";

import { adminAPI, type AdminMedia } from "@/features/admin/admin-api";

export function MediaPicker({ value, onChange, label = "Pilih media", informativeOnly = false }: { value?: string; onChange: (media: AdminMedia) => void; label?: string; informativeOnly?: boolean }) {
  const [items, setItems] = useState<AdminMedia[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    adminAPI.listMedia().then(
      (response) => { if (active) setItems(response.data.filter((item) => item.status === "ready" && (!informativeOnly || !item.decorative))); },
      (reason) => { if (active) setError(reason instanceof Error ? reason.message : "Media tidak dapat dimuat."); },
    );
    return () => { active = false; };
  }, [informativeOnly]);

  return (
    <fieldset className="admin-media-picker">
      <legend>{label}</legend>
      {error ? <p className="admin-form-error">{error}</p> : null}
      {items.length === 0 && !error ? <p>Belum ada media berstatus ready.</p> : (
        <div className="admin-media-picker-grid">
          {items.map((item) => (
            <label key={item.id} className="admin-media-picker-option">
              <input type="radio" name={`media-${label}`} checked={value === item.id} onChange={() => onChange(item)} />
              <Image unoptimized width={item.width} height={item.height} src={`/admin-api/public-media/${item.id}/display.webp`} alt="" />
              <span>{item.decorative ? "Gambar dekoratif" : item.altText}</span>
            </label>
          ))}
        </div>
      )}
    </fieldset>
  );
}
