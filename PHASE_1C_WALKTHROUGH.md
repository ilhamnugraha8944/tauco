# Phase 1C Walkthrough

## Admin CMS, Authentication, dan Publishing Lokal

| Atribut | Nilai |
| --- | --- |
| Tanggal mulai | 4 Agustus 2026 |
| Branch | `feature/phase-1c` |
| Baseline | `520d315` |
| Status | C0-C9 complete; C10 pending |
| Production | Tidak berubah |

Dokumen ini menjadi ledger pelaksanaan Phase 1C. Setiap gate diperbarui setelah
perubahan dan pemeriksaan relevan benar-benar selesai.

## Format Update Gate

```markdown
## Update C<n>: <Nama Gate>

**Tanggal:**
**Status:** In Progress | Complete | Blocked

### Tujuan
### Perubahan
### Keputusan arsitektur
### Command yang dijalankan
### Hasil pengujian
### Evidence
### Known limitations
### Next gate
```

## Update C0: Scope Freeze

**Tanggal:** 4 Agustus 2026
**Status:** Complete

### Tujuan

Membekukan scope, boundary, kontrak pelaksanaan, security baseline, gate, dan
acceptance Phase 1C sebelum migration, endpoint, auth, atau UI admin dibuat.

### Perubahan

- Membuat `PHASE_1C_PLAN.md` sebagai kontrak implementasi C0-C10.
- Membuat `PHASE_1C_WALKTHROUGH.md` sebagai progress/evidence ledger.
- Mengubah status PRD menjadi `IMPLEMENTING-1C-LOCAL`.
- Menambahkan status dan tautan Phase 1C pada README.
- Membuat branch lokal `feature/phase-1c` dari Phase 1B commit `520d315`.

### Keputusan arsitektur

- Phase 1C tetap local-first shadow-mode.
- Website production tetap memakai `LocalContentSource` dan Netlify Forms.
- Home dan About editable; `/tauco` dan shell katalog read-only.
- Product, media, inbox, activity, dan account termasuk scope.
- Password dan TOTP wajib; akun awal dibuat melalui CLI.
- Form terstruktur; tidak ada raw JSON, rich text, Markdown, atau autosave.
- Setiap Save Draft menghasilkan immutable revision.
- Tidak memakai Supabase Auth.
- Tidak membuat unit-test file; regression memakai integration/contract/E2E.
- Deployment dan production cutover tetap Phase 1D.

### Command yang dijalankan

```powershell
git status --short --branch
git rev-parse HEAD
git log -1 --oneline
git switch -c feature/phase-1c
rg -n "Phase 1C|IMPLEMENTING-1C" PRD.md README.md PHASE_1C_PLAN.md PHASE_1C_WALKTHROUGH.md
git diff --check
```

### Hasil pengujian

```text
baseline commit: 520d315d422c4b543e87f32ef0d33a945cd8673b
working tree sebelum C0: bersih
branch: feature/phase-1c
production source adapter: tidak berubah
production contact adapter: tidak berubah
migration/API/auth/admin UI: tidak dibuat pada C0
```

### Evidence

- Plan mencatat scope, security, data/API target, gate, dan acceptance.
- Walkthrough menyediakan checklist C0-C10 dan format update wajib.
- PRD membedakan status Phase 1A, Phase 1B, Phase 1C lokal, serta Phase 1D.
- README mengarahkan contributor ke plan dan walkthrough Phase 1C.
- Final `git diff --check` wajib lulus sebelum commit C0.

### Known limitations

- Belum ada migration auth/RBAC/session/MFA.
- Belum ada endpoint atau UI admin.
- Belum ada akun admin atau credential.
- Inbox Phase 1C kelak hanya membaca database lokal sampai contact cutover.
- Tidak ada deployment atau perubahan production.

### Next gate

C1: OpenAPI dan database foundation.

## C0 Checklist

- [x] Baseline adalah Phase 1B commit `520d315`.
- [x] Working tree bersih sebelum C0.
- [x] Branch `feature/phase-1c` dibuat.
- [x] Scope local shadow-mode dibekukan.
- [x] Production adapter tetap dibekukan.
- [x] Editable content dibekukan.
- [x] Authentication dan TOTP policy dibekukan.
- [x] Revision dan editor policy dibekukan.
- [x] Testing policy dibekukan.
- [x] `PHASE_1C_PLAN.md` dibuat.
- [x] `PHASE_1C_WALKTHROUGH.md` dibuat.
- [x] PRD diamendemen.
- [x] README diperbarui.
- [x] Final document/link/diff verification.
- [x] Commit C0.

## Update C1: OpenAPI dan Database Foundation

**Tanggal:** 4 Agustus 2026

**Status:** Complete

### Tujuan

Membuat kontrak API admin dan fondasi data/security CMS yang dapat dipakai gate
C2-C8 tanpa mengaktifkan route admin, mengubah frontend, atau menyentuh
production Phase 1A.

### Perubahan

- OpenAPI v1.1 memuat auth, Home/About, product, media, inbox, activity log,
  public ready-media path, cursor pagination, RFC7807, CSRF, dan `If-Match`.
- Generated Go contract diregenerasi secara deterministik.
- Migration `000005_admin_cms_foundation` menambah admin user, RBAC, session,
  refresh-token hash, encrypted TOTP storage, recovery-code hash, dan index.
- Menambah `page_revision_media`, `product_revision_media`, serta
  `products.archived_at`.
- Semua page/product revision dan revision-media link immutable setelah insert.
- Menambah NOLOGIN role `tauco_admin_runtime` dan login lokal terpisah
  `tauco_admin_local` melalui `ADMIN_DATABASE_URL`.
- Integration suite memeriksa migration up/down/up, schema, revision
  immutability, role membership, dan privilege publik/admin.
- Dokumentasi status Phase 1C dan setup backend diperbarui.

### Keputusan arsitektur

- C1 hanya contract dan persistence foundation; generated admin handler tidak
  diregistrasikan ke Gin.
- Public runtime tetap memakai `tauco_runtime`; CMS kelak memakai
  `tauco_admin_runtime`; migrator tetap satu-satunya schema owner.
- Password hanya menerima hash Argon2id. Refresh token, CSRF token, dan recovery
  code hanya disimpan sebagai hash. TOTP secret disimpan sebagai ciphertext,
  nonce, dan key identifier.
- Recovery-code regeneration memakai `revoked_at`, bukan hard delete.
- Publish kelak membuat immutable published snapshot dan mengubah pointer;
  revision draft yang sudah tersimpan tidak di-update menjadi published.
- `/tauco` tetap read-only dan tidak memiliki route admin.

### Command yang dijalankan

```powershell
docker compose -f backend/compose.yaml up -d --wait
npm.cmd run backend:generate
go test ./internal/delivery/api -run "TestEmbeddedOpenAPIContract|TestAdminOpenAPISecurityAndConcurrencyContract|TestOpenAPIOperationRequirements" -count=1
npm.cmd run backend:test
npm.cmd run backend:test:integration
npm.cmd run backend:migrate:down
npm.cmd run backend:migrate:up
npm.cmd run backend:migrate:version
npm.cmd run backend:generate:check
npm.cmd run backend:vet
npm.cmd run backend:build
git diff --check
```

### Hasil pengujian

```text
OpenAPI validation + admin security contract: PASS
generated OpenAPI artifact reproducible: PASS
backend regression suite: PASS
fresh database integration dan migration round-trip: PASS
public/admin database privilege matrix: PASS
local migration: version=5 dirty=false
go vet: PASS
backend cmd build: PASS
```

Windows Application Control memblokir satu executable test temporary pada run
awal. Runner repository otomatis mengulang package tersebut dari workspace;
hasil fallback `internal/architecture` adalah PASS.

### Evidence

- Kontrak memiliki 41 operasi unik dan operation ID unik.
- Semua admin operation selain login memerlukan cookie auth; seluruh unsafe
  operation juga memerlukan CSRF.
- Mutation yang rawan lost update mewajibkan strong `If-Match`.
- Fresh DB test memverifikasi 11 tabel C1, 12 permission `super_admin`, role
  terpisah, draft immutability, serta penolakan DDL/hard delete admin.
- Migration lokal final berada pada v5 dan tidak dirty.
- Integration test memastikan admin login dan public media C1 masih `404` pada
  runtime Phase 1B karena handler belum diregistrasikan.
- File `backend/.env` lokal diperbarui tetapi tetap ignored dan tidak masuk Git.
- Tidak ada unit-test file baru; evidence memakai contract dan integration test.
- Production adapter, Netlify Forms, canonical, sitemap, dan deploy tidak diubah.

### Known limitations

- Belum ada Argon2/JWT/TOTP implementation atau bootstrap admin CLI; itu C2.
- Belum ada auth middleware atau handler admin aktif; itu C3.
- Belum ada BFF, UI CMS, upload handler, editor, inbox UI, atau cutover.
- OpenAPI public media path baru merupakan contract dan belum diregistrasikan.
- Password dan secret contoh di `.env.example` hanya untuk loopback lokal.

### Next gate

C2: auth domain dan CLI.

## C1 Checklist: OpenAPI dan Database Foundation

- [x] Admin OpenAPI paths dan schemas.
- [x] Auth/RBAC/session/MFA migration.
- [x] Admin DB authorization role.
- [x] Revision-media relationship.
- [x] Product archive field.
- [x] Revision immutability enforcement.
- [x] Migration up/down.
- [x] Fresh database integration.
- [x] Privilege matrix.
- [x] OpenAPI drift check.
- [x] Walkthrough update C1.

## Update C2: Auth Domain dan CLI

**Tanggal:** 4 Agustus 2026

**Status:** Complete

### Tujuan

Menyediakan domain authentication dan recovery operator yang dapat diuji tanpa
mengaktifkan endpoint admin atau mengubah production.

### Perubahan

- Password memakai Argon2id dengan parameter yang dibekukan di plan.
- Access token memakai RS256 dengan validasi `alg`, `typ`, `kid`, issuer,
  audience, seluruh claim waktu, user ID, session ID, dan JWT ID.
- Refresh/CSRF token dibuat dari CSPRNG, hanya hash SHA-256 disimpan, dan reuse
  mencabut seluruh session secara transaction-safe.
- TOTP RFC 6238 disimpan terenkripsi AES-256-GCM dan counter dipakai sekali.
- Sepuluh recovery code disimpan sebagai HMAC-SHA256 dan dipakai sekali.
- CLI lokal menyediakan keygen, bootstrap, reset password, reset TOTP, dan
  revoke session. Password dibaca dari stdin, bukan process argument.
- Migration v6 menambah authentication level session dan lifecycle pencabutan
  credential MFA.
- Koneksi `ADMIN_DATABASE_URL` diverifikasi sebagai role admin least-privilege.

### Keputusan arsitektur

- Secret key lokal berada di `backend/.local-auth/` dan di-ignore Git.
- Runtime public dan admin memakai pool serta identity assertion terpisah.
- Refresh reuse harus commit pencabutan sebelum mengembalikan error generik;
  rollback transaksi tidak boleh membatalkan tindakan protektif tersebut.
- C2 belum mendaftarkan route admin apa pun.

### Command yang dijalankan

```powershell
npm.cmd run backend:admin -- keygen .local-auth/jwt-private.pem .local-auth/jwt-public.pem
npm.cmd run backend:migrate:up
npm.cmd run backend:migrate:version
npm.cmd run backend:format
npm.cmd run backend:build
npm.cmd run backend:test:integration
git diff --check
```

### Hasil pengujian

```text
migration: version=6 dirty=false
backend command build: PASS
fresh database auth lifecycle: PASS
Argon2id bootstrap/login: PASS
TOTP setup/enable/replay rejection: PASS
recovery-code one-time use: PASS
refresh rotation/reuse revocation: PASS
Windows Application Control fallback: PASS
```

### Evidence

- Integration suite membuat database disposable dan role admin terpisah.
- Access baru valid setelah MFA, TOTP/recovery replay ditolak dengan error
  generik, dan reuse refresh lama membuat access session hasil rotasi invalid.
- RSA private key, `.env`, dan nilai runtime lokal tidak masuk Git.
- Tidak ada unit-test file baru dan production Phase 1A tidak berubah.

### Known limitations

- Endpoint auth/cookie/CSRF/rate limit belum diaktifkan; itu C3.
- Belum ada akun admin permanen karena email/password owner belum diberikan.
- Key C2 hanya untuk local development dan bukan key production.

### Next gate

C3: auth API dan security middleware.

## C2 Checklist: Auth Domain dan CLI

- [x] Argon2id password hashing.
- [x] RS256 key generation/loading.
- [x] JWT claims dan validation.
- [x] Refresh token hashing/rotation model.
- [x] AES-GCM TOTP secret storage.
- [x] TOTP replay protection.
- [x] Recovery code generation/consumption.
- [x] Admin bootstrap CLI.
- [x] Password/TOTP/session recovery CLI.
- [x] Auth integration evidence.
- [x] Walkthrough update C2.

## Update C3: Auth API dan Security Middleware

**Tanggal:** 4 Agustus 2026

**Status:** Complete

### Tujuan

Mengaktifkan kontrak auth admin pada API lokal dengan browser security boundary,
authorization, rate limiting, dan evidence abuse tanpa membuat UI atau BFF.

### Perubahan

- Mengaktifkan tujuh route auth: login, setup/enable TOTP, refresh, logout, me,
  dan recovery-code regeneration.
- Access/refresh disimpan pada cookie HttpOnly HostOnly; CSRF memakai cookie
  terpisah. Seluruh cookie `SameSite=Strict`, `Path=/`, dan Secure configurable.
- Mutation memerlukan exact Origin atau Referer, menolak Fetch Metadata
  `cross-site`, serta membandingkan header/cookie/hash CSRF constant-time.
- Login dibatasi 5/15 menit untuk HMAC IP+email; authenticated admin dibatasi
  120/menit per admin melalui Redis dengan bounded local fallback.
- Middleware permission reusable memeriksa permission dari session principal.
- Seluruh response auth memakai `Cache-Control: no-store` dan error RFC7807.
- Refresh reuse, login, setup, enable, refresh, logout, dan recovery dicatat
  sebagai activity event tanpa memasukkan token atau secret.
- Composition membuka pool admin terpisah hanya ketika konfigurasi C3 tersedia.

### Keputusan arsitektur

- API Go tetap authorization authority; UI/BFF C4 tidak akan menentukan role.
- Route C3 menggunakan application service, tidak mengakses GORM dari handler.
- Production tetap tidak berubah dan tidak menerima credential C3.
- `Secure=false` hanya untuk HTTP loopback lokal; deployment HTTPS kelak wajib
  `ADMIN_COOKIE_SECURE=true` pada Phase 1D.

### Command yang dijalankan

```powershell
npm.cmd run backend:format
npm.cmd run backend:test
npm.cmd run backend:test:integration
npm.cmd run backend:generate:check
npm.cmd run backend:vet
npm.cmd run backend:build
npm.cmd run backend:migrate:version
git diff --check
```

### Hasil pengujian

```text
seven auth routes: PASS
cookie HostOnly/HttpOnly/SameSite/Path policy: PASS
TOTP setup dan enable: PASS
refresh rotation dan logout: PASS
recovery-code regeneration: PASS
CSRF missing: 403 PASS
cross-site Origin/Fetch Metadata: 403 PASS
permission missing: 403 PASS
login attempt ke-6: 429 PASS
auth activity evidence: PASS
Phase 1B integration regression: PASS
```

### Evidence

- Security flow dijalankan melalui HTTP terhadap database disposable dan
  session/cookie jar nyata.
- Permission dicabut melalui migrator fixture; endpoint account mengembalikan
  403 sebelum use case dijalankan.
- Login response tidak membedakan email, password, TOTP, atau recovery code
  yang salah.
- Local fallback limiter tetap bounded jika Redis tidak tersedia.
- Tidak ada unit-test file baru, push, deploy, atau production cutover.

### Known limitations

- Belum ada Next.js BFF, halaman login, atau admin shell; itu C4.
- Belum ada bootstrap akun owner permanen.
- HTTPS cookie behavior diverifikasi lewat atribut/configuration, bukan deploy.
- C3 tetap shadow-mode lokal.

### Next gate

C4: same-origin BFF dan admin shell.

## C3 Checklist: Auth API dan Security Middleware

- [x] Login.
- [x] TOTP setup/enable.
- [x] Refresh/logout/me.
- [x] Recovery code regeneration.
- [x] Cookie policy.
- [x] CSRF dan exact origin.
- [x] Fetch Metadata enforcement.
- [x] Login/admin rate limit.
- [x] RBAC/permission middleware.
- [x] Auth activity audit.
- [x] Security abuse evidence.
- [x] Walkthrough update C3.

## Update C4: Same-Origin BFF dan Admin Shell

**Tanggal:** 4 Agustus 2026

**Status:** Complete

### Tujuan

Menyediakan pintu masuk CMS lokal yang aman dan dapat digunakan melalui
browser, tanpa memindahkan authorization dari Go API dan tanpa membuka route
admin pada production Phase 1A.

### Perubahan

- Menambahkan feature flag server-only `ADMIN_CMS_ENABLED` dengan default
  `false`; semua route `/admin` dan `/admin-api` menghasilkan 404 ketika flag
  tidak aktif.
- Menambahkan BFF catch-all dengan allowlist path dan method untuk tujuh auth
  route C3. BFF meneruskan header yang dibutuhkan saja, termasuk cookie,
  `Set-Cookie`, CSRF, Origin, dan request ID.
- Menambahkan halaman login, enrollment TOTP, recovery code satu kali, shell
  navigasi, halaman akun, regenerasi recovery code, dan logout.
- Menambahkan refresh-and-retry satu kali pada response 401 tanpa menyimpan
  access atau refresh token di Web Storage.
- Menambahkan `noindex`, `nofollow`, `noarchive`, `no-store`, batas payload 64
  KiB, timeout upstream, dan response 502 generik.
- Menambahkan layout responsive light/dark berbasis preferensi sistem dengan
  keyboard focus, reduced-motion, native form control, dan Phosphor Icons.
- Menambahkan suite Playwright C4 berbasis HTTP fixture terpisah. Suite admin
  dan suite publik memakai konfigurasi/environment yang terpisah.

### Keputusan arsitektur

- Go API tetap menjadi satu-satunya authorization authority; Next.js BFF tidak
  menafsirkan role atau permission.
- `ADMIN_API_ORIGIN` bersifat server-only dan tidak memakai prefix
  `NEXT_PUBLIC_`.
- BFF tidak menjadi proxy generik. Path dan method di luar kontrak auth C3
  ditolak sebelum request mencapai backend.
- UI C4 tidak menambah component library, form library, state library, atau
  dependency baru.
- Menu C5-C8 ditampilkan sebagai status belum tersedia, bukan route kosong.
- CMS lokal memakai visual publik yang sudah ada dan tidak membuat aset brand
  atau dokumentasi produk baru.

### Command yang dijalankan

```powershell
npm.cmd run lint
npm.cmd run typecheck
npm.cmd run test:admin
$env:ADMIN_CMS_ENABLED='false'; npm.cmd run build
npm.cmd run test:e2e
git diff --check
```

Production-disabled smoke dijalankan pada build dengan flag `false`:

```text
GET /             -> 200
GET /admin/login  -> 404
```

### Hasil pengujian

```text
ESLint: PASS
TypeScript: PASS
admin production build: PASS
admin desktop/mobile E2E: 6 passed
admin axe light/dark: PASS
BFF unknown path: 404 PASS
BFF unsupported method: 405 PASS
production-disabled admin route: 404 PASS
Phase 1A regression: 79 passed, 13 expected browser-project skips
```

### Evidence

- Login menghasilkan cookie melalui response upstream dan TOTP enrollment
  mengarahkan user ke shell hanya setelah faktor kedua aktif.
- Navigasi langsung ke shell dengan session password-level dikembalikan ke
  setup TOTP; shell baru terbuka setelah `mfaEnabled` terverifikasi.
- Recovery code ditampilkan sekali pada enrollment dan dapat diregenerasi dari
  halaman akun dengan TOTP.
- Header admin membawa `Cache-Control: private, no-store` dan
  `X-Robots-Tag: noindex, nofollow, noarchive`; metadata admin juga noindex.
- Browser desktop dan emulasi Pixel 7 menyelesaikan login, setup, navigation,
  account, recovery regeneration, serta logout.
- Axe tidak menemukan pelanggaran WCAG A/AA pada login light/dark maupun shell.
- Screenshot desktop dan mobile ditinjau; tidak ada overflow, broken image,
  atau public header/footer di area admin.
- Build mode default mempertahankan homepage 200 dan menyembunyikan CMS dengan
  true 404.
- Seluruh 79 regression test publik tetap lulus dengan CMS disabled.
- Tidak ada deployment, push, content cutover, contact cutover, atau credential
  lokal yang masuk Git.

### Known limitations

- C4 hanya menyediakan auth dan shell. Media, editor page/product, inbox, dan
  activity log mulai C5-C8.
- E2E frontend memakai fixture HTTP deterministik; behavior auth Go terhadap
  PostgreSQL/Redis sudah diuji terpisah pada C3.
- Akun owner lokal tetap harus dibuat melalui CLI dan TOTP harus dihubungkan
  ke aplikasi autentikator milik owner.
- HTTPS Secure-cookie dan remote origin belum diuji karena deployment tetap
  milik Phase 1D.

### Next gate

C5: Media CMS.

## C4 Checklist: BFF dan Admin Shell

- [x] `ADMIN_CMS_ENABLED` default false.
- [x] Same-origin admin BFF.
- [x] Exact path/method allowlist.
- [x] Cookie/CSRF/request ID forwarding.
- [x] Login dan setup TOTP UI.
- [x] Admin layout/navigation/account/logout.
- [x] Noindex/no-store.
- [x] Production-disabled 404/build.
- [x] Responsive/accessibility baseline.
- [x] Walkthrough update C4.

## Update C5: Media CMS

**Tanggal:** 4 Agustus 2026

**Status:** Complete

### Tujuan

Menyediakan pustaka media lokal yang aman, asynchronous, dapat dipakai ulang
oleh editor CMS, dan tidak pernah mengekspos file original melalui route publik.

### Perubahan

- Mengaktifkan list, detail, upload multipart, retry, display, dan variant route
  dari kontrak C1 dengan auth MFA, RBAC, CSRF, exact Origin, dan rate limit.
- Memakai kembali normalizer, object store, durable job, worker, serta generator
  WebP C1/B8; HTTP response upload tidak menunggu resize selesai.
- Menambahkan query admin dengan cursor HMAC, status, error code, dan varian.
- Menambahkan public route yang hanya membaca varian dari asset `ready`, membawa
  ETag, dan tidak memiliki route untuk original.
- Memperbaiki sumber lebih kecil dari 320px agar tetap menghasilkan satu WebP
  display pada lebar aslinya tanpa upscaling.
- Menambahkan BFF allowlist khusus media dengan batas upload 10 MiB, penerusan
  binary/ETag, serta tetap menolak path lain.
- Menambahkan UI pustaka media, upload, status polling hanya saat diperlukan,
  retry eksplisit, empty/error/loading state, dan media picker reusable.

### Keputusan arsitektur

- API Go tetap memegang validasi, authorization, state transition, dan URL
  publik; Next.js hanya BFF same-origin dan presentation.
- File diperiksa lewat magic bytes dan full decode. JPEG, PNG, dan WebP statis
  dinormalisasi menjadi PNG private sebelum worker membuat WebP 320/640/1280.
- Upload identik menggunakan object key checksum dan tidak membuat job ganda.
- Retry hanya berlaku untuk status `failed`, mereset durable job secara
  transaction-safe, dan mencatat actor admin ke activity log.
- Tidak ada dependency frontend/backend baru untuk C5.

### Command yang dijalankan

```powershell
npm.cmd run backend:format
npm.cmd run backend:generate:check
npm.cmd run backend:test
npm.cmd run backend:test:integration
npm.cmd run backend:vet
npm.cmd run backend:build
npm.cmd run typecheck
npm.cmd run lint
npm.cmd run test:admin
git diff --check
```

### Hasil pengujian

```text
OpenAPI generated drift: PASS
backend repository/delivery/composition regression: PASS
PostgreSQL media integration: PASS
valid image ingest, dedupe, durable job, WebP worker: PASS
admin list/detail, ready display, failed retry/conflict: PASS
invalid magic bytes dan payload >10 MiB: REJECTED
frontend TypeScript/ESLint/build: PASS
desktop/mobile admin E2E: 6 passed
media upload, thumbnail, polling state, axe WCAG A/AA: PASS
go vet dan backend command build: PASS
```

Windows Application Control kembali memblokir beberapa executable Go temporary;
runner resmi mengulang package terdampak dari workspace dan seluruh fallback
berakhir `PASS`.

### Evidence

- Integration database disposable membuktikan satu upload membuat satu asset,
  satu durable job, varian deterministik, dan activity log idempotent.
- Replay worker tidak menggandakan varian atau activity log.
- Admin list hanya memakai pool `tauco_admin_runtime`; worker tetap memakai
  pool runtime yang terpisah.
- Asset `processing` atau `failed` menghasilkan 404 pada route display publik.
- Original tersimpan di object key private dan tidak memiliki HTTP route.
- E2E desktop dan mobile menyelesaikan login MFA, membuka Media, upload file,
  melihat thumbnail siap, dan tetap axe-clean.
- BFF tetap memiliki exact path/method allowlist dan feature flag default false.
- Tidak ada deploy, push, public content cutover, contact cutover, atau secret
  lokal yang masuk Git.

### Known limitations

- Pemrosesan asynchronous nyata memerlukan API dan worker berjalan bersamaan.
- Storage C5 masih filesystem lokal; adapter S3-compatible tetap disiapkan untuk
  Phase 1D tanpa mengubah use case.
- Media picker baru dipakai oleh editor mulai C6.
- Production Phase 1A tetap tidak berubah dan CMS masih shadow-mode lokal.

### Next gate

C6: structured editor, immutable revision, preview, dan publishing Home/About.

## C5 Checklist: Media CMS

- [x] Authenticated multipart upload.
- [x] Existing image pipeline reused.
- [x] Media list/detail/status polling.
- [x] Retry failed media.
- [x] Media picker.
- [x] Ready display/variant route.
- [x] Original remains private.
- [x] Upload abuse evidence.
- [x] Walkthrough update C5.

## Update C6: Homepage dan About CMS

**Tanggal:** 4 Agustus 2026

**Status:** Complete

### Tujuan

Menyediakan editor terstruktur untuk Home dan About dengan revision immutable,
optimistic concurrency, preview terautentikasi, serta publish/unpublish lokal
tanpa mengalihkan website production dari konten Phase 1A.

### Perubahan

- Menambahkan use case dan repository admin untuk membaca halaman, menyimpan
  draft immutable, melihat revision, publish, dan unpublish.
- Mengunci singleton page saat mutation serta mewajibkan strong `ETag` dan
  `If-Match`, sehingga draft dari tab lama ditolak dengan `412`.
- Memvalidasi payload dengan schema OpenAPI dan aturan konten yang sama dengan
  frontend sebelum revision disimpan.
- Menolak publish bila media yang direferensikan belum berstatus `ready`.
- Membuat snapshot published baru tanpa mengubah revision lama, mencatat actor
  ke activity log, dan mengantrekan durable job `content.invalidate_cache`.
- Menambahkan BFF allowlist khusus Home/About serta menolak mutation `/tauco`.
- Menambahkan editor form terstruktur, media picker, revision history/detail,
  fact-check confirmation, preview, publish, dan unpublish.
- Memakai komponen presentation yang sama pada preview CMS dan halaman publik
  agar struktur hasil preview tidak menyimpang dari implementasi website.

### Keputusan arsitektur

- Next.js tetap bertindak sebagai same-origin BFF dan presentation layer; Go
  tetap memegang validation, concurrency, transaction, audit, dan publishing.
- Setiap Save Draft selalu menghasilkan revision baru. Revision lama tidak
  pernah diperbarui atau dihapus oleh workflow C6.
- Preview membaca revision tertentu, bukan mutable page state.
- Checkbox fact-check wajib dikonfirmasi ulang setelah isi berubah dan sebelum
  Save Draft atau Publish.
- `/tauco` tetap dibaca dari sumber lokal dan tidak mempunyai editor/mutation.
- Tidak ada dependency baru; form native dan Zod yang sudah ada digunakan
  kembali sesuai keputusan Ponytail/YAGNI.

### Command yang dijalankan

```powershell
npm.cmd run backend:format
npm.cmd run backend:generate:check
npm.cmd run backend:test
npm.cmd run backend:test:integration
npm.cmd run backend:vet
npm.cmd run backend:build
npm.cmd run typecheck
npm.cmd run lint
npm.cmd run test:e2e
npm.cmd run test:admin
git diff --check
```

### Hasil pengujian

```text
OpenAPI generated drift: PASS
PostgreSQL content lifecycle integration: PASS
concurrent Save Draft: satu sukses, satu PRECONDITION_FAILED
immutable revision dan published snapshot: PASS
media processing saat publish: REJECTED
publish/unpublish, audit, durable invalidation enqueue: PASS
frontend TypeScript/ESLint/build: PASS
Phase 1A regression: 79 passed, 13 configured skips
desktop/mobile admin E2E: 6 passed
editor Home/About, preview, publish/unpublish, light/dark axe: PASS
go vet dan backend command build: PASS
```

### Evidence

- Dua writer dengan ETag sama diuji terhadap PostgreSQL disposable; hanya satu
  draft yang tersimpan dan writer kedua menerima precondition failure.
- Publish menghasilkan revision snapshot baru, memperbarui pointer published,
  mencatat activity log, dan membuat satu job invalidasi durable.
- Unpublish membersihkan pointer published dan membuat audit serta job
  invalidasi terpisah.
- E2E desktop dan mobile mengedit Home dan About melalui form terstruktur,
  menyimpan draft, membuka preview terautentikasi, publish, lalu unpublish.
- Mutation `/tauco-guide` melalui BFF menghasilkan 404.
- Seluruh 79 regression test publik tetap lulus setelah presentation component
  Home/About dipakai bersama oleh public page dan preview.
- Tidak ada deploy, push, content cutover, contact cutover, atau secret lokal
  yang masuk Git.

### Known limitations

- Handler worker untuk menjalankan job `content.invalidate_cache` baru dibuat
  pada C9; C6 sudah membuat durable job secara transaction-safe.
- Editor dan API masih local shadow-mode. Production Phase 1A tetap membaca
  `LocalContentSource` dan tidak berubah saat C6 selesai.
- Product, inbox, activity viewer, recovery, dan operational closeout masih
  menjadi scope C7-C10.

### Next gate

C7: Product CMS dengan lifecycle draft, publish, archive, stable slug, media,
dan public exclusion.

## C6 Checklist: Homepage dan About CMS

- [x] Structured Home editor.
- [x] Structured About editor.
- [x] Immutable Save Draft.
- [x] Revision history/detail.
- [x] Optimistic conflict.
- [x] Authenticated preview.
- [x] Publish/unpublish.
- [x] Media readiness.
- [x] Fact-check confirmation.
- [x] Audit dan cache invalidation job.
- [x] `/tauco` tetap read-only.
- [x] Walkthrough update C6.

## C7 Checklist: Product CMS

**Tanggal:** 5 Agustus 2026

**Status:** Complete

### Hasil C7

- Go API dan Next.js CMS menyediakan list, create, detail, identity update,
  structured editor, immutable draft, history, preview, publish/unpublish,
  serta archive/unarchive produk.
- Slug menjadi immutable setelah publish pertama; SKU uppercase dan sort order
  divalidasi pada browser dan service.
- Canonical path wajib mengikuti identity slug. Referensi media divalidasi dan
  publish menolak asset yang belum `ready`.
- Publish/unpublish mencatat audit dan mengantrekan invalidasi untuk tag
  `products` serta `product:{slug}` dalam transaction yang sama.
- Query public mengecualikan produk archived.
- Preview memakai presentation component yang sama dengan halaman publik.

### Verifikasi C7

```text
OpenAPI drift, format, typecheck, ESLint, Go build, Next build: PASS
Repository integration via Linux container: PASS
Admin Playwright desktop/mobile: 6 passed
Product editor dan preview axe A/AA: PASS
Windows Application Control: native test blocked; container fallback PASS
```

- [x] Product list/create/detail.
- [x] Identity update.
- [x] Structured product editor.
- [x] Draft/history/preview.
- [x] Publish/unpublish.
- [x] Archive/unarchive.
- [x] Stable slug after first publish.
- [x] SKU/sort validation.
- [x] Media/canonical validation.
- [x] Archived public exclusion.
- [x] Walkthrough update C7.

### Limitation dan next gate

- Handler worker untuk job invalidasi tetap menjadi scope C9.
- Deployment dan public content cutover tetap menjadi scope Phase 1D.
- Gate berikutnya: C8 Inbox dan Activity Log lokal.

## C8 Checklist: Inbox dan Activity Log

**Tanggal:** 5 Agustus 2026

**Status:** Complete

### Hasil C8

- Inbox memiliki list ber-cursor, filter status, detail, dan explicit status
  mutation `unread`, `read`, atau `archived`.
- GET inbox murni read-only. Status berubah hanya lewat PATCH dengan CSRF dan
  strong ETag; update stale mendapat `412`.
- Activity viewer memiliki cursor serta filter exact `eventType` dan
  `entityType` yang terikat pada signature cursor.
- API activity hanya memilih field allowlist dan sengaja tidak membaca atau
  mengirim `metadata_json`. Audit status hanya menyimpan status lama/baru.
- UI Inbox dan Activity responsif, keyboard-accessible, dan axe-clean pada
  desktop serta mobile.

### Verifikasi C8

```text
OpenAPI generated drift, Go build, typecheck, ESLint, Next build: PASS
PostgreSQL repository integration via Linux container: PASS
GET read-only, ETag conflict, status/filter, PII allowlist: PASS
Admin Playwright desktop/mobile: 6 passed
Inbox dan Activity axe A/AA: PASS
```

- [x] Inbox cursor/filter.
- [x] Message detail.
- [x] Explicit status mutation.
- [x] GET remains read-only.
- [x] Activity cursor/filter.
- [x] Metadata allowlist.
- [x] PII redaction.
- [x] Local inbox limitation documented.
- [x] Walkthrough update C8.

### Limitation dan next gate

- Inbox ini hanya membaca `contact_messages` lokal. Pesan production masih
  berada di Netlify Forms dan tidak disalin otomatis selama Phase 1C.
- Contact cutover dan migrasi inbox production tetap menjadi Phase 1D.
- Gate berikutnya: C9 publishing worker, recovery, runbook, dan operations.

## C9 Checklist: Publishing, Recovery, dan Operations

**Tanggal:** 5 Agustus 2026

**Status:** Complete

### Hasil C9

- Worker kini memiliki handler `content.invalidate_cache` dan Redis menjadi
  dependency readiness worker bersama PostgreSQL, media storage, dan SMTP.
- Payload legacy page (`generationTag`) dan payload product
  (`generationTags`) divalidasi ketat. Tag yang diizinkan hanya `home`,
  `about`, `products`, dan `product:{slug}`; field asing ditolak.
- Retry invalidasi aman: generation hanya bergerak maju sehingga eksekusi
  ulang tidak dapat menghidupkan cache lama. Dead job tetap direplay melalui
  command ops yang sudah tersedia setelah penyebab diperbaiki.
- Metrics menambahkan jumlah session admin `active/expired/revoked` dan state
  publishing page/product dengan label bounded tanpa PII.
- `PHASE_1C_RUNBOOK.md` mendokumentasikan password/TOTP/session recovery,
  revision conflict, media retry, dead job replay, cache recovery, dan metrics.
- Boundary Phase 1D tetap eksplisit: tidak ada Next.js revalidation, remote
  provider, deploy, atau production cutover pada C9.

### Verifikasi C9

```text
Worker readiness PostgreSQL/Redis/media/SMTP: PASS
OpenAPI drift, Go format/build, TypeScript, ESLint: PASS
Cache payload allowlist dan repeat-safe generation integration: PASS
PostgreSQL repository integration via Linux container: PASS
Composition API, Redis, metrics publishing integration: PASS
```

- [x] Durable cache invalidation handler.
- [x] Page/product generation tags.
- [x] Idempotent retry/dead-letter recovery.
- [x] Auth/TOTP recovery runbook.
- [x] Revision conflict runbook.
- [x] Media retry runbook.
- [x] Admin/session/publish metrics.
- [x] Phase 1D revalidation boundary preserved.
- [x] Walkthrough update C9.

### Limitation dan next gate

- Metrics dan recovery masih untuk lingkungan lokal; alerting dan remote
  incident response menjadi scope Phase 1D.
- C10 tetap diperlukan untuk quality gate menyeluruh dan local closeout.

## C10 Checklist: Quality Gate dan Local Closeout

- [ ] Migration/integration.
- [ ] OpenAPI contract/drift.
- [ ] Auth/security suite.
- [ ] Race, vet, lint, vulnerability.
- [ ] Container build.
- [ ] Admin Playwright E2E.
- [ ] Admin axe/accessibility.
- [ ] Frontend Phase 1A regression.
- [ ] Production Phase 1A smoke.
- [ ] Quality report.
- [ ] PRD `IMPLEMENTED-1C-LOCAL`.
- [ ] README dan runbook final.
- [ ] Backend documentation MD/PDF final.
- [ ] No deployment/cutover.
- [ ] Walkthrough update C10.
