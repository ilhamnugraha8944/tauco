# Phase 1C Walkthrough

## Admin CMS, Authentication, dan Publishing Lokal

| Atribut | Nilai |
| --- | --- |
| Tanggal mulai | 4 Agustus 2026 |
| Branch | `feature/phase-1c` |
| Baseline | `520d315` |
| Status | C0 complete; C1-C10 pending |
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

## C1 Checklist: OpenAPI dan Database Foundation

- [ ] Admin OpenAPI paths dan schemas.
- [ ] Auth/RBAC/session/MFA migration.
- [ ] Admin DB authorization role.
- [ ] Revision-media relationship.
- [ ] Product archive field.
- [ ] Revision immutability enforcement.
- [ ] Migration up/down.
- [ ] Fresh database integration.
- [ ] Privilege matrix.
- [ ] OpenAPI drift check.
- [ ] Walkthrough update C1.

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
