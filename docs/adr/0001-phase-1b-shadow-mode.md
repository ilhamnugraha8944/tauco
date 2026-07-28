# ADR-0001: Phase 1B Shadow-Mode di Monorepo

- Status: Accepted
- Tanggal: 28 Juli 2026
- Pemilik: Product / Engineering

## Context

Phase 1A sudah live, terindeks, dan tidak bergantung pada database atau API.
Phase 1B perlu membangun backend tanpa membuat availability, build, SEO, dan
contact production bergantung pada fondasi yang belum melewati quality gate.

Repository saat ini menempatkan Next.js di root dan konfigurasi Netlify
bergantung pada layout tersebut.

## Decision

- Pertahankan Next.js di root.
- Tambahkan backend di `backend/`.
- Jalankan Phase 1B dalam shadow-mode.
- Production tetap memakai `LocalContentSource` dan Netlify Forms.
- Import konten JSON ke PostgreSQL hanya satu arah selama Phase 1B.
- Tambahkan `GET /api/v1/tauco-guide` agar API dapat memenuhi seluruh kontrak
  `ContentSource`.
- Buat adapter API frontend hanya untuk contract test/preview, bukan default
  production.
- Tunda content/contact cutover sampai Phase 1D.

## Consequences

Positif:

- Phase 1A tidak terpengaruh outage/cold-start backend.
- Parity dapat dibuktikan sebelum cutover.
- Tidak ada dual-write contact.
- Struktur Netlify tidak perlu dipindah.

Trade-off:

- JSON dan database hidup bersamaan sementara.
- Importer dan checksum parity wajib mencegah drift.
- Fitur CMS Phase 1C belum otomatis terlihat di public production sampai
  cutover Phase 1D.
