# Backend Operations Runbook

Dokumen ini berlaku untuk Phase 1B local shadow-mode. Tidak ada perintah di
sini yang mengubah website production.

## Health dan metrics

```powershell
Invoke-RestMethod http://127.0.0.1:8080/health/live
Invoke-RestMethod http://127.0.0.1:8080/health/ready
npm.cmd run backend:worker:ready
Invoke-WebRequest http://127.0.0.1:8080/internal/metrics `
  -Headers @{ Authorization = "Bearer $env:METRICS_BEARER_TOKEN" }
```

Liveness tidak memanggil dependency. API readiness wajib gagal saat PostgreSQL
gagal; Redis dan media storage hanya menghasilkan status `degraded`. Worker
readiness mewajibkan PostgreSQL, media storage yang writable, dan SMTP.

## Replay dead job

1. Pastikan job berstatus `dead` dan catat ID-nya tanpa membuka payload PII.
2. Buat request ID operasional, lalu jalankan:

```powershell
npm.cmd run backend:ops -- job-replay <JOB_UUID> <REQUEST_ID>
```

Replay mengatur ulang attempt, menjadwalkan job, dan menulis activity log. Untuk
media retry, replay job `media.generate_variants` milik asset terkait. Jangan
mengubah status media atau job langsung melalui DBeaver.

## Cache purge

Invalidasi memakai generation tag dan tidak melakukan Redis key scan:

```powershell
npm.cmd run backend:ops -- cache-purge home about tauco-guide products
npm.cmd run backend:ops -- cache-purge product:tauco-cap-badak
```

Redis boleh di-flush hanya pada database lokal disposable. Production harus
selalu memakai targeted generation tags.

## Contact retention

Worker menjalankan purge saat startup lalu setiap 24 jam, maksimal 100 record
per batch sampai seluruh record jatuh tempo habis. Contact message yang belum
mencapai `retention_delete_at` tidak boleh dihapus oleh operator.

## Backup design

- PostgreSQL adalah satu-satunya data source yang wajib dibackup.
- Remote pilot memakai backup provider bila tersedia; sebelum itu lakukan
  `pg_dump --format=custom` dari migration connection ke media terenkripsi.
- File backup yang memuat contact PII wajib ikut dihapus paling lambat 12 bulan.
- Media original/variant dicadangkan terpisah bersama checksum object.
- Redis tidak dibackup karena hanya cache dan rate-limit state.
- Restore drill dilakukan ke database kosong, lalu migration version, row
  count, constraint, dan checksum media diverifikasi sebelum dianggap lulus.
- Jangan menaruh dump, credential, atau recovery token di repository.

## Incident order

1. Hentikan perubahan manual dan simpan request ID/log terkait.
2. Periksa liveness, readiness, dan metrics tanpa menyalin PII.
3. Pulihkan PostgreSQL lebih dahulu; Redis dapat dibiarkan fail-open.
4. Replay hanya job yang terbukti aman dan idempotent.
5. Catat tindakan dan hasil restore/replay pada incident log eksternal.
