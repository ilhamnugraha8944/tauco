# Phase 1B Local Quality Report

**Tanggal:** 3 Agustus 2026
**Status:** B0-B10 complete, B11 deferred
**Boundary:** local shadow-mode, tanpa push atau deployment

## Ringkasan gate

| Gate | Evidence | Hasil |
| --- | --- | --- |
| Format/OpenAPI | gofmt 128 file, generated code reproducible | Lulus |
| Unit/contract/abuse (historical, sebelum cleanup) | Go suite dan 85 frontend unit tests | Lulus |
| Integration/migration | PostgreSQL + Redis nyata, migration v4 clean | Lulus |
| Race | `go test -race ./...` pada Linux container | Lulus |
| Static analysis | `go vet`, golangci-lint v2.11.4 | 0 issue |
| Vulnerability | govulncheck v1.6.0, npm production audit | 0 reachable / 0 npm |
| Container | scratch runtime image `tauco-api:phase1b-local` | Lulus |
| Warm/cold baseline | cold 65 ms; 200 read; p95 47 ms; p99 49 ms; error 0% | Lulus |
| Frontend | lint, typecheck, unit, Next build | Lulus |
| E2E lokal | Playwright | 79 lulus, 13 intentional skip |
| Production smoke | Netlify Phase 1A | 29/29 lulus |

Empat vulnerability reachable yang pertama kali ditemukan telah ditutup dengan
upgrade AWS SDK S3/EventStream, pgx, dan quic-go sebelum scan final.

## Evidence acceptance 1-33

| # | Evidence |
| --- | --- |
| 1 | Migration integration membuat database kosong dan menerapkan v1-v4 tanpa AutoMigrate. |
| 2 | Architecture dependency test dan golangci-lint lulus. |
| 3 | OpenAPI mencakup public, contact, health, metrics, schema, example, problem; drift check lulus. |
| 4 | Tauco guide route dan frontend fixture parity lulus. |
| 5 | Seed/importer replay idempotent dan parity Phase 1A lulus. |
| 6 | Repository/API hanya membaca published revision. |
| 7 | Unknown product menghasilkan RFC 7807 404 pada API dan true 404 frontend. |
| 8 | Catalog deterministic, bounded, signed cursor, dan tamper test lulus. |
| 9 | Contact message serta dua job dibuat dalam satu transaction integration. |
| 10 | Idempotent replay/conflict tidak menggandakan message/job. |
| 11 | Strict validation dan structured-log redaction tidak menyimpan payload invalid. |
| 12 | Redis hit menghindari origin read. |
| 13 | Redis outage/failure fail-open ke PostgreSQL. |
| 14 | Concurrent miss memakai singleflight. |
| 15 | Generation invalidation hanya menaikkan tag target. |
| 16 | Limit menghasilkan 429 dan Retry-After. |
| 17 | Trusted/untrusted proxy normalization lulus. |
| 18 | Dua worker memakai SKIP LOCKED tanpa double claim. |
| 19 | Lease expiry dan crash reclaim lulus. |
| 20 | Backoff bounded, dead-letter, serta audited replay lulus. |
| 21 | API dan worker berhenti melalui context/grace period; race detector lulus. |
| 22 | Media valid menghasilkan normalized original dan non-upscaled WebP variants. |
| 23 | Spoof/corrupt/oversized/dimension/animated input ditolak. |
| 24 | Normalized output di-encode ulang tanpa EXIF/GPS. |
| 25 | Media transition dan conditional replay idempotent lulus. |
| 26 | Liveness hanya memeriksa process. |
| 27 | PostgreSQL failure memetakan readiness 503; Redis/storage hanya degraded. |
| 28 | Protected Prometheus metrics memakai label bounded tanpa PII. |
| 29 | Worker purge saat startup/24 jam dan repository retention integration lulus. |
| 30 | Unit, integration, race, contract, abuse, migration, lint, vet, vuln, container lulus. |
| 31 | Frontend check, E2E lokal, dan production smoke lulus. |
| 32 | Production tetap LocalContentSource dan Netlify Forms. |
| 33 | Tidak ada deployment/push; secret lokal tetap ignored. |

## Catatan setelah gate

Sesuai instruksi owner, file unit test dihapus setelah seluruh evidence di atas
direkam. Integration, architecture, contract, migration, E2E, dan production
smoke tests dipertahankan sebagai regression gate yang executable.
