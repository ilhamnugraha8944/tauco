# Phase 1B Walkthrough

## Tauco Cap Badak Backend Foundation

| Atribut | Nilai |
| --- | --- |
| Mulai | 28 Juli 2026 |
| Status keseluruhan | B0-B4 complete, B5 pending |
| Mode | Local-first, shadow-mode |
| Production Phase 1A | Tidak berubah |
| Dokumen rencana | `PHASE_1B_PLAN.md` |

## Cara Membaca

Walkthrough ini adalah progress ledger. Setiap gate hanya boleh berubah menjadi
`Complete` setelah output dan test gate tersebut mempunyai evidence.

Status:

- `Complete`: output dan acceptance gate lulus.
- `In progress`: sedang dikerjakan.
- `Blocked`: tidak dapat dilanjutkan tanpa dependency/keputusan.
- `Pending`: belum dimulai.
- `Deferred`: sengaja ditempatkan di phase lain.

## Ringkasan Progress

| Gate | Status | Hasil |
| --- | --- | --- |
| B0 Scope freeze dan ADR | Complete | PRD v1.2 dan lima ADR diterima |
| B1 Backend skeleton | Complete | Runtime Go, boundary, test, dan local dependency scaffold lulus |
| B2 OpenAPI dan contract | Complete | Executable contract, generated types, cursor, dan parity fixtures lulus |
| B3 Database dan seed | Complete | Migration, role, schema, repository, deterministic seed, dan PostgreSQL integration gate lokal selesai |
| B4 Public read API | Complete | Lima public read route, cursor, ETag/304, 404, dan PostgreSQL-to-HTTP integration lulus |
| B5 Contact transaction | Pending | Belum dimulai |
| B6 Durable worker | Pending | Belum dimulai |
| B7 Media pipeline | Pending | Belum dimulai |
| B8 Redis dan security | Pending | Belum dimulai |
| B9 Observability/operations | Pending | Belum dimulai |
| B10 Quality gate | Pending | Belum dimulai |
| B11 Optional remote pilot | Deferred | Membutuhkan instruksi owner |

## Update 1: Deep Analysis Sebelum Implementasi

**Tanggal:** 28 Juli 2026
**Status:** Complete

### Yang diperiksa

- `PRD.md` Phase 1B dan 1C.
- Interface frontend `ContentSource`.
- Interface frontend `ContactGateway`.
- Local content schema dan published filtering.
- Static product route dan sitemap behavior.
- Static image registry.
- Netlify production boundary.
- Ketersediaan Go/Docker/tool lokal.
- Current free-tier Supabase, Upstash, Cloud Run, R2, Resend, Render, Koyeb,
  dan Netlify Functions dari dokumentasi resmi.

### Temuan utama

1. PRD mencampur requirement Phase 1B dan Phase 1C.
2. API belum mempunyai endpoint untuk `getTaucoGuide()`.
3. Belum ada source data API sebelum CMS.
4. Contact Go dan Netlify Forms belum mempunyai cutover rule.
5. Durable job belum mempunyai scale-to-zero wake-up mechanism.
6. Retensi contact 12 bulan belum masuk schema backend.
7. Media worker belum mempunyai trigger karena admin upload baru Phase 1C.
8. Frontend belum dapat langsung menerima remote media URL.
9. Product publishing dinamis belum dapat bekerja karena
   `dynamicParams = false`.
10. Docker dan database tooling belum tersedia lokal.
11. Seluruh runtime gratis tidak dapat sekaligus memberikan SLA, backup, HA,
    dan hard zero-cost guarantee.

### Keputusan arsitektur yang direkomendasikan

- Backend dibuat shadow-mode.
- Production Next.js tetap memakai local content.
- Production contact tetap memakai Netlify Forms.
- Tambah `GET /api/v1/tauco-guide`.
- Gunakan importer satu arah dari JSON Phase 1A.
- PostgreSQL tetap source of truth job.
- Redis hanya cache/rate-limit dan selalu fallible.
- Job mempunyai durable lease/retry/dead-letter.
- Worker core tidak bergantung pada runtime provider.
- Supabase Storage menjadi remote pilot sebelum R2.
- Tidak ada Admin/Auth pada Phase 1B.
- Tidak ada deploy atau push tanpa instruksi owner.

### Evidence

- Rencana detail: `PHASE_1B_PLAN.md`.
- PRD utama: `PRD.md`.
- Repo berada pada root Next.js dan belum mempunyai folder `backend/`.
- Go lokal: `1.26.5`.
- Docker: command tidak tersedia pada saat audit.

### File yang dibuat

- `PHASE_1B_PLAN.md`.
- `PHASE_1B_WALKTHROUGH.md`.

### File runtime yang diubah

- Tidak ada.

### Test yang dijalankan

- Tidak ada test runtime karena tahap ini read-only analysis dan dokumentasi.

### Known blocker

- Scope freeze B0 belum disetujui.
- Docker dibutuhkan sebelum integration gate B3 dapat dinyatakan complete.

### Next

Setelah default B0 disetujui:

1. amend `PRD.md` untuk boundary Phase 1B;
2. buat ADR;
3. mulai B1 backend skeleton;
4. update walkthrough segera setelah B1 selesai dan test lulus.

## Update 2: B0 Scope Freeze dan ADR

**Tanggal:** 28 Juli 2026
**Status:** Complete

### Persetujuan owner

Owner menyetujui untuk melanjutkan B1 dengan default:

- local-first;
- shadow-mode;
- production Phase 1A tetap memakai local content dan Netlify Forms;
- endpoint tauco guide ditambahkan;
- Admin/Auth tetap Phase 1C;
- agent tidak melakukan push atau deploy;
- Docker Desktop akan dipasang oleh owner.

### Perubahan

- `PRD.md` dinaikkan ke versi 1.2.
- Status Phase 1B menjadi `IMPLEMENTING-1B`.
- Boundary Phase 1B, 1C, dan 1D dipisahkan eksplisit.
- `GET /api/v1/tauco-guide` masuk public API contract.
- Contact cutover dipindahkan ke Phase 1D.
- Remote pilot storage awal menjadi Supabase Storage dengan S3 port.

### ADR yang diterima

- `docs/adr/0001-phase-1b-shadow-mode.md`.
- `docs/adr/0002-runtime-deployment-profiles.md`.
- `docs/adr/0003-postgres-durable-jobs.md`.
- `docs/adr/0004-portable-image-processing.md`.
- `docs/adr/0005-s3-compatible-storage.md`.

### Validation

```text
git diff --check: lulus
PRD version/status/boundary check: lulus
ADR inventory check: 5 file ditemukan
```

Tidak ada runtime test pada B0 karena output B0 seluruhnya berupa contract dan
architecture decision.

### Next

B1 backend skeleton.

## Update 3: B1 Backend Skeleton

**Tanggal:** 28 Juli 2026
**Status:** Complete

### Output

- Go module dan toolchain dibuat di `backend/`.
- Boundary fitur `content`, `catalog`, `contact`, `media`, `jobs`, dan `audit`
  disiapkan dengan layer `domain` dan `application`.
- Dependency rule Clean Architecture dijaga oleh architecture test.
- Config loader env-only memvalidasi environment, bind address, port, timeout,
  shutdown grace, service name, log level, dan log format.
- Zap logger memakai field allowlist serta redaksi credential, token, email,
  telepon, IP, multiline, dan teks terlalu panjang.
- Gin runtime mempunyai request ID, liveness, RFC 7807 problem response dasar,
  recovery global, access log terstruktur, trusted proxy kosong, dan release
  mode permanen agar tidak ada log framework non-JSON di production.
- Composition root menghubungkan config, logger, middleware, router, HTTP
  server, signal handling, dan graceful shutdown.
- Multi-stage Dockerfile memakai runtime `scratch` non-root.
- Compose lokal menyediakan PostgreSQL, Redis, dan Mailpit opsional dengan
  pinned image, loopback port, healthcheck, volume, dan bounded log rotation.
- Root command wrapper menyediakan format, test, vet, build, dev, dan Compose.
- Runner test tetap menjalankan `go test ./...` terlebih dahulu serta mempunyai
  fallback terbatas untuk Windows Application Control.

### File utama

- `backend/go.mod`, `backend/go.sum`, dan `backend/.go-version`.
- `backend/cmd/api/main.go`.
- `backend/internal/composition/`.
- `backend/internal/platform/config/`.
- `backend/internal/platform/logging/`.
- `backend/internal/platform/httpserver/`.
- `backend/internal/platform/httpmiddleware/`.
- `backend/internal/architecture/dependencies_test.go`.
- `backend/internal/{content,catalog,contact,media,jobs,audit}/`.
- `backend/Dockerfile`, `backend/compose.yaml`, `backend/.env.example`,
  `backend/.dockerignore`, dan `backend/README.md`.
- `scripts/run-go-tests.mjs`.
- Root `package.json`, `.gitignore`, `eslint.config.mjs`, dan `README.md`.

### Hardening hasil audit silang

1. Gin dipaksa release mode di level kode, termasuk ketika container tidak
   mengatur `GIN_MODE`.
2. Panic log membawa request ID, route template, method, status, dan latency,
   tetapi tidak membawa nilai panic yang mungkin sensitif.
3. Access log memakai route template, bukan raw URL atau query.
4. Semua variasi `backend/.env*` di-ignore kecuali `.env.example`.
5. Architecture test menolak dependency domain/application/delivery yang
   mengarah ke outer layer.
6. Graceful shutdown menguji request aktif selesai sebelum proses berhenti dan
   koneksi dipaksa tutup ketika deadline terlampaui.
7. Local default hanya bind ke `127.0.0.1`; staging dan production mewajibkan
   network binding dan service name eksplisit.

### Command dan hasil

```text
npm.cmd run backend:format: lulus
go -C backend mod tidy: lulus
npm.cmd run backend:test: lulus
npm.cmd run backend:vet: lulus
npm.cmd run backend:build: lulus
npm.cmd run backend:check: lulus
npm.cmd run check:frontend: lulus
npm.cmd run check: lulus
production-mode API smoke /health/live: lulus
```

Evidence test:

- 37 top-level Go test mencakup config, redaction, logger adapter, request ID,
  error response, recovery, access log, dependency direction, composition,
  timeout, graceful drain, dan forced shutdown.
- Frontend Phase 1A tetap lulus: lint, typecheck, 73/73 unit test, dan Next.js
  production build.
- Smoke runtime production menghasilkan dua record JSON valid, tidak
  menghasilkan `[GIN-debug]`, dan `/health/live` mengembalikan `status=ok`.
- `backend:check` dan agregat root `check` berakhir dengan exit code `0`.

### Windows Application Control

Kebijakan Windows pada mesin lokal memblokir satu executable test yang dibuat
Go di temporary directory. Ini bukan assertion failure. Runner
`scripts/run-go-tests.mjs`:

1. selalu mencoba `go test ./...`;
2. hanya aktif pada Windows dan hanya jika pesan Application Control terdeteksi;
3. mengulang hanya package yang diblokir melalui test binary di workspace;
4. menghapus binary fallback setelah selesai;
5. tetap gagal untuk assertion, compile error, atau error lain.

Package yang terdampak telah dijalankan melalui fallback dan seluruh
assertion-nya lulus.

### Known limitation

- Docker Desktop 4.84.0, Docker CLI 29.6.2, dan Compose 5.3.1 sudah berhasil
  dipasang melalui package resmi `Docker.DockerDesktop`.
- Pemeriksaan awal menemukan WSL 2.6.1 terpasang, hypervisor aktif, RAM sekitar
  16 GB, serta ruang kosong drive sistem sekitar 228 GB.
- Initial engine startup gagal karena `Virtual Machine Platform not enabled`,
  sesuai log `com.docker.backend.exe.log`.
- Fitur `Microsoft-Windows-Subsystem-Linux` dan `VirtualMachinePlatform` sudah
  diaktifkan melalui DISM tanpa restart otomatis. DISM mengembalikan exit code
  `3010`, sehingga restart Windows masih wajib sebelum daemon, Compose config,
  image pull, healthcheck nyata, dan Dockerfile build dapat diverifikasi.
- Container build merupakan acceptance B10. PostgreSQL nyata mulai wajib pada
  B3; container Redis boleh di-smoke-test di B3, sedangkan adapter dan
  integration test Redis menjadi acceptance B8.
- Belum ada endpoint bisnis, database adapter, Redis runtime, worker, atau
  production cutover. Ini sesuai batas B1.
- Tidak ada push, deploy, perubahan Netlify, atau secret yang dibuat.

### Next

B2 OpenAPI dan domain contract. B3 tidak boleh dinyatakan complete sebelum
Docker Desktop aktif dan integration test PostgreSQL nyata lulus. Redis
adapter dan integration test nyata baru diwajibkan pada B8.

## Update 4: B2 OpenAPI dan Domain Contract

**Tanggal:** 28 Juli 2026
**Status:** Complete

### Boundary eksekusi

- B2 dikerjakan tanpa Docker karena belum menyentuh PostgreSQL, Redis, atau
  container runtime.
- Website production Phase 1A, Netlify Forms, canonical, dan sitemap tidak
  diubah.
- Tidak ada push, deploy, route admin, atau registrasi placeholder business
  handler.
- Restart Windows dan verifikasi Docker engine tetap wajib sebelum B3.

### Output kontrak

- OpenAPI 3.0.3 mendefinisikan tepat sembilan operasi:
  lima public read route, contact intake, liveness, readiness, dan protected
  metrics.
- `GET /api/v1/tauco-guide` tersedia sehingga seluruh method
  `ContentSource` mempunyai future API route.
- Response publik sukses memakai envelope versioned dengan `requestId`;
  response error memakai `application/problem+json`.
- Seluruh response mendefinisikan `X-Request-ID`.
- Public read contract mempunyai ETag, `304`, dan cache policy awal.
- Contact contract membatasi body 32 KiB, menolak field asing, mewajibkan
  `Idempotency-Key`, dan mencakup status `400`, `409`, `413`, `415`, `422`,
  `429`, `500`, serta `503`.
- Product list membatasi page size ke default 20 dan maksimum 50.
- Cursor membawa version, SHA-256 query hash, posisi `sort_order ASC, id ASC`,
  lalu ditandatangani HMAC-SHA256.
- Contact form input dinormalisasi lebih dahulu, sedangkan wire schema menolak
  leading/trailing whitespace agar payload tersimpan dalam satu bentuk
  canonical.

### Code generation dan parity

- `oapi-codegen` v2.8.0 dikunci sebagai Go tool dan runtime v1.6.0 dikunci
  sebagai dependency.
- Generated models, Gin server interface, strict server interface, dan embedded
  spec berada di `backend/internal/delivery/api/api.gen.go`.
- Handwritten factory pada package yang sama membangun generated metadata dan
  list response, lalu memvalidasinya melalui actual generated response visitor.
- Drift check meregenerasi file ke temporary directory lalu membandingkan
  output dengan file repository.
- Delapan fixture bersama mencakup lima content response, contact request,
  contact response, dan validation problem.
- Fixture content dibandingkan dengan output `LocalContentSource`.
- Fixture yang sama divalidasi oleh Zod, embedded OpenAPI schema, dan generated
  Go types.

### Security dan hardening

1. Cursor memakai unpadded base64url `payload.signature`, secret minimal 32
   byte, canonical JSON, strict unknown-field rejection, size bound, dan
   constant-time signature comparison.
2. Cursor yang dimodifikasi, memakai secret/filter berbeda, versi asing,
   payload noncanonical, atau posisi tidak valid selalu menghasilkan sentinel
   generik tanpa membocorkan input maupun secret.
3. OpenAPI validation juga memvalidasi seluruh committed example.
4. Contract test menolak route admin, operation ID kosong/duplikat, response
   tanpa request ID, error non-problem JSON, dan status wajib yang hilang.
5. Frontend contract menolak metadata, envelope, page, dan problem field yang
   tidak dikenal.
6. Zero-value query hash ditolak agar cursor tidak pernah lepas dari query
   binding.
7. Request ID hasil middleware ditulis kembali ke request header sehingga
   generated parameter binder, context, response header, dan logger melihat ID
   canonical yang sama.
8. Generated default error handler tidak dipakai. Safe strict/transport handler
   selalu mengirim problem detail generik, termasuk ketika implementation
   mengembalikan `(nil, nil)`, dan tidak mengekspos `err.Error()`; static guard
   menolak pemakaian default constructor/registrar pada seluruh production code
   backend, termasuk `cmd/api`.
9. Router menyediakan seam untuk menyerahkan ownership `/health/live` kepada
   generated delivery tanpa duplicate-route panic.
10. Tiga puluh lima content/problem string memakai pola canonical nonblank
    yang setara dengan Zod. Problem instance memakai escaped absolute path
    bounded yang sama pada platform writer dan generated safe writer.

### File utama

- `backend/openapi/openapi.yaml`.
- `backend/openapi/oapi-codegen.yaml`.
- `backend/internal/delivery/api/api.gen.go`.
- `backend/internal/delivery/api/contract_test.go`.
- `backend/internal/delivery/api/response_factory.go`.
- `backend/internal/delivery/api/safe_handler.go`.
- `backend/internal/catalog/application/pagination.go`.
- `backend/internal/catalog/delivery/cursor/hmac_sha256.go`.
- `backend/internal/contract/requestmeta/`.
- `backend/internal/platform/httpserver/response.go`.
- `contracts/fixtures/*.json`.
- `src/features/api-contract/`.
- `src/features/content/api-contract-schemas.ts`.
- `src/features/contact/api-contract-schemas.ts`.
- `tests/unit/api-contract.test.ts`.
- `scripts/check-openapi-generated.mjs`.

### Command dan hasil

```text
npm.cmd run backend:generate: lulus
npm.cmd run backend:generate:check: lulus; output reproducible
npm.cmd run backend:test: lulus untuk seluruh package
npm.cmd run backend:vet: lulus
npm.cmd run backend:build: lulus
npm.cmd run backend:check: lulus
go -C backend mod verify: lulus
npm.cmd run lint: lulus
npm.cmd run typecheck: lulus
tests/unit/api-contract.test.ts: 12/12 lulus
npm.cmd run check:frontend: lulus; 85/85 unit test dan production build
npm.cmd run check: lulus; frontend dan backend gate gabungan exit code 0
```

Evidence test:

- 65 top-level Go test function tersedia pada suite setelah B2.
- Enam Go contract test memvalidasi inventory operasi, requirement per operasi,
  OpenAPI examples, 35 canonical nonblank string contract, shared fixtures,
  generated model round-trip, problem writer, dan static guard untuk unsafe
  generated handler defaults pada seluruh source production backend.
- Empat pagination test dan delapan cursor test mencakup boundary, tamper,
  canonical encoding, query binding, secret ownership, dan non-leakage.
- Generated response factory dan actual visitor test mencakup metadata,
  pagination invariant, strong ETag, cache header, request ID, dan page bound.
- Safe generated handler test mencakup binder, handler, `(nil, nil)`, dan
  serialization error tanpa kebocoran detail internal.
- Problem writer test mencakup field error, `no-store`, bounded escaped
  instance, uppercase path, trailing slash, dan percent-encoded path.
- Clean Architecture boundary test lulus setelah coverage writer dipisahkan per
  layer tanpa dependency terbalik.

### Final audit closure

- Independent review menemukan dan menutup `(nil, nil)` strict-handler edge
  case serta memperluas static guard dari `internal` ke seluruh backend.
- OpenAPI dan Zod disamakan untuk generic image alt, canonical nonblank text,
  serta problem instance yang escaped dan bounded.
- Full regression pertama setelah hardening menangkap dependency test dari
  delivery ke platform. Coverage kemudian dipisahkan ke test masing-masing
  layer, targeted architecture test lulus, dan full `npm.cmd run check`
  berikutnya selesai dengan exit code `0`.

### Known limitation

- B2 adalah executable contract, bukan business endpoint runtime. Public read
  handler mulai B4 dan contact transaction mulai B5.
- Generated Gin binder belum otomatis menegakkan unknown query, Content-Type,
  strict unknown JSON field, body 32 KiB, dan security scheme. B4/B5/B8 wajib
  memasang middleware serta negative E2E sebelum route terkait diregistrasikan.
- Docker Desktop sudah terpasang dan Windows feature sudah aktif, tetapi restart
  masih diperlukan sebelum daemon dapat diverifikasi.
- Tidak ada cloud credential, database data, production cutover, push, atau
  deploy pada gate ini.

### Next

Restart Windows, verifikasi `docker version` serta `docker compose version`,
lalu mulai B3 migration, deterministic seed, repository, dan integration test
PostgreSQL.

## Update 5: B3 Database dan Seed

**Tanggal:** 28 Juli 2026
**Status:** Complete

### Milestone B3.0: local dependency readiness

- Docker Desktop engine aktif setelah WSL selesai diperbarui.
- Docker Client dan Server `29.6.2` dapat diakses.
- Docker Compose `v5.3.1` dapat diakses.
- PostgreSQL `17.10-alpine3.22` berstatus `healthy`.
- Redis `8.8.0-alpine3.23` berstatus `healthy`.
- `pg_isready` menerima koneksi dan Redis menjawab `PONG`.
- Database awal belum memiliki application table sehingga migration test dapat
  dimulai dari state kosong.

### Keputusan B3 yang dibekukan

1. Application table ditempatkan di schema privat `tauco_app`, bukan `public`,
   untuk menghindari eksposur tidak sengaja melalui PostgREST/Supabase.
2. Login migrasi, authorization role migrator, authorization role runtime, dan
   login runtime dipisahkan. Runtime tidak memiliki hak DDL atau hak tulis ke
   published revision.
3. Migration memakai versioned SQL. GORM hanya dipakai oleh connection dan
   repository adapter; `AutoMigrate` dilarang.
4. Published revision immutable dan current pointer wajib menunjuk revision
   berstatus published milik entity yang sama.
5. Seed Phase 1A memakai UUID tetap, canonical JSON, checksum SHA-256, dan satu
   transaksi idempotent. Replay identik tidak menambah atau mengubah row.
6. Catalog shell dari `content/products.json` disimpan sebagai singleton page
   key `products`; setiap published product tetap disimpan sebagai product
   revision terpisah. Koreksi ini diperlukan agar B4 dapat menghasilkan catalog
   yang parity dengan `LocalContentSource`.
7. Product list memakai keyset `(sort_order, id)`, melakukan probe satu row
   tambahan, dan mengembalikan `hasMore` tanpa count query.

### Milestone B3.1: deterministic importer

**Status:** Complete

- Manifest menetapkan UUIDv7 stabil untuk empat singleton page dan satu
  product awal.
- Seluruh source dibaca dengan batas ukuran, strict unknown-field rejection,
  duplicate-key rejection, dan path traversal protection.
- Generated Go type dan embedded OpenAPI memvalidasi bentuk content yang sama
  dengan frontend.
- Product source wajib berstatus `published`; field source `status` tidak ikut
  tersimpan di JSON public product detail.
- Canonical JSON v1 menolak invalid UTF-8, whitespace noncanonical, float,
  trailing value, dan checksum drift.
- Reconciliation membedakan insert, replay no-op, serta conflict untuk UUID
  ownership, revision, checksum, published pointer, dan product `sort_order`.
- Golden parity terhadap response fixture Phase 1A lulus.

Evidence:

```text
go test ./internal/content/domain ./internal/content/application ./internal/content/importer: lulus
go vet ./internal/content/domain ./internal/content/application ./internal/content/importer: lulus
15 top-level focused test beserta subtest: lulus
git diff --check: lulus
```

### Milestone B3.2: migration, schema, dan database role

**Status:** Complete

- Tiga versioned SQL migration membentuk sembilan application table di schema
  privat `tauco_app`; tidak ada application table di schema `public`.
- Authorization role `tauco_migrator` dan `tauco_runtime` tidak dapat login
  langsung. Concrete login role lokal dipisahkan dari ownership dan privilege.
- Runtime tidak mempunyai hak DDL, temporary table, atau hak mengubah published
  revision.
- Ownership `schema_migrations` tetap stabil saat credential migrator diputar.
- Primary key memakai UUIDv7, public payload memakai JSONB canonical, dan waktu
  disimpan sebagai `timestamptz` UTC.
- Foreign key, check constraint, index, dan integrity trigger diterapkan untuk
  page, product, media, contact, job, serta activity.
- Published revision bersifat immutable dan activity log bersifat append-only.
- Retensi contact tidak boleh melewati 12 bulan sejak consent diberikan.
- Durable job mempunyai batas percobaan default delapan kali.

Migration dari database kosong, rollback dan re-apply, rotasi login migrator,
serta privilege matrix telah diverifikasi pada disposable PostgreSQL nyata.

### Milestone B3.3: GORM repository dan atomic seed

**Status:** Complete

- GORM digunakan untuk connection dan repository tanpa `AutoMigrate`.
- Koneksi mendukung PostgreSQL pooler melalui simple protocol serta memakai
  batas connection pool eksplisit.
- Seed berjalan dalam satu transaksi, memakai advisory lock, role eksplisit,
  UUID global stabil, natural key, revision, checksum, dan snapshot canonical.
- Replay source identik menjadi no-op. Conflict ownership/checksum/pointer
  membatalkan seluruh transaksi.
- Timestamp revision ditetapkan secara deterministik hanya saat insert.
- JSONB dicanonicalisasi saat dibaca dan checksum diverifikasi kembali sebelum
  data diteruskan oleh repository.
- Repository page dan product hanya membaca revision published.
- Product list memakai keyset pagination, probe `limit + 1`, dan `hasMore`
  tanpa count query.
- CLI seed dan wrapper migration tersedia melalui root npm scripts.

### Milestone B3.4: local apply

**Status:** Complete

Migration dan seed diterapkan hanya ke PostgreSQL lokal:

```text
npm.cmd run backend:migrate:up: lulus
npm.cmd run backend:migrate:version: version=3 dirty=false
seed pertama: inserted=5 unchanged=0
seed replay: inserted=0 unchanged=5
```

Tidak ada cloud database, production deploy, push, atau remote mutation.

### Evidence integration

- Disposable PostgreSQL integration suite lulus tanpa cache sebelum content
  drift terjadi.
- Suite tersebut mencakup migration dari kosong, up/down, rotasi migrator,
  privilege dan trigger, atomic rollback ketika seed conflict, idempotent
  replay, repository parity, serta keyset pagination.
- Focused importer, canonical JSON, reconciliation, repository, migration, dan
  database package test lulus pada gate B3.

### Catatan content drift setelah gate

- Setelah migration dan seed replay lokal selesai, copy pada
  `content/home.json` berubah dari versi saat seed. Fixture homepage sudah
  diselaraskan kembali ke copy aktif “Tauco dari Cianjur” pada B4.
- Copy About yang mengandung typo dan klaim tahun belum terverifikasi sudah
  dikembalikan ke copy aman yang sama dengan golden fixture.
- Importer parity dan API fixture parity kembali lulus pada B4.

### Test yang tidak diulang atas arahan owner

- Frontend lint, typecheck, unit test, production build, dan E2E tidak diulang
  setelah penutupan B3.
- B3 tidak mengubah runtime, route, maupun adapter production Phase 1A, sehingga
  regression frontend tidak menjadi blocker untuk menutup database gate lokal.
- Full frontend regression tetap wajib dijalankan kembali saat B4
  mengintegrasikan public read API dan pada B10 sebelum cutover/deploy.
- Pengecualian ini tidak menghapus evidence frontend Phase 1A dan B2 yang sudah
  lulus.

### Boundary

- Redis hanya melewati health smoke pada B3. Adapter cache/rate-limit dan
  integration test Redis tetap milik B8.
- Production Next.js, Netlify Forms, dan deployment Phase 1A tidak berubah.
- Tidak ada commit, push, deploy, atau cloud database mutation.

### Next

B4 public read API dapat dimulai.

## Update 6: B4 Public Read API

**Tanggal:** 28 Juli 2026
**Status:** Complete

### Output

- Lima route publik aktif pada runtime lokal:
  - `GET /api/v1/home`;
  - `GET /api/v1/about`;
  - `GET /api/v1/tauco-guide`;
  - `GET /api/v1/products`;
  - `GET /api/v1/products/{slug}`.
- Delivery memakai generated OpenAPI request/response visitor dengan selective
  safe registration. Contact, readiness, dan metrics tidak ikut diregistrasikan.
- Handler hanya berbicara dengan `PublishedReader` application use case.
  Concrete GORM repository tetap dirangkai di composition root.
- Page dan product hanya mengambil revision yang ditunjuk oleh
  `published_revision_id`, berstatus `published`, dan mempunyai canonical JSON
  checksum yang valid.
- Catalog shell digabungkan dengan published product summary dari repository.
- Default page limit 20 dan maksimum 50. Keyset pagination memakai
  `(sort_order, id)` dan HMAC-SHA256 cursor yang terikat query hash.
- Unknown, duplicate, invalid limit, dan invalid cursor query menghasilkan
  problem response tanpa detail internal.
- Unknown product slug menghasilkan true `404 PRODUCT_NOT_FOUND`.
- Setiap response sukses membawa API version, request ID, strong `ETag`, dan
  public cache policy.
- `If-None-Match`, termasuk weak validator yang cocok, menghasilkan `304`
  tanpa body.
- Database unavailable dipetakan ke generic `503` dengan `Retry-After`.
- API process membuka least-privilege runtime connection, menutup pool pada
  shutdown, dan mewajibkan `CURSOR_HMAC_SECRET` minimal 32 byte.
- Root `backend:dev` sekarang memuat `backend/.env` melalui wrapper lokal.

### Evidence

```text
npm.cmd run backend:test: lulus
npm.cmd run backend:test:integration: lulus tanpa cache
npm.cmd run backend:generate:check: generated code reproducible
npm.cmd run backend:vet: lulus
npm.cmd run backend:build: lulus
tests/unit/api-contract.test.ts: 12/12 lulus
```

HTTP acceptance mencakup:

- parity lima response terhadap fixture Phase 1A;
- unknown dan duplicate query;
- invalid cursor serta limit;
- published-only read;
- product `404`;
- ETag dan `304`;
- contact route tetap tidak terdaftar;
- real disposable PostgreSQL sampai generated HTTP response.

Local process smoke terhadap database utama:

```text
200 /api/v1/home
200 /api/v1/about
200 /api/v1/tauco-guide
200 /api/v1/products
200 /api/v1/products/tauco-cap-badak
304 /api/v1/home
404 /api/v1/products/tidak-ada
```

Windows Application Control sempat memblokir satu temporary test executable.
Test diulang memakai workspace Go build cache dan seluruh package lulus.

### Dokumentasi kode

- `BACKEND_CODE_DOCUMENTATION.md`.
- `BACKEND_CODE_DOCUMENTATION.pdf`.
- Generator tersedia melalui `npm.cmd run backend:docs:pdf`.
- Dokumen menjelaskan kode aktual, bukan implementation plan.

### Boundary

- Redis cache belum dipasang; itu tetap B8.
- Contact transaction dan Netlify Forms cutover belum dilakukan; itu B5 dan
  Phase 1D.
- Production Next.js tetap memakai local content.
- Tidak ada commit, push, deploy, secret cloud, atau Supabase mutation.

### Next

B5 contact transaction.

## B0 Checklist

- [x] Audit PRD.
- [x] Audit repository.
- [x] Audit interface frontend.
- [x] Audit free-tier dan deployment alternatives.
- [x] Tentukan recommended scope Phase 1B.
- [x] Tentukan acceptance criteria Phase 1B.
- [x] Buat implementation plan.
- [x] Buat walkthrough ledger.
- [x] Owner menerima shadow-mode.
- [x] Owner menerima endpoint tauco guide.
- [x] Owner menerima production adapter tetap.
- [x] Owner menerima Phase 1C boundary.
- [x] PRD diamendemen.
- [x] ADR dibuat.
- [x] B0 final tests/document checks lulus.

## B1 Checklist

- [x] Go module dan toolchain.
- [x] Config loader/validation.
- [x] Clean Architecture folder boundary.
- [x] Gin composition root.
- [x] Zap logger/redaction.
- [x] Graceful shutdown.
- [x] Dockerfile.
- [x] Compose local dependency.
- [x] Root command wrapper.
- [x] Unit test.
- [x] Walkthrough update B1.

## B2 Checklist

- [x] OpenAPI v1.
- [x] `/api/v1/tauco-guide`.
- [x] Success envelope.
- [x] RFC 7807 problem details.
- [x] Request ID.
- [x] Cursor.
- [x] Code generation.
- [x] Safe generated handler dan response factory.
- [x] Contract fixtures.
- [x] Frontend Zod parity.
- [x] Canonical content string dan bounded escaped problem instance.
- [x] OpenAPI validation.
- [x] Clean Architecture dan full regression gate.
- [x] Walkthrough update B2.

## B3 Checklist

- [x] SQL migration.
- [x] Runtime/migration database role.
- [x] Page/revision schema.
- [x] Product/revision schema.
- [x] Media schema.
- [x] Contact schema.
- [x] Job/activity schema.
- [x] Published immutability.
- [x] GORM repository adapter.
- [x] Deterministic seed/import.
- [x] PostgreSQL integration test.
- [x] Migration test dari kosong.
- [x] Parity test awal lulus; guard kemudian menangkap live content drift.
- [x] Walkthrough update B3.

## B4 Checklist

- [x] Home.
- [x] About.
- [x] Tauco guide.
- [x] Product list.
- [x] Product detail.
- [x] Published-only rule.
- [x] Unknown slug 404.
- [x] Cursor pagination.
- [x] ETag/304.
- [x] Cache header.
- [x] API E2E.
- [x] Walkthrough update B4.

## B5 Checklist

- [ ] Contact validation parity.
- [ ] Mandatory idempotency.
- [ ] Message + email job + activity job transaction.
- [ ] Consent metadata.
- [ ] 12-month retention field.
- [ ] Honeypot.
- [ ] PII-free log.
- [ ] Duplicate/conflict tests.
- [ ] Retention test.
- [ ] Netlify production tetap aktif.
- [ ] Walkthrough update B5.

## B6 Checklist

- [ ] Atomic claim.
- [ ] Lease.
- [ ] Heartbeat.
- [ ] Bounded channel.
- [ ] Goroutine pool.
- [ ] Retry/backoff/jitter.
- [ ] Dead-letter.
- [ ] Replay.
- [ ] Graceful shutdown.
- [ ] Two-worker concurrency test.
- [ ] Crash/reclaim test.
- [ ] Walkthrough update B6.

## B7 Checklist

- [ ] Object storage port.
- [ ] Local/test adapter.
- [ ] S3 adapter.
- [ ] Magic/decode validation.
- [ ] Pixel/dimension limit.
- [ ] EXIF orientation.
- [ ] Metadata stripping.
- [ ] Normalized original.
- [ ] 320/640/1280 WebP.
- [ ] No-upscale.
- [ ] Idempotent object key.
- [ ] Media state machine.
- [ ] Abuse/failure tests.
- [ ] Walkthrough update B7.

## B8 Checklist

- [ ] Redis adapter.
- [ ] Cache-aside.
- [ ] TTL jitter.
- [ ] Singleflight.
- [ ] Generation invalidation.
- [ ] Redis fail-open.
- [ ] Public rate limit.
- [ ] Contact rate limit.
- [ ] Local fallback.
- [ ] Trusted proxy.
- [ ] CORS.
- [ ] Body/method/content-type limit.
- [ ] Recovery/error mapper.
- [ ] Walkthrough update B8.

## B9 Checklist

- [ ] Liveness.
- [ ] API readiness.
- [ ] Worker readiness.
- [ ] Prometheus metrics.
- [ ] Trace propagation.
- [ ] DB/cache/job/media/email metrics.
- [ ] Runbook replay.
- [ ] Runbook cache purge.
- [ ] Runbook media retry.
- [ ] Backup design.
- [ ] Walkthrough update B9.

## B10 Checklist

- [ ] Go unit.
- [ ] Go integration.
- [ ] Race.
- [ ] Vet.
- [ ] Lint.
- [ ] Vulnerability scan.
- [ ] OpenAPI drift.
- [ ] Migration.
- [ ] Container build.
- [ ] Warm load.
- [ ] Cold-start measurement.
- [ ] Existing `npm.cmd run check`.
- [ ] Existing E2E.
- [ ] Phase 1A production smoke test.
- [ ] Acceptance 1 sampai 33 mempunyai evidence.
- [ ] Walkthrough update B10.

## B11 Checklist

- [ ] Owner meminta remote pilot.
- [ ] Deployment profile dipilih.
- [ ] Billing/quota risk diterima.
- [ ] Region ditetapkan.
- [ ] Secret dibuat di provider.
- [ ] Remote migration.
- [ ] Shadow smoke test.
- [ ] Health/metrics/log diperiksa.
- [ ] Quota/cost diperiksa.
- [ ] Rollback diuji.
- [ ] Tidak ada frontend/contact cutover.
- [ ] Walkthrough update B11.
