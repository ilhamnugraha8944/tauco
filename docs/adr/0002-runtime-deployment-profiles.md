# ADR-0002: Runtime-Neutral Core dan Deployment Profiles

- Status: Accepted
- Tanggal: 28 Juli 2026
- Pemilik: Product / Engineering

## Context

Pemilik menginginkan biaya awal Rp0. Netlify Legacy Free sudah tersedia dan
mempunyai hard quota, sedangkan Cloud Run lebih cocok untuk container dan
worker tetapi membutuhkan billing account. Tidak ada free tier yang sekaligus
memberikan hard spending cap, SLA, backup, dan enterprise observability.

## Decision

- B1 sampai B10 local-first dan tidak dideploy otomatis.
- Backend core tidak mengimpor Netlify atau Google Cloud pada domain/use case.
- Sediakan executable HTTP/container standar sebagai primary development
  profile.
- Netlify synchronous Go Function + Scheduled Function menjadi optional
  hard-capped pilot.
- Cloud Run API/worker + Cloud Tasks/Scheduler menjadi optional scale-ready
  pilot.
- Pemilik memilih profile pada B11 setelah local acceptance lulus.
- Agent tidak melakukan deploy atau push.

## Consequences

- Composition root/runtime adapter dapat berbeda tanpa mengubah business rule.
- Netlify scheduled worker harus mematuhi execution limit 30 detik.
- Cloud Run profile memerlukan explicit billing-risk approval.
- Operational SLA tidak diklaim pada free pilot.
