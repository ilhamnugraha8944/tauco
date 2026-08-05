# Phase 1C Local Operations Runbook

Runbook ini untuk recovery CMS lokal. Jalankan dari root repository dengan
Docker Desktop, PostgreSQL, Redis, API, dan worker lokal. Jangan gunakan
credential production pada Phase 1C.

**Status:** Phase 1C C0-C10 complete lokal. Runbook ini bukan instruksi
deployment atau production cutover.

## 1. Pemeriksaan awal

```powershell
npm.cmd run backend:compose:up
npm.cmd run backend:migrate:version
npm.cmd run backend:worker:ready
```

`worker:ready` memeriksa PostgreSQL, Redis, media storage, dan SMTP. Jalankan
Mailpit profile bila readiness SMTP belum tersedia.

## 2. Password, TOTP, dan session

Ganti password bila credential hilang atau diduga bocor:

```powershell
npm.cmd run backend:admin -- reset-password ilhamnugraha8944@gmail.com
npm.cmd run backend:admin -- revoke-sessions ilhamnugraha8944@gmail.com
```

Reset TOTP bila authenticator hilang dan recovery code tidak tersedia:

```powershell
npm.cmd run backend:admin -- reset-totp ilhamnugraha8944@gmail.com
npm.cmd run backend:admin -- revoke-sessions ilhamnugraha8944@gmail.com
```

Setelah reset, login ulang dan selesaikan setup TOTP. Password dimasukkan lewat
prompt tersembunyi, bukan argument command atau file log.

## 3. Konflik revision

Jika UI menampilkan bahwa konten telah berubah:

1. Jangan mengulang request lama atau mengganti `If-Match` secara manual.
2. Salin perubahan lokal yang belum tersimpan bila masih diperlukan.
3. Muat ulang editor untuk mengambil revision dan ETag terbaru.
4. Terapkan kembali perubahan, konfirmasi fact-check, lalu Save Draft.
5. Buka preview sebelum Publish.

Revision lama tetap immutable dan dapat dibuka dari riwayat.

## 4. Media gagal

1. Buka **Media** dan periksa `lastErrorCode` tanpa menyalin data sensitif.
2. Perbaiki sumber bila format, ukuran, atau dimensi tidak valid.
3. Untuk kegagalan transient, gunakan **Retry** satu kali dari detail media.
4. Pastikan status berubah menjadi `ready` sebelum Publish.
5. Jika job menjadi `dead`, gunakan prosedur job replay setelah akar masalah
   diperbaiki.

Original tetap private dan tidak boleh diberi public route.

## 5. Dead job dan replay

Cari job `dead` melalui DBeaver pada database lokal:

```sql
SELECT id, kind, attempts, last_error_code, updated_at
FROM tauco_app.background_jobs
WHERE status = 'dead'
ORDER BY updated_at DESC;
```

Replay satu job setelah akar masalah diperbaiki:

```powershell
npm.cmd run backend:ops -- job-replay <JOB_ID> c9-local-recovery-001
```

Jangan memperbarui row job secara manual. Invalidasi generation aman dijalankan
ulang karena cache key lama tetap tidak dapat dipilih setelah generation maju.

## 6. Cache recovery

Worker memproses `content.invalidate_cache` untuk tag `home`, `about`,
`products`, dan `product:{slug}`. Recovery operator lokal:

```powershell
npm.cmd run backend:ops -- cache-purge home
npm.cmd run backend:ops -- cache-purge products product:tauco-cap-badak
```

Gunakan tag spesifik. Jangan menghapus seluruh Redis database.

## 7. Metrics lokal

Endpoint `/internal/metrics` memerlukan bearer token lokal. Periksa minimal:

- `tauco_background_jobs` untuk retry/dead;
- `tauco_admin_sessions` untuk active/expired/revoked;
- `tauco_publishing_entities` untuk page/product published state;
- `tauco_media_assets` dan `tauco_contact_retention_due`.

Label metrics dibatasi dan tidak boleh berisi email, nama, isi pesan, token,
atau metadata audit mentah.

## 8. Final local verification

Jalankan sebelum handoff ke Phase 1D atau setelah perubahan lintas modul:

```powershell
npm.cmd run backend:format:check
npm.cmd run backend:generate:check
npm.cmd run backend:test:integration
npm.cmd run backend:race
npm.cmd run backend:vet
npm.cmd run backend:lint
npm.cmd run backend:vuln
npm.cmd run backend:load
npm.cmd run backend:container:build
npm.cmd run check:frontend
npm.cmd run test:admin
npm.cmd run test:e2e
```

Production smoke `npm.cmd run qa:g6` bersifat read-only. Jangan menjalankan
live form test tanpa sengaja karena akan membuat submission nyata.

Evidence baseline final ada di `PHASE_1C_QUALITY_REPORT.md`.

## 9. Boundary Phase 1D

Phase 1C tidak mengubah production Netlify, `LocalContentSource`, Netlify
Forms, canonical, atau Search Console. Revalidation Next.js/ISR, remote API,
Supabase, Upstash, object storage, backup/restore, dan production cutover hanya
dikerjakan pada Phase 1D dengan checklist deployment terpisah.
