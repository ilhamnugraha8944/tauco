import Image from "next/image";
import type { ReactNode } from "react";

import { getImageAsset } from "@/lib/images";

export function AdminAuthFrame({ children }: { children: ReactNode }) {
  return (
    <section className="admin-root admin-auth-page">
      <div className="admin-auth-visual" aria-hidden="true">
        <Image
          src={getImageAsset("/images/tauco-hero-provisional.webp")}
          alt=""
          fill
          priority
          sizes="(max-width: 767px) 100vw, 46vw"
          className="object-cover"
        />
        <div className="admin-auth-visual-copy">
          <p>Tauco Cap Badak</p>
          <strong>Kelola cerita produk dari satu tempat.</strong>
        </div>
      </div>

      <div className="admin-auth-panel">
        <a href="/" className="admin-wordmark" aria-label="Tauco Cap Badak, buka website publik">
          Tauco Cap Badak
        </a>
        <div className="admin-auth-form-wrap">{children}</div>
        <p className="admin-local-note">CMS lokal. Konten production belum berubah.</p>
      </div>
    </section>
  );
}
