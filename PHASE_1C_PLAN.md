# Phase 1C Implementation Plan

## Admin CMS, Authentication, dan Publishing Lokal

| Atribut | Nilai |
| --- | --- |
| Versi | 1.0 |
| Tanggal | 4 Agustus 2026 |
| Status | C0-C5 complete; C6-C10 pending |
| Branch | `feature/phase-1c` |
| Baseline | Phase 1B commit `520d315` |
| Mode | Local-first shadow-mode |

## 1. Tujuan

Phase 1C membangun Admin CMS di atas backend Phase 1B agar pemilik dapat
mengelola konten, produk, media, inbox, dan activity log tanpa mengubah source
code secara manual.

Phase ini belum mengalihkan website production. Next.js production tetap
membaca `LocalContentSource`, formulir production tetap memakai Netlify Forms,
dan seluruh canonical serta route public tetap sama sampai Phase 1D.

## 2. Keputusan yang Dibekukan

- Implementasi dan acceptance dilakukan lokal dalam shadow-mode.
- Page yang editable hanya Homepage dan About.
- `/tauco` dan shell katalog tetap read-only.
- Product, media, inbox, activity log, dan account management termasuk scope.
- Hanya ada role aktif `super_admin`; schema tetap mendukung RBAC.
- Akun pertama dibuat melalui CLI, tanpa public registration.
- Password dan TOTP wajib sebelum CMS dapat digunakan.
- Editor berupa form terstruktur, tanpa raw JSON, rich text, Markdown, atau
  autosave.
- Setiap Save Draft membuat immutable revision baru.
- Product tidak dapat di-hard-delete; archive bersifat reversible.
- Tidak memakai Supabase Auth.
- Tidak membuat unit-test file. Quality evidence memakai contract,
  integration, migration, security, race, dan Playwright E2E.
- Tidak ada deployment, public content cutover, contact cutover, atau remote
  cloud mutation.

## 3. Scope

### 3.1 In scope

- Admin identity, session, RBAC foundation, TOTP, recovery code, dan CLI
  recovery.
- Same-origin BFF untuk browser admin.
- Admin UI pada `/admin`.
- Homepage dan About draft, preview, publish, serta unpublish.
- Product create, edit, draft, preview, publish, unpublish, archive, dan
  unarchive.
- Media upload, processing status, retry, picker, dan ready-variant delivery.
- Inbox list/detail/status dan activity log viewer.
- Immutable revision, optimistic concurrency, audit, dan targeted Redis
  invalidation.
- Local operations, security review, dan documentation.

### 3.2 Deferred ke Phase 1D

- Supabase PostgreSQL/Storage atau provider remote lain.
- Upstash Redis dan remote API/worker deployment.
- Production switch ke `ApiContentSource`.
- Production switch dari Netlify Forms ke contact API.
- Next.js production revalidation/ISR.
- Remote backup/restore drill, alert policy, load test, dan security test.
- Public media renderer cutover dan final image delivery profile.

### 3.3 Out of scope

- Inventory, warehouse, stock ledger, order, checkout, payment, dan customer
  account.
- Public registration, organization, atau multi-tenant CMS.
- WYSIWYG, Markdown, collaboration, approval workflow, dan analytics dashboard.
- Hard delete content, product, media, inbox, atau activity log.
- Email password reset.

## 4. Target Architecture

```text
Browser admin
  -> Next.js /admin
  -> same-origin /admin-api BFF
  -> Go /api/v1/admin
  -> auth / permission / CSRF / rate limit
  -> application use case
  -> admin PostgreSQL role
  -> audit + durable job

Public production selama Phase 1C
  -> Netlify Next.js
  -> LocalContentSource
  -> Netlify Forms

Local public API
  -> published pointer
  -> Redis cache-aside
  -> PostgreSQL
```

Backend membuka connection terpisah:

- `DATABASE_URL` untuk public/runtime operation Phase 1B;
- `ADMIN_DATABASE_URL` untuk authenticated CMS mutation;
- `MIGRATION_DATABASE_URL` untuk versioned schema migration.

`tauco_runtime` tidak memperoleh permission menulis content/product revision.
Admin handler tidak mengakses GORM secara langsung dan seluruh authorization
dilakukan pada application boundary.

## 5. Authentication dan Session

### 5.1 Password

- Panjang 12-128 karakter tanpa composition rule.
- Hash Argon2id: memory 19 MiB, iteration 2, parallelism 1, salt 16 byte, output
  32 byte.
- Parameter dan versi tersimpan bersama hash agar dapat direhash kemudian.
- Login menghasilkan error generik untuk email, password, TOTP, dan recovery
  code yang salah.

### 5.2 JWT dan refresh token

- Access JWT memakai RS256 dan `golang-jwt/jwt/v5`.
- TTL access token 10 menit.
- Refresh session maksimum 30 hari.
- Claims wajib: `iss`, `aud`, `sub`, `sid`, `jti`, `iat`, `nbf`, `exp`.
- Header wajib: `alg=RS256`, explicit `typ`, dan allowlisted `kid`.
- Refresh token opaque berasal dari CSPRNG, hanya hash yang disimpan.
- Setiap refresh merotasi token dalam transaksi.
- Penggunaan ulang refresh token yang sudah dipakai mencabut session.

### 5.3 TOTP dan recovery

- TOTP RFC 6238, enam digit, interval 30 detik, tolerance satu interval sebelum
  dan sesudah waktu saat ini.
- Counter yang sama tidak dapat dipakai ulang.
- Secret unik per admin dan dienkripsi AES-256-GCM memakai
  `MFA_ENCRYPTION_KEY`.
- Sepuluh recovery code acak ditampilkan satu kali.
- Recovery code disimpan sebagai HMAC-SHA256 dan hanya dapat dipakai sekali.
- Reset password, reset TOTP, dan revoke session tersedia melalui CLI lokal.

### 5.4 Cookie dan CSRF

- Access dan refresh cookie: HttpOnly, HostOnly, `SameSite=Strict`, `Path=/`.
- `Secure` wajib pada HTTPS dan dimatikan hanya pada local HTTP.
- Token tidak disimpan di localStorage atau sessionStorage.
- Mutation membutuhkan exact Origin/Referer, `X-CSRF-Token`, dan Fetch Metadata
  yang tidak menunjukkan cross-site request.
- CSRF token dibuat acak per session dan dibandingkan constant-time.

### 5.5 Rate limit

- Login: 5 attempt per 15 menit untuk kombinasi HMAC IP dan normalized email.
- Admin authenticated: 120 request per menit per admin.
- Redis limiter memakai local bounded fallback saat Redis gagal.

## 6. Database Target

Migration Phase 1C menambahkan:

- `admin_users`;
- `roles`;
- `permissions`;
- `user_roles`;
- `role_permissions`;
- `admin_sessions`;
- `admin_refresh_tokens`;
- `mfa_credentials`;
- `mfa_recovery_codes`;
- `page_revision_media`;
- `product_revision_media`;
- `products.archived_at`;
- foreign key nullable `created_by` ke `admin_users` untuk revision baru;
- fixed authorization role `tauco_admin_runtime`.

Semua ID menggunakan UUIDv7 dan waktu menggunakan `timestamptz` UTC. Refresh
token, recovery code, dan idempotency secret tidak disimpan plaintext.

### 6.1 Revision rule

- Save Draft melakukan `INSERT`, bukan update.
- Published revision dan draft revision tidak dapat diubah atau dihapus.
- Revision number dialokasikan setelah parent entity dikunci `FOR UPDATE`.
- Mutation membawa `baseRevisionId` atau `If-Match`.
- Jika state terbaru berbeda, API mengembalikan `409 REVISION_CONFLICT`.
- Publish membuat published revision baru dari draft dalam satu transaksi.
- Unpublish hanya mengosongkan published pointer.
- Activity log mutation berada dalam transaksi yang sama.

### 6.2 Product identity

- Slug unik dan dapat berubah hanya sebelum first publish.
- SKU opsional dan unik ketika tersedia.
- Sort order integer non-negatif.
- Archive mengisi `archived_at` dan mengosongkan published pointer.
- Unarchive mengosongkan `archived_at`; hard delete tidak tersedia.

### 6.3 Revision-media relationship

Setiap relasi menyimpan revision ID, media asset ID, field path/slot, dan
position. Publish menolak asset yang tidak `ready`.

Static asset lama pada `/images/...` tetap valid. Media baru menggunakan path
stabil:

```text
/api/v1/media/{assetId}/display.webp
```

Original selalu privat. Public route hanya menyajikan variant milik asset
`ready` dengan ETag dan immutable cache header.

## 7. REST API Contract

### 7.1 Auth

```text
POST /api/v1/admin/auth/login
POST /api/v1/admin/auth/totp/setup
POST /api/v1/admin/auth/totp/enable
POST /api/v1/admin/auth/refresh
POST /api/v1/admin/auth/logout
GET  /api/v1/admin/auth/me
POST /api/v1/admin/auth/recovery-codes/regenerate
```

### 7.2 Page CMS

Hanya key `home` dan `about`:

```text
GET  /api/v1/admin/pages/{key}
POST /api/v1/admin/pages/{key}/drafts
GET  /api/v1/admin/pages/{key}/revisions/{revisionId}
POST /api/v1/admin/pages/{key}/revisions/{revisionId}/publish
POST /api/v1/admin/pages/{key}/unpublish
```

### 7.3 Product CMS

```text
GET   /api/v1/admin/products
POST  /api/v1/admin/products
GET   /api/v1/admin/products/{id}
PATCH /api/v1/admin/products/{id}
POST  /api/v1/admin/products/{id}/drafts
GET   /api/v1/admin/products/{id}/revisions/{revisionId}
POST  /api/v1/admin/products/{id}/revisions/{revisionId}/publish
POST  /api/v1/admin/products/{id}/unpublish
POST  /api/v1/admin/products/{id}/archive
POST  /api/v1/admin/products/{id}/unarchive
```

### 7.4 Media

```text
GET  /api/v1/admin/media
POST /api/v1/admin/media
GET  /api/v1/admin/media/{id}
POST /api/v1/admin/media/{id}/retry

GET /api/v1/media/{id}/display.webp
GET /api/v1/media/{id}/variants/{width}.webp
```

Upload tetap dibatasi 10 MiB dan hanya menerima JPEG, PNG, atau static WebP
yang lolos magic-byte dan full-decode validation.

### 7.5 Inbox dan audit

```text
GET   /api/v1/admin/contact-messages
GET   /api/v1/admin/contact-messages/{id}
PATCH /api/v1/admin/contact-messages/{id}/status
GET   /api/v1/admin/activity-logs
```

List memakai opaque cursor. Response error memakai RFC 7807 dan request ID.
GET tidak pernah melakukan mutation.

## 8. Admin Frontend

Route yang direncanakan:

```text
/admin/login
/admin/setup-totp
/admin/content
/admin/content/[key]
/admin/products
/admin/products/new
/admin/products/[id]
/admin/media
/admin/inbox
/admin/inbox/[id]
/admin/activity
/admin/account
/admin/preview/[type]/[revisionId]
```

- `ADMIN_CMS_ENABLED=false` secara default.
- Admin route menghasilkan 404 ketika feature disabled.
- Seluruh admin page memakai `noindex, nofollow` dan `Cache-Control: no-store`.
- `/admin-api/[...path]` menjadi same-origin BFF dengan backend origin
  server-only.
- BFF hanya meneruskan exact allowlisted admin/media path dan method.
- BFF meneruskan cookie, Set-Cookie, CSRF, Origin, dan request ID; hop-by-hop
  header dibuang.
- Go API tetap menjadi authorization authority.
- UI memakai Tailwind dan Phosphor yang sudah tersedia.
- Tidak menambah component library atau form library.
- Form memakai native input, textarea, select, repeatable group, reorder
  control, dan media picker.
- Public page presentation diekstrak menjadi view component berbasis props agar
  preview memakai tampilan yang sama tanpa mengubah production content source.

## 9. Publishing dan Cache

- Publish/unpublish memasukkan durable job `content.invalidate_cache`.
- Job menaikkan generation tag `home`, `about`, `products`, atau
  `product:{slug}` secara idempotent.
- Preview membaca revision langsung dan tidak memakai public cache.
- Public API tetap hanya membaca published pointer.
- Next.js production revalidation tetap milik Phase 1D.

## 10. Gate Implementasi

### C0 - Scope freeze

- Plan, walkthrough, PRD amendment, README, branch, dan baseline evidence.

### C1 - OpenAPI dan database foundation

- Admin contract, auth/RBAC/session/MFA schema, media relationship, archive,
  admin DB role, migration/privilege evidence.

### C2 - Auth domain dan CLI

- Argon2id, JWT key management, refresh token, encrypted TOTP, recovery code,
  bootstrap/reset/revoke commands.

### C3 - Auth API dan security middleware

- Login, TOTP enrollment, refresh, logout, current user, permission, CSRF,
  Origin, rate limit, audit, dan abuse evidence.

### C4 - Same-origin BFF dan admin shell

- Feature flag, BFF, login/setup/account/navigation, cookie forwarding, noindex,
  no-store, dan production-disabled build.

### C5 - Media CMS

- Authenticated upload, list/detail/status/retry, picker, polling, ready variant,
  dan abuse evidence.

### C6 - Homepage dan About CMS

- Structured editor, immutable draft, history, preview, publish, unpublish,
  media readiness, fact-check confirmation, audit, dan invalidation.

### C7 - Product CMS

- List/create/update, draft/history/preview, publish/unpublish,
  archive/unarchive, stable slug, media, canonical, dan public exclusion.

### C8 - Inbox dan activity log

- Cursor/filter/detail/status, privacy boundary, append-only viewer, dan no-PII
  metrics/logging.

### C9 - Publishing, recovery, dan operations

- Durable invalidation, dead-job recovery, auth/TOTP recovery, revision
  conflict, media retry, metrics, dan runbook.

### C10 - Quality gate dan local closeout

- Migration, contract, integration, security, race, lint, vulnerability,
  container, frontend, admin E2E, accessibility, Phase 1A regression, report,
  dan documentation final.

## 11. Testing dan Acceptance

Tidak ada unit-test file baru. Suite executable wajib mencakup:

- fresh migration, down/up, privilege matrix, dan revision immutability;
- bootstrap satu admin dan secret tidak muncul pada process argument/log;
- generic invalid login response;
- TOTP setup, valid/expired/replay code, clock tolerance, dan recovery sekali
  pakai;
- JWT wrong alg/signature/issuer/audience/type/kid/session/expiry;
- refresh rotation, reuse detection, logout, dan CLI revoke;
- CSRF, Origin/Referer, Fetch Metadata, cookie flag, dan permission 403;
- concurrent Save Draft menghasilkan satu success dan satu 409;
- published pointer, unpublish, audit, serta cache job atomic;
- stable product slug dan archived-product public exclusion;
- media spoof/corrupt/animated/oversized/dimension abuse;
- publish dengan media non-ready ditolak;
- public media hanya menyajikan ready variant, bukan original;
- inbox cursor/status dan activity metadata tanpa PII;
- admin disabled menghasilkan production 404;
- noindex, no-store, keyboard, focus, label, validation, responsive, dark mode,
  serta axe A/AA;
- public Phase 1A route, canonical, sitemap, robots, JSON-LD, build, E2E, dan
  Netlify production adapter tetap lulus.

Quality command akan memperluas `phase1b:quality`, bukan mengganti gate yang
sudah ada.

## 12. Walkthrough Rule

`PHASE_1C_WALKTHROUGH.md` diperbarui setelah setiap gate dengan:

1. tanggal dan status;
2. tujuan;
3. perubahan;
4. keputusan arsitektur;
5. command yang dijalankan;
6. hasil pengujian;
7. evidence;
8. known limitation;
9. next gate.

Gate hanya berstatus Complete ketika pemeriksaan relevan lulus. Setiap gate
direncanakan sebagai commit terpisah. Push, merge, dan deployment membutuhkan
instruksi owner terpisah.

## 13. Security References

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [RFC 8725 JWT Best Current Practices](https://www.rfc-editor.org/info/rfc8725/)
- [RFC 6238 TOTP](https://www.rfc-editor.org/rfc/rfc6238.html)
- [Next.js Authentication Guide](https://nextjs.org/docs/app/guides/authentication)
