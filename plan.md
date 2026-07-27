# Phase 1A Launch Completion Plan

## Tauco Cap Badak Website

| Atribut | Nilai |
| --- | --- |
| Dokumen | Deployment dan launch completion plan |
| Versi | 1.0 |
| Tanggal | 27 Juli 2026 |
| Status awal | Ready for owner execution |
| Scope | Phase 1A public website |
| Target hosting | Netlify Free |
| Production branch | `main` |
| Pelaksana deployment | Pemilik project |
| Peran assistant | Persiapan lokal, audit read-only, dan pembaruan dokumentasi bila diminta |

> Dokumen ini hanya menutup Phase 1A. Backend Go, Supabase/PostgreSQL, Redis,
> autentikasi, dan Admin CMS tetap berada di Phase 1B-1D.

## 1. Tujuan

Membawa implementasi Phase 1A yang sudah lulus validasi lokal menjadi website
production yang:

- tersedia pada origin HTTPS yang stabil;
- indexable hanya pada production;
- memiliki canonical, sitemap, robots, Open Graph, dan JSON-LD yang benar;
- menerima pesan melalui Netlify Forms;
- memiliki inbox owner dan prosedur privasi yang nyata;
- terhubung ke Google Search Console;
- mempunyai bukti QA, deployment, dan rollback yang dapat diaudit.

## 2. Aturan eksekusi

1. Seluruh login, koneksi GitHub, perubahan Netlify, deployment, Search Console,
   commit, dan push dilakukan oleh pemilik project.
2. Jangan mengganti subdomain production setelah Google mulai mengindeks situs,
   kecuali menggunakan migration plan dan redirect.
3. Jangan menampilkan alamat, harga, sertifikasi, legalitas, atau klaim produk
   yang belum terverifikasi.
4. Jangan menghapus origin guard jika build Netlify gagal karena environment
   variable belum tersedia.
5. Jangan menggunakan `npm audit fix --force`.
6. Jangan menandai gate selesai tanpa bukti yang disebutkan pada dokumen ini.
7. Ranking halaman pertama Google dan field Core Web Vitals bukan gate yang
   dapat dibuktikan pada hari deployment.

## 3. Snapshot kesiapan

Snapshot berikut harus diverifikasi ulang sebelum eksekusi:

| Pemeriksaan | Snapshot 27 Juli 2026 |
| --- | --- |
| Working tree | Bersih |
| Branch | `main` |
| Remote | `origin` ke repository GitHub `tauco` |
| Local vs remote | `main` sinkron dengan `origin/main` |
| Commit | `e2676d4` |
| Unit test terakhir | 73 lulus |
| Playwright terakhir | 79 lulus, 13 skip yang disengaja |
| Lighthouse terakhir | 12 laporan lulus |
| Performance minimum lokal | 90 |
| SEO, Accessibility, Best Practices | 100 pada route audit |
| Production dependency audit | 0 vulnerability |

### Catatan advisori development tool

Full `npm audit` pada 27 Juli 2026 melaporkan entri high severity pada dependency
development yang berakar pada advisori `brace-expansion` melalui tree ESLint.
Dependency tersebut tidak masuk runtime production dan npm belum menawarkan
automatic fix.

Keputusan launch yang direkomendasikan:

- production audit `npm audit --omit=dev` harus tetap 0;
- catat full-audit finding sebagai accepted development-tool risk;
- jangan menyatakan full dependency audit bernilai 0;
- pantau pembaruan upstream dan lakukan controlled upgrade terpisah;
- setiap upgrade dependency wajib diikuti seluruh quality gate.

## 4. Alur gate

```text
G0 Scope dan baseline
  -> G1 Persetujuan bisnis dan privasi
  -> G2 Release candidate lokal
  -> G3 Konfigurasi project Netlify
  -> G4 Deploy Preview dan QA
  -> G5 Go/No-Go production
  -> G6 Production deployment dan smoke test
  -> G7 Forms, Search Console, dan operasional
  -> G8 Dokumentasi dan penutupan Phase 1A
```

Jika satu gate gagal, hentikan progres pada gate tersebut. Perbaiki penyebabnya,
jalankan ulang pemeriksaan yang relevan, lalu lanjutkan.

---

## G0. Scope dan baseline

**Owner:** Pemilik project  
**Status:** [ ] Belum disetujui

### Tindakan

- [ ] Konfirmasi bahwa target saat ini hanya Phase 1A public website.
- [ ] Konfirmasi target hosting Netlify Free.
- [ ] Konfirmasi `main` sebagai production branch.
- [ ] Konfirmasi deployment dan akses akun dilakukan oleh pemilik project.
- [ ] Jalankan:

```powershell
git status -sb
git remote -v
git log -1 --oneline
```

- [ ] Pastikan working tree bersih sebelum membuat release candidate.
- [ ] Pastikan remote mengarah ke repository GitHub yang benar.

### Bukti

- Screenshot atau salinan output `git status -sb`.
- Commit SHA release candidate.
- URL repository GitHub.

### Lulus jika

- Scope dipahami sebagai Phase 1A.
- Branch, remote, dan commit release candidate jelas.
- Tidak ada perubahan lokal yang tidak sengaja tertinggal.

### Stop condition

- Remote salah.
- Branch production belum disepakati.
- Ada perubahan lokal yang belum dipahami.

---

## G1. Persetujuan bisnis dan privasi

**Owner:** Pemilik bisnis dan pemilik operasional inbox  
**Status:** [ ] Belum lengkap

### Keputusan minimum yang wajib diisi

| Keputusan | Nilai |
| --- | --- |
| Nama pengendali/pengelola data |  |
| Pemilik inbox operasional |  |
| Email notifikasi utama |  |
| Email notifikasi cadangan |  |
| Retensi maksimal 12 bulan disetujui | Ya / Tidak |
| Penanggung jawab penghapusan berkala |  |
| Penanggung jawab access/correction/deletion request |  |
| Copy saat ini disetujui untuk launch | Ya / Tidak |
| Visual provisional disetujui untuk launch | Ya / Tidak |
| Data belum terverifikasi tetap dihilangkan | Ya / Tidak |

### Checklist

- [ ] Tetapkan siapa yang boleh membuka submission Netlify.
- [ ] Tetapkan email utama dan cadangan untuk notifikasi.
- [ ] Setujui retensi pesan paling lama 12 bulan.
- [ ] Buat jadwal penghapusan, minimal review bulanan atau kuartalan.
- [ ] Tetapkan prosedur verifikasi permintaan akses, koreksi, dan penghapusan.
- [ ] Setujui bahwa website saat ini menggunakan wordmark teks dan visual
      provisional, bukan logo atau dokumentasi fasilitas resmi.
- [ ] Setujui launch minimal tanpa alamat, nomor WhatsApp, harga, SKU, ukuran,
      sertifikasi, dan klaim legal yang belum terverifikasi.
- [ ] Tinjau `FACT_CHECK.md` dan pastikan tidak ada klaim terlarang yang masuk.

### Bukti

- Keputusan pada tabel di atas terisi.
- Persetujuan tertulis pemilik bisnis.
- SOP inbox dan retensi singkat.

### Lulus jika

- Ada owner data dan inbox yang jelas.
- Retensi 12 bulan dapat benar-benar dilaksanakan.
- Copy, aset provisional, dan omission data mendapat persetujuan.

### Stop condition

- Tidak ada orang yang bertanggung jawab atas inbox.
- Retensi dan penghapusan belum disetujui.
- Pemilik belum menyetujui konten atau aset provisional.

---

## G2. Release candidate lokal

**Owner:** Pemilik project  
**Status:** [ ] Belum dijalankan untuk release

### Persiapan

`npm.cmd ci` dijalankan sekali untuk membuat instalasi dependency bersih dari
`package-lock.json`. Perintah ini tidak perlu dijalankan setiap kali memakai
`npm.cmd run dev`.

### Perintah wajib

```powershell
npm.cmd ci
npm.cmd run lint
npm.cmd run typecheck
npm.cmd run test
npm.cmd run build
npm.cmd run test:e2e
npm.cmd run lighthouse
npm.cmd audit --omit=dev
npm.cmd audit
git diff --check
git status -sb
```

### Expected result

| Gate | Expected |
| --- | --- |
| Lint | Exit code 0, tanpa warning |
| Typecheck | Exit code 0 |
| Unit test | Semua lulus |
| Build | Exit code 0 |
| E2E | Semua test wajib lulus; skip hanya yang terdokumentasi |
| Lighthouse | Seluruh assertion lulus |
| Production audit | 0 vulnerability |
| Full audit | Finding dev-tool ditinjau dan dicatat |
| Diff check | Tidak ada whitespace error |
| Git status | Hanya perubahan release yang disengaja |

### Bukti

- Tanggal dan waktu eksekusi.
- Ringkasan setiap command.
- Screenshot hasil Lighthouse.
- Salinan production dependency audit.
- Accepted-risk note untuk full audit.

### Lulus jika

- Seluruh runtime, test, build, dan performance gate lulus.
- Tidak ada vulnerability production yang diketahui.
- Full-audit finding sudah dipahami dan diterima secara eksplisit.

### Stop condition

- Build gagal.
- Production audit tidak 0.
- Ada regression test, accessibility, SEO, atau Lighthouse.
- Ada perubahan tidak dikenal pada worktree.

---

## G3. Konfigurasi project Netlify

**Owner:** Pemilik project  
**Status:** [ ] Belum dilakukan

### 3.1 Akun dan biaya

- [ ] Login ke team/account yang akan menjadi owner jangka panjang.
- [ ] Buka **Usage & billing > Plan details**.
- [ ] Pastikan plan Free atau legacy plan yang memang ingin dipertahankan.
- [ ] Jangan mengaktifkan paid add-on atau pindah plan tanpa review.
- [ ] Catat tanggal reset credit bulanan.

### 3.2 Import repository

1. Buka Netlify.
2. Pilih **Add new project**.
3. Pilih **Import an existing project**.
4. Pilih GitHub.
5. Beri akses hanya ke repository yang diperlukan bila opsi tersedia.
6. Pilih repository `tauco`.
7. Pilih `main` sebagai production branch.

Jika build awal gagal karena `NEXT_PUBLIC_SITE_URL` belum tersedia, biarkan
gagal. Itu adalah proteksi yang disengaja. Jangan retry production sebelum nama
site dan environment variable benar.

### 3.3 Finalisasi origin

Urutan wajib:

1. Pilih nama project final.
2. Pastikan nama tersedia.
3. Salin exact Netlify URL.
4. Baru set environment variable.

Pilihan nama:

```text
tauco-cap-badak
tauco-cap-badak-cianjur
```

Catat hasil final:

| Data | Nilai |
| --- | --- |
| Nama project Netlify |  |
| Production origin | `https://________________.netlify.app` |
| Production branch | `main` |
| Netlify project ID |  |

Tambahkan environment variable dengan scope Builds dan seluruh deploy context:

```text
NEXT_PUBLIC_SITE_URL=https://nama-final.netlify.app
```

Aturan nilai:

- HTTPS;
- hostname publik;
- tanpa path tambahan;
- tanpa query atau hash;
- bukan Deploy Preview URL;
- sama persis dengan production origin.

### 3.4 Search Console token lebih awal

Langkah ini opsional sebelum preview, tetapi menghemat satu production deploy:

1. Setelah origin final diketahui, buat Search Console URL-prefix property.
2. Pilih metode HTML tag.
3. Salin nilai token dari atribut `content`.
4. Tambahkan ke environment Netlify:

```text
GOOGLE_SITE_VERIFICATION=token_google
```

Verification baru dilakukan setelah production online.

### 3.5 Build dan Forms

Pastikan source-of-truth dari `netlify.toml` terbaca:

```text
Build command: npm run build
Publish directory: .next
Node version: 22
```

- [ ] Jangan pin plugin Next.js Netlify.
- [ ] Buka menu **Forms**.
- [ ] Aktifkan **Form detection**.
- [ ] Pastikan Deploy Previews aktif.
- [ ] Jangan retry production deploy dahulu jika ingin menjalankan strict
      preview-first flow.

### Bukti

- Screenshot nama project dan production URL.
- Screenshot environment variable, dengan token sensitif disamarkan.
- Screenshot build settings.
- Screenshot Form detection aktif.

### Lulus jika

- Origin final sudah stabil.
- Environment variable tepat.
- Build setting benar.
- Form detection dan Deploy Previews aktif.

### Stop condition

- Nama site masih sementara.
- `NEXT_PUBLIC_SITE_URL` berbeda dari origin final.
- Environment variable hanya tersedia untuk satu context.
- Form detection belum aktif.

---

## G4. Deploy Preview dan QA

**Owner:** Pemilik project dan pemilik bisnis  
**Status:** [ ] Belum dilakukan

### 4.1 Membuat Deploy Preview

Gunakan perubahan dokumentasi launch yang nyata sebagai release branch:

```powershell
git switch -c launch/phase-1a
git add plan.md plan.pdf README.md WALKTHROUGH.md PRD.md FACT_CHECK.md
git commit -m "docs: prepare phase 1a launch"
git push -u origin launch/phase-1a
```

Tambahkan hanya file yang memang berubah. Jangan memakai `git add .` tanpa
memeriksa `git status`.

Di GitHub:

1. Buat Pull Request dari `launch/phase-1a` ke `main`.
2. Tunggu status Netlify Deploy Preview selesai.
3. Buka URL dengan format:

```text
https://deploy-preview-<nomor-pr>--<nama-site>.netlify.app
```

4. Jangan merge sebelum seluruh checklist G4 dan G5 lulus.

### 4.2 Functional QA

- [ ] `/`
- [ ] `/tauco`
- [ ] `/tentang-kami`
- [ ] `/produk`
- [ ] `/produk/tauco-cap-badak`
- [ ] `/kontak`
- [ ] `/kebijakan-privasi`
- [ ] Unknown product slug menghasilkan 404.
- [ ] Seluruh internal link bekerja.
- [ ] Mobile navigation bekerja.
- [ ] Dark mode mengikuti sistem.
- [ ] Image tidak broken dan tidak menyebabkan layout shift.
- [ ] Copy utama tersedia pada initial HTML.

### 4.3 Preview SEO isolation

- [ ] Page source mempunyai `noindex, nofollow`.
- [ ] `/robots.txt` melarang crawling.
- [ ] Canonical tetap menunjuk production origin, bukan preview URL.
- [ ] Sitemap tidak memakai preview origin.
- [ ] Tidak ada localhost, placeholder domain, atau random Netlify URL.
- [ ] Netlify response juga memiliki `X-Robots-Tag: noindex`.

### 4.4 Form QA

- [ ] Form `kontak` terlihat pada Active forms.
- [ ] Field dan label lengkap.
- [ ] Native validation bekerja.
- [ ] Email invalid ditolak.
- [ ] Pesan terlalu pendek ditolak.
- [ ] Double-submit tidak menghasilkan dua submission.
- [ ] Valid submission masuk verified submissions.
- [ ] Submission tidak masuk spam.
- [ ] Success feedback terlihat.
- [ ] Network error mempertahankan input.
- [ ] Honeypot tersedia.
- [ ] Tidak ada file upload.

Gunakan data test yang jelas:

```text
Nama: QA Phase 1A
Email: alamat email milik penguji
Topik: Pertanyaan umum
Pesan: Pengujian deployment Phase 1A, bukan pertanyaan pelanggan.
```

### 4.5 Accessibility dan keyboard

- [ ] Navigasi dengan `Tab` dan `Shift+Tab`.
- [ ] Seluruh focus indicator terlihat.
- [ ] `Enter` dan `Space` mengaktifkan control yang relevan.
- [ ] `Escape` menutup menu mobile bila berlaku.
- [ ] Form error memindahkan focus secara masuk akal.
- [ ] Zoom browser 200 persen tetap dapat digunakan.
- [ ] Reduced motion tidak menampilkan motion berlebihan.

### 4.6 Browser dan perangkat

| Target | Status | Catatan |
| --- | --- | --- |
| Edge desktop | [ ] |  |
| Firefox desktop | [ ] |  |
| Safari asli di iPhone/iPad/Mac | [ ] |  |
| Android Chrome | [ ] |  |
| Viewport sekitar 320 px | [ ] |  |
| Viewport sekitar 768 px | [ ] |  |
| Viewport sekitar 1024 px | [ ] |  |
| Desktop lebar | [ ] |  |

Playwright WebKit boleh digunakan sebagai pemeriksaan tambahan, tetapi bukan
pengganti penuh Safari pada perangkat Apple.

### Bukti

- URL Deploy Preview.
- Screenshot desktop, mobile, dark mode, dan form success.
- Screenshot `noindex` dan `robots.txt`.
- ID submission test.
- Tabel browser yang sudah diisi.
- Daftar defect dan resolution.

### Lulus jika

- Tidak ada defect blocker atau major.
- Preview terisolasi dari indexing.
- Form terdeteksi dan menerima verified submission.
- Browser, keyboard, mobile, dan dark mode lulus.

### Stop condition

- Preview indexable.
- Canonical salah.
- Submission tidak muncul.
- Safari atau browser wajib belum diuji.
- Ada broken route, broken image, atau accessibility blocker.

---

## G5. Go/No-Go production

**Owner:** Pemilik project dan pemilik bisnis  
**Status:** [ ] Belum diputuskan

### Go checklist

- [ ] G0 lulus.
- [ ] G1 lulus.
- [ ] G2 lulus.
- [ ] G3 lulus.
- [ ] G4 lulus.
- [ ] Seluruh defect blocker dan major ditutup.
- [ ] Production origin dikunci.
- [ ] Owner bisnis menyetujui copy dan visual final.
- [ ] Inbox owner siap menerima pesan.
- [ ] Rollback owner mengetahui prosedur rollback.
- [ ] Credit Netlify masih cukup.

### Keputusan

| Field | Nilai |
| --- | --- |
| Keputusan | GO / NO-GO |
| Diputuskan oleh |  |
| Tanggal dan waktu |  |
| Commit SHA |  |
| Preview URL |  |
| Catatan risiko yang diterima |  |

### No-Go jika

- Ada satu saja mandatory gate yang belum lulus.
- Pemilik bisnis belum menyetujui launch.
- Form belum masuk verified submissions.
- Origin/canonical belum benar.
- Tidak ada owner inbox.

---

## G6. Production deployment dan smoke test

**Owner:** Pemilik project  
**Status:** [ ] Belum dilakukan

### 6.1 Deployment

1. Pastikan Pull Request menunjukkan commit yang telah diuji.
2. Merge Pull Request ke `main`.
3. Tunggu Netlify production deploy.
4. Pastikan status **Published**.
5. Catat deploy ID dan commit SHA.
6. Jangan rename project setelah deployment.

### 6.2 Route smoke test

| URL | Expected | Status |
| --- | ---: | --- |
| `/` | 200 | [ ] |
| `/tauco` | 200 | [ ] |
| `/tentang-kami` | 200 | [ ] |
| `/produk` | 200 | [ ] |
| `/produk/tauco-cap-badak` | 200 | [ ] |
| `/kontak` | 200 | [ ] |
| `/kebijakan-privasi` | 200 | [ ] |
| `/produk/slug-tidak-ada` | 404 | [ ] |

### 6.3 Production SEO verification

- [ ] Production page tidak mempunyai `noindex`.
- [ ] Canonical seluruh halaman memakai exact production origin.
- [ ] Open Graph URL dan image URL memakai production origin.
- [ ] `/robots.txt` mengizinkan crawling.
- [ ] `/robots.txt` memblokir `/__forms.html`.
- [ ] `/robots.txt` menunjuk production sitemap.
- [ ] `/sitemap.xml` berisi tujuh published URL.
- [ ] Sitemap tidak memuat localhost atau preview URL.
- [ ] Homepage JSON-LD memuat `WebSite` dan `Organization`.
- [ ] Interior pages mempunyai breadcrumb yang relevan.
- [ ] Product JSON-LD tidak membuat Offer, rating, atau review palsu.
- [ ] Schema Markup Validator tidak menemukan syntax error.
- [ ] Rich Results Test digunakan hanya untuk tipe yang eligible.
- [ ] Initial source HTML memuat heading, copy, dan link utama.

### 6.4 Production quality

- [ ] Security headers tersedia.
- [ ] HTTPS valid.
- [ ] Tidak ada mixed content.
- [ ] Image CDN bekerja.
- [ ] Tidak ada error console blocker.
- [ ] Lighthouse production diperiksa pada route representatif.
- [ ] Form production dapat dibuka.
- [ ] Mobile, dark mode, dan keyboard spot-check lulus.

### Bukti

| Evidence | Nilai |
| --- | --- |
| Production URL |  |
| Netlify deploy ID |  |
| Commit SHA |  |
| Published timestamp |  |
| Robots URL |  |
| Sitemap URL |  |
| Schema validation URL/result |  |
| Lighthouse result |  |

### Lulus jika

- Deploy berstatus Published.
- Route, 404, SEO, headers, image, dan interaction lulus pada production.

### Stop condition dan rollback trigger

Rollback segera bila:

- homepage atau route utama 5xx;
- canonical menunjuk domain salah;
- production masih `noindex`;
- sitemap memakai origin salah;
- form menyebabkan error besar atau kebocoran data;
- asset utama gagal dimuat;
- ada regression accessibility yang menghalangi penggunaan.

---

## G7. Forms, Search Console, dan operasional

**Owner:** Pemilik inbox dan pemilik project  
**Status:** [ ] Belum dilakukan

### 7.1 Production form acceptance

- [ ] Submit pesan test dari production.
- [ ] Pastikan submission berada di verified submissions.
- [ ] Pastikan bukan spam.
- [ ] Aktifkan email notification utama.
- [ ] Aktifkan backup notification.
- [ ] Pastikan email notifikasi benar-benar diterima.
- [ ] Balasan tidak dikirim kepada data test yang tidak sah.
- [ ] Hapus submission test jika tidak lagi dibutuhkan.
- [ ] Buat pengingat review retensi.

Path notifikasi Netlify:

```text
Project configuration
  > Notifications
  > Emails and webhooks
  > Form submission notifications
```

### 7.2 Google Search Console

- [ ] Tambahkan exact production URL sebagai URL-prefix property.
- [ ] Gunakan akun bisnis yang akan menjadi owner jangka panjang.
- [ ] Verifikasi ownership.
- [ ] Pastikan verification meta tetap ada setelah deploy berikutnya.
- [ ] Submit `https://<production-origin>/sitemap.xml`.
- [ ] URL Inspection homepage.
- [ ] URL Inspection `/tauco`.
- [ ] URL Inspection `/produk`.
- [ ] URL Inspection detail produk.
- [ ] Request indexing hanya setelah konten production final.
- [ ] Catat baseline Pages dan Enhancements.

### 7.3 Operasional

- [ ] Owner inbox mengetahui cara melihat verified dan spam submission.
- [ ] Owner mengetahui cara menghapus submission.
- [ ] Owner menjalankan SOP access/correction/deletion request.
- [ ] Owner mengetahui jadwal penghapusan maksimal 12 bulan.
- [ ] Netlify usage/credit diperiksa.
- [ ] Notification recipient diuji ulang jika alamat berubah.

### Bukti

- Screenshot Active forms.
- ID dan timestamp production test submission.
- Bukti email notification diterima.
- Screenshot Search Console verified.
- Screenshot sitemap submitted.
- Nama owner operasional.

### Lulus jika

- Form bekerja end-to-end.
- Notification diterima owner.
- Privacy SOP dapat dijalankan.
- Search Console verified dan sitemap berhasil dikirim.

---

## G8. Dokumentasi dan penutupan Phase 1A

**Owner:** Pemilik project  
**Status:** [ ] Belum dilakukan

### File yang harus diperbarui

- [ ] `PRD.md`
- [ ] `WALKTHROUGH.md`
- [ ] `README.md`
- [ ] `FACT_CHECK.md`
- [ ] `plan.md`

### Pembaruan minimum

- [ ] Ubah status target deployment Phase 1A menjadi implemented.
- [ ] Catat production URL.
- [ ] Catat deploy ID, commit SHA, dan timestamp.
- [ ] Catat hasil production route dan SEO smoke test.
- [ ] Catat Netlify Forms detection dan test submission.
- [ ] Catat notification owner.
- [ ] Catat hasil manual Edge, Firefox, Safari, dan mobile.
- [ ] Catat Search Console verification dan sitemap submission.
- [ ] Perbaiki pernyataan lama bahwa repository belum commit/push.
- [ ] Perbarui dependency audit agar membedakan production dan development.
- [ ] Tandai data resmi yang benar-benar sudah dikonfirmasi.
- [ ] Jangan menandai data yang masih dihilangkan sebagai verified.

### Commit dokumentasi penutupan

Sebelum commit:

```powershell
git status --short
git diff --check
git diff -- PRD.md WALKTHROUGH.md README.md FACT_CHECK.md plan.md
```

Commit dan push dilakukan oleh pemilik project. Jika continuous deployment tetap
aktif, push ke `main` akan memicu production deployment baru.

### Definition of Done Phase 1A

Phase 1A hanya boleh diberi status **Complete** jika seluruh item berikut benar:

- [ ] Production deployment berstatus Published.
- [ ] Origin production stabil dan terdokumentasi.
- [ ] Seluruh published route 200 dan unknown product 404.
- [ ] Production indexable dan preview non-indexable.
- [ ] Canonical, robots, sitemap, Open Graph, dan JSON-LD benar.
- [ ] Form terdeteksi dan verified submission masuk.
- [ ] Notification utama dan cadangan teruji.
- [ ] Owner inbox dan SOP retensi tersedia.
- [ ] Manual keyboard, Edge, Firefox, Safari, dan mobile lulus.
- [ ] Search Console verified dan sitemap dikirim.
- [ ] Tidak ada unverified brand claim.
- [ ] Production dependency audit 0.
- [ ] Evidence log lengkap.
- [ ] PRD, WALKTHROUGH, README, FACT_CHECK, dan plan diperbarui.

---

## 5. Rollback plan

### Kapan rollback

Gunakan rollback bila production mempunyai blocker yang tidak dapat diperbaiki
dalam waktu singkat, terutama:

- 5xx atau blank page;
- accidental `noindex`;
- canonical atau sitemap salah;
- broken navigation;
- broken critical image;
- form mengirim ke tujuan yang salah;
- privacy atau data exposure issue.

### Cara rollback Netlify

1. Buka **Deploys**.
2. Pilih production deploy sukses sebelumnya.
3. Buka detail deploy.
4. Pilih **Publish Deploy**.
5. Verifikasi homepage, robots, sitemap, dan form.
6. Catat rollback timestamp dan alasan.
7. Perbaiki source pada branch terpisah.
8. Jalankan ulang gate terkait sebelum redeploy.

Rollback Netlify memublikasikan atomic deploy sebelumnya. Push production baru
akan menggantikan hasil rollback jika auto publishing masih aktif.

### Rollback record

| Field | Nilai |
| --- | --- |
| Incident time |  |
| Bad deploy ID |  |
| Restored deploy ID |  |
| Trigger |  |
| Owner |  |
| Recovery verified at |  |
| Follow-up issue |  |

---

## 6. Free-tier guardrail

Untuk credit-based Free plan berdasarkan informasi Netlify pada 27 Juli 2026:

| Meter | Nilai saat dokumen dibuat |
| --- | --- |
| Monthly credit | 300 credits dengan hard limit |
| Production deploy | 15 credits per published production deploy |
| Deploy Preview | Tidak memakai deployment credits |
| Branch deploy | Tidak memakai deployment credits |
| Failed deploy | Tidak memakai deployment credits |
| Form submission | Gratis dan unlimited pada credit-based plans |
| Bandwidth | 20 credits per GB |
| Web requests | 2 credits per 10.000 requests |
| Compute | 10 credits per GB-hour |

Aturan:

- [ ] Periksa plan aktual di dashboard karena legacy plan dapat berbeda.
- [ ] Jangan mengaktifkan paid add-on tanpa persetujuan.
- [ ] Gunakan Deploy Preview untuk QA.
- [ ] Hindari production deploy berulang yang tidak perlu.
- [ ] Catat credit sebelum dan sesudah launch.
- [ ] Periksa usage secara rutin setelah Search Console mulai mengirim traffic.

Free plan dapat pause ketika hard limit habis. Free tier bukan SLA bisnis.

---

## 7. Post-launch monitoring

Item berikut dimulai setelah launch dan tidak menahan penutupan engineering pada
hari deployment:

| Metric | Waktu review | Target/tujuan |
| --- | --- | --- |
| Search Console indexing | Mingguan pada bulan pertama | Seluruh intended page ditemukan |
| Impression dan query | Setiap 28 hari | Baseline dan tren non-brand |
| Average position | Setiap 28 hari | Tren, bukan fluktuasi harian |
| LCP p75 | Setelah field data tersedia | <= 2,5 detik |
| INP p75 | Setelah field data tersedia | <= 200 ms |
| CLS p75 | Setelah field data tersedia | <= 0,1 |
| Valid form submission | Mingguan | Tidak ada pesan yang terlewat |
| Netlify credits | Mingguan pada bulan pertama | Tetap di bawah hard limit |
| Broken links | Setiap content release | 0 |

Target halaman pertama Google untuk kueri `tauco` dalam 6-12 bulan adalah KPI
aspiratif, bukan jaminan atau acceptance criterion pada hari launch.

---

## 8. Evidence log

| Gate | Status | Owner | Tanggal | Evidence/link | Catatan |
| --- | --- | --- | --- | --- | --- |
| G0 Scope dan baseline | [ ] |  |  |  |  |
| G1 Bisnis dan privasi | [ ] |  |  |  |  |
| G2 Release candidate lokal | [ ] |  |  |  |  |
| G3 Konfigurasi Netlify | [ ] |  |  |  |  |
| G4 Deploy Preview QA | [ ] |  |  |  |  |
| G5 Go/No-Go | [ ] |  |  |  |  |
| G6 Production dan smoke | [ ] |  |  |  |  |
| G7 Forms dan Search Console | [ ] |  |  |  |  |
| G8 Dokumentasi dan closure | [ ] |  |  |  |  |

## 9. Final handoff record

| Field | Nilai |
| --- | --- |
| Production URL |  |
| Netlify project name |  |
| Netlify project ID |  |
| Production deploy ID |  |
| Production commit SHA |  |
| Published timestamp |  |
| Owner Netlify |  |
| Owner inbox |  |
| Search Console owner |  |
| Sitemap submitted at |  |
| Phase 1A closed at |  |
| Closed by |  |

## 10. Referensi

- [PRD Phase 1A](./PRD.md)
- [Implementation walkthrough](./WALKTHROUGH.md)
- [Fact-check dan publishing guard](./FACT_CHECK.md)
- [Repository runbook](./README.md)
- [Next.js on Netlify](https://docs.netlify.com/build/frameworks/framework-setup-guides/nextjs/overview/)
- [Deploy Previews](https://docs.netlify.com/deploy/deploy-types/deploy-previews/)
- [Manage and roll back deploys](https://docs.netlify.com/deploy/manage-deploys/manage-deploys-overview/)
- [Netlify Forms setup](https://docs.netlify.com/manage/forms/setup/)
- [Next.js Forms workaround](https://opennext.js.org/netlify/forms)
- [Form notifications](https://docs.netlify.com/manage/forms/notifications/)
- [Netlify credit-based plans](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-credit-based-plans/credit-based-pricing-plans/)
- [Google Search Console property](https://support.google.com/webmasters/answer/34592)
- [Google sitemap documentation](https://developers.google.com/search/docs/crawling-indexing/sitemaps/build-sitemap)
- [Schema Markup Validator](https://validator.schema.org/)
- [Google Rich Results Test](https://search.google.com/test/rich-results)
- [Next.js July 2026 Security Release](https://nextjs.org/blog/july-2026-security-release)
