# Walkthrough Implementasi Phase 1A

Dokumen ini mencatat implementasi dan bukti verifikasi website publik
SEO-first Tauco Cap Badak. Scope berhenti pada Phase 1A dan seluruh pekerjaan
berjalan di local, Deploy Preview, dan production.

**Pembaruan terakhir:** 27 Juli 2026

**Status deployment:** Production tersedia di
`https://tauco-cap-badak.netlify.app`. G6 production smoke test dan G7 Forms,
Search Console, serta operasional sudah lulus. G8 selesai dan Phase 1A berstatus
**Complete** pada 28 Juli 2026. Tidak ada koneksi Supabase atau layanan backend.

## Ringkasan progres

| Poin | Milestone | Status | Bukti utama |
| --- | --- | --- | --- |
| 1 | Fondasi proyek | Selesai | Next.js, TypeScript, Tailwind, quality tooling |
| 2 | Konten tervalidasi dan aset provisional | Selesai | Async content source, Zod, tiga visual |
| 3 | Halaman publik dan design system | Selesai | Tujuh route, true 404, responsive light/dark |
| 4 | SEO teknis dan structured data | Selesai | Metadata, canonical, sitemap, JSON-LD, initial HTML |
| 5 | Form kontak dan privacy flow | Production acceptance selesai | Verified form dan email utama lulus |
| 6 | Automated tests dan pre-flight desain | Selesai | Local, preview, dan production gate hijau |
| 7 | Build, deployment, dan walkthrough | Production aktif | Static/SSG build, screenshot, dan G6 lulus |

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
- Production dependency audit melaporkan 0 vulnerability. Full audit
  melaporkan 9 high severity pada dev-only tree ESLint.

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
- Production dependency audit melaporkan 0 vulnerability. Full audit
  melaporkan 9 high severity pada dev-only chain
  `ESLint -> minimatch -> brace-expansion`; `npm audit fix --force` tidak
  dijalankan karena memaksa upgrade major.
- `next-env.d.ts` dipulihkan setelah audit dan tidak menjadi source change.
- Dokumentasi content workflow, privacy fact-check, test evidence, serta
  deferred launch gate telah diselaraskan.
- Remediasi awal tidak melakukan deployment. Deployment preview berikutnya
  dilakukan oleh pemilik project melalui PR #1.

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

Pada checkpoint local, hal berikut belum dapat diverifikasi:

- Deteksi form di dashboard Netlify.
- Submission masuk ke verified inbox Netlify.
- Email notification operasional.

Ketiga poin tersebut saat itu ditunda sesuai instruksi local-only dan kemudian
ditutup melalui G4 serta G7.

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

## Progress deployment: G4 automated QA

**Status: [x] 53 automated test dan owner evidence lulus**

Pada 27 Juli 2026, suite khusus G4.2 dijalankan terhadap:

```text
Pull Request: #1
Commit: 3a07ee0b56a5feb04deb23596833df76a6fc5bb8
Deploy Preview: https://deploy-preview-1--tauco-cap-badak.netlify.app
```

Command agregat:

```powershell
$env:PLAYWRIGHT_BASE_URL = "https://deploy-preview-1--tauco-cap-badak.netlify.app"
npm.cmd run qa:g4
Remove-Item Env:PLAYWRIGHT_BASE_URL
```

Hasil otomatis:

| Suite | Hasil |
| --- | --- |
| Functional | 13/13 lulus |
| Preview SEO isolation | 9/9 lulus |
| Form non-live | 6 lulus, 1 live test sengaja skip |
| Accessibility dan keyboard | 15/15 lulus |
| Browser matrix | 10 lulus, 6 duplikasi sengaja skip |
| Total | 53 lulus, 7 skip terdokumentasi |

Functional suite membuktikan tujuh route, true 404, internal link, mobile menu,
system dark mode, image/CLS, dan initial HTML. SEO suite membuktikan noindex,
robots, canonical, sitemap, serta tidak adanya preview/localhost origin.
Accessibility suite menjalankan Axe WCAG A/AA pada seluruh route, light/dark,
state form dinamis, keyboard focus, reduced motion, dan responsive reflow.

Browser matrix menggunakan Microsoft Edge terpasang, Playwright Firefox 151,
WebKit 26.5 sebagai pemeriksaan supplemental, dan emulasi Pixel 7 Android
Chrome. WebKit bukan pengganti Safari asli dan emulasi bukan pengganti perangkat
Android fisik.

Evidence lokal tersedia di:

```text
playwright-report/g4/functional
playwright-report/g4/seo-isolation
playwright-report/g4/form
playwright-report/g4/accessibility
playwright-report/g4/browser-matrix
```

### Netlify Forms live evidence

Satu, dan hanya satu, submission sintetis dikirim ke Deploy Preview:

| Field | Nilai |
| --- | --- |
| Waktu | 27 Juli 2026, 10:34:56 WIB |
| Nama | `QA Phase 1A` |
| Email | `qa-phase1a@example.com` |
| Run ID | `2026-07-27-G4-PR1` |
| HTTP response | 200 |
| Netlify request ID | `01KYGT7R41AA1QXM1TG26AJMK8` |

UI success tampil setelah response 200. Pemilik project mengonfirmasi form
`kontak` terlihat pada Active forms dan submission berstatus verified serta
tidak masuk spam. Submission sintetis sekarang boleh dihapus.

### Release gate lokal terbaru

| Pemeriksaan | Hasil |
| --- | --- |
| Lint | Lulus, 0 warning |
| TypeScript | Lulus |
| Unit | 73/73 lulus |
| Build | Lulus, 13 route Static/SSG |
| E2E lokal | 79 lulus, 13 intentional skip |
| Lighthouse | 12/12 lulus |
| Performance Lighthouse | 95-97 |
| Accessibility, SEO, Best Practices | 100 |
| Production dependency audit | 0 vulnerability |
| Full dependency audit | 9 high, dev-only ESLint chain |
| Git diff check | Lulus |

Runner Lighthouse sekarang mencetak Performance, LCP, dan TBT setiap run.
Median LCP seluruh route memenuhi batas 2,5 detik dan report atomik berstatus
`passed` di `.lighthouseci/run-status.json`.

## Konfirmasi pemilik dan status G5

Pemilik mengonfirmasi Netlify Free, environment variable seluruh context, form
aktif, verified submission, Safari asli, Android Chrome asli, zoom 200 persen,
owner inbox, retensi 12 bulan, copy/visual provisional, dan penerimaan risiko
dev-only advisory. Pemilik juga menyetujui bahwa data yang belum terverifikasi
tetap dihilangkan. Dengan konfirmasi tersebut, G1, G3, dan G4 dinyatakan lulus.

Alamat email operasional dimask pada dokumen repository publik. Nilai lengkap
tetap berada pada dashboard Netlify atau catatan privat pemilik.

Owner kemudian melakukan merge PR #1. Netlify memublikasikan commit
`2ce0a310075224b2cb8bb470d0e0ba4d0d301b98` pada 27 Juli 2026 pukul
13:24:42 WIB. Tindakan owner tersebut dicatat sebagai keputusan GO. Pemahaman
prosedur rollback tetap menjadi item operasional yang perlu dikonfirmasi.

Screenshot Usage & billing mengonfirmasi Free Legacy: bandwidth 70 MB dari
100 GB, build 3 dari 300 menit, concurrent build 0 dari 1, dan satu anggota.
Seluruh meter yang terlihat jauh di bawah threshold internal 90 persen, sehingga
gate usage quota lulus.

### Hasil G6 production smoke test

Production QA dijalankan dengan guard yang hanya menerima exact origin
`https://tauco-cap-badak.netlify.app`.

```powershell
npm.cmd run qa:g6
npm.cmd run lighthouse:production
```

Hasilnya:

| Pemeriksaan | Hasil |
| --- | --- |
| Route, true 404, SEO, schema, security, form contract, dan accessibility | 29/29 lulus |
| Lighthouse, empat route kali tiga run | 12/12 lulus |
| Performance | 96-100 |
| Accessibility dan SEO | 100 |
| LCP terburuk | 2.132 ms |
| CLS | 0 |
| Schema.org Validator homepage | 0 error, 0 warning |
| Schema.org Validator detail produk | 0 error, 0 warning |

Seluruh tujuh route menghasilkan 200 dan slug produk yang tidak dikenal
menghasilkan true 404. Production indexable, canonical dan Open Graph memakai
origin production, robots/sitemap benar, JSON-LD faktual, HTTPS serta security
headers aktif, dan tidak ada mixed content atau console blocker.

Report lokal tersimpan di:

```text
playwright-report/production
.lighthouseci-production
```

### Progress G7 Forms dan Search Console

Tepat satu submission sintetis dikirim ke form production:

| Field | Nilai |
| --- | --- |
| Waktu | 27 Juli 2026, 15:31:43 WIB |
| Nama | `QA Production Phase 1A` |
| Email | `qa-phase1a@example.com` |
| Run ID | `2026-07-27-G7-PRODUCTION` |
| HTTP response | 200 |
| Netlify request ID | `01KYHB76A0M1N2HJQY4HVSZQDH` |

HTTP acceptance sudah lulus. Screenshot owner mengonfirmasi submission G7
berada di Verified submissions dan bukan Spam. Owner masih perlu memastikan
kedua notification email diterima, lalu menghapus submission sintetis jika
tidak lagi dibutuhkan.

Owner menambahkan token `google-site-verification` dan memublikasikan deploy
`6a671c07dba5f22c9cfab616` pada 27 Juli 2026 pukul 15:52:14 WIB. Pemeriksaan
publik setelah deploy membuktikan homepage HTTP 200, exact verification meta
tersedia, tidak ada `noindex`, dan sitemap tetap berisi tujuh production URL
tanpa preview atau localhost. Owner selanjutnya perlu menekan **Verify** di
Search Console, mengirim sitemap, dan menjalankan URL Inspection.

Owner kemudian menyelesaikan ownership verification dan mengirim
`sitemap.xml`. Screenshot Search Console pada 27 Juli 2026 menunjukkan status
**Sukses**, sitemap telah dibaca, dan tepat tujuh halaman ditemukan. G7.2
selanjutnya berfokus pada URL Inspection dan request indexing empat URL
prioritas.

Owner mengonfirmasi URL Inspection dan request indexing selesai untuk
homepage, `/tauco`, `/produk`, dan `/produk/tauco-cap-badak`. Request indexing
adalah permintaan crawl, bukan jaminan halaman langsung terindeks atau mendapat
peringkat tertentu.

Setelah owner mengaktifkan notification email utama dan cadangan, satu
submission acceptance tambahan dikirim tanpa retry:

| Field | Nilai |
| --- | --- |
| Waktu | 27 Juli 2026, 16:10:13 WIB |
| Run ID | `2026-07-27-G7-NOTIFICATION` |
| HTTP response | 200 |
| Netlify request ID | `01KYHDDP0DCHWME30K7EV0NY3J` |

Transport form lulus dan owner mengonfirmasi notification email utama diterima.
Notification cadangan belum ditambahkan dan sengaja ditunda oleh owner sebagai
follow-up operasional.

Pada 28 Juli 2026, owner mengonfirmasi ketiga submission QA telah dihapus
permanen dan pengingat review retensi sudah dibuat. Baseline Search Console
mencatat sitemap sukses dengan tujuh halaman, empat URL prioritas telah diminta
untuk diindeks, dan data historis belum tersedia karena property masih baru.
Peringatan Product enhancement diterima untuk Phase 1A karena harga,
`Offer`, review, dan rating yang belum terverifikasi tidak boleh diterbitkan.

Owner mengonfirmasi akun Search Console akan dipertahankan sebagai akun owner
jangka panjang, memahami SOP akses/koreksi/penghapusan data, serta memahami
prosedur rollback Netlify. Dengan bukti tersebut, G7 dinyatakan **Lulus** pada
28 Juli 2026. Backup notification tetap menjadi follow-up yang diterima owner
dan tidak memblokir launch karena notification utama sudah lulus end-to-end.

## G8 dokumentasi dan penutupan

**Status: [x] Complete pada 28 Juli 2026**

Dokumen `PRD.md`, `README.md`, `FACT_CHECK.md`, `plan.md`, dan walkthrough ini
diselaraskan dengan keadaan production. `plan.pdf` diregenerasi langsung dari
`plan.md` agar checklist Markdown dan PDF identik.

Final quality gate:

| Pemeriksaan | Hasil |
| --- | --- |
| `npm.cmd run check` | Lulus |
| ESLint | Lulus, 0 warning |
| TypeScript | Lulus |
| Unit test | 73/73 lulus |
| Production build | Lulus, 13 route Static/SSG |
| Production smoke terbaru | 29/29 lulus |
| Production dependency audit | 0 vulnerability |
| Full dependency audit | 9 high pada development-only ESLint chain; accepted risk |
| `git diff --check` | Lulus |

Release production sudah commit, push, dan merge pada commit
`2ce0a310075224b2cb8bb470d0e0ba4d0d301b98`. Perubahan suite QA serta
dokumentasi closeout masih berada di worktree `launch/phase-1a` dan sengaja
belum di-commit atau di-push oleh assistant. Owner menangani version-control
handoff setelah review.

## Data resmi yang masih dibutuhkan

Data berikut hanya perlu disediakan sebelum ingin ditampilkan. Implementasi
sekarang tetap aman karena field tersebut tidak dipublikasikan:

- logo resmi dan panduan warna;
- foto produk yang hak pakainya jelas;
- alamat, email, telepon/WhatsApp, dan jam operasional;
- tahun berdiri, sejarah, dan identitas pemilik;
- daftar kemasan, ukuran, harga, serta availability;
- legalitas dan sertifikasi;
- proses produksi yang boleh dipublikasikan;
- social links;

Daftar rinci dan publishing guard ada di [FACT_CHECK.md](./FACT_CHECK.md).

## Batas Phase 1A

Tidak diimplementasikan:

- backend Go, Gin, GORM, PostgreSQL, Supabase, atau Redis;
- JWT/PASETO, Admin CMS, worker upload, dan activity log;
- inventory, stock concurrency, order, checkout, atau payment;
- production analytics khusus atau layanan email pihak ketiga;
- backend atau database deployment.

Implementasi Phase 1A berstatus **Complete**, production aktif, dan G0–G8
sudah lulus. Follow-up yang tidak memblokir adalah backup notification, data
brand resmi, serta monitoring Search Console/Netlify setelah launch.
