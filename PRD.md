# Product Requirements Document

## Tauco Cap Badak Website & Admin CMS — Phase 1

| Atribut | Nilai |
| --- | --- |
| Versi dokumen | 1.0 |
| Tanggal | 24 Juli 2026 |
| Status produk | Phase 1A implemented scope; Phase 1B–1D planned |
| Bahasa produk | Indonesia |
| Target pasar awal | Indonesia |
| Target deployment Phase 1A | Netlify Free setelah approval; belum dieksekusi |
| Pemilik dokumen | Product / Engineering |

## 1. Ringkasan

Tauco Cap Badak Website & Admin CMS adalah fondasi digital untuk memperkenalkan
brand, mengedukasi pengunjung tentang tauco, menampilkan produk, dan menerima
pertanyaan pelanggan. Sasaran bisnis jangka panjangnya adalah membangun
visibilitas organik untuk pencarian terkait `tauco`, lalu menyediakan CMS yang
dapat berkembang menjadi sistem inventory dan order management.

Delivery dibagi agar website publik dapat diluncurkan lebih cepat:

- **Phase 1A — implemented local scope:** website publik SEO-first dengan konten lokal
  tervalidasi, halaman yang diprerender, katalog awal, detail produk, formulir
  kontak berkontrak Netlify Forms, metadata SEO, sitemap, robots, structured
  data, dan privacy notice. Deployment serta validasi dashboard ditunda.
- **Phase 1B — future:** fondasi REST API Go, PostgreSQL, Redis, media storage,
  job processing, observability, dan security middleware.
- **Phase 1C — future:** Admin CMS, authentication, publishing workflow,
  product/media management, dan inbox.
- **Phase 1D — future:** hardening, migration konten lokal ke CMS, load/security
  test, backup, dan operational readiness.
- **Phase 2 — out of scope:** inventory, warehouse, order, payment, dan
  transaction processing.

Label status dalam dokumen ini:

- `IMPLEMENTED-1A`: termasuk implementasi repository dan verifikasi local
  Phase 1A, bukan bukti deployment atau integrasi dashboard eksternal.
- `PLANNED`: desain target yang belum diimplementasikan pada Phase 1A.
- `OUT-OF-SCOPE`: bukan bagian delivery Phase 1.

## 2. Latar Belakang dan Problem Statement

### 2.1 Masalah bisnis

- Tauco Cap Badak belum memiliki pusat informasi digital yang dapat dikontrol
  dan dijadikan sumber rujukan.
- Informasi publik tentang sejarah, alamat, kontak, legalitas, dan produk saat
  ini tersebar serta sebagian tidak dapat diverifikasi.
- Pencarian kata `tauco` memiliki intent yang luas: definisi, sejarah, resep,
  produk, dan oleh-oleh. Satu halaman promosi tidak cukup untuk menjawab seluruh
  intent tersebut.
- Pengelolaan konten secara manual tidak memadai untuk ekspansi produk dan
  operasi pada fase berikutnya.

### 2.2 Masalah engineering

- Website harus menghasilkan HTML yang dapat dirayapi tanpa menunggu
  client-side fetch.
- Arsitektur frontend perlu dapat berpindah dari konten lokal ke REST API tanpa
  mengganti URL publik atau komponen presentasi secara besar.
- Backend mendatang harus memiliki batas modul, transaksi, job durability, dan
  observability yang cukup untuk tumbuh menuju inventory serta order management.

## 3. Tujuan dan Ukuran Keberhasilan

### 3.1 Tujuan bisnis

1. Menyediakan sumber informasi resmi Tauco Cap Badak yang jelas dan mudah
   ditemukan.
2. Menjawab intent pencarian pengunjung terkait tauco dan Tauco Cap Badak.
3. Mengarahkan pengunjung untuk mengenal produk dan mengirim pertanyaan.
4. Membangun fondasi authority organik secara bertahap.

### 3.2 Tujuan engineering

1. Menghasilkan website public-facing dengan Next.js App Router, Server
   Components sebagai default, dan HTML yang dapat dirayapi.
2. Memisahkan content source dari UI agar migrasi ke API tidak mengubah URL,
   slug, atau kontrak presentasi.
3. Menjaga Core Web Vitals, accessibility, semantic HTML, dan metadata pada
   standar production.
4. Mendefinisikan target Clean Architecture backend sebelum CMS dibangun.

### 3.3 KPI

#### KPI teknis yang menjadi acceptance gate

- Seluruh route published dapat diakses dan tidak memiliki broken internal link.
- Lighthouse mobile pada halaman utama: Performance ≥ 90, SEO ≥ 95, dan
  Accessibility ≥ 90 pada kondisi pengujian yang disepakati.
- Gate synthetic local: median LCP ≤ 2,5 detik dan CLS ≤ 0,1 pada route
  representatif.
- Target field Core Web Vitals setelah launch: LCP p75 ≤ 2,5 detik,
  INP p75 ≤ 200 ms, dan CLS p75 ≤ 0,1.
- `robots.txt` dan `sitemap.xml` dapat diakses serta memakai production origin
  yang benar.
- Canonical, title, description, Open Graph, dan structured data tidak memakai
  domain placeholder.
- Konten utama tetap tersedia ketika JavaScript dinonaktifkan.
- Kontrak form, validation state, dan payload lulus test local. Penerimaan
  dashboard Netlify menjadi deferred launch gate.
- Tidak ada klaim brand yang melanggar kebijakan pada
  [FACT_CHECK.md](./FACT_CHECK.md).

#### KPI bisnis yang dipantau setelah launch

- Indexed pages, impressions, clicks, CTR, dan average position melalui Google
  Search Console.
- Pertumbuhan impression untuk kueri non-brand terkait tauco.
- Jumlah valid contact submission dan klik CTA produk.
- Target halaman pertama Google Indonesia untuk kueri `tauco` dalam 6–12 bulan
  adalah **KPI aspiratif**, bukan acceptance criterion atau jaminan engineering.
  Ranking dipengaruhi kompetisi, kualitas konten, backlink, domain authority,
  perilaku pengguna, dan perubahan algoritma Google.

## 4. Pengguna dan Kebutuhan

### 4.1 Pengunjung umum

Ingin memahami apa itu tauco, mengenal Tauco Cap Badak, melihat produk, dan
mengetahui cara menghubungi pemilik usaha.

### 4.2 Calon pembeli atau distributor

Ingin melihat bentuk produk, informasi dasar, ketersediaan, dan kanal untuk
meminta harga atau informasi pemesanan.

### 4.3 Pemilik usaha

Pada Phase 1A membutuhkan company profile yang dapat diluncurkan cepat. Pada
fase berikutnya membutuhkan pengelolaan konten tanpa perubahan source code.

### 4.4 Admin operasional

`PLANNED` — membutuhkan login, content publishing, product/media management,
inbox, dan audit trail.

## 5. Scope dan Status Delivery

| Kapabilitas | Status | Catatan |
| --- | --- | --- |
| Homepage | `IMPLEMENTED-1A` | Konten lokal, prerendered |
| Tentang Kami | `IMPLEMENTED-1A` | Hanya fakta yang aman dipublikasikan |
| Panduan tauco | `IMPLEMENTED-1A` | Konten edukatif untuk intent informasional |
| Katalog produk | `IMPLEMENTED-1A` | Katalog awal berbasis konten lokal |
| Detail produk dengan slug | `IMPLEMENTED-1A` | Slug harus dipertahankan |
| Kontak | `IMPLEMENTED-1A` | Contract/blueprint local; dashboard belum diuji |
| Kebijakan privasi | `IMPLEMENTED-1A` | Mencakup data formulir |
| Technical SEO | `IMPLEMENTED-1A` | Metadata, canonical, sitemap, robots, JSON-LD |
| Responsive dan accessibility baseline | `IMPLEMENTED-1A` | Semantic HTML dan keyboard flow |
| REST API Go | `PLANNED` | Phase 1B |
| PostgreSQL/Supabase | `PLANNED` | Phase 1B |
| Redis/Upstash | `PLANNED` | Phase 1B |
| Object storage dan image worker | `PLANNED` | Phase 1B |
| Admin authentication | `PLANNED` | Phase 1C |
| CMS homepage/about | `PLANNED` | Phase 1C |
| Product/media CMS | `PLANNED` | Phase 1C |
| Inbox dan activity log | `PLANNED` | Phase 1C |
| Inventory dan order | `OUT-OF-SCOPE` | Phase 2 |

## 6. Functional Requirements — Phase 1A

### 6.1 Global navigation dan layout

**Status:** `IMPLEMENTED-1A`

- Header menyediakan akses ke Beranda, Mengenal Tauco, Tentang Kami, Produk,
  dan Kontak.
- Logo/wordmark kembali ke homepage.
- Navigasi desktop dan mobile mempunyai urutan serta destination yang sama.
- Menu mobile dapat dibuka, ditutup, dioperasikan dengan keyboard, dan
  mengembalikan focus secara layak.
- Footer menyediakan internal link utama, privacy notice, dan identitas brand.
- Informasi alamat, telepon, email, social media, jam operasional, dan legalitas
  hanya boleh ditampilkan setelah masuk status terverifikasi di
  `FACT_CHECK.md`.
- Semua halaman memiliki satu primary heading yang menjelaskan tujuan halaman.

### 6.2 Homepage

**Status:** `IMPLEMENTED-1A`

Homepage harus:

1. Menjelaskan bahwa Tauco Cap Badak merupakan produk tauco asal Cianjur
   menggunakan wording yang telah diverifikasi.
2. Memiliki satu CTA utama pada hero menuju katalog; CTA kontak ditempatkan
   pada section terpisah.
3. Menampilkan ringkasan brand tanpa mencantumkan sejarah yang belum
   terkonfirmasi.
4. Menampilkan produk unggulan dari content source.
5. Memberi konteks edukatif singkat mengenai tauco serta internal link menuju
   konten yang lebih relevan.
6. Menampilkan trust signal hanya jika bukti dan hak pakainya tersedia.

### 6.3 Tentang Kami

**Status:** `IMPLEMENTED-1A`

- Halaman menjelaskan konteks Tauco Cap Badak dan keterkaitannya dengan Cianjur.
- Sejarah umum tauco Cianjur harus dipisahkan dari sejarah spesifik brand.
- Tahun berdiri, nama pendiri, generasi pengelola, metode produksi, dan klaim
  warisan tidak boleh disajikan sebagai fakta sebelum dikonfirmasi pemilik.
- Halaman memiliki link menuju panduan tauco dan Produk; Kontak tetap tersedia
  pada navigasi global.

### 6.4 Panduan tauco

**Status:** `IMPLEMENTED-1A`

- Route `/tauco` menjawab intent informasional “apa itu tauco”.
- Halaman membahas pengertian, proses fermentasi secara umum, ragam, konteks
  Cianjur, penggunaan, dan penyimpanan.
- Proses yang dijelaskan harus diberi konteks umum dan tidak boleh
  diatribusikan sebagai proses khusus Tauco Cap Badak.
- Saran penggunaan bukan pengganti petunjuk label produk.
- Health claim tidak boleh ditambahkan hanya untuk memperluas keyword.
- Halaman mempunyai sumber, internal link ke brand, dan link ke produk.

### 6.5 Katalog produk

**Status:** `IMPLEMENTED-1A`

- Hanya produk berstatus published pada content source yang ditampilkan.
- Setiap card minimal memiliki nama, deskripsi ringkas, image/placeholder dengan
  alt text yang tepat, dan link detail.
- Harga bersifat opsional. Apabila harga resmi belum tersedia, UI memakai CTA
  kontak, bukan harga marketplace pihak ketiga.
- Ukuran, berat bersih, komposisi, sertifikasi, shelf life, dan instruksi
  penyimpanan tidak boleh ditebak.
- Empty state harus menjelaskan bahwa informasi produk belum tersedia tanpa
  menampilkan error teknis.

### 6.6 Detail produk

**Status:** `IMPLEMENTED-1A`

- Route menggunakan slug manusiawi dan stabil: `/produk/[slug]`.
- Build menghasilkan parameter untuk seluruh produk published.
- Slug yang tidak dikenal menghasilkan not found, bukan detail kosong.
- Detail minimal mencakup nama, deskripsi, image, dan CTA kontak.
- Structured data `Product` hanya memuat property yang mempunyai sumber.
- `Offer`, rating, review, availability, dan harga tidak boleh disertakan jika
  datanya tidak resmi.

### 6.7 Contact Us

**Status:** `IMPLEMENTED-1A`

Field:

- Nama — wajib.
- Email — wajib dan harus valid.
- Nomor telepon/WhatsApp — opsional.
- Subjek — wajib.
- Pesan — wajib.
- Persetujuan privacy notice — wajib.
- `bot-field` — honeypot, tidak boleh diisi pengguna.
- `form-name` — hidden field untuk Netlify Forms.

Behavior:

- Form melakukan validasi client-side dan tetap memakai semantic label.
- Submit dikirim sebagai `application/x-www-form-urlencoded` ke static Netlify
  form endpoint yang telah dideteksi saat deploy.
- Tombol submit tidak dapat dikirim berulang selama request berlangsung.
- Sukses dan gagal ditampilkan secara inline dan diumumkan ke assistive
  technology.
- Pesan sukses tidak mengungkap data internal.
- Kegagalan network tidak menghapus input pengguna.
- Form tidak menjanjikan waktu balas yang belum disepakati bisnis.
- Submission hanya dapat diverifikasi penuh pada Deploy Preview/production
  Netlify; local development bukan bukti integrasi.

### 6.8 Kebijakan privasi

**Status:** `IMPLEMENTED-1A`

Privacy notice minimal menjelaskan:

- data yang dikirim melalui form;
- tujuan penggunaan data;
- platform pemroses formulir pada fase berjalan;
- siapa yang dapat mengakses submission;
- prinsip penyimpanan dan penghapusan;
- cara meminta koreksi atau penghapusan setelah kontak resmi tersedia.

Scope Phase 1A menetapkan retensi pesan maksimal 12 bulan. Pemilik operasional
data dan prosedur penghapusan tetap harus dikonfirmasi sebelum deployment.

### 6.9 Error dan not found

**Status:** `IMPLEMENTED-1A`

- Route yang tidak ada menghasilkan status dan UI not found yang dapat dipahami.
- Pesan error tidak menampilkan stack trace, internal path, atau configuration.
- Pengunjung selalu mendapat jalur kembali ke homepage atau katalog.

## 7. Content Architecture — Phase 1A

**Status:** `IMPLEMENTED-1A`

- Konten disimpan lokal dalam format terstruktur dan divalidasi saat
  build/runtime boundary.
- UI tidak mengimpor bentuk file mentah secara tersebar; akses dilakukan melalui
  interface content source.
- Interface bersifat asynchronous agar adapter lokal dapat diganti REST adapter
  tanpa mengubah page contract.
- DTO inti mencakup homepage, about, product summary, product detail, money
  opsional, dan media.
- Produk published mempunyai unique slug.
- Setiap image mempunyai alt text; decorative image menggunakan empty alt.
- Perubahan konten harus mengikuti prosedur di
  [README.md](./README.md#memperbarui-konten).
- Fakta brand wajib mengikuti [FACT_CHECK.md](./FACT_CHECK.md).

## 8. SEO Requirements — Phase 1A

**Status:** `IMPLEMENTED-1A`

### 8.1 Crawlability dan indexability

- Konten utama tersedia pada HTML response.
- Production pages tidak memakai `noindex`.
- Non-production deployment tidak boleh menjadi canonical origin.
- `robots.txt` mengizinkan route publik dan mereferensikan sitemap.
- `sitemap.xml` hanya memuat URL canonical yang intended untuk index.
- URL menggunakan huruf kecil, kata Bahasa Indonesia, dan tanda hubung.

### 8.2 On-page SEO

- Setiap route memiliki title dan meta description unik.
- H1 unik dan sesuai intent halaman.
- Canonical dibentuk dari production site URL.
- Internal link menggunakan anchor yang deskriptif.
- Image yang membawa informasi memiliki alt text kontekstual.
- Copy tidak melakukan keyword stuffing atau membuat halaman doorway.

### 8.3 Structured data

- `Organization`/`LocalBusiness` hanya menggunakan field identitas yang
  terverifikasi.
- `WebSite` menggunakan production origin.
- Breadcrumb digunakan pada hierarchy produk.
- `Product` tidak mengandung harga, rating, sertifikasi, atau stok yang tidak
  bersumber.
- JSON-LD harus valid dan konsisten dengan konten yang terlihat.

### 8.4 Social metadata

- Open Graph dan Twitter metadata menggunakan title, description, URL, dan image
  yang sesuai.
- Production social image harus merupakan aset yang dimiliki atau berlisensi.
- Tidak menggunakan foto marketplace atau user-generated content tanpa izin.

### 8.5 Keyword dan content strategy

- Kueri utama: `tauco`.
- Supporting intent: tauco Cianjur, apa itu tauco, produk tauco Cianjur, dan
  penggunaan tauco.
- Homepage berfokus pada brand + category.
- Halaman edukasi menjawab intent informasional, bukan mengulang homepage.
- Katalog/detail menjawab intent produk.
- Search Console menjadi sumber pengukuran; posisi manual atau incognito tidak
  menjadi sumber KPI.

## 9. Non-Functional Requirements — Phase 1A

### 9.1 Performance

- Server Components digunakan sebagai default.
- Client Components dibatasi pada interaksi yang membutuhkan browser state.
- Image menggunakan dimensi eksplisit dan responsive source.
- Font tidak menyebabkan layout shift yang material.
- Third-party script tidak ditambahkan tanpa kebutuhan dan review dampak.
- Build tidak bergantung pada API eksternal untuk konten awal.

### 9.2 Accessibility

- Target WCAG 2.2 AA.
- Seluruh flow dapat dioperasikan dengan keyboard.
- Focus state terlihat.
- Heading hierarchy, landmark, label, error association, dan live region benar.
- Contrast teks dan interactive control memenuhi minimum.
- Motion menghormati `prefers-reduced-motion`.
- Automated axe test melengkapi, bukan menggantikan, manual keyboard test.

### 9.3 Browser dan responsive behavior

- Automated browser test berjalan pada Chromium desktop dan mobile. Verifikasi
  manual Edge, Firefox, dan Safari menjadi deferred launch gate.
- Layout diuji minimal pada 320 px, 768 px, 1024 px, dan desktop lebar.
- Tidak ada horizontal scroll akibat content overflow pada viewport target.

### 9.4 Security dan privacy

- Tidak ada secret di source code atau `NEXT_PUBLIC_*`.
- User input tidak dirender kembali sebagai HTML.
- Form memakai honeypot dan platform spam filtering.
- Dependency dan Next.js security advisory ditinjau sebelum launch.
- Security header diaktifkan tanpa memblokir resource aplikasi yang sah.
- Data form tidak disalin ke log frontend.

### 9.5 Reliability dan operability

- Production build harus reproducible dari lockfile.
- Website informational tetap dapat disajikan tanpa database atau Go API.
- Netlify credit usage dipantau; free tier tidak dianggap SLA.
- Form submission diperiksa di dashboard dan mempunyai notification owner.

## 10. Target Deployment Phase 1A

**Status:** `PLANNED` — konfigurasi repository siap, deployment belum
dieksekusi atas instruksi pemilik project.

- Runtime target: Netlify dengan adapter Next.js modern.
- Output tetap `.next`; aplikasi tidak menggunakan legacy `next export`.
- Node production version dikunci pada major yang didukung project.
- Production environment wajib mempunyai `NEXT_PUBLIC_SITE_URL` yang sama
  persis dengan public origin.
- Production subdomain `*.netlify.app` dipilih sekali dan dijaga stabil sampai
  custom domain tersedia.
- Deploy Preview dapat dipakai untuk QA setelah pemilik project memberi
  approval; production deploy hanya dilakukan setelah launch gate lulus.
- Saat deployment diizinkan, Netlify Form Detection harus aktif dan deploy
  harus mendeteksi static form blueprint.
- Panduan operasional lengkap tersedia di [README.md](./README.md).

### 10.1 Domain strategy

Memakai subdomain Netlify memungkinkan biaya hosting awal Rp0 dan tidak
menghalangi prerendering atau technical SEO. Namun custom domain tetap
direkomendasikan ketika anggaran tersedia karena memberi ownership, trust,
portabilitas, dan brand authority yang lebih baik.

Jika berpindah ke custom domain:

1. pertahankan path dan slug;
2. pasang permanent redirect satu-ke-satu;
3. ubah production environment URL;
4. regenerate canonical, sitemap, structured data, dan Open Graph URL;
5. verifikasi property baru di Search Console;
6. gunakan Change of Address bila sesuai;
7. pertahankan redirect domain lama selama mungkin.

## 11. Future Architecture — Phase 1B dan 1C

Seluruh bagian ini berstatus `PLANNED` dan bukan dependency runtime Phase 1A.

### 11.1 Topologi target

- Next.js tetap menangani public web dan `/admin`.
- Go 1.21+ dengan Gin menyediakan versioned REST API.
- GORM hanya digunakan di repository/infrastructure layer.
- PostgreSQL menjadi source of truth.
- Redis digunakan sebagai cache dan distributed rate-limit store.
- Object storage menyimpan original serta image variants.
- API stateless dan worker dapat di-scale terpisah.
- Target free-tier awal: Supabase PostgreSQL, Upstash Redis, Cloudflare R2,
  Cloud Run scale-to-zero, dan provider email ber-free-tier. Seluruh quota,
  billing requirement, dan terms wajib ditinjau ulang saat implementasi.

### 11.2 Clean Architecture

Setiap bounded module memisahkan:

1. **Domain/entity:** business rule dan value object tanpa dependency framework.
2. **Use case/service:** orchestration, authorization, transaction boundary.
3. **Repository interface:** port untuk persistence/cache/storage.
4. **Delivery/handler:** HTTP parsing, validation, response mapping.
5. **Infrastructure:** GORM, Redis, object storage, email, logging, dan job
   adapter.

Dependency injection menggunakan constructor. Handler tidak mengakses GORM
secara langsung dan domain tidak bergantung pada Gin.

Modul awal:

- `auth`
- `content`
- `catalog`
- `media`
- `contact`
- `jobs`
- `audit`

### 11.3 Public REST API

Base path: `/api/v1`

| Method | Path | Tujuan |
| --- | --- | --- |
| GET | `/home` | Published homepage |
| GET | `/about` | Published about content |
| GET | `/products` | Published product list |
| GET | `/products/{slug}` | Published product detail |
| POST | `/contact-messages` | Persist visitor message |

Response sukses memakai envelope `{ "data": ..., "meta": ... }`. Error memakai
RFC 7807 `application/problem+json` dan memiliki `requestId`. Pagination list
menggunakan opaque cursor.

### 11.4 Admin REST API

- login, refresh, logout, current user;
- setup/enable/disable TOTP;
- read/save draft, preview, publish, dan unpublish page;
- product create/read/update, preview, publish, archive;
- media upload dan status polling;
- inbox list/detail serta read/unread;
- operational health dan protected cache revalidation.

OpenAPI menjadi contract dan divalidasi dalam CI.

### 11.5 Authentication dan authorization

- Role aktif Phase 1: `super_admin`; schema tetap siap untuk RBAC.
- Tidak ada public registration.
- Password menggunakan Argon2id.
- Access JWT RS256 berumur pendek dan disimpan pada secure HttpOnly cookie.
- Opaque refresh token dirotasi, disimpan dalam bentuk hash, dan dapat dicabut
  per session.
- TOTP opsional bagi bisnis tetapi wajib diverifikasi saat login apabila aktif.
- Cookie mutation dilindungi Origin check, CSRF token, SameSite policy, dan CORS
  allowlist.
- Admin browser mengakses API melalui same-origin BFF/proxy agar cookie tidak
  menjadi third-party cookie lintas platform.

### 11.6 Content publishing

- Homepage dan About merupakan singleton content.
- Draft dan published revision dipisahkan.
- Public site selalu membaca last published revision.
- Preview hanya dapat diakses authenticated user dan diberi `noindex`.
- Product slug stabil setelah publish pertama.
- Publish ditolak jika media wajib belum `ready`.

### 11.7 Data model target

- `admin_users`, `roles`, `permissions`, `user_roles`
- `admin_sessions`, `mfa_credentials`
- `pages`, immutable `page_revisions`
- `products`, immutable `product_revisions`
- `media_assets`, ordered media relationships
- `contact_messages`
- `background_jobs`
- append-only `activity_logs`

Entity memakai UUIDv7 dan `timestamptz`. Uang memakai integer minor unit dan
currency `IDR`, bukan floating point. GORM `AutoMigrate` dilarang di production;
migration memakai versioned SQL.

### 11.8 Concurrency dan durable background jobs

Goroutine tidak boleh dipakai sebagai fire-and-forget setelah HTTP response.

#### Image processing

1. API menyimpan media record dan durable job.
2. Worker mengklaim job secara atomik, misalnya dengan
   `FOR UPDATE SKIP LOCKED`.
3. Bounded channel mendistribusikan pekerjaan ke configurable goroutine pool.
4. Worker menghasilkan WebP 320, 640, dan 1280 piksel serta mempertahankan
   original.
5. Status bergerak `processing` → `ready` atau `failed`.

#### Contact side effects

1. Contact message harus committed sebelum API mengembalikan `201`.
2. Email notification dan activity event diproses oleh durable job.
3. Gagal email tidak membatalkan message yang sudah diterima.

Semua job memiliki idempotency key, attempts, `run_at`, lock owner, last error,
exponential backoff, dead-letter state, graceful shutdown, dan manual replay.
Pada runtime scale-to-zero, external task trigger boleh membangunkan worker;
handler harus menyelesaikan atau menyerahkan kembali job sebelum request
berakhir.

### 11.9 Redis caching

- Public home, about, product list, dan product detail memakai cache-aside.
- Key mengandung version/locale/query dimension yang relevan.
- Default TTL lima menit ditambah jitter.
- Request coalescing mencegah cache stampede.
- Redis failure bersifat fail-open ke PostgreSQL untuk public read.
- Publish/unpublish melakukan targeted invalidation dan memicu Next.js
  revalidation.
- Tag minimal: `home`, `about`, `products`, dan `product:{slug}`.

### 11.10 Security middleware

- Distributed rate limiting:
  - public GET: baseline 60 request/menit/IP;
  - contact: baseline 5 request/jam/IP;
  - login: baseline 5 percobaan/15 menit per IP + account;
  - admin: baseline 120 request/menit/user.
- Local limiter menjadi fallback saat Redis tidak tersedia.
- Structured logging menggunakan Zap dengan request ID, route, status, latency,
  admin/job ID, dan stable error code.
- Token, password, TOTP, raw message body, dan sensitive PII harus direduksi.
- Global error handler memetakan domain error tanpa mengekspos internals.
- Upload maksimal 10 MB dan hanya JPEG/PNG/WebP yang lolos magic-byte serta
  decode validation; SVG tidak diterima.

### 11.11 Observability dan operations

- Metrics: request latency/error, cache hit, DB pool, queue depth, retry/dead
  jobs, resize duration, dan email result.
- Liveness dan readiness dipisahkan.
- Alert tersedia untuk elevated error, queue backlog, dead jobs, database
  exhaustion, dan revalidation failure.
- Runbook mencakup job replay, cache purge, signing-key rotation, TOTP recovery,
  media retry, backup, dan restore.
- Database backup disimpan di lokasi yang terpisah dari primary service.

## 12. Phase 2 Readiness

**Status:** `OUT-OF-SCOPE` untuk implementasi saat ini.

Phase 1 tidak membangun tabel stok atau order lebih awal. Readiness dicapai
dengan:

- stable product identity dan SKU;
- module boundaries;
- explicit transaction boundary;
- versioned migration;
- idempotency;
- durable jobs/outbox;
- auditability;
- integer money type;
- optimistic/pessimistic locking strategy yang dapat ditambahkan pada bounded
  context inventory.

Phase 2 diperkirakan mencakup warehouse, stock ledger, reservation, stock
movement, order, transaction, race-condition handling, dan reconciliation.

## 13. Testing Strategy

### 13.1 Phase 1A quality gate

- ESLint.
- TypeScript type check.
- Unit test untuk schema, content source, helper metadata, dan form behavior.
- Production build.
- Playwright route/navigation/form test.
- Axe accessibility automation serta manual keyboard test.
- Lighthouse API dengan tiga synthetic run untuk setiap route representatif.
- Broken-link dan structured-data validation.
- Post-deploy Netlify Forms smoke test sebagai deferred launch gate.

### 13.2 Future backend test

- Unit test domain/use case dengan fake interface.
- Repository integration test dengan PostgreSQL dan Redis.
- API contract test terhadap OpenAPI.
- Auth, token rotation, TOTP, CSRF, CORS, rate limit, dan upload abuse test.
- Worker retry, duplicate delivery, idempotency, crash recovery, dan graceful
  shutdown test.
- Cache invalidation serta Redis failure test.
- Load test public API pada warm traffic; cold start diukur terpisah.

## 14. Acceptance Criteria Phase 1A

1. **Given** production environment URL valid, **when** build selesai, **then**
   canonical, sitemap, robots, structured data, dan Open Graph URL memakai
   origin tersebut.
2. **Given** JavaScript dinonaktifkan, **when** visitor membuka public route,
   **then** heading, copy utama, link, dan product information tetap tersedia.
3. **Given** product published, **when** build dijalankan, **then** katalog dan
   detail slug dibuat serta saling terhubung.
4. **Given** slug tidak dikenal, **when** route dibuka, **then** visitor mendapat
   not-found response dan jalur kembali.
5. **Given** input form valid, **when** local contract test mengintersep
   submission, **then** payload URL-encoded benar dan UI memberi status sukses.
   Visibilitas submission di dashboard Netlify menjadi acceptance tambahan
   setelah deployment diizinkan.
6. **Given** input tidak valid atau network gagal, **when** visitor submit,
   **then** error dapat dipahami, focus dapat diarahkan, dan input tidak hilang.
7. **Given** keyboard-only user, **when** menavigasi seluruh flow, **then**
   seluruh control dapat dicapai dan focus terlihat.
8. **Given** fakta berstatus belum terverifikasi, **when** content ditinjau,
   **then** fakta tersebut tidak disajikan sebagai klaim brand.

## 15. Risks dan Mitigasi

| Risiko | Dampak | Mitigasi |
| --- | --- | --- |
| Data brand belum lengkap | Copy salah dan local SEO lemah | Fact-check gate dan owner questionnaire |
| Foto/logo tanpa source file | Kualitas visual atau hak pakai bermasalah | Minta original asset dan usage approval |
| Subdomain gratis | Brand authority dan portability lebih lemah | Nama stabil; migrasi 301 saat domain tersedia |
| Ketergantungan free tier | Pause/quota/terms dapat berubah | Usage alert, static-first, review pricing berkala |
| Broad keyword sangat kompetitif | KPI ranking tidak tercapai cepat | Topic cluster, Search Console iteration, backlinks |
| Netlify form salah terdeteksi | Pesan tidak masuk | Static blueprint dan production smoke test |
| Marketplace dianggap sumber resmi | Harga/klaim salah | Jangan hardcode; gunakan CTA kontak |
| Future migration mengubah URL | Ranking dan backlink hilang | Stable slug dan parity test saat CMS migration |

## 16. Dependencies dari Pemilik Bisnis

Sebelum public launch final, pemilik bisnis perlu menyediakan atau menyetujui:

- logo resmi dan panduan penggunaan;
- foto produk dan foto usaha beserta hak penggunaan;
- nama badan/pelaku usaha;
- narasi sejarah yang disetujui;
- alamat, map pin, jam operasional, telepon, WhatsApp, dan email resmi;
- daftar produk, SKU, ukuran, harga, komposisi, allergen, dan penyimpanan;
- bukti sertifikat halal, SPP-IRT/BPOM, serta status merek bila hendak
  ditampilkan;
- privacy contact, pemilik operasional inbox, dan persetujuan retensi 12 bulan;
- akun Netlify dan Google Search Console.

Daftar status rinci berada di [FACT_CHECK.md](./FACT_CHECK.md).

## 17. Assumptions

- Bahasa Phase 1 hanya Indonesia; kontrak konten dibuat agar locale dapat
  ditambahkan kemudian.
- Phase 1A tidak memiliki database, API Go, Redis, login, atau CMS.
- Netlify Forms adalah transport kontak sementara dan dapat diganti contact
  endpoint Go mulai Phase 1B tanpa mengubah UI publik.
- Konten awal dapat diperbarui melalui repository oleh developer.
- Harga dan order online tidak termasuk Phase 1A.
- Availability 99,9% adalah aspirasi arsitektur, bukan SLA free-tier.
- Seluruh vendor future dapat diganti melalui adapter jika terms atau quota
  berubah.
