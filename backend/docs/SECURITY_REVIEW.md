# Phase 1B Security Review

**Hasil:** lulus untuk local shadow-mode pada 3 Agustus 2026.

## Kontrol aktif

- PostgreSQL runtime role least-privilege dan schema privat.
- Secret hanya dari environment; `.env` tidak masuk Git.
- Strict JSON/body/content-type/query validation dan RFC 7807 response aman.
- Exact-origin CORS, trusted-proxy allowlist, security headers, serta recovery.
- Redis rate limit dengan HMAC IP dan bounded local fallback.
- Contact idempotency HMAC, honeypot, atomic jobs, log tanpa PII, retensi 12 bulan.
- Media magic/decode/dimension validation, metadata stripping, private original,
  conditional object write, dan no-upscale variants.
- Metrics bearer token khusus dibandingkan constant time.
- W3C trace ID dan request ID tidak membawa payload atau credential.

## Temuan yang ditutup

- 26 unchecked close/cleanup findings production dan 6 staticcheck findings
  diperbaiki; final golangci-lint melaporkan 0 issue.
- AWS SDK EventStream/S3, pgx, dan quic-go dinaikkan setelah govulncheck
  menemukan empat reachable vulnerabilities. Final scan: 0 reachable.
- Docker build context 1,15 GB dipangkas menjadi sekitar 11 KB dengan
  mengecualikan build cache, tool binary, test binary, dan local media.

## Risiko residual yang diterima

- Local HTTP tidak memakai TLS; remote TLS menjadi tanggung jawab runtime B11.
- Metrics/tracing belum dikirim ke remote collector.
- Backup provider dan restore drill remote belum dapat diuji sebelum B11.
- S3 adapter baru contract-tested; belum memakai credential/bucket remote.
- Contact production tetap Netlify Forms dan belum cutover ke Go API.

Tidak ada blocker local Phase 1B. Seluruh risiko residual berada di luar
shadow-mode dan harus dievaluasi kembali sebelum remote pilot/cutover.
