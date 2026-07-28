# Phase 1B Implementation Plan

## Tauco Cap Badak Backend Foundation

| Atribut | Nilai |
| --- | --- |
| Versi | 0.3 |
| Tanggal | 28 Juli 2026 |
| Status | Accepted; B0-B4 complete, B5 pending |
| Mode delivery | Local-first, shadow-mode |
| Production cutover | Tidak termasuk Phase 1B |
| Git push/deploy | Dilakukan pemilik repository, bukan bagian implementasi otomatis |
| Acuan utama | `PRD.md` versi 1.2 |

## 1. Keputusan Eksekutif

Phase 1B membangun fondasi backend yang lengkap dan dapat diuji tanpa
mengubah perilaku production Phase 1A.

Default yang direkomendasikan:

1. Next.js tetap berada di root repository dan tetap memakai
   `LocalContentSource` serta Netlify Forms di production.
2. Backend baru ditempatkan di `backend/`.
3. Backend memakai Go, Gin, GORM, PostgreSQL, Redis, object storage
   S3-compatible, dan durable job worker.
4. Seluruh public API berjalan dalam shadow-mode sampai cutover terpisah.
5. Tambahkan endpoint `GET /api/v1/tauco-guide` karena interface frontend
   sudah mewajibkan `getTaucoGuide()`.
6. Konten Phase 1A diimpor satu arah ke PostgreSQL melalui seed yang
   deterministic. Selama Phase 1B, JSON lokal tetap menjadi source of truth
   production.
7. Endpoint contact Go dibuat dan diuji, tetapi production tetap mengirim ke
   Netlify Forms sampai anti-spam, email sender, retensi, dan rollback
   tervalidasi pada cutover.
8. Tidak ada Admin CMS, login, JWT/PASETO, TOTP, RBAC runtime, atau admin CRUD
   pada Phase 1B.
9. Tidak ada deployment, push, atau perubahan provider tanpa instruksi
   eksplisit dari pemilik.

Pendekatan ini menjaga hasil Phase 1A:

- HTML utama tetap tersedia saat build dan tidak bergantung pada API;
- URL, canonical, sitemap, JSON-LD, serta Search Console tidak berubah;
- produk yang sudah terindeks tidak berubah slug;
- form production tidak mengalami dual-write atau email ganda;
- backend dapat dikembangkan dan diuji tanpa risiko regresi SEO.

## 2. Hasil Audit Sebelum Implementasi

### 2.1 Kondisi repository

- Branch lokal bersih dan sinkron dengan `origin/main`.
- Next.js tetap berada di root dan terhubung ke Netlify.
- `ContentSource` sudah asynchronous, tetapi page masih mengimpor
  `localContentSource` secara konkret.
- `ContactGateway` sudah tersedia, tetapi form masih mengimpor adapter
  Netlify secara konkret.
- Produk baru hanya dapat dibuat saat build karena detail page memakai static
  params dan `dynamicParams = false`.
- Image frontend hanya menerima aset yang tercatat dalam static image map.
- Docker, `psql`, migration CLI, dan linter Go belum tersedia lokal.
- Go `1.26.5` tersedia lokal.

### 2.2 Gap P0 pada PRD

| Gap | Dampak | Resolusi Phase 1B |
| --- | --- | --- |
| Requirement Phase 1B dan 1C tercampur pada PRD §11 | Scope dapat berkembang menjadi Admin CMS | Bekukan boundary pada §3 dokumen ini |
| Endpoint panduan tauco tidak ada | `ContentSource` tidak dapat dimigrasikan penuh | Tambah `GET /api/v1/tauco-guide` |
| Source data API belum ditetapkan | API kosong atau menjadi source of truth kedua | Importer satu arah dari JSON Phase 1A |
| Contact cutover ambigu | Pesan/email dapat terduplikasi | Endpoint dibuat shadow-only; Netlify tetap production |
| Worker scale-to-zero belum konkret | Job dapat tertinggal setelah HTTP response | PostgreSQL job table + request-bound wake-up + scheduled reconciliation |
| Retensi 12 bulan belum masuk model backend | Risiko privacy dan data tidak terhapus | `retention_delete_at` + purge job + PII-free job payload |
| Media worker ada, upload admin belum ada | Tidak ada trigger aman pada Phase 1B | Trigger internal CLI/test fixture; tidak ada public upload |
| Cursor dan problem response belum detail | Kontrak API tidak executable | Bekukan kontrak pada §7 |
| Storage target gratis tidak pasti | Risiko card/overage R2 | Supabase Storage sebagai pilot, tetap melalui S3 port |

### 2.3 Konsekuensi free-tier

Kode dapat dibuat enterprise-grade, tetapi kombinasi runtime gratis tidak
memberikan SLA enterprise.

- Supabase Free dapat pause karena inactivity, berukuran database 500 MB, dan
  tidak menyediakan backup harian yang dapat dipulihkan pemilik.
- Upstash Free memiliki quota command dan dapat diarsipkan setelah inactivity.
- Cloud Run mempunyai free allowance, tetapi membutuhkan billing account dan
  penggunaan di atas allowance atau egress tertentu dapat ditagih.
- Cloudflare R2 mempunyai free allowance besar, tetapi merupakan layanan
  usage-based dan aktivasi dapat membutuhkan payment method.
- Netlify Legacy Free mempunyai hard quota dan mendukung Go Functions serta
  Scheduled Functions, tetapi scheduled execution maksimal 30 detik,
  background functions tidak tersedia, dan observability terbatas.
- Provider email gratis tetap membutuhkan domain sendiri untuk pengiriman
  production ke recipient umum.

Karena itu, Phase 1B default adalah local-first. Deployment diperlakukan
sebagai gate terpisah setelah implementasi dan pengujian.

## 3. Scope Freeze

### 3.1 In scope Phase 1B

- Clean Architecture backend Go.
- Public REST API dan OpenAPI contract.
- PostgreSQL schema dan versioned SQL migrations.
- Importer/seed deterministic dari konten Phase 1A.
- Published-only public content dan catalog read.
- Contact persistence, consent metadata, idempotency, dan retention.
- Durable background job engine.
- Media processing worker melalui internal seam.
- Redis cache-aside, request coalescing, dan rate limiting.
- Object storage port dan S3-compatible adapter.
- Email port, local adapter, dan fake adapter.
- Request ID, RFC 7807 errors, structured logging, CORS, security middleware.
- Liveness, readiness, metrics, dan operational runbook skeleton.
- Unit, integration, contract, failure, race, security, dan load-test harness.
- `ApiContentSource` serta `ApiContactGateway` dalam shadow/test mode.
- CI workflow yang dapat dijalankan setelah pemilik push ke GitHub.

### 3.2 Deferred ke Phase 1C

- Admin login.
- JWT/PASETO, refresh session, cookie, CSRF, TOTP, dan RBAC enforcement.
- Admin content/product CRUD.
- Draft/publish UI dan workflow.
- Public admin upload endpoint.
- Inbox UI dan admin inbox API.
- Activity log viewer.
- Publish-triggered Next.js revalidation.

Schema dapat menyediakan extension point untuk Phase 1C, tetapi Phase 1B tidak
boleh membuat endpoint atau kredensial admin setengah jadi.

### 3.3 Deferred ke Phase 1D

- Production switch dari `LocalContentSource` ke `ApiContentSource`.
- Production switch dari Netlify Forms ke contact API.
- ISR/tag revalidation dan dynamic product publishing.
- Remote image support pada renderer production.
- Automated off-site backup dan restore drill.
- Active alert policy.
- Production load/security test.
- Final production API deployment/cutover.

### 3.4 Tetap out of scope

- Inventory.
- Warehouse.
- Stock ledger dan reservation.
- Order.
- Checkout.
- Payment.
- Customer account.

Tidak ada tabel stok atau order yang dibuat lebih awal. Product identity hanya
disiapkan agar tabel Phase 2 kelak dapat mereferensikannya.

## 4. Target Architecture

```text
Phase 1A production, tidak berubah

Crawler / visitor
       |
       v
Netlify + Next.js
  |-- LocalContentSource
  `-- Netlify Forms


Phase 1B shadow backend

Client / contract test
       |
       v
Delivery: Gin HTTP
       |
       v
Application use cases
  |-- Content
  |-- Catalog
  |-- Contact
  |-- Media
  |-- Jobs
  `-- Audit
       |
       v
Domain entities + ports
       |
       +--------------+--------------+----------------+
       |              |              |                |
       v              v              v                v
PostgreSQL         Redis       S3-compatible       Email
source of truth    optional    object storage      adapter
       |
       v
Durable background_jobs
       |
       v
Worker claim + bounded goroutine pool
```

Dependency rule:

```text
delivery/http -> application/usecase -> domain + ports
infrastructure adapters -------------> ports
composition root --------------------> seluruh concrete dependency
```

Larangan:

- domain tidak mengimpor Gin;
- domain tidak mengimpor GORM;
- use case tidak mengimpor vendor Redis, S3, atau email;
- GORM model tidak menjadi public API DTO;
- HTTP handler tidak berisi business rule;
- goroutine tidak boleh hidup tanpa durable job setelah HTTP response;
- production code tidak menggunakan `AutoMigrate`.

## 5. Struktur Repository

Next.js tidak dipindah agar konfigurasi Netlify Phase 1A tetap stabil.

```text
/
|-- src/                         # Next.js existing
|-- content/                     # source Phase 1A
|-- public/
|-- tests/
|-- contracts/
|   `-- fixtures/                # parity fixture TS/OpenAPI/Go
|-- backend/
|   |-- go.mod
|   |-- go.sum
|   |-- .go-version
|   |-- .env.example
|   |-- cmd/
|   |   |-- api/
|   |   |   `-- main.go
|   |   |-- worker/
|   |   |   `-- main.go
|   |   |-- migrate/
|   |   |   `-- main.go
|   |   `-- seed/
|   |       `-- main.go
|   |-- internal/
|   |   |-- composition/
|   |   |-- platform/
|   |   |   |-- config/
|   |   |   |-- database/
|   |   |   |-- cache/
|   |   |   |-- storage/
|   |   |   |-- email/
|   |   |   |-- logging/
|   |   |   |-- telemetry/
|   |   |   `-- httpserver/
|   |   |-- content/
|   |   |   |-- domain/
|   |   |   |-- application/
|   |   |   |-- repository/
|   |   |   `-- delivery/
|   |   |-- catalog/
|   |   |-- contact/
|   |   |-- media/
|   |   |-- jobs/
|   |   `-- audit/
|   |-- migrations/
|   |-- openapi/
|   |   |-- openapi.yaml
|   |   `-- oapi-codegen.yaml
|   |-- testdata/
|   |-- Dockerfile
|   `-- compose.yaml
|-- docs/
|   |-- adr/
|   |-- runbooks/
|   `-- phase-1b/
|-- PHASE_1B_PLAN.md
`-- PHASE_1B_WALKTHROUGH.md
```

## 6. Toolchain dan Dependency Policy

### 6.1 Go

- Local toolchain: Go `1.26.5`.
- Module compatibility: `go 1.25.0`.
- Toolchain directive: `toolchain go1.26.5`.
- `.go-version` dikunci ke patch version yang sama.
- Semua dependency dikunci melalui `go.mod` dan `go.sum`.
- Tidak menggunakan floating `latest` pada CI atau container.

### 6.2 Library baseline

| Concern | Pilihan awal | Alasan |
| --- | --- | --- |
| HTTP | Gin | Requirement PRD |
| ORM | GORM + PostgreSQL driver | Requirement PRD; hanya repository |
| SQL driver | pgx melalui GORM driver | Pooling dan PostgreSQL support |
| Migration | golang-migrate | Versioned up/down SQL |
| Logging | Zap | Structured logging dan performa |
| Redis | go-redis v9 | TLS, context, pooling |
| Object storage | AWS SDK for Go v2 S3 | Vendor-neutral S3 subset |
| OpenAPI | oapi-codegen | Contract-first delivery types |
| Metrics | Prometheus client | Vendor-neutral metric endpoint |
| UUID | google/uuid UUIDv7 | Stable app-generated IDs |
| Coalescing | `x/sync/singleflight` | Mencegah local cache stampede |
| Image resize | `x/image/draw` | Portable dan CGO-free |
| WebP | gen2brain/webp | CGO-free fallback untuk serverless/container |

Image processor tetap berada di balik interface. Bila benchmark menunjukkan
memory atau throughput tidak memenuhi gate, adapter container berbasis
libvips dapat ditambahkan tanpa mengubah use case.

## 7. REST API Contract

### 7.1 Route

| Method | Route | Scope |
| --- | --- | --- |
| `GET` | `/api/v1/home` | Published homepage |
| `GET` | `/api/v1/about` | Published about |
| `GET` | `/api/v1/tauco-guide` | Published tauco guide |
| `GET` | `/api/v1/products` | Published product catalog |
| `GET` | `/api/v1/products/{slug}` | Published product detail |
| `POST` | `/api/v1/contact-messages` | Contact intake |
| `GET` | `/health/live` | Process liveness |
| `GET` | `/health/ready` | Dependency readiness |
| `GET` | `/internal/metrics` | Protected metrics |

Tidak ada route `/api/v1/admin/*` pada Phase 1B.

### 7.2 Success envelope

```json
{
  "data": {},
  "meta": {
    "requestId": "01...",
    "apiVersion": "v1"
  }
}
```

List response menambahkan:

```json
{
  "meta": {
    "requestId": "01...",
    "apiVersion": "v1",
    "page": {
      "nextCursor": "opaque-or-null",
      "hasMore": false,
      "limit": 20
    }
  }
}
```

### 7.3 Error contract

Media type: `application/problem+json`.

```json
{
  "type": "urn:tauco-cap-badak:problem:validation",
  "title": "Permintaan tidak valid",
  "status": 422,
  "detail": "Periksa field yang ditandai.",
  "instance": "/api/v1/contact-messages",
  "code": "CONTACT_VALIDATION_FAILED",
  "requestId": "01...",
  "errors": [
    {
      "field": "email",
      "code": "INVALID_EMAIL",
      "message": "Masukkan alamat email yang valid."
    }
  ]
}
```

Status:

- `400` untuk malformed JSON, header, atau cursor;
- `404` untuk slug/route resource yang tidak ditemukan;
- `409` untuk idempotency key yang dipakai dengan payload berbeda;
- `413` untuk body terlalu besar;
- `415` untuk media type tidak didukung;
- `422` untuk semantic field validation;
- `429` untuk rate limit dan selalu menyertakan `Retry-After`;
- `500` untuk unexpected error dengan detail generik;
- `503` untuk dependency kritis yang tidak siap.

Response selalu menyertakan header `X-Request-ID`.

### 7.4 Pagination

- Default `limit=20`.
- Maksimum `limit=50`.
- Sort catalog: `sort_order ASC, id ASC`.
- Cursor adalah base64url payload bertanda HMAC.
- Cursor mengikat version, filter/query hash, posisi sort, dan product ID.
- Cursor yang dimodifikasi menghasilkan `400 INVALID_CURSOR`.
- Unknown query parameter ditolak agar cache dimension stabil.

### 7.5 Cache validator

- Content response mempunyai ETag dari revision checksum.
- `If-None-Match` dapat menghasilkan `304`.
- Public cache header awal:
  `public, max-age=0, s-maxage=300, stale-while-revalidate=60`.
- No-store digunakan untuk contact dan health detail.

### 7.6 Contact contract

Request JSON:

```json
{
  "name": "Nama pengirim",
  "email": "pengirim@example.com",
  "phone": "",
  "subject": "Informasi produk",
  "message": "Pesan minimal dua puluh karakter.",
  "privacyConsent": true,
  "botField": ""
}
```

Rule harus parity dengan frontend:

- nama 2 sampai 100 karakter;
- email maksimal 160 karakter dan format valid;
- telepon opsional, 7 sampai 30 karakter;
- subject hanya enum yang sudah tersedia;
- pesan 20 sampai 2.000 karakter;
- privacy consent wajib `true`;
- honeypot wajib kosong;
- unknown JSON field ditolak;
- request body maksimal 32 KiB.

Header `Idempotency-Key` wajib:

- 16 sampai 128 printable ASCII characters;
- hanya hash HMAC yang disimpan;
- key sama + payload sama tidak membuat record/job baru;
- key sama + payload berbeda menghasilkan `409`;
- replay mengembalikan response semantik yang sama dan
  `Idempotency-Replayed: true`.

Response `201`:

```json
{
  "data": {
    "status": "received"
  },
  "meta": {
    "requestId": "01...",
    "apiVersion": "v1"
  }
}
```

Internal message ID tidak perlu diekspos ke publik pada Phase 1B.

## 8. Data Model

### 8.1 General rule

- Primary ID memakai UUIDv7.
- Semua timestamp memakai `timestamptz` dan disimpan UTC.
- Semua status memakai `CHECK` constraint atau lookup yang versioned.
- Foreign key eksplisit.
- Index dibuat berdasarkan query path yang diuji.
- Published revision immutable pada application layer dan dilindungi trigger
  database untuk update/delete tidak sah.
- Money menggunakan integer minor unit dan currency `IDR`.
- SKU nullable sampai data resmi tersedia, lalu unique melalui partial index.
- Tidak ada GORM `AutoMigrate`.

### 8.2 `pages`

```text
id
key                         unique: home | about | tauco-guide | products
published_revision_id      nullable FK
created_at
updated_at
```

### 8.3 `page_revisions`

```text
id
page_id                    FK
revision_number
status                     draft | published | archived
schema_version
content_json               JSONB
content_checksum
created_by                 nullable, extension Phase 1C
created_at
published_at               nullable
```

Constraint:

- unique `(page_id, revision_number)`;
- `pages.published_revision_id` hanya menunjuk revision dari page yang sama;
- revision published tidak dapat dimutasi.

### 8.4 `products`

```text
id
slug                       stable, lowercase, unique
sku                        nullable
sort_order
published_revision_id      nullable FK
first_published_at         nullable
created_at
updated_at
```

Slug bukan primary key. Product UUID menjadi stable identity untuk relasi
Phase 2.

### 8.5 `product_revisions`

```text
id
product_id                 FK
revision_number
status                     draft | published | archived
schema_version
content_json               JSONB
content_checksum
created_by                 nullable, extension Phase 1C
created_at
published_at               nullable
```

### 8.6 `media_assets`

```text
id
status                     processing | ready | failed
original_object_key        unique
original_mime_type
original_width
original_height
original_bytes
sha256
alt_text
decorative
last_error_code            nullable, redacted
created_at
updated_at
```

### 8.7 `media_variants`

```text
id
media_asset_id             FK
width
height
format                     webp
object_key                 unique
mime_type
bytes
sha256
created_at
```

Unique `(media_asset_id, width, format)`.

### 8.8 `contact_messages`

```text
id
idempotency_key_hash       unique
request_payload_hash
name
email
phone                      nullable
subject
message
status                     unread | read | archived
privacy_consent
privacy_notice_version
consent_at
retention_delete_at
created_at
updated_at
```

Honeypot dan raw IP tidak disimpan sebagai business data.

### 8.9 `background_jobs`

```text
id
kind
payload_json               hanya entity ID, tanpa PII
idempotency_key            unique
status                     pending | running | succeeded | retry | dead
priority
attempts
max_attempts
run_at
locked_at                  nullable
lock_owner                 nullable
lease_expires_at           nullable
last_error_code            nullable
last_error_message         nullable, sanitized
created_at
updated_at
completed_at               nullable
dead_at                    nullable
```

### 8.10 `activity_logs`

```text
id
event_type
entity_type
entity_id                  nullable
actor_type                 system | visitor | admin
actor_id                   nullable
metadata_json              allowlist dan tanpa PII
request_id                 nullable
created_at
```

Log bersifat append-only.

## 9. Content Import dan Shadow Parity

Importer membaca:

- `content/home.json`;
- `content/about.json`;
- `content/tauco-guide.json`;
- `content/products.json`.

Rule:

1. File tetap divalidasi oleh Zod pada sisi frontend.
2. Importer Go memvalidasi bentuk yang sama melalui generated contract dan
   domain validation.
3. Seed memakai stable preassigned UUID dan revision number.
4. Checksum canonical JSON disimpan pada revision.
5. Import bersifat idempotent.
6. Seed tidak mengubah published revision bila checksum sama.
7. Konflik slug atau fakta terlarang menghentikan seed.
8. Hanya produk `published` yang dapat menjadi published revision.
9. Golden fixture membandingkan hasil:
   local JSON -> LocalContentSource -> API DTO -> Zod parse.
10. Production Next.js tidak membaca database pada Phase 1B.

## 10. Redis Design

### 10.1 Cache-aside

Cache digunakan untuk:

- homepage;
- about;
- tauco guide;
- catalog dimension;
- product detail per slug.

Contoh key:

```text
tcb:v1:tag:home:generation
tcb:v1:content:home:id-ID:g{n}
tcb:v1:tag:products:generation
tcb:v1:catalog:id-ID:{queryHash}:g{n}
tcb:v1:product:{slug}:id-ID:g{n}
```

Policy:

- TTL dasar 300 detik;
- jitter acak ±30 detik;
- schema/version berada dalam key;
- JSON payload memiliki checksum;
- local `singleflight` menggabungkan concurrent miss;
- Redis timeout ketat;
- public read fail-open ke PostgreSQL;
- corrupt cache dihapus lalu diambil ulang;
- negative cache unknown slug maksimum 30 detik;
- invalidation menaikkan tag generation;
- stale key dibiarkan expire agar invalidation tidak memerlukan scan.

Publish belum ada pada Phase 1B. Invalidation port dan test tetap dibuat agar
Phase 1C dapat memanggilnya.

### 10.2 Rate limit

| Endpoint | Distributed limit |
| --- | --- |
| Public GET | 60 request/menit/IP |
| Contact | 5 request/jam/IP |

Rule:

- key menggunakan HMAC dari normalized client IP;
- IPv6 dinormalisasi;
- trusted proxy ditetapkan eksplisit;
- Redis Lua script melakukan atomic counter/window;
- response `429` mempunyai `Retry-After`;
- bila Redis gagal, public GET memakai local limiter dan fail-open;
- contact memakai local limiter yang lebih ketat saat degraded;
- metric mencatat fallback tanpa mencatat IP mentah.

Upstash command quota harus dimetrikkan karena quota dihitung per command, bukan
per HTTP request.

## 11. Durable Job dan Concurrency

### 11.1 Transactional enqueue

Contact flow:

```text
BEGIN
  insert contact message
  insert email notification job
  insert activity log job
COMMIT
optional request-bound wake-up
return 201
```

Commit message dan job adalah acceptance boundary. Email failure tidak pernah
membatalkan atau menghapus contact message.

Media flow Phase 1B:

```text
internal CLI/test fixture
  -> validate source
  -> store sanitized full-resolution original
  -> insert media processing record + job
  -> worker processes variants
```

Tidak ada public/admin upload endpoint sebelum Phase 1C.

### 11.2 Claim

Worker mengklaim job dengan transaksi dan
`SELECT ... FOR UPDATE SKIP LOCKED`.

Default awal yang harus divalidasi test:

- batch claim: 10;
- worker goroutine: 2;
- channel capacity: 20;
- lease: 120 detik;
- heartbeat: 30 detik;
- shutdown grace: 10 detik;
- max attempts: 8;
- initial backoff: 30 detik;
- exponential factor: 2;
- max backoff: 30 menit;
- jitter: ±20 persen.

### 11.3 Delivery semantics

- At-least-once delivery diterima.
- Handler wajib idempotent.
- Object key variant deterministic.
- Activity event mempunyai unique idempotency key.
- Email adapter menerima idempotency reference bila provider mendukung.
- Lease expired dapat direclaim.
- Worker yang shutdown berhenti claim job baru.
- Running job diselesaikan atau lease dilepas sebelum exit.
- Setelah max attempts, job menjadi `dead`.
- Replay membuat transition tercatat dan tidak menghapus history.

### 11.4 Scale-to-zero wake-up

PostgreSQL tetap source of truth job.

Profile container:

1. Setelah commit, API mencoba membuat Cloud Task dalam request yang sama.
2. Payload Cloud Task hanya `jobId`.
3. Cloud Task memanggil private worker endpoint menggunakan OIDC.
4. Scheduled reconciler mencari pending job yang belum dibangunkan.
5. Bila wake-up provider gagal, job tetap aman di PostgreSQL.

Profile Netlify Legacy Free:

1. Scheduled Go Function tersedia pada seluruh plan.
2. Function mengklaim batch job dari PostgreSQL.
3. Execution maksimal 30 detik.
4. Job panjang harus dilepas/retry sebelum timeout.
5. Profile ini hanya pilot, bukan runtime production enterprise.

Worker core tidak boleh bergantung pada Cloud Tasks atau Netlify.

## 12. Media Pipeline

### 12.1 Validation

- Maksimum upload: 10 MB.
- Input yang diterima: JPEG, PNG, WebP.
- SVG, GIF, PDF, dan format lain ditolak.
- Extension tidak dipercaya.
- MIME header tidak dipercaya.
- Magic byte dan actual decode harus cocok.
- Decode config dijalankan sebelum full decode.
- Maksimum decoded pixels awal: 40 megapixel.
- Maksimum sisi awal: 12.000 pixel.
- Animated WebP ditolak pada Phase 1B.
- Corrupt/truncated image ditolak.

### 12.2 Normalization

- Terapkan EXIF orientation.
- Re-encode full-resolution image agar metadata EXIF/GPS tidak ikut disajikan.
- Raw upload hanya berada di memory/temp bounded selama processing awal dan
  tidak dipertahankan sebagai public object.
- "Original" dalam model berarti normalized full-resolution original.
- Temp file memakai directory khusus, permission minimum, dan selalu dibersihkan.

### 12.3 Variants

- Target width: 320, 640, 1280.
- Aspect ratio dipertahankan.
- Image tidak di-upscale.
- Width yang lebih besar dari source dilewati dan dicatat pada metadata.
- Output WebP.
- Default quality awal ditetapkan melalui benchmark, bukan asumsi.
- Object key immutable dan content-addressed.
- Upload variant memakai conditional/idempotent put.
- Variant baru selesai seluruhnya sebelum media menjadi `ready`.
- Partial failure menghasilkan retry tanpa menggandakan object.

### 12.4 Bucket policy

- Normalized original private.
- Variant dapat dibuat public-read saat cutover.
- Phase 1B shadow tidak mengubah image production.
- Object memakai explicit `Content-Type` dan immutable cache header.
- Delete tidak dilakukan langsung dari public request.
- Orphan cleanup dilakukan job terpisah dan memakai grace period.

## 13. Security dan Privacy

### 13.1 HTTP

- Exact CORS allowlist.
- Production public access dirancang same-origin melalui proxy/BFF.
- Local allowlist hanya origin development yang eksplisit.
- Tidak menggunakan `*` bersama credential.
- Request timeout, header timeout, idle timeout, dan graceful shutdown.
- Maximum body size per route.
- JSON decoder menolak unknown field.
- Security header relevan diterapkan.
- Method yang tidak didukung menghasilkan problem response.

### 13.2 Trusted client IP

- Gin tidak boleh mempercayai seluruh proxy.
- Trusted proxy CIDR/config ditentukan per runtime.
- Resolver platform-specific berada di delivery/infrastructure.
- `X-Forwarded-For` hanya dipakai dari proxy tepercaya.
- Raw IP tidak ditulis ke application log.

### 13.3 Secret

- Tidak ada secret di repository.
- Backend env tidak memakai prefix `NEXT_PUBLIC_`.
- Config gagal start bila secret wajib kosong.
- Secret mempunyai minimum length dan tidak boleh memakai placeholder.
- Rotation dapat dilakukan tanpa mengubah domain/use case.
- `.env.example` hanya memuat nama dan contoh non-secret.

### 13.4 Logging redaction

Log tidak boleh memuat:

- password;
- token;
- cookie;
- authorization header;
- API key;
- raw email;
- raw phone;
- contact message;
- TOTP secret;
- raw IP;
- storage credential;
- database URL.

Field log minimum:

```text
timestamp
level
service
environment
request_id atau job_id
route atau job_kind
status
latency_ms
stable_error_code
```

### 13.5 Contact privacy

- `consent_at` dan `privacy_notice_version` wajib.
- `retention_delete_at = consent_at + 12 bulan`.
- Retention job berjalan harian.
- Setelah batas retensi, message dan direct PII dihapus.
- Job payload hanya menyimpan message ID.
- Activity log tidak menyimpan name/email/phone/message.
- Backup policy Phase 1D harus mempunyai batas retensi terpisah.

### 13.6 Spam strategy

Phase 1B:

- honeypot;
- rate limit;
- payload/body limit;
- idempotency;
- origin check bila header tersedia;
- local/contract tests.

Sebelum contact cutover:

- evaluasi Cloudflare Turnstile atau kontrol anti-abuse ekuivalen;
- review privacy notice dan CSP;
- validasi server-side token;
- sediakan accessibility fallback;
- jangan mematikan Netlify Forms sampai gate lulus.

## 14. Database Connection dan Supabase Profile

Local:

- PostgreSQL container;
- runtime role dan migration role terpisah;
- integration test memakai database disposable.

Supabase pilot:

- region Singapore;
- runtime melalui Supavisor transaction mode port `6543`;
- prepared statement dimatikan;
- GORM `PrepareStmt: false`;
- pgx `PreferSimpleProtocol: true`;
- `MaxOpenConns=5`;
- `MaxIdleConns=2`;
- connection lifetime dan idle time dibatasi;
- Cloud Run max instance awal 3, sehingga target maksimum sekitar 15 connection;
- migration memakai session pooler/direct connection terpisah;
- TLS certificate verification wajib.

Database outage adalah readiness failure. Redis outage bukan readiness failure
untuk public read karena mempunyai fail-open path.

## 15. Storage dan Email Profile

### 15.1 Local

- PostgreSQL dan Redis melalui Docker Compose.
- Mailpit sebagai SMTP capture.
- Object storage memakai S3-compatible local test service atau contract fake
  yang dibekukan melalui ADR sebelum B7.
- Tidak ada credential cloud untuk unit test.

Docker Desktop sudah dipasang setelah audit. Fitur WSL dan Virtual Machine
Platform juga sudah diaktifkan, tetapi Windows masih harus direstart sebelum
daemon dan integration suite lokal dapat diverifikasi. B2 tidak memakai
container; Docker mulai menjadi gate wajib pada B3.

### 15.2 Remote pilot

Default tanpa R2:

- Supabase PostgreSQL Free;
- Supabase Storage Free melalui S3-compatible adapter;
- Upstash Redis Free;
- email backend tetap disabled/recording sampai sender domain tersedia;
- Netlify Forms tetap mengirim notification production.

Supabase Storage dipilih untuk pilot karena:

- tidak membutuhkan vendor tambahan;
- menyediakan 1 GB Free storage;
- S3-compatible subset cukup untuk object put/get/head/delete;
- aplikasi tetap membatasi file 10 MB.

Keterbatasan:

- tidak ada object versioning;
- delete permanen;
- image transform tidak tersedia pada Free;
- quota egress organisasi terbatas.

### 15.3 Upgrade path

Cloudflare R2 dapat menggantikan storage adapter ketika:

- pemilik menerima billing-enabled subscription;
- custom domain tersedia;
- bucket policy, CORS, lifecycle, dan backup disetujui.

Resend dapat diaktifkan ketika:

- domain pengirim resmi tersedia;
- SPF/DKIM terverifikasi;
- recipient dan privacy processor disetujui;
- end-to-end email test lulus.

## 16. Observability

### 16.1 Health

`/health/live`:

- hanya membuktikan process dapat merespons;
- tidak melakukan network call;
- tidak memuat detail secret/dependency.

`/health/ready` API:

- PostgreSQL wajib sehat;
- Redis dapat berstatus degraded;
- storage dapat degraded untuk read/contact API;
- response detail aman dan tidak memuat endpoint/credential.

Worker readiness:

- PostgreSQL wajib;
- storage wajib untuk media job;
- email dependency hanya wajib ketika email adapter production aktif.

### 16.2 Metrics

Minimum:

- request count/latency/status per route template;
- in-flight request;
- problem code count;
- DB pool open/idle/wait;
- cache hit/miss/error;
- Redis command/fallback;
- rate-limit allowed/blocked/fallback;
- queue depth per status/kind;
- job duration/retry/dead/replay;
- lease reclaim;
- image decode/resize duration dan failure;
- email attempt/success/failure;
- retention delete count.

Tidak ada label email, slug bebas, request ID, IP, atau value high-cardinality.

### 16.3 Trace dan log

- W3C `traceparent` diterima/dipropagasikan.
- Request ID tetap tersedia walau tracing provider tidak aktif.
- Zap menulis JSON stdout.
- Local memakai human-readable development encoder.
- Unexpected panic direcover, dicatat tanpa PII, lalu dipetakan ke RFC 7807.

## 17. Testing Strategy

### 17.1 Unit

- domain invariant;
- use case dengan fake port;
- validation;
- cursor encode/decode/tamper;
- error mapping;
- config validation;
- retry/backoff;
- media state transition;
- retention calculation;
- log redaction.

### 17.2 Integration

- migration dari database kosong;
- migration up/down yang didukung;
- repository PostgreSQL;
- JSONB/revision immutability;
- `SKIP LOCKED` dengan dua worker;
- Redis cache hit/miss/fail-open;
- Redis rate-limit atomicity;
- object storage contract;
- email capture melalui Mailpit.

SQLite tidak digunakan sebagai pengganti PostgreSQL.

### 17.3 Contract

- OpenAPI lint/validate;
- generated server type tidak drift;
- response fixture parse di Zod frontend;
- Go validation parity dengan TypeScript;
- RFC 7807 schema;
- unknown slug `404`;
- ETag/304;
- pagination/cursor.

### 17.4 Contact

- valid request;
- malformed JSON;
- unknown field;
- every frontend field boundary;
- honeypot;
- idempotent replay;
- idempotency conflict;
- atomic message + jobs;
- email failure;
- rate limit;
- log redaction;
- retention purge.

### 17.5 Worker

- two-worker no double claim;
- crash after claim;
- lease expiry/reclaim;
- duplicate delivery;
- retry/backoff;
- max attempts/dead-letter;
- manual replay;
- graceful SIGTERM;
- channel bound;
- context cancellation;
- poisoned job isolation.

### 17.6 Media abuse

- MIME spoof;
- extension spoof;
- corrupt/truncated file;
- oversized byte payload;
- excessive decoded dimensions;
- animated input;
- SVG;
- no-upscale;
- EXIF orientation;
- EXIF/GPS stripping;
- deterministic variant key;
- partial upload retry;
- cleanup temporary file.

### 17.7 Quality command

Target commands:

```powershell
npm.cmd run check
npm.cmd run test:e2e

Set-Location backend
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
govulncheck ./...
```

Tambahkan root wrapper agar owner tidak perlu mengingat seluruh command.

### 17.8 Load

- Warm API read test terpisah dari cold-start.
- Contact tidak di-load-test ke email provider nyata.
- Verify p95/p99, DB pool wait, Redis command, queue depth, dan memory.
- Initial local target, bukan SLA:
  - public cached p95 < 100 ms;
  - public DB fallback p95 < 300 ms;
  - contact transaction p95 < 500 ms;
  - error rate < 1 persen pada agreed local load.

Target akan dikalibrasi ulang berdasarkan hardware dan remote pilot.

## 18. CI Gates

Workflow backend setelah dipush pemilik:

1. checkout;
2. setup exact Go;
3. restore module/build cache;
4. OpenAPI validation;
5. code generation drift check;
6. format check;
7. unit test;
8. PostgreSQL integration dan Redis integration pada gate masing-masing;
9. race test;
10. vet/lint;
11. vulnerability scan;
12. migration test;
13. container build;
14. frontend parity fixture test;
15. existing Phase 1A `npm.cmd run check`.

Repository saat ini public sehingga GitHub Actions dapat menjadi fallback
integration environment. Docker Desktop lokal sudah terpasang, tetapi daemon
masih menunggu restart Windows dan verifikasi. Workflow tidak dapat dibuktikan
sampai pemilik melakukan push.

## 19. Deployment Profiles

Deployment bukan bagian default implementasi Phase 1B.

### 19.1 Profile A: hard-capped pilot pada Netlify Legacy Free

- Go API sebagai synchronous Netlify Function.
- Go worker sebagai Scheduled Function.
- Same-origin route melalui Netlify rewrite.
- 125.000 serverless function invocation/site/bulan pada plan yang terdeteksi.
- Scheduled Function tersedia, tetapi maksimal 30 detik.
- Tidak ada Background Function pada Legacy Free.
- Tidak ada enterprise observability/SLA.

Kelebihan:

- akun dan hard quota sudah tersedia;
- tidak membutuhkan Google billing account;
- same-origin mengurangi CORS.

Kekurangan:

- worker panjang tidak cocok;
- runtime dan observability terbatas;
- API/backend deployment terikat pada site deploy;
- bukan target ideal untuk media processing production.

### 19.2 Profile B: scale-ready pilot pada Cloud Run

- API dan worker service terpisah.
- Request-based billing.
- `min-instances=0`, `max-instances=3`.
- Region Singapore.
- Cloud Tasks OIDC untuk wake-up.
- Cloud Scheduler untuk reconciliation/retention.

Kelebihan:

- container portable;
- API/worker dapat diskalakan terpisah;
- timeout dan observability lebih sesuai.

Kekurangan:

- billing account wajib;
- free allowance bukan hard spending cap;
- Singapore/egress dapat menghasilkan biaya;
- budget alert tidak menghentikan layanan otomatis.

### 19.3 Keputusan deployment

Backend core harus mendukung kedua profile melalui adapter. Pilihan runtime
baru dibuat setelah local acceptance lulus. Tidak ada deployment otomatis.

## 20. Acceptance Criteria Phase 1B

Phase 1B baru dinyatakan selesai bila seluruh poin berikut mempunyai evidence.

1. Fresh PostgreSQL dapat dimigrasikan dari kosong tanpa `AutoMigrate`.
2. Domain/use case tidak mengimpor Gin, GORM, Redis, atau vendor SDK.
3. OpenAPI mencakup seluruh public route, health, schemas, examples, dan
   problem details.
4. Endpoint tauco guide tersedia dan parity dengan frontend.
5. Seed/importer idempotent dan parity dengan JSON Phase 1A.
6. Public endpoint hanya mengembalikan published revision.
7. Unknown product slug menghasilkan RFC 7807 `404`.
8. Product list deterministic, bounded, dan cursor tamper-resistant.
9. Contact membuat message + email job + activity job dalam satu transaction.
10. Idempotent replay tidak membuat message atau job ganda.
11. Payload invalid tidak menyimpan PII dan tidak menulis raw data ke log.
12. Redis hit menghindari DB read.
13. Redis outage tetap melayani public content dari PostgreSQL.
14. Concurrent cache miss terkoalesi.
15. Targeted invalidation hanya mengubah tag relevan.
16. Rate limit menghasilkan `429` dan `Retry-After`.
17. Trusted proxy/IP normalization diuji.
18. Dua worker tidak memproses job yang sama pada saat bersamaan.
19. Crash dan lease expiry dapat direcover.
20. Retry bounded; max attempt masuk dead-letter; replay tercatat.
21. Graceful shutdown selesai dalam grace period.
22. Media valid menghasilkan normalized original dan seluruh non-upscaled
    variant 320/640/1280 WebP.
23. Spoofed/corrupt/oversized/decompression-bomb image ditolak.
24. EXIF/GPS tidak terdapat pada normalized public output.
25. Media state transition valid dan retry idempotent.
26. Liveness tidak bergantung pada vendor.
27. Readiness gagal aman bila PostgreSQL tidak tersedia.
28. Metrics minimum tersedia tanpa high-cardinality/PII.
29. Retention job menghapus contact PII maksimal 12 bulan.
30. Unit, integration, race, contract, abuse, migration, lint, vet,
    vulnerability, dan container build lulus.
31. Existing frontend checks dan Phase 1A smoke test tetap lulus.
32. Production website masih memakai local content dan Netlify Forms.
33. Tidak ada deployment, push, atau secret yang masuk repository.

## 21. Work Breakdown dan Walkthrough Gate

### B0: Scope freeze dan ADR

Output:

- dokumen ini disetujui;
- amendment PRD;
- ADR architecture boundary;
- ADR deployment profiles;
- ADR job wake-up;
- ADR image processor;
- ADR storage contract;
- walkthrough dibuat.

### B1: Backend skeleton

Output:

- Go module/toolchain;
- package boundary;
- config validation;
- Zap logger;
- Gin server;
- graceful shutdown;
- composition root;
- Dockerfile/compose;
- root command wrappers;
- initial unit test.

### B2: OpenAPI dan domain contract — Complete

Output:

- OpenAPI v1;
- generated types;
- request ID;
- success/problem envelope;
- cursor;
- safe generated handler dan response factory;
- canonical content string dan bounded escaped problem instance;
- frontend contract fixture;
- `/tauco-guide`.

### B3: Database dan seed — Complete

Output:

- versioned migrations;
- roles/connection policy;
- GORM repository only;
- revision immutability;
- deterministic importer;
- migration/repository/parity tests.

### B4: Public read API — Complete

Output:

- home/about/tauco guide;
- catalog/detail;
- published-only;
- 404;
- pagination;
- ETag/cache header;
- API E2E.

### B5: Contact transaction

Output:

- validation parity;
- idempotency;
- contact/message/job atomic transaction;
- privacy metadata;
- retention;
- test fake email;
- no production cutover.

### B6: Durable worker

Output:

- claim/lease/heartbeat;
- bounded pool;
- retry/backoff/dead-letter;
- replay;
- graceful shutdown;
- concurrency/crash tests.

### B7: Media pipeline

Output:

- object storage port/adapters;
- secure decode;
- normalization;
- WebP variants;
- metadata stripping;
- idempotent state machine;
- abuse tests.

### B8: Redis dan security middleware

Output:

- cache-aside;
- fail-open;
- singleflight;
- generation invalidation;
- rate limiter;
- CORS/trusted proxy/body limit;
- global error/recovery.

### B9: Observability dan operations

Output:

- health;
- metrics;
- trace propagation;
- queue/retention metrics;
- runbook skeleton;
- backup design.

### B10: Quality gate

Output:

- seluruh acceptance test;
- warm/cold load baseline;
- security review;
- dependency audit;
- frontend regression;
- documentation final.

### B11: Optional remote pilot

Hanya dilakukan setelah instruksi eksplisit pemilik.

Output:

- selected deployment profile;
- environment/account checklist;
- secret setup;
- migration;
- shadow smoke test;
- cost/quota verification;
- rollback;
- tidak melakukan frontend/contact production cutover.

## 22. Walkthrough Update Rule

`PHASE_1B_WALKTHROUGH.md` wajib diperbarui setiap satu gate selesai:

1. status dan timestamp;
2. file yang dibuat/diubah;
3. keputusan arsitektur;
4. command yang dijalankan;
5. hasil test;
6. evidence;
7. known limitation;
8. next gate.

Gate tidak boleh ditandai selesai hanya karena kode ditulis. Test yang relevan
harus lulus atau blocker harus dicatat eksplisit.

## 23. Prasyarat dan Keputusan Owner

Sebelum B1 dimulai, default berikut perlu diterima:

- Phase 1B local-first dan shadow-mode: **direkomendasikan Ya**.
- Tambah endpoint `/api/v1/tauco-guide`: **direkomendasikan Ya**.
- Website production tetap memakai local content: **direkomendasikan Ya**.
- Netlify Forms tetap production: **direkomendasikan Ya**.
- Tidak membuat Admin/Auth pada Phase 1B: **direkomendasikan Ya**.
- Gunakan Supabase Storage sebagai remote pilot sebelum R2:
  **direkomendasikan Ya**.
- Tidak deploy atau push oleh agent: **Ya**.

Prasyarat teknis:

- Docker Desktop atau compatible Docker engine untuk integration test lokal;
- cukup disk untuk image/container;
- port local database/cache/mail/storage tersedia;
- akun cloud belum dibutuhkan untuk B1 sampai B10;
- GitHub Actions baru berjalan setelah pemilik push.

B1 dan B2 tidak memerlukan Docker. B3 tidak boleh dinyatakan selesai hanya
dengan mock; PostgreSQL integration test nyata wajib. Redis adapter dan
integration test nyata menjadi gate B8.

## 24. Referensi Resmi yang Diverifikasi

- [Go release history](https://go.dev/doc/devel/release)
- [Go 1.26 release notes](https://go.dev/doc/go1.26)
- [Gin security guidance](https://gin-gonic.com/en/docs/middleware/security-guide/)
- [Gin trusted proxies](https://gin-gonic.com/en/docs/server-config/trusted-proxies/)
- [GORM PostgreSQL](https://gorm.io/docs/connecting_to_the_database.html)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Supabase pricing](https://supabase.com/pricing)
- [Supabase project pausing](https://supabase.com/docs/guides/platform/free-project-pausing)
- [Supabase connections](https://supabase.com/docs/guides/database/connecting-to-postgres)
- [Supabase backups](https://supabase.com/docs/guides/platform/backups)
- [Supabase Storage S3 compatibility](https://supabase.com/docs/guides/storage/s3/compatibility)
- [Supabase regions](https://supabase.com/docs/guides/platform/regions)
- [Upstash Redis pricing](https://upstash.com/pricing/redis)
- [Upstash inactivity behavior](https://upstash.com/docs/redis/help/faq)
- [Cloud Run pricing](https://cloud.google.com/run/pricing)
- [Cloud Run execution model](https://docs.cloud.google.com/run/docs/container-contract)
- [Cloud Tasks pricing](https://cloud.google.com/tasks/pricing)
- [Cloud Tasks authenticated HTTP targets](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks)
- [Cloud Scheduler pricing](https://cloud.google.com/scheduler/pricing)
- [Cloudflare R2 pricing](https://developers.cloudflare.com/r2/pricing/)
- [Cloudflare billing policy](https://developers.cloudflare.com/billing/understand/billing-policy/)
- [Resend pricing](https://resend.com/pricing)
- [Netlify Go Functions](https://docs.netlify.com/build/functions/lambda-compatibility/)
- [Netlify Scheduled Functions](https://docs.netlify.com/build/functions/scheduled-functions/)
- [Netlify Legacy limits](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-legacy-plans/legacy-pricing-plans/)
