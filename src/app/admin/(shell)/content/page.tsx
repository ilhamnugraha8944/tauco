import type { Metadata } from "next";

export const metadata: Metadata = { title: "Konten CMS | Tauco Cap Badak" };

export default function AdminContentPage() {
  return (
    <div className="admin-page-stack">
      <header className="admin-page-header">
        <h1>Pengelolaan konten</h1>
        <p>Fondasi CMS sudah aman. Editor Homepage dan Tentang Kami tersedia pada Gate C6.</p>
      </header>
      <section className="admin-empty-state">
        <h2>Belum ada editor pada gate ini</h2>
        <p>C4 menyiapkan login, session, navigasi, dan account management tanpa mengubah konten production.</p>
      </section>
    </div>
  );
}
