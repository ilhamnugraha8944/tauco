# Walkthrough Implementasi Phase 1A

Dokumen ini mencatat implementasi dan bukti verifikasi website publik
SEO-first Tauco Cap Badak. Scope berhenti pada Phase 1A dan seluruh pekerjaan
berjalan di local.

**Pembaruan terakhir:** 24 Juli 2026

**Status deployment:** tidak dilakukan. Tidak ada site Netlify atau Vercel,
Deploy Preview, production deploy, koneksi Supabase, maupun perubahan layanan
eksternal. Konfigurasi Netlify hanya disiapkan sebagai future runbook.

## Ringkasan progres

| Poin | Milestone | Status | Bukti utama |
| --- | --- | --- | --- |
| 1 | Fondasi proyek | Selesai | Next.js, TypeScript, Tailwind, quality tooling |
| 2 | Konten tervalidasi dan aset provisional | Selesai | Async content source, Zod, tiga visual |
| 3 | Halaman publik dan design system | Selesai | Tujuh route, true 404, responsive light/dark |
| 4 | SEO teknis dan structured data | Selesai | Metadata, canonical, sitemap, JSON-LD, initial HTML |
| 5 | Form kontak dan privacy flow | Selesai secara lokal | Contract, native validation, UI states, blueprint |
| 6 | Automated tests dan pre-flight desain | Selesai | 33 unit, 69 E2E, Lighthouse gate hijau |
| 7 | Build dan walkthrough local | Selesai | Static/SSG build dan tujuh screenshot |

## Poin 1. Fondasi proyek

**Status: selesai**

Yang diimplementasikan:

- Next.js 16 App Router, React 19.2, TypeScript, dan Node 22.
- Tailwind CSS 4 melalui `@tailwindcss/postcss`.
- ESLint, Vitest, Playwright, axe, dan Lighthouse API.
- Script `dev`, `lint`, `typecheck`, `test`, `test:e2e`, `build`,
  `lighthouse`, `images:optimize`, `walkthrough:capture`, dan `check`.
- Build standar Next.js dengan Webpack, bukan static export, agar tetap siap
  untuk image optimization dan migrasi ISR/API pada fase berikutnya.
- Security headers, image output WebP/AVIF, dan inline critical CSS.
- `.env.example`, `.nvmrc`, `.gitignore`, `netlify.toml`, dan lockfile npm.
- Runner Lighthouse mandiri yang membuat audit build terpisah, menjalankan
  server local, mengaudit empat route sebanyak tiga kali, lalu menghentikan
  server.

File utama:

- `package.json`
- `next.config.ts`
- `postcss.config.mjs`
- `eslint.config.mjs`
- `vitest.config.ts`
- `playwright.config.ts`
- `lighthouserc.json`
- `scripts/run-lighthouse.mjs`

Keputusan arsitektur:

- Konten publik dirender sebagai Server Components.
- Navigasi mobile memakai elemen native `<details>`, sehingga tidak membutuhkan
  JavaScript client.
- Hanya form kontak yang menjadi Client Component.
- Link publik utama tetap dapat digunakan ketika JavaScript dimatikan.

## Poin 2. Konten tervalidasi dan aset provisional

**Status: selesai**

Yang diimplementasikan:

- Empat sumber konten lokal pada folder `content`.
- Interface async `ContentSource`.
- Adapter `LocalContentSource` yang dapat diganti dengan API pada fase
  berikutnya tanpa mengubah page component.
- Schema Zod untuk metadata, canonical, slug, alt text, internal link, relasi
  produk, dan tanggal artikel.
- Validasi slug unik serta unknown product yang menghasilkan `null`.
- Pemisahan bukti penelitian 2022 sebagai `researchEvidence`, lengkap dengan
  batas konteks agar tidak terbaca sebagai formula produk saat ini.
- Tiga visual provisional hasil image generation yang telah diperiksa:
  ilustrasi hero, fermentasi umum, dan penggunaan tauco dalam masakan.
- Master image disimpan di `assets/image-sources`; file publik hanya format
  WebP/AVIF teroptimasi.

Publishing guard:

- Tidak ada logo badak buatan baru.
- Tidak ada foto marketplace yang disalin.
- Ilustrasi tidak memuat logo, teks, sertifikasi, atau kemasan bermerek.
- Caption “Ilustrasi penyajian” dipakai saat visual dapat dianggap dokumenter.
- Harga, SKU, berat, shelf life, legalitas, sertifikasi, dan availability tidak
  ditampilkan sebelum dikonfirmasi pemilik.

File utama:

- `content/home.json`
- `content/about.json`
- `content/tauco-guide.json`
- `content/products.json`
- `src/features/content`
- `public/images`
- `assets/image-sources`
- `FACT_CHECK.md`

## Poin 3. Halaman publik dan design system

**Status: selesai**

Route final:

| Route | Hasil |
| --- | --- |
| `/` | Homepage entity-first dengan hero, pengantar tauco, produk, dan CTA |
| `/tauco` | Pillar article informasional dengan daftar isi dan referensi |
| `/tentang-kami` | Profil singkat tanpa sejarah brand yang belum terbukti |
| `/produk` | Katalog satu produk yang sudah dipublikasikan |
| `/produk/tauco-cap-badak` | Detail produk tanpa field komersial rekaan |
| `/kontak` | Form pertanyaan produk atau peluang kerja sama |
| `/kebijakan-privasi` | Tujuan, persetujuan, dan retensi data 12 bulan |
| Slug produk asing | True HTTP 404 |

Design direction yang diterapkan:

- Modern heritage.
- `DESIGN_VARIANCE: 7`.
- `MOTION_INTENSITY: 3`.
- `VISUAL_DENSITY: 3`.
- Geist Sans, tanpa serif.
- Cool off-white, charcoal, dan forest green sebagai palet provisional.
- Dark mode mengikuti preferensi sistem.
- Motion hanya hover, focus, active, dan feedback form berbasis CSS.
- Komposisi asimetris dan editorial, tanpa generic three-card layout.
- Satu CTA utama pada hero.
- Responsive collapse telah diperiksa pada 320 px, 390 px, 768 px, dan desktop.

Hasil pre-flight visual:

- Satu `h1` per halaman.
- H1 homepage tetap maksimal dua baris pada desktop.
- Nama produk dan CTA tampil sebelum ilustrasi pada tablet/mobile.
- Focus state dan CTA memiliki kontras yang jelas.
- Tidak ada overflow horizontal.
- Tidak ada em dash atau en dash pada visible copy.
- Dark mode form dan article tetap terbaca.

## Poin 4. SEO teknis dan structured data

**Status: selesai**

Yang diimplementasikan:

- `html lang="id-ID"`.
- Title, description, canonical, dan Open Graph unik.
- `NEXT_PUBLIC_SITE_URL` sebagai satu-satunya sumber canonical origin.
- Parser origin menolak credentials, path, query, dan hash.
- Production build menolak canonical yang tidak valid.
- `robots.ts`, `sitemap.ts`, manifest, dan favicon.
- Local normal memakai `noindex, nofollow`; tidak ada risiko local URL dianggap
  indexable.
- Harness Lighthouse memakai audit build terpisah yang indexable hanya selama
  pengukuran SEO.
- Homepage memakai JSON-LD `WebSite` dan `Organization`.
- Interior route memakai `BreadcrumbList`.
- `/tauco` memakai `Article` dengan tanggal, author, dan sumber yang juga
  terlihat di halaman.
- Detail produk memakai `Product` tanpa `Offer`, image produk resmi, rating,
  atau review.
- Sitemap berisi exact unique route set dan tanggal konten yang relevan.
- Copy utama, heading, dan link tersedia di initial HTML.
- `next/image` memakai dimensi tetap, kandidat responsif, preload hanya pada
  hero, dan format WebP/AVIF.

Catatan ranking:

Technical SEO, content relevance, dan Core Web Vitals sudah disiapkan, tetapi
tidak dapat menjamin halaman pertama Google. Ranking tetap dipengaruhi
kompetisi, authority, backlink, kualitas fakta resmi, dan data lapangan setelah
site benar-benar diluncurkan.

## Poin 5. Form kontak dan privacy flow

**Status: selesai secara lokal**

Yang diimplementasikan:

- Field nama, email, telepon opsional, topik, pesan, persetujuan privasi, dan
  honeypot.
- Constraint native `required`, panjang minimum/maksimum, type email, dan
  pattern telepon tersedia walaupun JavaScript dimatikan.
- Validator client ringan berbagi constant dan limit dengan contract model.
- Zod dipertahankan untuk contract/server-side boundary dan unit test, tetapi
  tidak ikut masuk browser bundle.
- State idle, pending, success, validation error, dan network error.
- Error summary dengan `role="alert"` dan fokus otomatis ke field invalid
  pertama.
- Prefill `/kontak?topik=produk`.
- Blueprint statis `public/__forms.html` dengan field dan constraint yang sama.
- Submission URL-encoded dengan hidden `form-name`.
- Kebijakan Privasi menetapkan retensi pesan maksimum 12 bulan.

Yang sudah diuji secara lokal:

- Payload dan field contract.
- State sukses dan network error melalui request interception.
- Native fallback tanpa JavaScript.
- Keyboard/focus behavior dan baseline Axe.
- Data input tidak hilang ketika request gagal.

Yang sengaja belum dapat diverifikasi:

- Deteksi form di dashboard Netlify.
- Submission masuk ke verified inbox Netlify.
- Email notification operasional.

Ketiga poin tersebut memerlukan deployment dan akun eksternal, sehingga ditunda
sesuai instruksi local-only.

## Poin 6. Automated tests dan pre-flight desain

**Status: selesai**

Hasil gate final:

| Gate | Hasil |
| --- | --- |
| `npm.cmd run lint` | Lulus, 0 warning |
| `npm.cmd run typecheck` | Lulus |
| `npm.cmd run test` | 33/33 lulus |
| `npm.cmd run test:e2e` | 69 lulus, 9 skip yang disengaja |
| `npm.cmd run build` | Lulus, seluruh route Static/SSG |
| `npm.cmd run lighthouse` | Lulus, 12/12 audit |
| `npm.cmd audit --omit=dev` | 0 vulnerability |
| `npm.cmd audit` | 0 vulnerability |

Sembilan E2E skip berasal dari pemeriksaan global yang sengaja dijalankan sekali
pada project desktop agar tidak diduplikasi pada project mobile. Test route,
rendering, form, dan Axe tetap berjalan pada desktop dan mobile.

Coverage E2E meliputi:

- seluruh route dan navigation target;
- true 404;
- satu `h1`;
- initial HTML dengan JavaScript disabled;
- mobile menu;
- contact form state dan native constraint;
- exact sitemap dan robots;
- canonical dan local noindex;
- JSON-LD entity, article, breadcrumb, dan product;
- larangan commerce field rekaan;
- security headers;
- tidak ada broken internal link;
- Axe WCAG A/AA automated rules;
- tidak ada overflow 320/768 px;
- visible copy tidak memakai em dash atau en dash.

### Hasil Lighthouse mobile

Setiap route diukur tiga kali. Angka Performance adalah skor terendah dari tiga
run, sedangkan LCP adalah median.

| Route | Performance min. | Accessibility min. | SEO min. | Best Practices min. | Median LCP | Median CLS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `/` | 93 | 100 | 100 | 100 | 2.401 ms | 0 |
| `/tauco` | 94 | 100 | 100 | 100 | 2.363 ms | 0 |
| `/produk/tauco-cap-badak` | 97 | 100 | 100 | 100 | 2.445 ms | 0 |
| `/kontak` | 96 | 100 | 100 | 100 | 2.257 ms | 0 |

Penulisan `2.401 ms` mengikuti pemisah ribuan Indonesia, yaitu 2.401
milidetik. Semua median LCP berada di bawah gate 2.500 ms.

Target INP p75 belum dapat dibuktikan dari Lighthouse local karena membutuhkan
field data pengguna nyata. Hal tersebut menjadi post-launch measurement, bukan
klaim pada handoff ini.

## Poin 7. Build dan walkthrough local

**Status: selesai**

Build final menghasilkan:

- Static route untuk seluruh halaman publik.
- SSG untuk `/produk/tauco-cap-badak`.
- True 404 untuk slug produk yang tidak dikenal.
- Tidak ada REST API, database, cache, auth, CMS, inventory, order, checkout,
  atau akun pelanggan.

### Menjalankan aplikasi

Di PowerShell:

```powershell
npm.cmd ci
Copy-Item .env.example .env.local
npm.cmd run dev
```

Buka:

```text
http://localhost:3000
```

Untuk memeriksa production build secara lokal:

```powershell
npm.cmd run build
npm.cmd run start
```

Untuk menjalankan seluruh gate utama:

```powershell
npm.cmd run lint
npm.cmd run typecheck
npm.cmd run test
npm.cmd run test:e2e
npm.cmd run lighthouse
npm.cmd audit
```

### Screenshot walkthrough

- [Homepage desktop light](./artifacts/walkthrough/home-desktop-light.png)
- [Homepage mobile light](./artifacts/walkthrough/home-mobile-light.png)
- [Pillar article desktop dark](./artifacts/walkthrough/tauco-desktop-dark.png)
- [Tentang Kami desktop light](./artifacts/walkthrough/about-desktop-light.png)
- [Detail produk tablet light](./artifacts/walkthrough/product-tablet-light.png)
- [Kontak desktop light](./artifacts/walkthrough/contact-desktop-light.png)
- [Kontak mobile dark](./artifacts/walkthrough/contact-mobile-dark.png)

Screenshot dapat dibuat ulang dengan:

```powershell
npm.cmd run walkthrough:capture
```

## Validasi yang ditunda sampai user mengizinkan deployment

Poin berikut bukan kegagalan Phase 1A local. Semuanya membutuhkan URL atau akun
eksternal:

1. Menentukan subdomain/domain production final.
2. Membuat site dan production deployment.
3. Smoke test Netlify Forms pada dashboard.
4. Mengaktifkan notifikasi inbox.
5. Memvalidasi canonical production dan robots production.
6. Rich Results Test/Schema validator pada URL publik.
7. Verifikasi Search Console dan submission sitemap.
8. Mengumpulkan field data Core Web Vitals, termasuk INP p75.
9. Cross-browser manual pada Safari dan Firefox.

## Data resmi yang masih dibutuhkan

Sebelum website diluncurkan, pemilik perlu menyediakan atau mengonfirmasi:

- logo resmi dan panduan warna;
- foto produk yang hak pakainya jelas;
- alamat, email, telepon/WhatsApp, dan jam operasional;
- tahun berdiri, sejarah, dan identitas pemilik;
- daftar kemasan, ukuran, harga, serta availability;
- legalitas dan sertifikasi;
- proses produksi yang boleh dipublikasikan;
- social links;
- pemilik operasional inbox dan persetujuan retensi 12 bulan.

Daftar rinci dan publishing guard ada di [FACT_CHECK.md](./FACT_CHECK.md).

## Batas Phase 1A

Tidak diimplementasikan:

- backend Go, Gin, GORM, PostgreSQL, Supabase, atau Redis;
- JWT/PASETO, Admin CMS, worker upload, dan activity log;
- inventory, stock concurrency, order, checkout, atau payment;
- production analytics, Search Console, atau layanan email;
- deployment ke platform mana pun.

Phase 1A selesai sebagai website publik yang berjalan dan tervalidasi di local.
