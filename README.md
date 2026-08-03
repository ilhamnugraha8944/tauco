# Tauco Cap Badak Website

Website company profile dan katalog awal Tauco Cap Badak yang berfokus pada
SEO, performance, dan accessibility. Production saat ini tetap menjalankan
**Phase 1A**: public website dengan konten lokal serta formulir kontak melalui
Netlify Forms.

Fondasi REST API Go untuk **Phase 1B** dikembangkan di `backend/` dalam
shadow-mode. Backend ini belum menjadi dependency website production dan tidak
mengubah canonical, sitemap, maupun alur form Phase 1A. Rencana serta evidence
implementasinya ada di [PHASE_1B_PLAN.md](./PHASE_1B_PLAN.md) dan
[PHASE_1B_WALKTHROUGH.md](./PHASE_1B_WALKTHROUGH.md).
Dokumentasi struktur dan cara kerja kode tersedia di
[BACKEND_CODE_DOCUMENTATION.md](./BACKEND_CODE_DOCUMENTATION.md) serta
[BACKEND_CODE_DOCUMENTATION.pdf](./BACKEND_CODE_DOCUMENTATION.pdf).

**Status handoff:** Phase 1A berstatus PRD-complete dan telah tersedia di
`https://tauco-cap-badak.netlify.app`. Gate G0–G8 sudah lulus dan Phase 1A
berstatus **Complete** pada 28 Juli 2026. Tidak ada koneksi Supabase atau
layanan backend. Bukti ada di
[WALKTHROUGH.md](./WALKTHROUGH.md).

## Status fitur

| Fitur | Status |
| --- | --- |
| Beranda | Phase 1A |
| Tentang Kami | Phase 1A |
| Katalog dan detail produk | Phase 1A |
| Contact form | Active; verified submission dan notification utama lulus |
| Privacy notice | Phase 1A |
| Metadata, canonical, sitemap, robots, JSON-LD | Phase 1A |
| Go REST API | Phase 1B local complete; public/contact/health/protected metrics aktif lokal |
| PostgreSQL | Phase 1B local complete; migration v4, durable job, media, retention, dan integration lulus |
| Redis | Phase 1B local complete; cache-aside, atomic limiter, fail-open, dan metrics lulus |
| Admin CMS | Future |
| Inventory dan order management | Out of scope Phase 1 |

## Tech stack

- Next.js App Router
- React 19
- TypeScript 5
- Tailwind CSS 4
- Zod untuk validasi konten lokal
- Playwright dan axe untuk end-to-end/accessibility test
- Lighthouse API untuk performance/SEO gate lokal
- Netlify sebagai production hosting dan transport form
- Go, Gin, dan Zap untuk fondasi API shadow-mode
- PostgreSQL dan Redis sebagai dependency lokal Phase 1B

Gunakan versi dependency yang dikunci oleh `package-lock.json`; jangan
mengandalkan versi global.

## Prasyarat

- Node.js 22
- npm yang disertakan bersama Node.js
- Git untuk workflow deploy berbasis repository
- Browser Chromium untuk end-to-end test
- Go sesuai `backend/go.mod`
- Docker Desktop mulai wajib pada Phase 1B gate B3

Versi lokal dapat diperiksa dengan:

```powershell
node --version
npm.cmd --version
go version
```

Dokumentasi ini memakai `npm.cmd`, bukan `npm`, agar command konsisten pada
PowerShell Windows dan tidak terhalang execution policy terhadap `npm.ps1`.

## Setup lokal di Windows

1. Buka PowerShell pada root repository.
2. Install dependency dari lockfile:

   ```powershell
   npm.cmd ci
   ```

   Jika sedang membuat ulang lockfile secara sengaja, gunakan
   `npm.cmd install`, review perubahan `package-lock.json`, lalu commit lockfile.

3. Salin contoh environment:

   ```powershell
   Copy-Item .env.example .env.local
   ```

4. Set origin lokal:

   ```dotenv
   NEXT_PUBLIC_SITE_URL=http://localhost:3000
   ```

5. Jalankan development server:

   ```powershell
   npm.cmd run dev
   ```

6. Buka `http://localhost:3000`.

Jangan commit `.env.local`.

## Environment variables

| Variable | Wajib | Contoh | Keterangan |
| --- | --- | --- | --- |
| `NEXT_PUBLIC_SITE_URL` | Ya pada setiap build Netlify | `https://nama-site.netlify.app` | Absolute canonical origin tanpa path dan query |
| `GOOGLE_SITE_VERIFICATION` | Tidak | Token Search Console | Isi setelah URL-prefix property dibuat |

Aturan `NEXT_PUBLIC_SITE_URL`:

- Lokal memakai `http://localhost:3000`.
- Setiap context Netlify wajib memakai origin production final yang benar.
- Netlify production indexable, sedangkan Deploy Preview dan branch deploy
  tetap `noindex`.
- Gunakan `https` dengan hostname publik. Localhost, IP literal, single-label
  hostname, dan reserved TLD ditolak pada build Netlify.
- Jangan memakai slash penutup jika parser konfigurasi mengharuskan origin.
- Jangan memakai URL Deploy Preview sebagai canonical.
- Setelah mengganti domain, build dan deploy ulang agar canonical, sitemap,
  robots, JSON-LD, dan social URL ikut berubah.

Phase 1A tidak membutuhkan secret. Nilai dengan prefix `NEXT_PUBLIC_` akan
masuk ke client bundle dan tidak boleh berisi credential.

## Scripts

Jalankan script menggunakan `npm.cmd run <nama>`.

| Script | Fungsi |
| --- | --- |
| `dev` | Menjalankan Next.js development server |
| `build` | Membuat production build |
| `start` | Menjalankan production build secara lokal |
| `lint` | Menjalankan ESLint |
| `typecheck` | Memeriksa TypeScript tanpa menulis output |
| `test:e2e` | Menjalankan Playwright end-to-end test |
| `test:e2e:ui` | Membuka Playwright UI; jalankan server local secara terpisah |
| `qa:g4:functional` | Menjalankan Functional QA terhadap Deploy Preview |
| `qa:g4:seo` | Memeriksa isolasi index preview, canonical, robots, dan sitemap |
| `qa:g4:form` | Memeriksa form tanpa membuat submission eksternal |
| `qa:g4:form:live` | Mengirim satu submission sintetis nyata; jalankan hanya sengaja |
| `qa:g4:a11y` | Menjalankan accessibility, keyboard, dark mode, dan reflow QA |
| `qa:g4:browsers` | Menjalankan Edge, Firefox, WebKit, dan emulasi Android |
| `qa:g4` | Menjalankan seluruh automated G4 kecuali live submission |
| `qa:g6` | Menjalankan production smoke, SEO, schema, security, dan accessibility QA |
| `qa:g7:form:live` | Mengirim tepat satu submission production; perlu opt-in eksplisit |
| `images:optimize` | Membuat WebP/AVIF dari master image provisional |
| `walkthrough:capture` | Build, start local, dan membuat tujuh screenshot |
| `plan:pdf` | Membuat ulang `plan.pdf` dari `plan.md` |
| `backend:docs:pdf` | Membuat ulang dokumentasi kode backend dalam PDF |
| `lighthouse` | Build audit terpisah dan mengaudit empat route local sebanyak 3x |
| `lighthouse:production` | Mengaudit empat route production sebanyak 3x |
| `backend:dev` | Menjalankan API lokal pada `127.0.0.1:8080` |
| `backend:worker` | Menjalankan durable background worker |
| `backend:worker:ready` | Memeriksa readiness dependency worker |
| `backend:media:import` | Ingest media internal tanpa HTTP upload |
| `backend:ops` | Dead-job replay atau targeted cache purge |
| `backend:load` | Mengukur cold-start dan warm-read baseline lokal |
| `backend:migrate:up` | Menerapkan migration PostgreSQL lokal |
| `backend:migrate:version` | Menampilkan versi dan dirty state migration |
| `backend:seed:phase1a` | Mengimpor konten Phase 1A secara idempotent |
| `backend:format` | Memformat seluruh package Go |
| `backend:format:check` | Menolak Go source yang belum diformat |
| `backend:generate` | Membuat ulang Go types/server contract dari OpenAPI |
| `backend:generate:check` | Menolak drift antara OpenAPI dan generated Go code |
| `backend:test` | Menjalankan regression Go yang dipertahankan, termasuk fallback Windows Application Control |
| `backend:test:integration` | Menjalankan PostgreSQL disposable dan Redis integration |
| `backend:race` | Menjalankan Go race detector di Linux container |
| `backend:vet` | Menjalankan static analysis standar Go |
| `backend:lint` | Menjalankan golangci-lint v2.11.4 di container |
| `backend:vuln` | Menjalankan govulncheck v1.6.0 di container |
| `backend:build` | Mengompilasi entrypoint API |
| `backend:container:build` | Membuat image API local scratch runtime |
| `backend:check` | Menjalankan codegen drift check, test, vet, dan build backend |
| `backend:quality` | Menjalankan quality gate backend B10 lengkap |
| `phase1b:quality` | Menjalankan backend, frontend, E2E, dan production smoke |
| `backend:compose:up` | Menyalakan dependency local setelah Docker tersedia |
| `backend:compose:down` | Menghentikan dependency local tanpa menghapus volume |
| `check:frontend` | Menjalankan gate cepat frontend Phase 1A |
| `check` | Menjalankan gate frontend dan backend berurutan |

Quality gate lengkap sebelum handoff atau deployment mendatang:

```powershell
npm.cmd run lint
npm.cmd run typecheck
npm.cmd run build
npm.cmd run test:e2e
npm.cmd run lighthouse
npm.cmd audit
```

Command `check` adalah gate cepat tanpa browser:

```powershell
npm.cmd run check
```

### Kontrak API Phase 1B

Sumber kebenaran kontrak berada di `backend/openapi/openapi.yaml`. File
`backend/internal/delivery/api/api.gen.go` dihasilkan oleh tool yang versinya
dikunci di `backend/go.mod`; jangan diedit manual.

Setelah mengubah OpenAPI:

```powershell
npm.cmd run backend:generate
npm.cmd run backend:generate:check
npm.cmd run backend:test
```

B2 mendefinisikan lima public read route, contact intake, dua health route, dan
protected metrics. Tidak ada route admin. B10 telah meregistrasikan public
read, contact, readiness, protected metrics, trace propagation, dan operations
tooling pada runtime lokal.

Generated defaults yang menulis `err.Error()` tidak boleh dipakai. B4 memakai
selective safe registration dan handwritten response factory pada package
generated API. Unknown query, duplicate query, cursor, limit, published-only,
ETag/304, true product 404, body limit, strict contact JSON, Content-Type,
cache, rate limit, CORS, dan trusted proxy sudah diuji. Authorization metrics
memakai dedicated bearer token dan constant-time compare.

Release evidence Phase 1A:

| Pemeriksaan | Hasil |
| --- | --- |
| Unit test | 73/73 lulus |
| Deploy Preview G4 | 53 lulus, 7 intentional skip |
| Production G6 | 29/29 lulus |
| Lighthouse production | 12/12 lulus; Performance 96–100, Accessibility dan SEO 100 |
| Manual browser/device | Edge, Firefox, Safari asli, Android Chrome asli, keyboard, zoom 200% lulus |
| Schema.org Validator | Homepage dan detail produk: 0 error, 0 warning |
| Production dependency audit | 0 vulnerability |
| Full dependency audit | Advisori development-only pada tree ESLint diterima owner |

Untuk pertama kali menjalankan Playwright pada mesin baru:

```powershell
npm.cmd exec -- playwright install chromium
npm.cmd run test:e2e
```

`test:e2e` dan `lighthouse` membuat production build serta mengelola server
local secara otomatis. Lighthouse memakai build `.next-lighthouse`, mencari
port kosong, dan mengaktifkan indexability hanya pada loopback audit. Manifest
baru dipublikasikan setelah seluruh assertion lulus. Runner juga membersihkan
staging, menghentikan server/Chrome, serta memulihkan `next-env.d.ts`. Skor
local bukan field data production.

Seluruh automated G4 dapat dijalankan langsung terhadap Deploy Preview tanpa
membuat build local:

```powershell
$env:PLAYWRIGHT_BASE_URL = "https://deploy-preview-1--tauco-cap-badak.netlify.app"
npm.cmd run qa:g4
Remove-Item Env:PLAYWRIGHT_BASE_URL
```

Runner memeriksa functional flow, preview SEO isolation, form tanpa submission
eksternal, WCAG/keyboard, responsive layout, serta browser matrix. Laporan
terpisah disimpan di `playwright-report/g4/`.

Live form test sengaja tidak termasuk agregat agar inbox tidak menerima pesan
berulang. Jalankan hanya bila memang ingin membuat satu submission sintetis:

```powershell
npm.cmd run qa:g4:form:live
```

Gunakan URL Deploy Preview aktif milik Pull Request yang sedang diuji. Jangan
menjalankan gate G4 terhadap origin production atau localhost.

## Routes

| Route | Tujuan |
| --- | --- |
| `/` | Homepage |
| `/tentang-kami` | Profil dan konteks brand |
| `/tauco` | Panduan informasional mengenai tauco |
| `/produk` | Katalog |
| `/produk/[slug]` | Detail produk |
| `/kontak` | Contact form |
| `/kebijakan-privasi` | Privacy notice |
| `/robots.txt` | Crawl directives |
| `/sitemap.xml` | Daftar URL canonical |

Slug produk merupakan public contract. Jangan mengganti slug setelah URL
dipublikasikan tanpa redirect permanent.

## Arsitektur Phase 1A

```text
Browser / crawler
       |
       v
Next.js App Router
  |-- Server Components untuk halaman dan konten utama
  |-- Navigasi mobile native <details> dari Server Component
  |-- Client Components untuk contact form dan error boundary Next.js
  |-- Metadata, robots, sitemap, dan JSON-LD
       |
       v
ContentSource interface
       |
       v
Local JSON adapter + Zod validation

Contact form ---- URL-encoded POST ----> Netlify Forms setelah deployment
```

Prinsip utama:

- Public content diprerender dan tidak menunggu browser melakukan API fetch.
- Page mengambil data melalui asynchronous content source.
- Konten mentah divalidasi sebelum dipresentasikan.
- Product detail dibuat dari daftar slug published; slug asing menjadi not
  found.
- Migrasi ke Go API nanti dilakukan dengan mengganti adapter content source,
  bukan mengganti URL atau komponen publik.
- Netlify Forms adalah solusi sementara sampai contact endpoint Go tersedia.

Struktur direktori konten:

```text
content/
  home.json
  about.json
  tauco-guide.json
  products.json

src/features/content/
  schemas.ts
  types.ts
  content-source.ts
  local-content-source.ts
  index.ts
```

Route dan komponen memakai export dari `src/features/content`, bukan mengimpor
JSON secara langsung. Aset provisional berada di `public/images`; hanya ganti
dengan file yang memang dimiliki atau berlisensi.

## Memperbarui konten

Phase 1A belum mempunyai CMS. Perubahan konten dilakukan melalui file konten
lokal dan melalui review repository.

Workflow:

1. Baca [FACT_CHECK.md](./FACT_CHECK.md).
2. Ubah record yang relevan:
   - `content/home.json`
   - `content/about.json`
   - `content/tauco-guide.json`
   - `content/products.json`
3. Setiap record produk wajib memiliki status eksplisit `draft` atau
   `published`. Status tidak mempunyai default.
4. Pertahankan ID dan slug produk published.
5. Pastikan setiap informative image memakai `decorative: false` dan alt text
   yang faktual. Gambar dekoratif wajib memakai `decorative: true` serta
   `alt: ""`.
6. Jangan menambahkan harga, sertifikasi, alamat, kontak, sejarah, testimoni,
   atau klaim kualitas tanpa source-of-truth.
7. Jalankan:

   ```powershell
   npm.cmd run check:frontend
   ```

8. Preview seluruh route yang terdampak, termasuk mobile viewport.
9. Setelah merge, periksa canonical dan sitemap pada deployment.

Produk draft tetap berada di repository dan bukan boundary untuk data rahasia.
Jangan menyimpan rahasia, credential, atau data pribadi dalam file konten.
Hanya produk published yang diproyeksikan ke katalog, detail route, homepage,
sitemap, dan static params.

Produk awal memakai slug `tauco-cap-badak` dan public URL
`/produk/tauco-cap-badak`. CTA pertanyaan produk menggunakan
`/kontak?topik=produk`; pertahankan query contract ini jika field kontak
diprefill.

Saat menambahkan image:

- gunakan file original dari pemilik atau aset dengan lisensi yang
  didokumentasikan;
- jangan mengambil foto dari marketplace, media, atau user-generated post tanpa
  izin;
- crop dari source resolusi tinggi dan simpan rasio/dimensi yang dibutuhkan;
- hindari menaruh teks penting hanya di dalam image;
- kompres secukupnya tanpa membuat label produk tidak terbaca.

## Netlify deployment dan redeploy

Production saat ini:

```text
https://tauco-cap-badak.netlify.app
```

Bagian ini menjadi runbook untuk deployment ulang atau recovery. Setiap push ke
production branch dapat memicu build baru bila continuous deployment aktif.

Netlify mendukung Next.js App Router melalui adapter modernnya. Project ini
memakai output `.next`, bukan static `out` dan bukan legacy plugin yang dipin
manual.

Referensi resmi:

- [Next.js on Netlify](https://docs.netlify.com/build/frameworks/framework-setup-guides/nextjs/overview/)
- [Netlify Forms setup](https://docs.netlify.com/manage/forms/setup/)

### Deploy dari Git

1. Push repository ke Git provider.
2. Di Netlify, pilih **Add new project** lalu import repository.
3. Gunakan production branch yang disepakati.
4. Pastikan build setting:
   - Build command: `npm run build`
   - Publish directory: `.next`
   - Node: `22`
5. Tambahkan environment variable pada production, Deploy Preview, dan branch
   deploy. Nilainya tetap origin production final:

   ```text
   NEXT_PUBLIC_SITE_URL=https://tauco-cap-badak.netlify.app
   ```

6. Pertahankan `GOOGLE_SITE_VERIFICATION`; penghapusan variable dapat
   menghilangkan metode ownership Search Console.
7. Jangan rename subdomain production tanpa migration plan.
8. Deploy.
9. Periksa deploy log, route, metadata, image, sitemap, robots, dan form.

Jika `netlify.toml` tersedia, file tersebut menjadi source-of-truth build
setting. Hindari konfigurasi yang berbeda antara dashboard dan repository.

### Menjaga biaya Rp0

- Gunakan Free plan dan pantau usage quota atau credit sesuai tipe plan pada
  dashboard.
- Jangan mengaktifkan paid add-on tanpa persetujuan.
- Batasi deploy production yang tidak perlu; gunakan local checks dan Deploy
  Preview.
- Periksa pricing dan terms saat ini sebelum mengandalkan angka quota tertentu,
  karena free-tier dapat berubah.
- Free tier tidak memberikan SLA bisnis.

## Netlify Forms

Form React/Next.js tidak cukup untuk dideteksi hanya dari runtime JSX. Project
memerlukan static HTML blueprint di `public/__forms.html` yang:

- mempunyai nama form yang sama dengan form UI;
- memakai `data-netlify="true"`;
- mencantumkan semua `name` field yang dapat dikirim UI;
- mencantumkan honeypot;
- tidak menjadi halaman navigasi publik.

Form UI mengirim:

- hidden `form-name`;
- field yang sama dengan blueprint;
- honeypot kosong;
- body URL-encoded;
- request ke static endpoint yang tidak ditangani route SSR.

### Aktivasi dan verifikasi

1. Buka site di Netlify.
2. Pastikan **Forms > Form detection** aktif.
3. Deploy ulang setelah detection diaktifkan atau field diubah.
4. Pastikan form muncul pada daftar Active forms.
5. Submit data test realistis dari Deploy Preview/production.
6. Verifikasi submission berada di daftar verified, bukan spam.
7. Konfigurasikan email notification kepada akun operasional.
8. Hapus submission test setelah verifikasi jika sudah tidak dibutuhkan.

Jika submission tidak muncul:

- cocokkan form name dan seluruh field antara UI dengan `__forms.html`;
- pastikan hidden `form-name` ikut dikirim;
- pastikan `Content-Type` dan body URL-encoded;
- pastikan honeypot dikirim kosong;
- periksa spam submissions;
- cek response network dan redirect;
- enable form detection lalu redeploy.

Local `next dev` tidak membuktikan bahwa Netlify telah mendeteksi atau menerima
form.

Acceptance Phase 1A membuktikan form `kontak` aktif, submission berada pada
Verified submissions, bukan spam, dan notification email utama diterima.
Backup recipient ditunda oleh owner. Seluruh submission QA sudah dihapus.

## SEO production checklist

Baseline berikut sudah lulus pada G6 dan wajib diulang setelah perubahan
metadata, origin, route, atau structured data:

- `NEXT_PUBLIC_SITE_URL` adalah origin production final.
- Semua route published menghasilkan `200`.
- Unknown product slug menghasilkan not found.
- Title, description, canonical, Open Graph, dan H1 unik.
- `https://<origin>/robots.txt` valid.
- `https://<origin>/sitemap.xml` berisi hanya canonical URL.
- JSON-LD lolos Rich Results Test/Schema validator yang relevan.
- Image memiliki dimensi dan alt text.
- Tidak ada placeholder, dummy contact, atau unverified claim.
- Tidak ada accidental `noindex` di production.
- Lighthouse dan accessibility gate lulus.
- Source HTML mengandung copy dan internal link utama.

Technical SEO membantu crawlability dan quality, tetapi tidak menjamin ranking
halaman pertama untuk kata `tauco`.

## Google Search Console

Untuk subdomain gratis Netlify, gunakan **URL-prefix property** dengan URL
production lengkap:

```text
https://tauco-cap-badak.netlify.app/
```

Status Phase 1A: ownership verified, verification meta aktif, sitemap berstatus
Sukses dengan tujuh halaman ditemukan, serta URL Inspection/request indexing
selesai untuk homepage, `/tauco`, `/produk`, dan detail produk. Property masih
baru sehingga data performa historis belum tersedia.

Panduan resmi:

- [Menambahkan property Search Console](https://support.google.com/webmasters/answer/34592)
- [Membangun dan mengirim sitemap](https://developers.google.com/search/docs/crawling-indexing/sitemaps/build-sitemap)

Langkah:

1. Login dengan akun bisnis yang akan menjadi owner jangka panjang.
2. Tambahkan URL-prefix property production.
3. Pilih metode **HTML tag**, lalu salin hanya nilai `content` dari meta tag.
4. Tambahkan nilai tersebut di Netlify sebagai environment variable
   `GOOGLE_SITE_VERIFICATION` untuk Builds dan production, lalu deploy ulang.
5. Pastikan meta verification tampil pada source production, kemudian klik
   **Verify** di Search Console.
6. Kirim
   `https://tauco-cap-badak.netlify.app/sitemap.xml` melalui laporan Sitemaps.
7. Gunakan URL Inspection untuk homepage, `/tauco`, `/produk`, dan detail
   produk.
8. Request indexing hanya setelah production content final.
9. Pantau Pages, Core Web Vitals, Enhancements, dan Manual Actions.
10. Catat baseline impression, click, CTR, serta average position.
11. Review query dan landing page setiap 28 hari; jangan mengubah strategi hanya
   karena fluktuasi harian.

Detail produk dapat diindeks sebagai hasil biasa, tetapi belum eligible untuk
seluruh Product enhancement karena Phase 1A tidak menerbitkan `Offer`, harga,
review, atau rating yang belum terverifikasi. Jangan membuat data palsu hanya
untuk menghilangkan warning Search Console.

Jika kelak memakai custom domain, buat Domain property melalui verifikasi DNS,
perbarui canonical/sitemap, dan ikuti migration checklist pada PRD.

## Batas fakta dan klaim

Sumber publik yang ditemukan belum cukup untuk memverifikasi seluruh identitas
bisnis. Aturan wajib:

- Jangan mengklaim “No. 1”, “terbaik”, “paling autentik”, “100% asli”, atau
  klaim ranking lain.
- Jangan mengklaim halal, izin aktif, tanpa pengawet, manfaat kesehatan, proses
  tradisional tertentu, tahun berdiri, atau warisan generasi tanpa bukti.
- Jangan mempublikasikan alamat, telepon, email, WhatsApp, map, jam buka, harga,
  atau SKU dari marketplace seolah-olah data resmi.
- Jangan menggunakan foto pihak ketiga tanpa izin.
- Penjelasan umum tauco Cianjur tidak boleh ditulis seolah-olah merupakan proses
  spesifik Tauco Cap Badak.

Daftar fakta, sumber, konflik, dan data yang masih dibutuhkan ada di
[FACT_CHECK.md](./FACT_CHECK.md). Jika ada konflik, dokumen tersebut dan bukti
terbaru dari pemilik bisnis mengalahkan copy pemasaran.

## Troubleshooting

### Canonical memakai localhost

- Periksa `NEXT_PUBLIC_SITE_URL` pada Netlify.
- Trigger deploy baru; environment public dibaca saat build.
- Periksa page source, sitemap, robots, dan JSON-LD setelah deploy.

### Build gagal karena konten

- Baca error schema dan path field.
- Perbaiki data; jangan melemahkan schema hanya agar build lolos.
- Jalankan `npm.cmd run check:frontend` dan E2E terkait setelah memperbaiki konten.

### Product route menjadi 404

- Pastikan record published.
- Pastikan slug cocok dan unik.
- Jalankan build ulang karena parameter detail diprerender dari content source.

### Contact form terlihat sukses tetapi submission tidak ada

- Jangan hanya mengandalkan state sukses UI.
- Ikuti checklist Netlify Forms di atas dan periksa response HTTP.
- Cocokkan static blueprint serta field form.

### Search Console belum menampilkan data

- Pastikan property dan protocol benar.
- Pastikan ownership sudah verified.
- Pastikan URL dapat dirayapi, tidak `noindex`, dan ada di sitemap.
- Data organik memerlukan waktu; submission sitemap bukan jaminan indexing.

## Dokumen terkait

- [PRD.md](./PRD.md) — scope, requirement, architecture target, dan acceptance.
- [FACT_CHECK.md](./FACT_CHECK.md) — source-of-truth fakta dan publishing guard.
- [WALKTHROUGH.md](./WALKTHROUGH.md) — progres, hasil gate, dan screenshot local.
