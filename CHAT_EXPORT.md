# Engineering Conversation Log

## Tauco Cap Badak Website dan Backend

| Atribut | Nilai |
| --- | --- |
| Tanggal ekspor | 28 Juli 2026 |
| Cakupan | Phase 1A sampai Phase 1B B4 |
| Bentuk | Ringkasan percakapan engineering |
| Repository | Public |

Dokumen ini merangkum keputusan, pekerjaan, pengujian, dan handoff yang muncul
dalam percakapan proyek. Ini bukan raw transcript platform. Email, token,
request ID, credential, dan data pribadi sengaja tidak disertakan.

## 1. Phase 1A

Permintaan awal adalah website publik SEO-first untuk Tauco Cap Badak dengan
Next.js App Router, Tailwind CSS, konten lokal tervalidasi, katalog awal, dan
form kontak.

Keputusan utama:

- frontend dipublikasikan di Netlify Free;
- dark mode mengikuti preferensi sistem;
- production memakai local content dan Netlify Forms;
- fakta yang belum terverifikasi tidak ditampilkan;
- backend, CMS, Redis, dan Supabase ditunda;
- deployment dilakukan oleh pemilik repository.

Hasil:

- tujuh public route aktif;
- metadata, canonical, sitemap, robots, dan structured data tersedia;
- form Netlify aktif dan submission terverifikasi;
- Search Console terverifikasi dan sitemap berhasil dibaca;
- gate G0 sampai G8 selesai;
- Phase 1A tersedia di `https://tauco-cap-badak.netlify.app`.

## 2. Arah Phase 1B

Phase 1B dibangun local-first dalam shadow-mode. Backend tidak mengubah
production Phase 1A dan belum memakai layanan cloud.

Target arsitektur:

- Go, Gin, dan OpenAPI untuk REST API;
- Clean Architecture dan dependency injection;
- PostgreSQL sebagai source of truth;
- Redis untuk cache dan rate limiting pada gate berikutnya;
- durable PostgreSQL jobs untuk proses asynchronous;
- Supabase dan Upstash hanya kandidat remote pilot setelah local gate selesai.

## 3. Gate B1: Backend Skeleton

Output:

- module Go dan entrypoint API;
- config tervalidasi;
- Gin server dengan request ID, recovery, dan structured logging;
- graceful shutdown;
- Docker Compose untuk PostgreSQL, Redis, dan Mailpit;
- package boundary untuk content, catalog, contact, media, jobs, dan audit;
- architecture test;
- local environment template dan root npm commands.

Keputusan:

- backend berada di folder `backend/`;
- Next.js tetap berada di root;
- tidak ada route admin;
- Docker belum diwajibkan sampai integration gate.

## 4. Gate B2: Executable API Contract

Output:

- OpenAPI v1 sebagai sumber kontrak;
- generated Go types dan server interface;
- success envelope dan RFC 9457 problem response;
- request ID contract;
- HMAC cursor contract;
- fixture JSON bersama Go dan TypeScript;
- Zod contract schemas pada frontend;
- generated-code reproducibility check.

Route yang didefinisikan:

- lima public read route;
- contact intake;
- liveness dan readiness;
- protected metrics.

Pada B2 route bisnis belum diaktifkan. Gate ini hanya membekukan dan menguji
kontrak.

## 5. Gate B3: Database dan Seed

Output:

- PostgreSQL migration versi 1 sampai 3;
- schema privat `tauco_app`;
- migrator dan least-privilege runtime role;
- revision-based page dan product schema;
- contact, media, background job, dan activity log tables;
- deterministic importer konten Phase 1A;
- idempotent seed;
- GORM repository;
- disposable PostgreSQL integration tests.

Database lokal diperiksa melalui Docker dan DBeaver. PostgreSQL serta Redis
berstatus sehat. Supabase belum dipakai.

## 6. Gate B4: Public Read API

Route runtime lokal:

```text
GET /api/v1/home
GET /api/v1/about
GET /api/v1/tauco-guide
GET /api/v1/products
GET /api/v1/products/{slug}
```

Behavior:

- hanya membaca published revision;
- catalog memakai keyset pagination dan HMAC-signed cursor;
- response membawa request ID, API version, cache policy, dan strong ETag;
- `If-None-Match` yang cocok menghasilkan `304`;
- unknown product menghasilkan true `404`;
- invalid query, limit, dan cursor menghasilkan `400`;
- database failure dipetakan menjadi generic `503`;
- contact, readiness, dan metrics tetap tidak diregistrasikan.

Verification:

- seluruh Go unit dan acceptance test lulus;
- PostgreSQL integration nyata lulus;
- frontend API contract test 12 dari 12 lulus;
- generated OpenAPI reproducible;
- `go vet` dan build lulus;
- local HTTP smoke menghasilkan lima `200`, satu `304`, dan satu expected
  product `404`.

Dokumentasi kode tersedia di `BACKEND_CODE_DOCUMENTATION.md` dan
`BACKEND_CODE_DOCUMENTATION.pdf`.

## 7. Menjalankan Backend

```powershell
npm.cmd run backend:compose:up
npm.cmd run backend:migrate:up
npm.cmd run backend:seed:phase1a
npm.cmd run backend:dev
```

Contoh pemeriksaan:

```powershell
curl.exe -i http://127.0.0.1:8080/api/v1/home
curl.exe -i http://127.0.0.1:8080/api/v1/products
curl.exe -i http://127.0.0.1:8080/api/v1/products/tauco-cap-badak
```

## 8. Status Handoff

- Phase 1A selesai di production.
- Phase 1B B0 sampai B4 selesai secara lokal.
- Production Next.js tetap memakai local content dan Netlify Forms.
- Tidak ada deployment backend atau mutation cloud database.
- Gate berikutnya adalah B5 contact transaction.
