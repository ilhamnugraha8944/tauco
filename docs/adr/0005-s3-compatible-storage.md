# ADR-0005: S3-Compatible Storage dengan Supabase sebagai Pilot

- Status: Accepted
- Tanggal: 28 Juli 2026
- Pemilik: Product / Engineering

## Context

Media harus berada di object storage. R2 mempunyai allowance besar tetapi
merupakan usage-based service, activation/billing perlu persetujuan, dan public
production sebaiknya memakai custom domain yang belum tersedia.

Supabase Free sudah menjadi target PostgreSQL dan menyediakan Storage
S3-compatible dengan quota pilot yang cukup.

## Decision

- Domain/use case bergantung pada `ObjectStorage` port.
- Infrastructure memakai subset operasi S3 umum melalui AWS SDK for Go v2.
- Remote pilot awal memakai Supabase Storage.
- Normalized original bersifat private.
- Variant belum menjadi source image production pada Phase 1B.
- Local/test memakai S3-compatible service atau contract fake yang dibekukan
  saat B7.
- R2 menjadi upgrade path setelah custom domain dan billing risk disetujui.
- Tidak bergantung pada provider-specific transform, lifecycle, atau versioning.

## Consequences

- Supabase dan R2 dapat ditukar di infrastructure layer.
- Supabase Free tidak menyediakan image transform atau object versioning.
- Delete harus diperlakukan irreversible.
- Backup object terpisah tetap menjadi requirement Phase 1D.
