# Phase 1C Walkthrough

## Admin CMS, Authentication, dan Publishing Lokal

| Atribut | Nilai |
| --- | --- |
| Tanggal mulai | 4 Agustus 2026 |
| Branch | `feature/phase-1c` |
| Baseline | `520d315` |
| Status | C0-C1 complete; C2-C10 pending |
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

## C2 Checklist: Auth Domain dan CLI

- [ ] Argon2id password hashing.
- [ ] RS256 key generation/loading.
- [ ] JWT claims dan validation.
- [ ] Refresh token hashing/rotation model.
- [ ] AES-GCM TOTP secret storage.
- [ ] TOTP replay protection.
- [ ] Recovery code generation/consumption.
- [ ] Admin bootstrap CLI.
- [ ] Password/TOTP/session recovery CLI.
- [ ] Auth integration evidence.
- [ ] Walkthrough update C2.

## C3 Checklist: Auth API dan Security Middleware

- [ ] Login.
- [ ] TOTP setup/enable.
- [ ] Refresh/logout/me.
- [ ] Recovery code regeneration.
- [ ] Cookie policy.
- [ ] CSRF dan exact origin.
- [ ] Fetch Metadata enforcement.
- [ ] Login/admin rate limit.
- [ ] RBAC/permission middleware.
- [ ] Auth activity audit.
- [ ] Security abuse evidence.
- [ ] Walkthrough update C3.

## C4 Checklist: BFF dan Admin Shell

- [ ] `ADMIN_CMS_ENABLED` default false.
- [ ] Same-origin admin BFF.
- [ ] Exact path/method allowlist.
- [ ] Cookie/CSRF/request ID forwarding.
- [ ] Login dan setup TOTP UI.
- [ ] Admin layout/navigation/account/logout.
- [ ] Noindex/no-store.
- [ ] Production-disabled 404/build.
- [ ] Responsive/accessibility baseline.
- [ ] Walkthrough update C4.

## C5 Checklist: Media CMS

- [ ] Authenticated multipart upload.
- [ ] Existing image pipeline reused.
- [ ] Media list/detail/status polling.
- [ ] Retry failed media.
- [ ] Media picker.
- [ ] Ready display/variant route.
- [ ] Original remains private.
- [ ] Upload abuse evidence.
- [ ] Walkthrough update C5.

## C6 Checklist: Homepage dan About CMS

- [ ] Structured Home editor.
- [ ] Structured About editor.
- [ ] Immutable Save Draft.
- [ ] Revision history/detail.
- [ ] Optimistic conflict.
- [ ] Authenticated preview.
- [ ] Publish/unpublish.
- [ ] Media readiness.
- [ ] Fact-check confirmation.
- [ ] Audit dan cache invalidation job.
- [ ] `/tauco` tetap read-only.
- [ ] Walkthrough update C6.

## C7 Checklist: Product CMS

- [ ] Product list/create/detail.
- [ ] Identity update.
- [ ] Structured product editor.
- [ ] Draft/history/preview.
- [ ] Publish/unpublish.
- [ ] Archive/unarchive.
- [ ] Stable slug after first publish.
- [ ] SKU/sort validation.
- [ ] Media/canonical validation.
- [ ] Archived public exclusion.
- [ ] Walkthrough update C7.

## C8 Checklist: Inbox dan Activity Log

- [ ] Inbox cursor/filter.
- [ ] Message detail.
- [ ] Explicit status mutation.
- [ ] GET remains read-only.
- [ ] Activity cursor/filter.
- [ ] Metadata allowlist.
- [ ] PII redaction.
- [ ] Local inbox limitation documented.
- [ ] Walkthrough update C8.

## C9 Checklist: Publishing, Recovery, dan Operations

- [ ] Durable cache invalidation handler.
- [ ] Page/product generation tags.
- [ ] Idempotent retry/dead-letter recovery.
- [ ] Auth/TOTP recovery runbook.
- [ ] Revision conflict runbook.
- [ ] Media retry runbook.
- [ ] Admin/session/publish metrics.
- [ ] Phase 1D revalidation boundary preserved.
- [ ] Walkthrough update C9.

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
