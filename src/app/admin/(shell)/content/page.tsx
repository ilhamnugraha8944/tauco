import { ArrowRight } from "@phosphor-icons/react/dist/ssr";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "Konten CMS | Tauco Cap Badak" };

const pages = [
  { key: "home", title: "Homepage", path: "/", description: "Hero, pengantar, produk unggulan, dan teaser halaman." },
  { key: "about", title: "Tentang Kami", path: "/tentang-kami", description: "Profil, bagian narasi, tautan terkait, dan sumber." },
] as const;

export default function AdminContentPage() {
  return <div className="admin-page-stack admin-content-index"><header className="admin-page-header"><p className="admin-kicker">Gate C6</p><h1>Pengelolaan konten</h1><p>Pilih halaman, simpan revision immutable, lalu preview sebelum publish.</p></header><div className="admin-content-page-list">{pages.map((page)=><Link key={page.key} href={`/admin/content/${page.key}`} className="admin-content-page-link"><div><span>{page.path}</span><h2>{page.title}</h2><p>{page.description}</p></div><ArrowRight size={22} aria-hidden="true" /></Link>)}</div><section className="admin-readonly-note"><h2>Read-only pada Phase 1C</h2><p>Halaman <code>/tauco</code> tetap dikelola sebagai pillar content terverifikasi dan tidak memiliki editor admin.</p></section></div>;
}
