# Tauco Cap Badak Backend

Backend Phase 1B adalah public REST API berbasis Go. Implementasi berjalan
local-first dan shadow-mode: website production tetap membaca konten lokal serta
tetap mengirim form kontak melalui Netlify Forms sampai ada gate cutover
tersendiri.

## Boundary Phase 1B

Phase 1B mencakup fondasi public API, PostgreSQL, Redis, durable worker, media
pipeline, security middleware, observability, dan kontrak OpenAPI.

Phase 1B tidak mencakup:

- Admin CMS;
- login, JWT, TOTP, session, atau RBAC;
- inventory, order, pembayaran, atau akun pelanggan;
- perpindahan website production ke API;
- deployment cloud.

Lihat `../PHASE_1B_PLAN.md` untuk rancangan lengkap dan
`../PHASE_1B_WALKTHROUGH.md` untuk evidence setiap gate.

Phase 1B B0-B10 telah selesai lokal. Final evidence tersedia di
`../PHASE_1B_QUALITY_REPORT.md`; runbook di `docs/OPERATIONS.md`; security review
di `docs/SECURITY_REVIEW.md`.

## Prasyarat

- Go sesuai versi pada `go.mod`;
- Docker Desktop atau Docker Engine dengan Compose v2;
- PowerShell 7 atau Windows PowerShell.

Docker mulai wajib pada gate B3. B1 dan B2 dapat dikompilasi serta diuji tanpa
Docker. PostgreSQL nyata wajib tersedia untuk integration gate B3. Container
Redis boleh di-smoke-test pada B3, sedangkan adapter dan integration test Redis
baru menjadi acceptance B8.

Setelah Docker Desktop selesai dipasang, pastikan aplikasinya sudah terbuka dan
engine berstatus running:

```powershell
docker version
docker compose version
```

Kedua perintah harus menampilkan versi tanpa error koneksi daemon.

## Setup lokal

Jalankan dari root repository:

```powershell
Set-Location backend
Copy-Item .env.example .env
go mod download
```

File `.env` hanya untuk komputer lokal dan tidak boleh di-commit. Nilai password
contoh pada `.env.example` sengaja hanya untuk dependency lokal. Gunakan secret
manager dan credential berbeda pada environment remote.

Docker Compose otomatis membaca file `.env`. Binary Go tidak memuat `.env`
secara otomatis. Wrapper migration, seed, dan integration pada root repository
memuat `backend/.env` tanpa mencetak credential. Konfigurasi HTTP B1 mempunyai
default lokal yang aman, sehingga `go run` dapat langsung dijalankan. Untuk
mengubah satu nilai pada sesi PowerShell, gunakan pola berikut:

```powershell
$env:PORT = "8081"
go run ./cmd/api
Remove-Item Env:PORT
```

### Menyalakan PostgreSQL dan Redis

```powershell
docker compose up -d postgres redis
docker compose ps
```

Tunggu sampai kolom health untuk keduanya menampilkan `healthy`. Bila salah satu
tidak sehat:

```powershell
docker compose logs --tail 100 postgres redis
```

Port hanya di-bind ke loopback host:

| Service | Host |
| --- | --- |
| PostgreSQL | `127.0.0.1:5432` |
| Redis | `127.0.0.1:6379` |

Port dapat diganti melalui `TAUCO_POSTGRES_PORT` dan `TAUCO_REDIS_PORT` di
`.env` bila port default sedang digunakan aplikasi lain.

## Database dan deterministic seed B3

B3 memakai schema privat `tauco_app` dan tiga identitas PostgreSQL yang
berbeda:

- login Compose `tauco_app` hanya untuk bootstrap/migration lokal;
- authorization role `tauco_migrator` menjadi stable schema owner;
- login `tauco_app_runtime` hanya mewarisi least-privilege
  `tauco_runtime`.

Runtime tidak dapat membuat schema/table, memakai temporary table, atau menulis
page/product revision. Migration SQL selalu versioned; GORM `AutoMigrate`
dilarang.

Jalankan dari root repository. Integration test memakai database dan login
disposable, kemudian menghapus keduanya. Database aplikasi utama `tauco` belum
diubah oleh test ini.

```powershell
npm.cmd run backend:test:integration
```

Setelah integration gate lulus, terapkan migration ke database lokal utama:

```powershell
npm.cmd run backend:migrate:up
npm.cmd run backend:migrate:version
```

Hasil version yang diharapkan adalah `version=4 dirty=false`. Import konten
Phase 1A:

```powershell
npm.cmd run backend:seed:phase1a
```

Run pertama menghasilkan `inserted=5 unchanged=0`. Replay command yang sama
harus menghasilkan `inserted=0 unchanged=5`; seed tidak melakukan upsert atau
menimpa published revision.

Rollback satu migration tersedia untuk latihan pada database disposable:

```powershell
npm.cmd run backend:migrate:down
```

Jangan menjalankan rollback pada database yang ingin dipertahankan tanpa
backup dan review versi. CLI tidak melakukan full rollback kecuali argument
`down --all` diberikan secara eksplisit.

Konten yang diimpor:

- singleton page `home`, `about`, `tauco-guide`, dan catalog shell `products`;
- satu published product `tauco-cap-badak`;
- UUIDv7, revision, canonical JSON, checksum, sort order, serta timestamp yang
  deterministik.

## Public read API B4

Lima route berikut aktif pada runtime lokal:

- `GET /api/v1/home`;
- `GET /api/v1/about`;
- `GET /api/v1/tauco-guide`;
- `GET /api/v1/products`;
- `GET /api/v1/products/{slug}`.

Response hanya membaca revision `published`, memakai generated OpenAPI type,
strong `ETag`, public cache header, dan mendukung conditional request `304`.
Catalog memakai HMAC-signed cursor dengan default limit 20 dan maksimum 50.
Unknown product menghasilkan true `404`.

Contact sudah aktif pada backend shadow. Readiness dan metrics baru dikerjakan
pada B9. Production Next.js belum memakai API ini.

Jalankan dari root repository agar `backend/.env` dimuat oleh wrapper:

```powershell
npm.cmd run backend:dev
```

Smoke lokal:

```powershell
Invoke-WebRequest http://127.0.0.1:8080/api/v1/home
Invoke-WebRequest http://127.0.0.1:8080/api/v1/products
```

## Contact, worker, media, cache, dan security B5-B8

Contact backend tersedia pada `POST /api/v1/contact-messages`. Request wajib
JSON, `Idempotency-Key`, consent, dan maksimal 32 KiB. Satu transaksi menyimpan
message serta membuat job email dan activity. Production tetap memakai Netlify
Forms sampai cutover Phase 1D.

Jalankan worker durable:

```powershell
npm.cmd run backend:worker
```

Worker memakai PostgreSQL lease, heartbeat, dua goroutine, retry dengan jitter,
dead-letter, dan replay. SMTP lokal dapat diarahkan ke Mailpit.

Media Phase 1B hanya dapat di-ingest melalui CLI internal:

```powershell
npm.cmd run backend:media:import -- --file "D:\gambar\produk.jpg" --alt "Foto produk"
```

Source JPEG/PNG/static WebP dinormalisasi menjadi PNG privat dan worker membuat
WebP 320/640/1280 tanpa upscale. Tidak ada endpoint upload HTTP sebelum Admin
CMS Phase 1C.

Public read memakai Redis cache-aside 5 menit dengan jitter dan generation
tags. Redis down otomatis jatuh ke PostgreSQL. Rate limit menggunakan Redis
dengan fallback lokal bounded: 60 request/menit untuk read dan 5 request/jam
untuk contact. CORS memakai exact allowlist dan forwarded IP hanya dipercaya
dari CIDR yang dikonfigurasi.

Dokumentasi kode lengkap tersedia di
`../BACKEND_CODE_DOCUMENTATION.md` dan
`../BACKEND_CODE_DOCUMENTATION.pdf`.

## Observability dan quality B9-B10

```powershell
Invoke-RestMethod http://127.0.0.1:8080/health/live
Invoke-RestMethod http://127.0.0.1:8080/health/ready
npm.cmd run backend:worker:ready
npm.cmd run backend:quality
```

Protected metrics berada di `/internal/metrics` dan memerlukan
`METRICS_BEARER_TOKEN`. Valid `traceparent` dipropagasikan ke response dan log.
Runbook lengkap berada di `docs/OPERATIONS.md`; security review di
`docs/SECURITY_REVIEW.md`; hasil akhir di `../PHASE_1B_QUALITY_REPORT.md`.

## OpenAPI dan generated contract

Kontrak B2 bersifat contract-first:

- sumber yang diedit: `openapi/openapi.yaml`;
- konfigurasi generator: `openapi/oapi-codegen.yaml`;
- output generated: `internal/delivery/api/api.gen.go`;
- shared parity fixture: `../contracts/fixtures/`.

Jalankan dari root repository:

```powershell
npm.cmd run backend:generate
npm.cmd run backend:generate:check
npm.cmd run backend:test
```

Generator `oapi-codegen` dan runtime-nya dikunci di `go.mod`. Jangan mengedit
`api.gen.go` secara manual. Drift check membuat output baru di temporary
directory dan membandingkannya byte-for-byte setelah normalisasi line ending.

Kontrak saat ini mempunyai route:

- `GET /api/v1/home`;
- `GET /api/v1/about`;
- `GET /api/v1/tauco-guide`;
- `GET /api/v1/products`;
- `GET /api/v1/products/{slug}`;
- `POST /api/v1/contact-messages`;
- `GET /health/live`;
- `GET /health/ready`;
- `GET /internal/metrics`.

Tidak ada route admin pada Phase 1B. Lima public read route B4 telah
diregistrasikan secara selektif. Contact transaction mulai B5, Redis cache mulai
B8, serta readiness dan protected metrics diselesaikan pada B9.

### Guardrail integrasi lanjutan

- Gunakan safe registration; static guard menolak generated default
  constructor/registrar di production code karena default tersebut
  mengembalikan `err.Error()`.
- Bangun list response melalui `api.NewListProducts200Response` agar request
  ID, API version, ETag, cache header, page limit, serta relasi
  `hasMore`/`nextCursor` tervalidasi sebelum generated response visitor menulis
  body.
- Tambahkan request validation middleware untuk unknown query, Content-Type,
  strict unknown contact JSON field, dan body maksimum 32 KiB pada B5.
  Generated binder tidak menegakkan seluruh constraint OpenAPI secara otomatis.
- Validasi response bisnis terhadap generated/OpenAPI contract sebelum emit;
  khusus image alt, parser boleh trim input authored tetapi response B4 wajib
  mengirim string canonical yang sudah tanpa whitespace di tepi.
- Metrics authentication memakai dedicated bearer token, constant-time
  comparison, serta negative/positive integration check.

### Menyalakan Mailpit

Mailpit disediakan melalui profile opsional untuk integration test email pada
gate B5:

```powershell
docker compose --profile mail up -d mailpit
docker compose --profile mail ps
```

SMTP tersedia pada `127.0.0.1:1025` dan inbox lokal pada
`http://127.0.0.1:8025`. Mailpit tidak mengirim email ke internet.

### Menjalankan API

Dengan dependency tetap hidup, jalankan dari root repository:

```powershell
npm.cmd run backend:dev
```

Default API hanya bind ke `127.0.0.1:8080`. Tekan `Ctrl+C` untuk memicu graceful
shutdown.

## Container production

Dockerfile memakai multi-stage build dan menghasilkan image runtime `scratch`
tanpa shell. Binary berjalan sebagai UID/GID `65532`, bukan root. CA certificate
dan timezone database tetap disertakan untuk koneksi TLS serta waktu yang benar.

Build lokal:

```powershell
docker build --tag tauco-cap-badak-api:local .
```

Build tersebut hanya memvalidasi image. Deployment dan push image tidak
dilakukan pada Phase 1B tanpa instruksi eksplisit pemilik repository.

## Lifecycle dependency

Menghentikan container tanpa menghapus data:

```powershell
docker compose --profile mail stop
```

Menyalakannya kembali:

```powershell
docker compose --profile mail start
```

Menghapus container dan network tetapi mempertahankan named volume:

```powershell
docker compose --profile mail down
```

Perintah berikut juga menghapus seluruh data PostgreSQL, Redis, dan Mailpit
lokal. Gunakan hanya saat benar-benar ingin reset:

```powershell
docker compose --profile mail down --volumes
```

## Image yang dikunci

Dependency lokal tidak memakai tag `latest`:

| Kegunaan | Image |
| --- | --- |
| Go builder | `golang:1.26.5-alpine3.23` |
| PostgreSQL | `postgres:17.10-alpine3.22` |
| Redis | `redis:8.8.0-alpine3.23` |
| Mail capture | `axllent/mailpit:v1.30.0` |

Object storage memakai port pada application layer. Adapter local aktif untuk
shadow-mode; adapter S3-compatible tersedia tetapi belum diberi credential atau
bucket remote sampai pilot B11.

## Troubleshooting singkat

- `docker` tidak dikenali: tutup dan buka kembali terminal setelah instalasi
  Docker Desktop.
- Tidak dapat terhubung ke daemon: buka Docker Desktop dan tunggu engine
  berstatus running.
- Port sudah digunakan: ubah port terkait pada `.env`, lalu jalankan
  `docker compose up -d` lagi.
- Credential berubah tetapi container masih memakai nilai lama: named volume
  PostgreSQL mempertahankan credential saat inisialisasi pertama. Reset volume
  hanya jika data lokal memang boleh dihapus.
- Windows Application Control memblokir executable test sementara: jalankan
  `npm.cmd run backend:test` dari root. Runner tetap mencoba `go test ./...`
  terlebih dahulu, lalu hanya mengulang package yang diblokir melalui binary
  sementara di workspace dan menghapusnya setelah selesai.
- API tidak menerima koneksi dari komputer lain: ini disengaja pada local
  profile karena `HTTP_HOST` dan dependency port dibatasi ke loopback.
