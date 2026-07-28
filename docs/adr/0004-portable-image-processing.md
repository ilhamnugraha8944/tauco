# ADR-0004: Portable CGO-Free Image Processing pada Phase 1B

- Status: Accepted
- Tanggal: 28 Juli 2026
- Pemilik: Product / Engineering

## Context

PRD meminta normalized original serta WebP width 320, 640, dan 1280. Libvips
cepat dan hemat memory, tetapi membawa CGO/native dependency yang mempersulit
Windows development serta optional serverless runtime.

## Decision

- Definisikan `ImageProcessor` port.
- Initial adapter memakai Go image packages, `x/image/draw`, dan WebP
  implementation dengan CGO-free fallback.
- Terapkan decode-config limit sebelum full decode.
- Tolak format di luar JPEG/PNG/WebP dan animated WebP.
- Terapkan EXIF orientation lalu re-encode untuk membuang EXIF/GPS.
- Jangan upscale.
- Benchmark memory, duration, dan output size menjadi B7 acceptance.
- Tambahkan libvips adapter container hanya bila benchmark membuktikan initial
  adapter tidak memenuhi gate.

## Consequences

- Binary lebih portable untuk local, Netlify Function, dan container.
- Throughput mungkin lebih rendah dibanding libvips.
- Interface membuat penggantian processor tidak mengubah media use case.
- Width yang lebih besar dari source dilewati, sehingga jumlah variant dapat
  kurang dari tiga untuk source kecil.
