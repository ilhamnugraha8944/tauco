# Phase 1D Walkthrough

## Remote Pilot dan Production Cutover

| Atribut | Nilai |
| --- | --- |
| Tanggal mulai | 6 Agustus 2026 |
| Branch | `feature/phase-1d` |
| Baseline | `216c6f9` |
| Status | D0-D2 complete lokal; D3-D10 pending |
| Production | Phase 1A tetap aktif dan tidak berubah |

Dokumen ini adalah ledger eksekusi Phase 1D. Setiap gate wajib diperbarui
setelah command dan acceptance benar-benar dijalankan. Secret, token, password,
backup, serta PII tidak boleh ditempelkan ke evidence.

## Format Update Gate

```markdown
## Update D<n>: <Nama Gate>

**Tanggal:**
**Status:** In Progress | Complete | Blocked

### Tujuan
### Perubahan
### Command yang dijalankan
### Hasil pengujian
### Evidence
### Known limitations
### Owner action
### Rollback
### Next gate
```

## Update D0: Scope dan Provider Freeze

**Tanggal:** 6 Agustus 2026
**Status:** Complete

### Tujuan

Membekukan provider, boundary, rollout, rollback, security baseline, dan gate
Phase 1D sebelum remote credential, migration, runtime, atau deployment dibuat.

### Perubahan

- Menormalkan generated `next-env.d.ts` sebelum membuat branch.
- Fast-forward local `main` ke Phase 1C merge commit `216c6f9`.
- Membuat branch lokal `feature/phase-1d`.
- Membuat `PHASE_1D_PLAN.md` dan walkthrough ini.
- Membuat ADR-0006 untuk Netlify remote pilot yang dibatasi waktu.
- Mengubah status Phase 1D pada PRD dan README.

### Keputusan arsitektur

- Netlify Legacy Free, Supabase Free, dan Upstash Free menjadi pilot provider.
- Netlify Forms tetap menjadi contact transport dan email sender.
- Verified submission disalin idempotently ke CMS; Go contact intake tetap off.
- Google Drive menjadi lokasi backup terenkripsi milik owner.
- Deployment dilakukan bertahap dan selalu memiliki local-content rollback.
- Owner mengoperasikan dashboard provider; agent menyiapkan kode, command, dan
  walkthrough.
- D0 documentation-only. Tidak ada runtime code atau cloud mutation.

### Command yang dijalankan

```powershell
git status --short --branch
git rev-parse main origin/main feature/phase-1c
git diff --exit-code -- next-env.d.ts
git switch main
git merge --ff-only origin/main
git switch -c feature/phase-1d
rg -n "Phase 1D|IMPLEMENTING-1D" PRD.md README.md PHASE_1D_PLAN.md PHASE_1D_WALKTHROUGH.md docs/adr
git diff --check
```

### Hasil pengujian

```text
Phase 1C merge baseline: 216c6f97020a74de3bb318c06c56c188967d58b5
branch: feature/phase-1d
working tree sebelum dokumentasi: bersih
runtime/migration/API/UI: tidak berubah pada D0
production deployment: tidak dilakukan
remote provider mutation: tidak dilakukan
```

### Evidence

- Plan mencatat D0-D10, provider, interface, rollout, rollback, dan acceptance.
- ADR mencatat alasan, batas waktu, serta konsekuensi Netlify pilot.
- PRD membedakan Phase 1A production, Phase 1B/1C local complete, dan Phase 1D
  implementing.
- README menautkan plan dan walkthrough Phase 1D.

### Known limitations

- Supabase dan Upstash belum dibuat.
- Belum ada production-safe configuration, remote runtime, atau adapter baru.
- Public production tetap memakai local content dan Netlify Forms.
- Go Lambda Compatibility mempunyai deadline provider sehingga pilot harus
  dievaluasi kembali sebelum 1 Juli 2027.

### Owner action

Tidak ada pada D0. Jangan membuat credential provider sebelum walkthrough D3
menentukan nama variable, role, dan penyimpanan secret.

### Rollback

D0 hanya dokumentasi dan branch lokal. Rollback dilakukan dengan tidak
melanjutkan branch; production tidak terdampak.

### Next gate

D1: production configuration, Ed25519 JWT, BFF trust boundary, dan
Supabase-safe database role profile.

## Maintenance Update D0.1: Local Worker Retention Fix

**Tanggal:** 6 Agustus 2026
**Status:** Complete

### Tujuan

Memperbaiki worker lokal yang berhenti segera setelah startup walaupun
readiness awal lulus.

### Root cause

Worker langsung menjalankan purge retensi saat startup. Repository mencoba
`DELETE` pada `contact_messages`, sedangkan least-privilege `tauco_runtime`
secara sengaja tidak memiliki privilege `DELETE`. Error dari retention
goroutine kemudian menghentikan seluruh worker dan hanya ditampilkan sebagai
pesan generik.

### Perubahan

- Migration v7 membuat bounded `SECURITY DEFINER` retention function.
- Public tidak mendapat `EXECUTE`; hanya `tauco_runtime` yang diizinkan.
- Fungsi menolak limit di luar 1-1000, nilai null, serta cutoff masa depan.
- Repository memanggil fungsi tersebut dan tetap tidak memiliki direct table
  `DELETE`.
- Worker readiness memeriksa capability retention.
- Startup error sekarang menyertakan penyebab internal yang aman.
- Bootstrap allowlist hanya menerima signature fungsi retensi yang tepat.
- Existing migration integration test memverifikasi direct delete ditolak,
  expired record dihapus, dan future cutoff ditolak.

### Command yang dijalankan

```powershell
npm.cmd run backend:worker:ready
npm.cmd run backend:test
npm.cmd run backend:test:integration
docker run ... go test -count=1 -run ^TestMigrationRoundTripAndRuntimePrivileges$ ./internal/platform/database
npm.cmd run backend:migrate:version
npm.cmd run backend:migrate:up
npm.cmd run backend:worker:ready
npm.cmd run backend:worker
npm.cmd run backend:vet
npm.cmd run backend:build
git diff --check
```

### Hasil pengujian

```text
targeted migration/privilege integration: PASS
migration local: version 6 -> 7, dirty=false
worker readiness setelah migration: PASS
worker process smoke: tetap hidup; tidak berhenti unexpectedly
direct DELETE contact_messages: false
bounded retention function EXECUTE: true
go vet: PASS
backend build: PASS
```

### Known limitations

- Repo-wide format checker melaporkan baseline CRLF/LF pada banyak file Go
  lama. File yang berubah sudah diformat dengan `gofmt`; tidak dilakukan
  rewrite massal.
- Full suite mengalami Windows Application Control dan satu Redis timeout
  fluktuatif. Targeted database integration, vet, build, readiness, dan live
  worker smoke lulus.
- Perbaikan ini hanya lokal dan tidak mengubah production.

### Rollback

Rollback membutuhkan code rollback dan migration down v7 secara bersamaan.
Menurunkan migration tanpa mengembalikan repository akan membuat worker gagal
readiness, sehingga mismatch tidak dapat berjalan diam-diam.

### Next gate

D1 tetap pending dan tidak diperluas oleh maintenance fix ini.

## Update D1: Production Configuration dan Security

**Tanggal:** 7 Agustus 2026
**Status:** Complete lokal

### Tujuan

Menutup konfigurasi yang sebelumnya dapat mengaktifkan remote Admin atau
contact intake secara implisit, lalu menyiapkan boundary autentikasi dan
database yang aman sebelum adapter media remote dibuat pada D2.

### Perubahan

- Menambah typed deployment config dengan boolean strict dan default aman:
  `ADMIN_REMOTE_ENABLED=false`, `CONTACT_API_ENABLED=false`, serta
  `FORM_SYNC_ENABLED=false`.
- Production/staging menolak origin non-HTTPS, cookie admin non-secure,
  local media storage, Redis non-TLS, secret contoh/pendek, dan environment
  aplikasi di atas budget konservatif 3072 byte.
- Contact route hanya dibangun dan didaftarkan ketika flag eksplisit aktif;
  secret contact tidak diperlukan ketika route mati.
- Admin route hanya dibangun ketika `ADMIN_REMOTE_ENABLED=true`; keberadaan
  `ADMIN_DATABASE_URL` saja tidak lagi membuka CMS.
- Next BFF menambahkan `ADMIN_BFF_SHARED_SECRET` dari server environment.
  Browser tidak dapat meneruskan atau mengganti header tersebut, dan Go
  menolak request admin tanpa nilai yang cocok menggunakan comparison
  constant-time.
- Production/staging memakai Ed25519/EdDSA dari satu raw-base64 environment
  secret. Local/test tetap memakai pasangan file RSA agar workflow lama tidak
  rusak.
- Menambah command operator `keygen-ed25519` tanpa dependency baru.
- Menambah profil database eksplisit `owned|supabase`. Profil Supabase
  mewajibkan TLS, migration port 5432, runtime/admin transaction port 6543,
  simple protocol, pool maksimal 2/0, dan bootstrap lokal nonaktif.
- Menambah command provisioning managed yang membuat hanya role/login Tauco
  dan schema private. Command tidak mencabut privilege `PUBLIC` atau mengubah
  role provider; tiga password dibaca interaktif dan tidak disimpan di env.
- Memperketat startup identity admin dengan exact membership, ownership,
  table privilege, dan function privilege matrix.
- Menambahkan toleransi clock skew lima detik pada pemanggilan purge retensi;
  SQL tetap menolak cutoff masa depan yang bermakna.
- Tidak membuat migration baru dan tidak membuat file unit-test baru.

### Command yang dijalankan

```powershell
npm.cmd run backend:build
npm.cmd run backend:test:integration
npm.cmd run typecheck
npm.cmd run lint
npm.cmd run test:admin
go -C backend test -count=1 -run '^TestD1' ./internal/platform/database ./internal/b4acceptance
git diff --check
```

### Hasil pengujian

```text
backend command build: PASS
targeted D1 config/JWT/database security: PASS
PostgreSQL/Redis integration suite: PASS
architecture test: PASS melalui Docker fallback untuk Windows Application Control
Admin BFF/browser E2E: 6/6 PASS (desktop dan mobile)
TypeScript: PASS
ESLint: PASS
production deployment/provider mutation: tidak dilakukan
schema migration: tetap version 7
```

### Evidence

- JWT matrix menolak wrong issuer, audience, type, key ID, key, expired token,
  missing required claim, invalid UUID, HS256, dan RS256 confusion terhadap
  manager EdDSA.
- Production auth loader tidak membaca RSA file dan menolak Ed25519 padded,
  malformed, hilang, atau berukuran salah.
- Request langsung ke Admin Go tanpa BFF secret menghasilkan `404` sebelum
  autentikasi; seluruh Admin E2E tetap lulus melalui same-origin BFF.
- Contact-disabled composition menghasilkan true `404` tanpa membutuhkan
  `CONTACT_HMAC_SECRET`.
- Supabase profile menolak bootstrap role lokal, TLS mati, port runtime 5432,
  pool terlalu besar, dan prepared statement.
- Existing local integration membuktikan exact runtime/admin role matrix tetap
  kompatibel dengan migration v1-v7.

### Known limitations

- Command managed provisioning baru divalidasi dan dikompilasi lokal; eksekusi
  terhadap akun Supabase asli baru dilakukan pada D3 oleh owner.
- Adapter S3 dan direct-upload diselesaikan pada D2; kontrak terhadap provider
  Supabase asli tetap harus dibuktikan pada D3.
- Supavisor `current_user`, inherited `TEMPORARY`, dan quota nyata harus
  dibuktikan ulang melalui remote contract test D3.
- `next-env.d.ts` adalah generated change yang sudah ada di working tree dan
  tidak menjadi bagian implementation D1.

### Owner action

Tidak ada. Jangan membuat atau menempelkan credential production pada D1.
Secret provider baru dibuat pada D3 mengikuti walkthrough yang sudah memiliki
nama variable final.

### Rollback

Set seluruh remote flag ke `false`. Local CMS tetap menggunakan RSA file,
local media, PostgreSQL owned profile, dan contact API lokal hanya bila flag
lokalnya eksplisit `true`. D1 tidak memiliki migration untuk diturunkan.

### Next gate

D2: direct media upload ke quarantine object storage, finalize, bounded
one-shot worker, variant processing, dan cleanup kedaluwarsa.

## Update D2: Direct Media Upload dan Scheduled Worker Foundation

**Tanggal:** 7 Agustus 2026

**Status:** Complete lokal

### Tujuan

Menyiapkan jalur media remote yang tidak memproksikan binary melalui Netlify
Function, sekaligus menyediakan worker singkat dan cleanup terukur yang aman
untuk dijadwalkan pada gate deployment berikutnya.

### Perubahan

- Migration v8 menambah `media_upload_intents` dengan status terbatas,
  quarantine key immutable, MIME, byte count, SHA-256, expiry, lifecycle, serta
  cleanup marker. Role admin dapat membuat intent; runtime hanya memproses dan
  tidak mempunyai direct `DELETE`.
- Admin API menambah create intent, status polling, dan finalize. Finalize
  memeriksa metadata object dan idempotently membuat durable
  `media.ingest_upload` job.
- Browser menghitung SHA-256, meminta intent melalui same-origin BFF, lalu
  mengirim binary langsung ke presigned object URL. BFF remote hanya meneruskan
  JSON kecil dan menolak multipart lama.
- S3-compatible adapter menambah presign PUT, bounded GET, HEAD, DELETE, dan
  bucket health check. Local adapter tetap tersedia untuk development.
- Media worker membaca quarantine object secara bounded, memvalidasi ulang
  ukuran, hash, dan MIME, lalu memakai pipeline image yang sudah ada untuk
  original serta variants. Invalid input gagal permanen; error dependency dapat
  di-retry.
- Source dibatasi 12,5 megapiksel dan sisi maksimal 6000 px. Normalisasi PNG
  memakai mode cepat agar setiap job CPU mempunyai headroom terhadap budget
  invocation 25 detik.
- Command one-shot memproses satu job dalam budget sekitar 25 detik. Cleanup
  memproses maksimal 25 intent per invocation, termasuk orphan dari ingest job
  yang sudah dead.
- Legacy multipart upload hanya menjadi fallback loopback lokal ketika endpoint
  intent tidak tersedia.
- Intent polling memakai backoff 5/10/15 detik; intent dan media polling berhenti
  otomatis setelah 15 menit. Media processing diperiksa maksimal setiap 15
  detik.

### Command yang dijalankan

```powershell
npm.cmd run backend:generate:check
npm.cmd run backend:format:check
npm.cmd run backend:vet
npm.cmd run backend:build
npm.cmd run backend:test:integration
npm.cmd run backend:migrate:version
npm.cmd run backend:worker:once
npm.cmd run backend:worker:ready
npm.cmd run typecheck
npm.cmd run lint
npm.cmd run test:admin
npm.cmd run build
git diff --check
```

### Hasil pengujian

```text
OpenAPI generated-code drift: PASS
Go format/vet/build: PASS
PostgreSQL dan Redis integration: PASS
migration lokal: version 8, dirty=false
worker one-shot: PASS
worker readiness DB/Redis/storage/SMTP: PASS
Admin direct-upload E2E: 7/7 PASS pada desktop, mobile, dan remote-origin BFF
TypeScript dan ESLint: PASS
Next.js production build: PASS
production deployment/provider mutation: tidak dilakukan
```

### Evidence

- Integration flow membuktikan create intent, quarantine PUT, finalize replay,
  durable ingest, variant processing, dan status `completed`.
- Ukuran, MIME, dan SHA-256 yang tidak cocok ditolak; source invalid menjadi
  kegagalan permanen tanpa retry loop.
- Expired intent dan orphan dengan dead ingest job diklaim secara bounded,
  object quarantine dihapus, lalu cleanup marker disimpan.
- E2E membuktikan remote direct PUT terjadi satu kali dan binary tidak melewati
  Admin BFF. Instance non-loopback terpisah membuktikan direct PUT/finalize
  lulus, multipart remote mendapat `404`, dan fixture mencatat legacy POST nol.
- Fixture maksimum 4000x3125 menghasilkan source 7.527.365 byte dan normalized
  object 63.560.433 byte. Normalisasi selesai 6,83 detik dan variant 4,57 detik;
  keduanya di bawah acceptance 10 detik per job.
- Polling intent dan asset memiliki interval hemat serta deadline eksplisit,
  sehingga tab Admin yang dibiarkan terbuka tidak melakukan polling selamanya.
- Readiness lulus setelah dependency lokal termasuk Mailpit dinyalakan melalui
  profile `mail`.

### Known limitations

- Belum ada bucket atau credential Supabase asli. S3 compatibility, CORS, dan
  privilege nyata baru diuji pada D3.
- Timing image harus diukur kembali pada runtime remote D4; angka lokal menjadi
  guardrail awal, bukan klaim performa provider.
- Invocation setiap dua menit dan daily cleanup belum dijadwalkan; wiring
  Netlify menjadi scope D4.
- Command cleanup terhadap database development tidak dijalankan karena audit
  read-only menemukan nol target. Penghapusan object dibuktikan pada disposable
  integration environment agar data lokal tidak dimutasi tanpa kebutuhan.
- Production Phase 1A, public content, dan Netlify Forms tidak berubah.
- `next-env.d.ts` tetap merupakan generated change lama di working tree dan
  bukan bagian D2.

### Owner action

Tidak ada pada D2. Jangan membuat atau menempelkan credential storage sebelum
langkah provisioning D3.

### Rollback

Sebelum remote cutover, nonaktifkan remote flag dan gunakan local media adapter.
Jika migration v8 perlu diturunkan, pastikan tidak ada intent aktif terlebih
dahulu lalu jalankan down-one bersama rollback kode. Belum ada schedule atau
production deployment yang perlu dibatalkan.

### Next gate

D3: owner membuat Supabase dan Upstash, lalu contract test membuktikan identity,
privilege, pooler, Redis TLS, private S3 bucket, CORS, quota, dan baseline backup.

## Gate Checklist

| Gate | Status | Exit utama |
| --- | --- | --- |
| D0 Scope/provider freeze | Complete | Plan, walkthrough, PRD, README, ADR konsisten |
| D1 Production security/config | Complete lokal | Fail-closed config dan remote role policy lulus |
| D2 Media/worker remote foundation | Complete lokal | Direct upload dan bounded worker lulus lokal |
| D3 Provider provisioning | Pending | Supabase/Upstash contract dan baseline backup lulus |
| D4 Netlify shadow runtime | Pending | Remote runtime hidup, seluruh cutover flag off |
| D5 Remote Admin CMS | Pending | Auth/TOTP/CMS/media E2E remote lulus |
| D6 API content/revalidation | Pending | API source, dynamic route, sitemap, ISR lulus |
| D7 Public content cutover | Pending | Production API content dan rollback lulus |
| D8 Verified form sync | Pending | Netlify email + exactly-once CMS mirror lulus |
| D9 Operations/DR | Pending | Monitoring, load, backup, restore, rotation lulus |
| D10 Quality closeout | Pending | Seluruh regression/evidence/docs complete |

## D0 Checklist

- [x] Generated `next-env.d.ts` dinormalkan.
- [x] Baseline adalah merge Phase 1C `216c6f9`.
- [x] Working tree bersih sebelum perubahan D0.
- [x] Branch `feature/phase-1d` dibuat.
- [x] Provider dan region pilot dibekukan.
- [x] Public content rollout dan rollback dibekukan.
- [x] Contact transport dan verified Inbox mirror dibekukan.
- [x] Media direct-upload boundary dibekukan.
- [x] Security dan test policy dibekukan.
- [x] Provider time-box dibekukan.
- [x] Plan dan walkthrough dibuat.
- [x] PRD dan README diperbarui.
- [x] ADR remote pilot dibuat.
- [x] Tidak ada deployment atau remote mutation.
- [x] Final Markdown/link verification.
- [x] `git diff --check`.
- [x] Commit lokal D0.

## D1 Checklist

- [x] Feature flag default aman dan boolean strict.
- [x] Production HTTPS, secure cookie, TLS Redis, secret, dan byte budget.
- [x] Contact API tidak diregistrasikan ketika flag mati.
- [x] Admin tidak aktif hanya karena database URL tersedia.
- [x] Ed25519 production dengan RSA fallback local/test.
- [x] Shared secret BFF diterapkan pada seluruh Admin Go route.
- [x] Supabase pooler profile dan pool kecil diterapkan.
- [x] Managed provisioning tidak mengubah privilege global provider.
- [x] Exact admin/runtime identity matrix lulus PostgreSQL integration.
- [x] Admin desktop/mobile E2E lulus.
- [x] Tidak ada deployment, provider mutation, atau schema migration baru.
- [x] `git diff --check`.

## D2 Checklist

- [x] Migration v8 upload intent dan quarantine lifecycle diterapkan.
- [x] Least-privilege matrix admin/runtime untuk intent lulus.
- [x] Create, poll, dan idempotent finalize API tersedia di OpenAPI.
- [x] Browser direct PUT tidak memproksikan binary melalui BFF remote.
- [x] S3/local adapter mendukung bounded read dan health operation.
- [x] Worker one-shot memproses maksimal satu job dalam budget invocation.
- [x] Fixture 12,5 MP membuktikan normalisasi dan variant di bawah 10 detik per job.
- [x] Invalid bytes, MIME, hash, serta oversize gagal aman.
- [x] Expired dan dead-job orphan cleanup lulus integration test.
- [x] Polling Admin memakai backoff, interval hemat, dan deadline 15 menit.
- [x] Admin desktop/mobile/remote-origin direct-upload E2E 7/7 lulus.
- [x] Multipart legacy remote ditolak dengan true `404`.
- [x] TypeScript, lint, Go checks, integration, dan production build lulus.
- [x] Tidak ada provider mutation, schedule, deployment, atau public cutover.
- [x] Walkthrough mencatat limitation, rollback, dan next gate D3.
- [x] `git diff --check`.
