# Dokumentasi Kode Backend Tauco Cap Badak

## Go API, PostgreSQL, dan OpenAPI

| Atribut | Nilai |
| --- | --- |
| Versi | 1.0 |
| Tanggal | 28 Juli 2026 |
| Status kode | Phase 1B B1-B4 |
| Runtime | Go 1.26.5, Gin, GORM, PostgreSQL 17 |
| Mode | Local-first, shadow-mode |

## 1. Fungsi Backend

Backend menyediakan REST API publik untuk membaca konten yang sudah
dipublikasikan. PostgreSQL menjadi source of truth, sedangkan kontrak HTTP
berasal dari OpenAPI.

Saat ini backend tidak menggantikan website production. Next.js production
masih membaca file konten lokal dan form kontak masih memakai Netlify Forms.

Route aktif:

```text
GET /health/live
GET /api/v1/home
GET /api/v1/about
GET /api/v1/tauco-guide
GET /api/v1/products
GET /api/v1/products/{slug}
```

Route contact, readiness, dan metrics sudah didefinisikan dalam OpenAPI tetapi
belum diregistrasikan sampai implementasinya tersedia.

## 2. Arsitektur

```text
HTTP request
    |
    v
Gin middleware
request ID, recovery, access log
    |
    v
Generated OpenAPI transport
parameter binding dan response visitor
    |
    v
Delivery handler
DTO, ETag, 304, problem response
    |
    v
Application use case
published read dan cursor pagination
    |
    v
Repository interface
    |
    v
GORM PostgreSQL adapter
    |
    v
Schema privat tauco_app
```

Aturan dependency:

- `domain` hanya memuat entity dan invariant.
- `application` memuat use case serta repository port.
- `delivery` menerjemahkan HTTP dan generated OpenAPI type.
- `repository` menjalankan query PostgreSQL.
- `platform` menangani database, HTTP server, config, dan logging.
- `composition` adalah satu-satunya tempat concrete dependency dirangkai.

Architecture test menolak dependency yang mengarah ke outer layer.

## 3. Struktur Kode

| Lokasi | Tanggung jawab |
| --- | --- |
| `backend/cmd/api` | Entry point HTTP API |
| `backend/cmd/migrate` | CLI migration |
| `backend/cmd/seed` | CLI deterministic seed |
| `backend/internal/content` | Page domain, use case, importer, repository |
| `backend/internal/catalog` | Product domain, pagination, cursor, repository |
| `backend/internal/delivery/api` | Generated contract dan B4 handler |
| `backend/internal/platform/database` | GORM, role bootstrap, migration |
| `backend/internal/platform/httpserver` | Gin router dan middleware dasar |
| `backend/internal/platform/logging` | Zap logger dan redaksi |
| `backend/internal/composition` | Dependency wiring dan lifecycle |
| `backend/migrations` | Versioned SQL migration |
| `backend/openapi/openapi.yaml` | Sumber kontrak API |
| `contracts/fixtures` | Fixture bersama Go dan TypeScript |

File `api.gen.go` dibuat oleh `oapi-codegen` dan tidak boleh diedit manual.

## 4. Menjalankan Backend

Prasyarat:

- Docker Desktop aktif.
- PostgreSQL dan Redis berstatus `healthy`.
- `backend/.env` tersedia dan tidak masuk Git.

Dari root repository:

```powershell
npm.cmd run backend:compose:up
npm.cmd run backend:migrate:up
npm.cmd run backend:migrate:version
npm.cmd run backend:seed:phase1a
npm.cmd run backend:dev
```

Migration yang benar menampilkan:

```text
version=3 dirty=false
```

API berjalan pada:

```text
http://127.0.0.1:8080
```

Hentikan API dengan `Ctrl+C`. Hentikan dependency tanpa menghapus volume:

```powershell
npm.cmd run backend:compose:down
```

## 5. Konfigurasi Runtime

| Environment variable | Fungsi |
| --- | --- |
| `APP_ENV` | `local`, `test`, `staging`, atau `production` |
| `HTTP_HOST`, `PORT` | Alamat HTTP server |
| `DATABASE_URL` | Login PostgreSQL runtime |
| `DATABASE_MAX_OPEN_CONNS` | Batas koneksi terbuka |
| `DATABASE_MAX_IDLE_CONNS` | Batas koneksi idle |
| `DATABASE_CONN_MAX_LIFETIME` | Umur maksimum koneksi |
| `DATABASE_CONN_MAX_IDLE_TIME` | Waktu idle maksimum |
| `CURSOR_HMAC_SECRET` | Kunci cursor, minimal 32 byte |
| `LOG_LEVEL`, `LOG_FORMAT` | Level dan format log |

Runtime memakai PostgreSQL simple protocol dan menonaktifkan prepared
statement agar kompatibel dengan Supavisor transaction pooler.

Nilai secret tidak boleh ditulis ke log, source code, atau Git.

## 6. Database

Application table berada di schema privat `tauco_app`.

| Tabel | Fungsi |
| --- | --- |
| `pages` | Identitas stabil singleton page |
| `page_revisions` | Revision konten page |
| `products` | Identitas, slug, dan urutan product |
| `product_revisions` | Revision detail product |
| `media_assets` | Metadata original media |
| `media_variants` | Hasil resize media |
| `contact_messages` | Pesan dan consent pengunjung |
| `background_jobs` | Durable asynchronous job |
| `activity_logs` | Append-only audit activity |

Role:

- `tauco_migrator`: owner schema dan migration, tidak dapat login langsung.
- `tauco_runtime`: authorization role aplikasi, tidak dapat login langsung.
- `tauco_app_runtime`: login lokal yang mewarisi least-privilege runtime role.

Runtime dapat membaca published content, tetapi tidak dapat membuat schema,
table, atau mengubah published revision.

Schema diubah hanya melalui SQL migration. `GORM AutoMigrate` dilarang.

## 7. Public Read API

Semua response sukses memakai envelope:

```json
{
  "data": {},
  "meta": {
    "apiVersion": "v1",
    "requestId": "request-id"
  }
}
```

Response list menambahkan:

```json
{
  "page": {
    "limit": 20,
    "hasMore": false,
    "nextCursor": null
  }
}
```

### Published-only

Repository hanya mengambil revision yang:

- dipilih oleh `published_revision_id`;
- dimiliki entity yang sama;
- mempunyai status `published`;
- lolos validasi checksum canonical JSON.

Draft dan archived revision tidak dapat keluar melalui public API.

### Pagination

Catalog diurutkan berdasarkan:

```text
sort_order ASC, id ASC
```

Cursor berisi posisi terakhir dan query hash, lalu ditandatangani HMAC-SHA256.
Cursor yang rusak, dimodifikasi, atau memakai secret berbeda menghasilkan:

```text
400 INVALID_CURSOR
```

Default limit adalah 20 dan maksimum 50.

### ETag

Response publik mengirim:

```text
Cache-Control: public, max-age=0, s-maxage=300, stale-while-revalidate=60
ETag: "sha256-..."
```

Client dapat mengirim `If-None-Match`. Jika data belum berubah, API
mengembalikan `304` tanpa response body.

### Error

Error memakai `application/problem+json` dan tidak mengekspos error internal.

| Kondisi | Status |
| --- | --- |
| Query, limit, atau header tidak valid | `400` |
| Cursor tidak valid | `400 INVALID_CURSOR` |
| Product tidak ditemukan | `404 PRODUCT_NOT_FOUND` |
| PostgreSQL tidak tersedia | `503 SERVICE_UNAVAILABLE` |
| Panic atau serialization failure | `500 INTERNAL_SERVER_ERROR` |

Setiap response membawa `X-Request-ID`.

## 8. Logging dan Lifecycle

Zap menghasilkan structured log. Access log memakai route template dan tidak
merekam query string atau request body.

Field sensitif seperti credential, token, email, telepon, IP, dan teks panjang
direduksi atau disensor.

Ketika menerima `Ctrl+C` atau `SIGTERM`, server:

1. berhenti menerima request baru;
2. menunggu request aktif sesuai shutdown deadline;
3. menutup pool PostgreSQL;
4. melakukan flush logger.

## 9. Perintah Pengembangan

```powershell
npm.cmd run backend:format
npm.cmd run backend:generate:check
npm.cmd run backend:test
npm.cmd run backend:test:integration
npm.cmd run backend:vet
npm.cmd run backend:build
```

`backend:test:integration` memakai PostgreSQL disposable untuk migration,
privilege, seed, repository, dan HTTP read API. Database disposable dihapus
setelah test.

Acceptance B4 memeriksa:

- lima public read route;
- parity dengan fixture Phase 1A;
- published-only;
- unknown product `404`;
- invalid cursor dan limit;
- unknown atau duplicate query;
- cursor pagination;
- ETag dan `304`;
- PostgreSQL-to-HTTP integration.

## 10. Menambah Fitur

Urutan perubahan backend:

1. ubah domain invariant jika diperlukan;
2. tambahkan use case dan port pada `application`;
3. implementasikan adapter pada `repository`;
4. ubah OpenAPI lalu generate ulang;
5. implementasikan delivery handler;
6. rangkai dependency pada `composition`;
7. tambahkan test terkecil yang membuktikan behavior.

Jangan:

- mengimpor Gin atau GORM ke domain/application;
- mengedit `api.gen.go`;
- memakai `AutoMigrate`;
- mengakses database langsung dari frontend;
- menaruh secret dalam source atau fixture.

## 11. Boundary Saat Ini

- Redis baru dependency lokal; cache dan rate limiting masuk B8.
- Contact API belum aktif; persistence masuk B5.
- Worker belum aktif; concurrency dan retry masuk B6.
- Media processing belum aktif; pipeline masuk B7.
- Admin, authentication, inventory, dan order tidak termasuk Phase 1B.
- Supabase belum dipasang. PostgreSQL Docker adalah environment lokal.
