# Phase 1D Implementation Plan

## Remote Pilot dan Production Cutover

| Atribut | Nilai |
| --- | --- |
| Versi | 1.0 |
| Tanggal | 6 Agustus 2026 |
| Status | D0 complete; D1-D10 planned |
| Branch | `feature/phase-1d` |
| Baseline | Phase 1C merge commit `216c6f9` |
| Mode | Remote pilot bertahap dengan rollback lokal |

## 1. Tujuan

Phase 1D memindahkan backend dan Admin CMS yang telah lulus secara lokal ke
remote pilot, lalu mengalihkan sumber konten production secara bertahap tanpa
mengubah URL publik atau menurunkan acceptance Phase 1A.

Target provider:

- Netlify Legacy Free untuk Next.js, Go Function, dan scheduled worker;
- Supabase Free untuk PostgreSQL dan private S3-compatible Storage;
- Upstash Redis Free untuk cache;
- Netlify Forms tetap menjadi transport dan pengirim email kontak;
- Google Drive pribadi untuk backup terenkripsi.

Free tier adalah pilot tanpa SLA. Quota, pause policy, dan perubahan terms
provider harus dipantau oleh pemilik.

## 2. Keputusan yang Dibekukan

- Production diubah melalui deployment terpisah: shadow, admin, content, lalu
  verified form sync.
- Pemilik melakukan klik dashboard, merge, dan production deploy berdasarkan
  walkthrough. Agent tidak mengubah akun vendor.
- Public form tetap memakai Netlify Forms. Go contact intake tidak diaktifkan.
- Hanya submission Netlify yang sudah verified yang disalin ke Inbox CMS.
- Email kontak tetap dikirim Netlify; backend tidak mengirim email duplikat.
- Retensi pesan maksimal 12 bulan berlaku pada Netlify dan Supabase.
- Public content mempunyai rollback eksplisit ke `LocalContentSource`.
- Media 10 MB diunggah langsung dari browser ke private object storage dengan
  presigned URL; binary tidak diproksikan melalui Netlify Function.
- Production authentication memakai Ed25519/EdDSA. File-based RSA tetap hanya
  untuk kompatibilitas lokal.
- PostgreSQL memakai role terpisah untuk migration, runtime, dan admin.
- Redis adalah optimization dan harus fail-open.
- Tidak ada unit-test file baru. Verification memakai integration, contract,
  migration, security, E2E, smoke, dan load check.
- Inventory, order, payment, customer account, dan custom domain tidak masuk
  Phase 1D.
- Netlify Go Lambda Compatibility hanya pilot. Review platform dilakukan pada
  1 Februari 2027, rehearsal migrasi pada Mei 2027, dan keluar paling lambat
  15 Juni 2027 sebelum deployment baru dihentikan pada 1 Juli 2027.

## 3. Target Architecture

```text
Public browser
  -> Netlify CDN / Next.js
  -> LocalContentSource (rollback) atau ApiContentSource
  -> Netlify Go API
  -> PostgreSQL + Redis

Admin browser
  -> Next.js /admin
  -> same-origin Admin BFF
  -> Netlify Go API
  -> admin PostgreSQL role
  -> durable jobs

Media upload
  -> upload intent API
  -> direct presigned PUT ke private Supabase Storage quarantine
  -> finalize API
  -> scheduled worker
  -> immutable original + variants

Contact
  -> Netlify Forms + Netlify email
  -> verified signed webhook
  -> idempotent internal importer
  -> Supabase Inbox + activity log
```

Runtime dan provider adapter tidak boleh masuk ke domain atau application use
case. Current public URL, canonical, sitemap origin, dan route Phase 1A tetap.

## 4. Gate Delivery

### D0: Scope dan Provider Freeze

- Membuat plan, walkthrough, dan ADR remote pilot.
- Mengubah status PRD dan README menjadi `IMPLEMENTING-1D`.
- Membuat branch dari Phase 1C merge commit.
- Tidak mengubah migration, API, UI, environment, atau production.

### D1: Production Configuration dan Security

- Production config fail-closed untuk HTTPS origin, secure cookie, secret,
  storage, contact flag, dan environment byte budget.
- Ed25519 JWT production dengan local RSA fallback.
- Shared secret untuk Admin BFF ke API.
- Supabase-safe role provisioning tanpa mencabut privilege global provider.
- Migration memakai session pooler; runtime/admin memakai transaction pooler
  tanpa prepared statement dan dengan pool kecil.

### D2: Direct Media Upload dan Scheduled Worker

- Migration upload intent/quarantine.
- Presigned direct PUT, finalize, validation, ingest, variant, dan cleanup.
- S3 adapter menambah presign, head, delete, dan health operation.
- Worker one-shot setiap dua menit, batch satu, budget sekitar 25 detik.
- Daily cleanup untuk intent dan quarantine kedaluwarsa.

### D3: Remote Provider Provisioning

- Owner membuat Supabase dan Upstash dari nol mengikuti walkthrough.
- Provision private schema/roles, migration, deterministic Phase 1A seed,
  private bucket, Redis TLS, quota notification, dan baseline backup.
- Remote contract test membuktikan DB identity/privilege, Supavisor, Redis,
  dan S3 operation.
- Local QA data serta Netlify QA submission lama tidak dimigrasikan.

### D4: Netlify Shadow Runtime

- Go API adapter membungkus `App.Handler()`.
- Scheduled worker dan retention function memakai one-shot application method.
- Tambahkan API rewrite, health/readiness, protected metrics, dan warm reuse.
- Deploy Preview pertama memakai local content serta seluruh cutover flag off.

### D5: Remote Admin Acceptance

- Bootstrap owner `ilhamnugraha8944@gmail.com`.
- Verifikasi password, TOTP, recovery, session, CSRF, Origin, BFF secret,
  secure cookie, `noindex`, dan `no-store`.
- Verifikasi content/product/media/inbox/activity/account flow remote.
- Hapus seluruh record QA setelah evidence dicatat.

### D6: ApiContentSource dan Revalidation

- Tambahkan selector `CONTENT_SOURCE=local|api`; default tetap `local`.
- Next fetch memakai TTL lima menit dan targeted cache tags.
- Protected revalidation dipanggil oleh durable invalidation job.
- Dynamic product slug, API-backed sitemap, timestamp revision, dan dynamic
  media tetap menghasilkan true 404 serta initial HTML yang valid.
- API mode fail-closed jika published content wajib tidak valid.

### D7: Public Content Cutover

- Deployment khusus mengaktifkan API content dan remote admin.
- Contact API serta form sync tetap off.
- Jalankan regression Phase 1A, SEO, accessibility, Lighthouse, dan CWV.
- Buktikan rollback ke local content sebelum API mode diaktifkan kembali.

### D8: Verified Netlify Forms Inbox Sync

- Signed Netlify form webhook hanya menerima form `kontak` yang verified.
- Netlify submission ID menjadi idempotency key.
- Import membuat Inbox message dan activity log tanpa email job backend.
- Tambahkan local reconciliation command memakai ignored Netlify token.
- Aktifkan form sync setelah signature, duplicate, spam, retention, privacy,
  email, dan cleanup test lulus.

### D9: Operations dan Disaster Recovery

- Backup PostgreSQL dan seluruh media dengan SHA-256 manifest.
- Arsip dienkripsi AES-256, diunggah ke private Google Drive, dan plaintext
  lokal dihapus setelah verifikasi.
- Restore drill memakai PostgreSQL lokal bersih dan temporary object store.
- Monitoring provider, protected metrics, quota threshold, smoke schedule,
  incident response, secret rotation, load, dan outage drill.

### D10: Quality Closeout

- Jalankan frontend/backend quality, remote E2E, security, production smoke,
  Lighthouse, secret scan, link check, dan `git diff --check`.
- Perbarui PRD, README, walkthrough, quality report, runbook, dokumentasi kode
  backend, struktur folder, flow aplikasi, dan PDF.
- Phase 1D selesai hanya setelah owner mengonfirmasi production dan evidence.

## 5. Interface dan Configuration Contract

Environment flags yang wajib memiliki default aman:

| Nama | Default | Fungsi |
| --- | --- | --- |
| `CONTENT_SOURCE` | `local` | Memilih sumber konten public |
| `ADMIN_REMOTE_ENABLED` | `false` | Membuka CMS remote |
| `CONTACT_API_ENABLED` | `false` | Menjaga Go contact intake tetap mati |
| `FORM_SYNC_ENABLED` | `false` | Mengaktifkan verified submission import |

Interface baru dibatasi pada:

- media upload intent dan finalize;
- internal verified-form importer;
- protected Next revalidation;
- remote readiness dan protected metrics.

Kontrak response public harus tetap kompatibel dengan schema TypeScript Phase
1C. Product response menyediakan published/revision timestamp untuk sitemap.

## 6. Rollout dan Rollback

Urutan deployment wajib:

1. Shadow runtime dengan semua cutover flag off.
2. Remote admin tanpa mengubah public content.
3. Public content API cutover.
4. Verified form mirror.

Kegagalan content cutover dipulihkan dengan `CONTENT_SOURCE=local` dan deploy
ulang last known-good. Form transport tidak bergantung pada backend, sehingga
contact tetap dapat diterima ketika API atau Supabase mengalami gangguan.

## 7. Acceptance

Phase 1D selesai jika:

- published public content berasal dari Go API dan rollback lokal terbukti;
- CMS remote mewajibkan password + TOTP dan tidak dapat melewati BFF;
- media private, direct upload, processing, dan public variant lulus;
- publish menginvalidasi Redis dan Next cache;
- verified form muncul tepat sekali di Inbox dan email tetap dari Netlify;
- Redis outage tidak menjatuhkan public read;
- backup terenkripsi dan restore drill lulus;
- tidak ada credential, PII, atau backup masuk Git;
- Phase 1A route, SEO, accessibility, dan performance gate tetap lulus;
- owner mengonfirmasi dashboard, deployment, serta production smoke.

## 8. Provider Constraints

- Netlify Legacy Free mempunyai fixed quota dan tidak memberikan SLA.
- Go synchronous/scheduled Function harus berada di bawah execution limit.
- Supabase Free dapat pause ketika tidak aktif dan tidak menyediakan automatic
  backup project; backup off-site wajib.
- Supabase Storage backup terpisah dari database backup.
- Upstash Free tidak memiliki SLA; Redis tidak boleh menjadi source of truth.
- Region pilot diselaraskan ke US East/Ohio karena Netlify Legacy Free tidak
  menyediakan custom function region.

Referensi provider:

- [Netlify Legacy plans](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-legacy-plans/legacy-pricing-plans/)
- [Netlify Functions configuration](https://docs.netlify.com/build/functions/configuration/)
- [Netlify Lambda compatibility](https://docs.netlify.com/build/functions/lambda-compatibility/)
- [Supabase database connections](https://supabase.com/docs/guides/database/connecting-to-postgres)
- [Supabase backups](https://supabase.com/docs/guides/platform/backups)
- [Supabase S3 compatibility](https://supabase.com/docs/guides/storage/s3/compatibility)
- [Upstash Redis pricing](https://upstash.com/pricing/redis)
