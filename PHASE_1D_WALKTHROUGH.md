# Phase 1D Walkthrough

## Remote Pilot dan Production Cutover

| Atribut | Nilai |
| --- | --- |
| Tanggal mulai | 6 Agustus 2026 |
| Branch | `feature/phase-1d` |
| Baseline | `216c6f9` |
| Status | D0 complete; D1-D10 pending |
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

## Gate Checklist

| Gate | Status | Exit utama |
| --- | --- | --- |
| D0 Scope/provider freeze | Complete | Plan, walkthrough, PRD, README, ADR konsisten |
| D1 Production security/config | Pending | Fail-closed config dan remote role policy lulus |
| D2 Media/worker remote foundation | Pending | Direct upload dan bounded worker lulus lokal |
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
