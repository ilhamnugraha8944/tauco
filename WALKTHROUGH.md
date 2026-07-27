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
| 6 | Automated tests dan pre-flight desain | Selesai | 73 unit, 79 E2E, Lighthouse gate hijau |
| 7 | Build dan walkthrough local | Selesai | Static/SSG build dan tujuh screenshot |

## Remediasi Phase 1A PRD-complete

Audit ulang menemukan bahwa baseline fungsional local sudah berjalan, tetapi
beberapa kontrak dan bukti acceptance masih perlu ditutup. Seluruh remediasi
berikut kini selesai tanpa deployment, commit, atau push.

Baseline sebelum remediasi:

- Git working tree bersih.
- ESLint dan TypeScript lulus.
- Unit test 33/33 lulus.
- Playwright 69 lulus dengan 9 expected skip.
- Dua belas audit Lighthouse lulus, dengan Performance minimum 91 serta
  Accessibility, SEO, dan Best Practices 100.
- Dependency audit melaporkan 0 vulnerability.

| Checkpoint | Scope | Status | Bukti |
| --- | --- | --- | --- |
| 0 | Baseline dan progress tracker | Selesai | Audit dan gate awal dicatat |
| 1 | Publication boundary dan DTO produk | Selesai | Content tests 21/21 dan TypeScript lulus |
| 2 | Empty state dan media contract | Selesai | Empty fixture dan image semantics tervalidasi |
| 3 | Kebijakan Privasi | Selesai | Privacy contract dan browser assertion lulus |
| 4 | Form dan keyboard interaction | Selesai | Targeted Playwright desktop/mobile 20/20 lulus |
| 5 | Acceptance test lengkap | Selesai | Quality browser suite 20/20 lulus |
| 6 | SEO environment dan audit integrity | Selesai | Environment tests 32/32 dan 12 Lighthouse report lulus |
| 7 | Final gate dan dokumentasi | Selesai | Seluruh local gate lulus |

### Hasil checkpoint 1

- Authored product wajib memiliki status eksplisit `draft` atau `published`.
- Public adapter memfilter produk sebelum data mencapai homepage, katalog,
  sitemap, static params, atau detail lookup.
- Draft, unknown slug, dan malformed slug tidak dapat dibaca melalui public
  content source.
- DTO `ProductSummary` hanya membawa field katalog, sedangkan
  `ProductDetail` membawa metadata dan field detail.
- Unique slug divalidasi terhadap seluruh record, termasuk lintas status.
- `LocalContentSource` menerima validated bundle melalui constructor agar
  publication boundary dapat diuji tanpa mengubah konten production.

### Hasil checkpoint 2

- Authored catalog dan public catalog mendukung nol produk.
- `featuredProductSlugs` dapat kosong dan hanya boleh menunjuk produk
  published.
- Empty fixture menghasilkan katalog kosong serta detail lookup `null`.
- Informative image wajib memakai `decorative: false` dan alt deskriptif.
- Decorative image wajib memakai `decorative: true` dan `alt=""`.
- Open Graph hanya menerima informative image.
- Decorative image tidak menghasilkan figcaption.
- Seluruh content JSON telah dimigrasikan ke media contract baru.
- Content tests 21/21, seluruh unit test 73/73, TypeScript, targeted
  ESLint, dan diff check lulus.

### Hasil checkpoint 3

- Privacy notice menjelaskan pengelola inbox yang ditunjuk dan Netlify sebagai
  pemroses teknis.
- Permintaan data mencakup akses, koreksi, dan penghapusan.
- Retensi ditetapkan paling lama 12 bulan tanpa pengecualian yang bertentangan
  dengan PRD.
- Browser test memverifikasi seluruh kontrak copy tersebut.

### Hasil checkpoint 4

- Contact form memakai lock sinkron untuk mencegah rapid double-submit.
- Pending state mempertahankan disabled button, `aria-busy`, dan live status.
- Network error mempertahankan seluruh nilai input serta mengaktifkan tombol
  kembali.
- Success state mereset seluruh field.
- Mobile menu dapat dibuka dan ditutup dengan keyboard serta mempertahankan
  focus pada summary.
- Seluruh tujuh route, true 404, menu native, dan constraint form diuji dengan
  JavaScript dinonaktifkan.
- Targeted ESLint, TypeScript, unit contact 8/8, production build, dan targeted
  Playwright desktop/mobile 20/20 lulus.

### Hasil checkpoint 5

- Overflow diuji pada 320, 390, 768, 1024, dan 1440 px untuk seluruh route.
- Boundary 1023/1024 px membuktikan pergantian mobile menu ke desktop
  navigation tanpa dua baris.
- Homepage hero diuji untuk batas dua baris, copy maksimal 20 kata, CTA dalam
  initial viewport, dan label CTA tanpa wrapping.
- Product hierarchy tablet, focus outline, serta reduced-motion diuji.
- Axe WCAG A/AA dijalankan pada seluruh route dalam light dan dark mode.
- Validation error, pending, success, dan network error form juga diaudit
  dengan Axe.
- Visible text, alt, title, aria-label, figcaption, dan button text dipindai
  terhadap em dash dan en dash.
- Keyboard-only walkthrough menjangkau seluruh control terlihat pada setiap
  route dengan Tab dan memverifikasi outline fokus.
- Targeted quality browser suite lulus 20/20.

### Hasil checkpoint 6

- Satu pure environment resolver mengendalikan origin dan indexability.
- Local dan provider tidak dikenal tetap noindex.
- Hanya Netlify production yang indexable; deploy preview dan branch deploy
  tetap noindex.
- SEO audit flag hanya bekerja pada loopback non-Netlify.
- Seluruh Netlify context membutuhkan origin HTTPS publik yang eksplisit.
- URL guard menolak localhost, IP literal, single-label hostname, reserved TLD,
  HTTP, path, query, hash, dan credentials.
- Lighthouse memakai port dinamis, memverifikasi canonical/robots audit,
  mempromosikan manifest hanya setelah gate lulus, membersihkan staging, dan
  memulihkan `next-env.d.ts`.
- Environment unit tests lulus 32/32.
- Full Lighthouse menghasilkan 12 report dengan Performance minimum 90 serta
  Accessibility, SEO, dan Best Practices minimum 100. Median LCP terburuk
  adalah 2498,9 ms dan CLS maksimum 0.

### Hasil checkpoint 7

- ESLint lulus tanpa warning.
- TypeScript typecheck lulus.
- Unit test lulus 73/73.
- Production build lulus dan menghasilkan route Static/SSG yang diharapkan.
- Full Playwright lulus 79 test dengan 13 desktop-only expected skip.
- Lighthouse lulus 12/12 report.
- Production dan seluruh dependency audit melaporkan 0 vulnerability dari 683
  dependency.
- `next-env.d.ts` dipulihkan setelah audit dan tidak menjadi source change.
- Dokumentasi content workflow, privacy fact-check, test evidence, serta
  deferred launch gate telah diselaraskan.
- Tidak ada commit, push, atau deployment.

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
- Client Component dibatasi pada form kontak dan error boundary yang diwajibkan
  Next.js untuk mekanisme retry.
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
- Responsive collapse telah diperiksa pada 320, 390, 768, 1024, dan 1440 px.

Hasil pre-flight visual:

- Satu `h1` per halaman.
- H1 homepage tetap maksimal dua baris pada viewport target.
- Nama produk dan CTA tampil sebelum ilustrasi pada tablet/mobile.
- Focus state, reduced-motion, dan CTA memiliki kontras yang jelas.
- Tidak ada overflow horizontal.
- Tidak ada em dash atau en dash pada visible maupun accessible copy.
- Seluruh route dan dynamic form state lolos Axe dalam mode yang relevan.

## Poin 4. SEO teknis dan structured data

**Status: selesai**

Yang diimplementasikan:

- `html lang="id-ID"`.
- Title, description, canonical, dan Open Graph unik.
- `NEXT_PUBLIC_SITE_URL` sebagai satu-satunya sumber canonical origin.
- Parser origin menolak credentials, path, query, hash, localhost, IP literal,
  single-label hostname, serta reserved TLD pada build Netlify.
- Seluruh context Netlify membutuhkan origin HTTPS publik. Hanya production
  yang indexable; Deploy Preview dan branch deploy tetap noindex.
- `robots.ts`, `sitemap.ts`, manifest, dan favicon.
- Local normal memakai `noindex, nofollow`; tidak ada risiko local URL dianggap
  indexable.
- Harness Lighthouse memakai audit build terpisah, port dinamis, promotion
  manifest setelah gate lulus, cleanup staging/process, dan pemulihan
  `next-env.d.ts`.
- Homepage memakai JSON-LD `WebSite` dan `Organization`.
- Interior route memakai `BreadcrumbList`.
- `/tauco` memakai `Article` dengan tanggal, author, dan sumber yang juga
  terlihat di halaman.
- Detail produk memakai `Product` tanpa `Offer`, image produk resmi, rating,
  atau review.
- Sitemap berisi exact unique route set serta tanggal publikasi Phase 1A yang
  eksplisit.
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
| `npm.cmd run test` | 73/73 lulus |
| `npm.cmd run test:e2e` | 79 lulus, 13 skip yang disengaja |
| `npm.cmd run build` | Lulus, seluruh route Static/SSG |
| `npm.cmd run lighthouse` | Lulus, 12/12 audit |
| `npm.cmd audit --omit=dev` | 0 vulnerability |
| `npm.cmd audit` | 0 vulnerability |

Tiga belas E2E skip berasal dari pemeriksaan global yang sengaja dijalankan sekali
pada project desktop agar tidak diduplikasi pada project mobile. Test route,
rendering, form, dan Axe tetap berjalan pada desktop dan mobile.

Coverage E2E meliputi:

- seluruh route dan navigation target;
- true 404;
- satu `h1`;
- seluruh public route dan true 404 dengan JavaScript disabled;
- mobile menu pointer dan keyboard, termasuk focus return;
- contact form validation, pending, duplicate lock, success, network error,
  input retention, dan native constraint;
- exact sitemap dan robots;
- canonical dan local noindex;
- JSON-LD entity, article, breadcrumb, dan product;
- larangan commerce field rekaan;
- security headers;
- tidak ada broken internal link;
- Axe WCAG A/AA automated rules pada light, dark, dan dynamic form state;
- tidak ada overflow pada 320, 390, 768, 1024, dan 1440 px;
- hero, CTA, navigation breakpoint, focus state, dan reduced-motion;
- visible serta accessible copy tidak memakai em dash atau en dash.

### Hasil Lighthouse mobile

Setiap route diukur tiga kali. Angka Performance adalah skor terendah dari tiga
run, sedangkan LCP adalah median.

| Route | Performance min. | Accessibility min. | SEO min. | Best Practices min. | Median LCP | Median CLS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `/` | 90 | 100 | 100 | 100 | 2.438 ms | 0 |
| `/tauco` | 91 | 100 | 100 | 100 | 2.402 ms | 0 |
| `/produk/tauco-cap-badak` | 93 | 100 | 100 | 100 | 2.499 ms | 0 |
| `/kontak` | 98 | 100 | 100 | 100 | 2.185 ms | 0 |

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
9. Cross-browser manual pada Edge, Firefox, dan Safari.

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

Phase 1A berstatus `PRD-complete` dan tervalidasi di local. Validasi launch yang
membutuhkan URL atau akun eksternal tetap ditunda sampai deployment diizinkan.
