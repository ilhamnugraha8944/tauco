# ADR-0006: Phase 1D Netlify Remote Pilot yang Dibatasi Waktu

- Status: Accepted
- Tanggal: 6 Agustus 2026
- Pemilik: Product / Engineering

## Context

Backend dan CMS Phase 1B-1C sudah lulus lokal. Pemilik meminta remote pilot
tanpa biaya dan production Phase 1A saat ini berada pada Netlify Legacy Free.
Cloud Run lebih sesuai untuk container dan long-running worker, tetapi
memerlukan billing account dan tidak memenuhi batas biaya keras yang dipilih.

Netlify Go Function memakai Lambda Compatibility. Netlify menyatakan deployment
baru dengan mode tersebut tidak diterima mulai 1 Juli 2027. Legacy Free juga
memiliki fixed quota dan tidak memberikan SLA.

## Decision

- Phase 1D memakai Netlify Legacy Free sebagai runtime pilot.
- API memakai synchronous Go Function; durable worker memakai scheduled
  one-shot invocation yang bounded.
- Supabase Free menyediakan PostgreSQL dan private S3-compatible Storage.
- Upstash Redis Free menyediakan cache yang fail-open.
- Netlify Forms tetap menjadi contact transport dan email sender.
- Rollout dipisah menjadi shadow runtime, remote admin, public content, lalu
  verified form mirror.
- Production selalu memiliki rollback ke local content.
- Review runtime dilakukan 1 Februari 2027, rehearsal migrasi pada Mei 2027,
  dan runtime Go dipindahkan paling lambat 15 Juni 2027.
- Kandidat pengganti dipilih berdasarkan kondisi provider saat review; Cloud
  Run tetap kandidat bila billing risk sudah disetujui.

## Consequences

- Pilot dapat berjalan tanpa menambah payment method, tetapi tidak memiliki
  SLA dan dapat terpengaruh quota atau perubahan terms.
- Scheduled worker tidak boleh menjadi long-running daemon.
- Media besar harus direct-upload karena payload Function terbatas.
- Ed25519 dipakai agar signing key dan seluruh environment tetap compact.
- Region runtime Legacy Free tidak dapat disesuaikan; database/cache pilot
  ditempatkan sedekat mungkin ke Ohio.
- Migration keluar dari Lambda Compatibility adalah pekerjaan wajib, bukan
  optional cleanup.

## References

- [Netlify Legacy plans](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-legacy-plans/legacy-pricing-plans/)
- [Netlify Lambda compatibility](https://docs.netlify.com/build/functions/lambda-compatibility/)
- [Netlify scheduled functions](https://docs.netlify.com/build/functions/scheduled-functions/)
