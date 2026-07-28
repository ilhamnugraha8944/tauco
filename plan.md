# Phase 1A Launch Completion Plan

## Tauco Cap Badak Website

| Atribut | Nilai |
| --- | --- |
| Dokumen | Deployment dan launch completion plan |
| Versi | 1.1 |
| Tanggal | 28 Juli 2026 |
| Status saat ini | Phase 1A Complete; G0–G8 lulus |
| Scope | Phase 1A public website |
| Target hosting | Netlify Free |
| Production branch | `main` |
| Pelaksana deployment | Pemilik project |
| Peran assistant | Implementasi/QA lokal, public production verification, dan dokumentasi; tanpa commit/push/deploy |

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

Snapshot berikut merangkum release 27 Juli dan closeout 28 Juli 2026. Worktree
lokal berisi suite QA serta dokumentasi closeout yang disengaja dan belum
di-commit oleh owner:

| Pemeriksaan | Snapshot 27 Juli 2026 |
| --- | --- |
| Working tree | Berisi perubahan QA dan dokumentasi yang disengaja |
| Branch | `launch/phase-1a` |
| Remote | `origin` ke repository GitHub `tauco` |
| Pull Request | [#1](https://github.com/ilhamnugraha8944/tauco/pull/1) |
| Preview commit yang diuji | `3a07ee0b56a5feb04deb23596833df76a6fc5bb8` |
| Unit test terakhir | 73 lulus |
| Playwright terakhir | 79 lulus, 13 skip yang disengaja |
| Automated G4 Preview | 53 lulus, 7 skip yang disengaja |
| Lighthouse terakhir | 12 laporan lulus |
| Performance minimum lokal | 95 |
| SEO, Accessibility, Best Practices | 100 pada route audit |
| Production dependency audit | 0 vulnerability |
| Production commit | `2ce0a310075224b2cb8bb470d0e0ba4d0d301b98` |
| Production QA | 29/29 Playwright dan 12/12 Lighthouse lulus |

### Catatan advisori development tool

Full `npm audit` pada 27 Juli 2026 melaporkan entri high severity pada dependency
development yang berakar pada advisori `brace-expansion` melalui tree ESLint.
Dependency tersebut tidak masuk runtime production. npm hanya menawarkan
`npm audit fix --force` yang akan memaksa upgrade major ESLint, sehingga tidak
diterapkan dalam release ini.

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
**Status:** [x] Lulus; release production dan closeout worktree teridentifikasi

### Tindakan

- [x] Konfirmasi bahwa target saat ini hanya Phase 1A public website.
- [x] Konfirmasi target hosting Netlify Free.
- [x] Konfirmasi `main` sebagai production branch.
- [x] Konfirmasi deployment dan akses akun dilakukan oleh pemilik project.
- [x] Jalankan:

```powershell
git status -sb
git remote -v
git log -1 --oneline
```

- [x] Release candidate production telah di-commit dan di-merge. Worktree
  closeout saat ini sengaja belum bersih sampai owner membuat commit G8.
- [x] Pastikan remote mengarah ke repository GitHub yang benar.

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
**Status:** Lulus berdasarkan konfirmasi pemilik

### Keputusan minimum yang wajib diisi

| Keputusan | Nilai |
| --- | --- |
| Nama pengendali/pengelola data | Pemilik project, dikelola sendiri |
| Pemilik inbox operasional | Pemilik project, dikelola sendiri |
| Email notifikasi utama | `ilhamnugraha***@gmail.com` |
| Email notifikasi cadangan | `nugraha***@gmail.com` |
| Retensi maksimal 12 bulan disetujui | Ya |
| Penanggung jawab penghapusan berkala | Pemilik project |
| Penanggung jawab access/correction/deletion request | Pemilik project |
| Copy saat ini disetujui untuk launch | Ya |
| Visual provisional disetujui untuk launch | Ya |
| Data belum terverifikasi tetap dihilangkan | Ya |

Alamat email dimask pada repository publik. Nilai lengkap dikonfirmasi langsung
oleh pemilik dan harus disimpan hanya pada konfigurasi notifikasi Netlify atau
catatan operasional privat.

### Checklist

- [x] Tetapkan siapa yang boleh membuka submission Netlify.
- [x] Tetapkan email utama dan cadangan untuk notifikasi.
- [x] Setujui retensi pesan paling lama 12 bulan.
- [x] Buat jadwal penghapusan dengan review kuartalan oleh pemilik project.
- [x] Tetapkan prosedur verifikasi permintaan akses, koreksi, dan penghapusan.
- [x] Setujui bahwa website saat ini menggunakan wordmark teks dan visual
      provisional, bukan logo atau dokumentasi fasilitas resmi.
- [x] Setujui launch minimal tanpa alamat, nomor WhatsApp, harga, SKU, ukuran,
      sertifikasi, dan klaim legal yang belum terverifikasi.
- [x] Tinjau `FACT_CHECK.md`; implementasi saat ini tidak menampilkan klaim
      terlarang atau data yang belum terverifikasi.

### SOP inbox dan privasi

1. Hanya pemilik project yang membuka submission dan menerima notifikasi.
2. Permintaan akses, koreksi, atau penghapusan diverifikasi melalui kecocokan
   alamat email dan detail pesan sebelum perubahan dilakukan.
3. Inbox direview minimal setiap kuartal.
4. Submission dihapus paling lambat 12 bulan setelah diterima.
5. Ekspor atau penerusan submission ke pihak lain tidak dilakukan tanpa tujuan
   operasional yang sah.

Data yang belum diverifikasi tidak boleh dipublikasikan sampai pemilik memberi
data resmi dan bukti yang cukup, lalu `FACT_CHECK.md` diperbarui.

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
**Status:** [x] Gate teknis lulus; release dipublikasikan

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

### Hasil audit worktree 27 Juli 2026

| Gate | Hasil aktual |
| --- | --- |
| Lint | Lulus, 0 warning |
| Typecheck | Lulus |
| Unit test | 73/73 lulus |
| Production build | Lulus, 13 route Static/SSG |
| E2E lokal | 79 lulus, 13 skip yang disengaja |
| Lighthouse | 12/12 lulus; Performance 95-97, A11y/SEO/Best Practices 100 |
| Production audit | 0 vulnerability |
| Full audit | 9 high pada dev-tool chain ESLint; runtime tidak terdampak |
| Diff check | Lulus |
| Git status | Perubahan QA/dokumentasi belum di-commit |

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
- Release production telah diuji. Suite post-launch dan dokumentasi menjadi
  bagian closure commit terpisah yang dibuat owner setelah G8 lulus.

### Stop condition

- Build gagal.
- Production audit tidak 0.
- Ada regression test, accessibility, SEO, atau Lighthouse.
- Ada perubahan tidak dikenal pada worktree.

---

## G3. Konfigurasi project Netlify

**Owner:** Pemilik project  
**Status:** Lulus berdasarkan deploy dan konfirmasi pemilik

### 3.1 Akun dan biaya

- [x] Login ke team/account yang akan menjadi owner jangka panjang.
- [x] Buka **Usage & billing > Plan details**.
- [x] Pastikan plan Free atau legacy plan yang memang ingin dipertahankan.
- [x] Jangan mengaktifkan paid add-on atau pindah plan tanpa review.
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
| Nama project Netlify | `tauco-cap-badak` |
| Production origin | `https://tauco-cap-badak.netlify.app` |
| Production branch | `main` |
| Netlify project ID | Belum dicatat, tidak memblokir deploy |

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

- [x] Jangan pin plugin Next.js Netlify.
- [x] Buka menu **Forms**.
- [x] Aktifkan **Form detection**.
- [x] Pastikan Deploy Previews aktif.
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

Konfirmasi pemilik pada 27 Juli 2026:

- Netlify plan **Free Legacy** aktif dan tidak ada payment method.
- Dashboard Usage & billing mengonfirmasi model Legacy dengan allowance
  bandwidth dan build minutes.
- Form `kontak` terlihat pada Active forms.
- `NEXT_PUBLIC_SITE_URL` berlaku untuk seluruh deploy context.
- Deploy Preview PR #1 berhasil dibuat.

### Stop condition

- Nama site masih sementara.
- `NEXT_PUBLIC_SITE_URL` berbeda dari origin final.
- Environment variable hanya tersedia untuk satu context.
- Form detection belum aktif.

---

## G4. Deploy Preview dan QA

**Owner:** Pemilik project dan pemilik bisnis  
**Status:** Lulus pada preview commit yang tercatat

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

- [x] `/`
- [x] `/tauco`
- [x] `/tentang-kami`
- [x] `/produk`
- [x] `/produk/tauco-cap-badak`
- [x] `/kontak`
- [x] `/kebijakan-privasi`
- [x] Unknown product slug menghasilkan 404.
- [x] Seluruh internal link bekerja.
- [x] Mobile navigation bekerja.
- [x] Dark mode mengikuti sistem.
- [x] Image tidak broken dan tidak menyebabkan layout shift.
- [x] Copy utama tersedia pada initial HTML.

Functional QA ini dapat diulang tanpa build local:

```powershell
$env:PLAYWRIGHT_BASE_URL = "https://deploy-preview-1--tauco-cap-badak.netlify.app"
npm.cmd run qa:g4:functional
Remove-Item Env:PLAYWRIGHT_BASE_URL
```

Evidence 27 Juli 2026:

| Field | Nilai |
| --- | --- |
| Pull Request | [#1](https://github.com/ilhamnugraha8944/tauco/pull/1) |
| Release branch | `launch/phase-1a` |
| Commit yang diuji | `3a07ee0b56a5feb04deb23596833df76a6fc5bb8` |
| Deploy Preview | `https://deploy-preview-1--tauco-cap-badak.netlify.app` |
| Netlify check | Success |
| Playwright | 13/13 lulus |
| Laporan lokal | `playwright-report/g4/functional/index.html` |

Suite agregat dapat dijalankan ulang dengan:

```powershell
$env:PLAYWRIGHT_BASE_URL = "https://deploy-preview-1--tauco-cap-badak.netlify.app"
npm.cmd run qa:g4
Remove-Item Env:PLAYWRIGHT_BASE_URL
```

### 4.3 Preview SEO isolation

- [x] Page source mempunyai `noindex, nofollow`.
- [x] `/robots.txt` melarang crawling.
- [x] Canonical tetap menunjuk production origin, bukan preview URL.
- [x] Sitemap tidak memakai preview origin.
- [x] Tidak ada localhost, placeholder domain, atau random Netlify URL.
- [x] Netlify response juga memiliki `X-Robots-Tag: noindex`.

Evidence: `qa:g4:seo` lulus 9/9 pada Deploy Preview PR #1.

### 4.4 Form QA

- [x] Form `kontak` terlihat pada Active forms.
- [x] Field dan label lengkap.
- [x] Native validation bekerja.
- [x] Email invalid ditolak.
- [x] Pesan terlalu pendek ditolak.
- [x] Double-submit dikunci pada UI.
- [x] Valid submission masuk verified submissions.
- [x] Submission tidak masuk spam.
- [x] Success feedback terlihat.
- [x] Network error mempertahankan input.
- [x] Honeypot tersedia.
- [x] Tidak ada file upload.

Gunakan data test yang jelas:

```text
Nama: QA Phase 1A
Email: alamat email milik penguji
Topik: Pertanyaan umum
Pesan: Pengujian deployment Phase 1A, bukan pertanyaan pelanggan.
```

Satu submission sintetis dikirim pada 27 Juli 2026 pukul 10:34:56 WIB:

| Field | Nilai |
| --- | --- |
| Nama | `QA Phase 1A` |
| Email | `qa-phase1a@example.com` |
| Run ID | `2026-07-27-G4-PR1` |
| HTTP response | 200 |
| Netlify request ID | `01KYGT7R41AA1QXM1TG26AJMK8` |

Pemilik mengonfirmasi submission terlihat pada **Forms > Active forms >
kontak**, berstatus verified, dan tidak masuk spam. Data sintetis boleh dihapus
setelah evidence ini dicatat.

### 4.5 Accessibility dan keyboard

- [x] Navigasi dengan `Tab` dan `Shift+Tab`.
- [x] Seluruh focus indicator terlihat.
- [x] `Enter` dan `Space` mengaktifkan control yang relevan.
- [x] `Escape` tidak berlaku karena menu memakai native `<details>`, bukan
      dialog atau modal.
- [x] Form error memindahkan focus secara masuk akal.
- [x] Reflow 200 persen lulus melalui viewport proxy otomatis dan zoom browser
      fisik dikonfirmasi pemilik.
- [x] Reduced motion tidak menampilkan motion berlebihan.

Evidence: 15/15 accessibility test lulus. Axe WCAG A/AA dijalankan pada seluruh
route, light/dark, dan state form dinamis. Iframe Netlify Drawer yang
diinjeksikan provider dikecualikan secara spesifik karena bukan DOM aplikasi.
Spot-check zoom browser fisik dikonfirmasi lulus oleh pemilik.

### 4.6 Browser dan perangkat

| Target | Status | Catatan |
| --- | --- | --- |
| Edge desktop | [x] | Microsoft Edge terpasang, seluruh matrix lulus |
| Firefox desktop | [x] | Playwright Firefox 151, seluruh route lulus |
| Safari asli di iPhone/iPad/Mac | [x] | Dikonfirmasi lulus oleh pemilik; WebKit 26.5 supplemental juga lulus |
| Android Chrome | [x] | Dikonfirmasi lulus oleh pemilik; emulasi Pixel 7 juga lulus |
| Viewport sekitar 320 px | [x] | Reflow dan overflow lulus |
| Viewport sekitar 768 px | [x] | Reflow dan overflow lulus |
| Viewport sekitar 1024 px | [x] | Pergantian navigasi dan layout lulus |
| Desktop lebar | [x] | 1440 px lulus |

Playwright WebKit boleh digunakan sebagai pemeriksaan tambahan, tetapi bukan
pengganti penuh Safari pada perangkat Apple.

Ringkasan automated G4:

| Suite | Hasil |
| --- | --- |
| Functional | 13 lulus |
| SEO isolation | 9 lulus |
| Form non-live | 6 lulus, 1 live test sengaja skip |
| Accessibility | 15 lulus |
| Browser matrix | 10 lulus, 6 duplikasi sengaja skip |
| Total | 53 lulus, 7 skip yang terdokumentasi |

Konfirmasi dashboard dan perangkat oleh pemilik diterima pada 27 Juli 2026.
Dengan bukti tersebut, seluruh acceptance condition G4 terpenuhi untuk preview
commit `3a07ee0b56a5feb04deb23596833df76a6fc5bb8`.

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
**Status:** GO telah dilaksanakan oleh owner melalui merge PR #1

### Go checklist

- [x] G0 lulus.
- [x] G1 lulus.
- [x] G2 lulus.
- [x] G3 lulus.
- [x] G4 lulus.
- [x] Tidak ditemukan defect aplikasi blocker atau major pada automated QA.
- [x] Production origin dikunci ke `https://tauco-cap-badak.netlify.app`.
- [x] Owner bisnis menyetujui copy dan visual provisional untuk launch.
- [x] Inbox owner siap menerima pesan.
- [x] Rollback owner mengetahui prosedur rollback.
- [x] Usage quota Netlify masih cukup.

### Keputusan

| Field | Nilai |
| --- | --- |
| Keputusan | GO dilaksanakan melalui merge PR #1 dan production publish |
| Diputuskan oleh | Pemilik project dan bisnis |
| Tanggal dan waktu | 27 Juli 2026; publish 13:24:42 WIB |
| Commit SHA | `2ce0a310075224b2cb8bb470d0e0ba4d0d301b98` |
| Preview URL | `https://deploy-preview-1--tauco-cap-badak.netlify.app` |
| Catatan risiko yang diterima | Dev-only ESLint advisory diterima pemilik; production audit 0 |

### Catatan keputusan

Owner telah melakukan merge PR #1 dan Netlify telah memublikasikan release.
Owner mengonfirmasi pemahaman prosedur rollback pada 28 Juli 2026. Production
smoke test G6 tidak menemukan trigger rollback.

### No-Go jika

- Ada satu saja mandatory gate yang belum lulus.
- Pemilik bisnis belum menyetujui launch.
- Form belum masuk verified submissions.
- Origin/canonical belum benar.
- Tidak ada owner inbox.

---

## G6. Production deployment dan smoke test

**Owner:** Pemilik project  
**Status:** [x] Lulus pada 27 Juli 2026

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
| `/` | 200 | [x] |
| `/tauco` | 200 | [x] |
| `/tentang-kami` | 200 | [x] |
| `/produk` | 200 | [x] |
| `/produk/tauco-cap-badak` | 200 | [x] |
| `/kontak` | 200 | [x] |
| `/kebijakan-privasi` | 200 | [x] |
| `/produk/slug-tidak-ada` | 404 | [x] |

### 6.3 Production SEO verification

- [x] Production page tidak mempunyai `noindex`.
- [x] Canonical seluruh halaman memakai exact production origin.
- [x] Open Graph URL dan image URL memakai production origin.
- [x] `/robots.txt` mengizinkan crawling.
- [x] `/robots.txt` memblokir `/__forms.html`.
- [x] `/robots.txt` menunjuk production sitemap.
- [x] `/sitemap.xml` berisi tujuh published URL.
- [x] Sitemap tidak memuat localhost atau preview URL.
- [x] Homepage JSON-LD memuat `WebSite` dan `Organization`.
- [x] Interior pages mempunyai breadcrumb yang relevan.
- [x] Product JSON-LD tidak membuat Offer, rating, atau review palsu.
- [x] Schema Markup Validator tidak menemukan syntax error.
- [x] Rich Results Test hanya digunakan untuk tipe yang eligible. Tidak ada
  klaim Product rich result karena data `Offer`, rating, dan review sengaja
  tidak diterbitkan.
- [x] Initial source HTML memuat heading, copy, dan link utama.

### 6.4 Production quality

- [x] Security headers tersedia.
- [x] HTTPS valid.
- [x] Tidak ada mixed content.
- [x] Image CDN bekerja.
- [x] Tidak ada error console blocker.
- [x] Lighthouse production diperiksa pada empat route, masing-masing tiga kali.
- [x] Form production dapat dibuka.
- [x] Mobile, dark mode, keyboard, dan Axe WCAG A/AA lulus.

### Bukti

| Evidence | Nilai |
| --- | --- |
| Production URL | `https://tauco-cap-badak.netlify.app` |
| Netlify deploy ID | `6a671c07dba5f22c9cfab616` |
| Commit SHA | `2ce0a310075224b2cb8bb470d0e0ba4d0d301b98` |
| Published timestamp | 27 Juli 2026, 15:52:14 WIB |
| Robots URL | `https://tauco-cap-badak.netlify.app/robots.txt` |
| Sitemap URL | `https://tauco-cap-badak.netlify.app/sitemap.xml` |
| Schema validation URL/result | Schema.org Validator: homepage dan detail produk, 0 error dan 0 warning |
| Lighthouse result | 12/12 lulus; Performance 96-100, Accessibility 100, SEO 100, LCP maksimum 2.132 ms, CLS 0 |

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
**Status:** [x] Lulus pada 28 Juli 2026

### 7.1 Production form acceptance

- [x] Submit tepat satu pesan test sintetis dari production.
- [x] Pastikan submission berada di verified submissions.
- [x] Pastikan bukan spam.
- [x] Aktifkan email notification utama.
- [x] Backup notification secara formal ditunda dan risikonya diterima owner
  untuk launch.
- [x] Pastikan email notifikasi utama benar-benar diterima.
- [x] Balasan tidak dikirim kepada alamat sintetis.
- [x] Hapus tiga submission QA setelah acceptance selesai.
- [x] Buat pengingat review retensi.

Path notifikasi Netlify:

```text
Project configuration
  > Notifications
  > Emails and webhooks
  > Form submission notifications
```

### 7.2 Google Search Console

- [x] Tambahkan exact production URL
  `https://tauco-cap-badak.netlify.app/` sebagai URL-prefix property.
- [x] Gunakan akun owner yang akan dipertahankan jangka panjang.
- [x] Verifikasi ownership.
- [x] Pastikan verification meta tetap ada setelah deploy berikutnya.
- [x] Submit `https://tauco-cap-badak.netlify.app/sitemap.xml`.
- [x] URL Inspection homepage.
- [x] URL Inspection `/tauco`.
- [x] URL Inspection `/produk`.
- [x] URL Inspection detail produk.
- [x] Request indexing hanya setelah konten production final.
- [x] Catat baseline Pages dan Enhancements.

Baseline 28 Juli 2026:

- Sitemap sukses dan menemukan tujuh halaman.
- URL Inspection serta request indexing selesai untuk empat URL prioritas.
- Data performa/index historis belum tersedia karena property baru.
- Detail produk tersedia untuk Google, tetapi belum eligible untuk seluruh
  Product enhancement. Ini diterima untuk Phase 1A karena `Offer`, review, dan
  rating yang belum terverifikasi sengaja tidak diterbitkan.

Status pemeriksaan publik 27 Juli 2026 pukul 15:53 WIB: production merespons
200 dan memuat exact meta `google-site-verification`. Deploy
`6a671c07dba5f22c9cfab616` dipublikasikan pukul 15:52:14 WIB dari commit
`2ce0a310075224b2cb8bb470d0e0ba4d0d301b98`. Ownership sudah diverifikasi
dan sitemap berhasil dikirim.

### 7.3 Operasional

- [x] Owner inbox mengetahui cara melihat verified dan spam submission.
- [x] Owner mengetahui cara menghapus submission.
- [x] SOP access/correction/deletion terdokumentasi dan dipahami owner.
- [x] Owner mengetahui jadwal penghapusan maksimal 12 bulan.
- [x] Netlify usage diperiksa; Free Legacy masih jauh di bawah kuota.
- [x] Prosedur mewajibkan notification recipient diuji ulang jika alamat
  berubah; belum ada perubahan alamat pada launch.

### Bukti

- Screenshot Active forms.
- Production test submission: request ID
  `01KYHB76A0M1N2HJQY4HVSZQDH`, 27 Juli 2026 pukul 15:31:43 WIB,
  run ID `2026-07-27-G7-PRODUCTION`, HTTP 200.
- Notification acceptance submission: request ID
  `01KYHDDP0DCHWME30K7EV0NY3J`, 27 Juli 2026 pukul 16:10:13 WIB,
  run ID `2026-07-27-G7-NOTIFICATION`, HTTP 200, satu attempt tanpa retry.
- Owner mengonfirmasi email notification utama diterima. Notification cadangan
  belum ditambahkan dan ditunda berdasarkan keputusan owner.
- Owner mengonfirmasi ketiga submission QA telah dihapus permanen dan pengingat
  review retensi telah dibuat pada 28 Juli 2026.
- Owner mengonfirmasi akun Search Console sebagai owner jangka panjang,
  memahami SOP privasi, dan memahami prosedur rollback pada 28 Juli 2026.
- Screenshot owner menunjukkan submission G7 berada di Verified submissions
  dan tidak dikategorikan sebagai spam.
- Bukti email notification diterima.
- Screenshot Search Console property dapat diakses setelah verification.
- Screenshot sitemap berstatus **Sukses**, terakhir dibaca 27 Juli 2026,
  dan menemukan tujuh halaman.
- Owner operasional: pemilik project.
- Public verification meta: exact token ditemukan, HTTP 200, tidak ada
  `noindex`; sitemap tetap berisi tujuh production URL tanpa preview/localhost.
- Owner mengonfirmasi URL Inspection dan request indexing selesai untuk
  homepage, `/tauco`, `/produk`, dan detail produk pada 27 Juli 2026.

### Lulus jika

- Form bekerja end-to-end.
- Notification diterima owner.
- Privacy SOP dapat dijalankan.
- Search Console verified dan sitemap berhasil dikirim.

---

## G8. Dokumentasi dan penutupan Phase 1A

**Owner:** Pemilik project  
**Status:** [x] Complete pada 28 Juli 2026

### File yang harus diperbarui

- [x] `PRD.md`
- [x] `WALKTHROUGH.md`
- [x] `README.md`
- [x] `FACT_CHECK.md`
- [x] `plan.md`
- [x] `plan.pdf` diregenerasi dari `plan.md`

### Pembaruan minimum

- [x] Ubah status target deployment Phase 1A menjadi implemented.
- [x] Catat production URL.
- [x] Catat deploy ID, commit SHA, dan timestamp.
- [x] Catat hasil production route dan SEO smoke test.
- [x] Catat Netlify Forms detection dan test submission.
- [x] Catat notification owner.
- [x] Catat hasil manual Edge, Firefox, Safari, dan mobile.
- [x] Catat Search Console verification dan sitemap submission.
- [x] Perbaiki pernyataan lama: release production sudah commit/push/merge,
  sedangkan perubahan closeout G8 menunggu commit owner.
- [x] Perbarui dependency audit agar membedakan production dan development.
- [x] Tandai data resmi yang benar-benar sudah dikonfirmasi.
- [x] Jangan menandai data yang masih dihilangkan sebagai verified.

### Commit dokumentasi penutupan

Sebelum commit:

```powershell
git status --short
git diff --check
git diff -- PRD.md WALKTHROUGH.md README.md FACT_CHECK.md plan.md
```

Commit dan push dilakukan oleh pemilik project. Jika continuous deployment tetap
aktif, push ke `main` akan memicu production deployment baru.

Status version control G8: seluruh perubahan closeout siap direview, tetapi
belum di-commit atau di-push oleh assistant sesuai pembagian tanggung jawab.

### Definition of Done Phase 1A

Phase 1A hanya boleh diberi status **Complete** jika seluruh item berikut benar:

- [x] Production deployment berstatus Published.
- [x] Origin production stabil dan terdokumentasi.
- [x] Seluruh published route 200 dan unknown product 404.
- [x] Production indexable dan preview non-indexable.
- [x] Canonical, robots, sitemap, Open Graph, dan JSON-LD benar.
- [x] Form terdeteksi dan verified submission masuk.
- [x] Notification utama teruji end-to-end; backup ditunda dan risikonya
  diterima owner sebagai follow-up operasional.
- [x] Owner inbox dan SOP retensi tersedia.
- [x] Manual keyboard, Edge, Firefox, Safari, dan mobile lulus.
- [x] Search Console verified dan sitemap dikirim.
- [x] Tidak ada unverified brand claim.
- [x] Production dependency audit 0.
- [x] Evidence log lengkap.
- [x] PRD, WALKTHROUGH, README, FACT_CHECK, plan, dan PDF diperbarui.

### Final validation G8

| Pemeriksaan | Hasil 28 Juli 2026 |
| --- | --- |
| `npm.cmd run check` | Lulus |
| ESLint dan TypeScript | Lulus |
| Unit test | 73/73 lulus |
| Production build | Lulus, 13 route Static/SSG |
| Production smoke test terbaru | 29/29 lulus |
| `npm.cmd audit --omit=dev` | 0 vulnerability |
| Full `npm.cmd audit` | 9 high pada development-only ESLint chain; accepted risk |
| `git diff --check` | Lulus |
| `npm.cmd run plan:pdf` | Lulus; PDF valid dan diregenerasi dari Markdown |

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

Screenshot pemilik pada 27 Juli 2026 mengonfirmasi **Free Legacy** tanpa payment
method.

| Meter Legacy Free | Allowance |
| --- | --- |
| Bandwidth | 100 GB per bulan, hard limit |
| Build minutes | 300 menit per bulan, hard limit |
| Serverless Functions | 125.000 invocation per site per bulan |
| Edge Functions | 1 juta invocation per bulan |
| Form submission | 100 submission per site per bulan |
| Concurrent build | 1 |

Snapshot sebelum launch:

| Meter | Pemakaian | Persentase |
| --- | --- | --- |
| Bandwidth | 70 MB / 100 GB | Kurang dari 0,1% |
| Build minutes | 3 / 300 menit | 1% |
| Concurrent builds | 0 / 1 | Tidak ada build aktif |
| Team members | 1 | Sesuai single-owner operation |

Seluruh meter yang terlihat jauh di bawah threshold internal 90 persen.

Aturan:

1. Dari team dashboard, buka halaman **Projects** untuk ringkasan usage.
2. Buka **Usage & billing > Account usage insights** untuk rincian.
3. Catat persentase bandwidth, build minutes, Functions, Edge Functions, dan
   Forms serta tanggal reset.
4. Launch hanya jika seluruh meter masih di bawah 90 persen. Jika satu meter
   mencapai 90 persen, hentikan production deploy dan evaluasi lebih dahulu.

- [x] Plan aktual dikonfirmasi Free oleh pemilik.
- [x] Tidak ada payment method pada akun.
- [x] Jangan mengaktifkan paid add-on tanpa persetujuan.
- [x] Gunakan Deploy Preview untuk QA.
- [x] Hindari production deploy berulang yang tidak perlu.
- [x] Catat usage sebelum launch.
- [ ] Catat usage sesudah launch.
- [ ] Periksa usage secara rutin setelah Search Console mulai mengirim traffic.

Pada Legacy Free, build limit menghentikan build baru. Limit meter lainnya dapat
menonaktifkan build dan mem-pause site. Free tier bukan SLA bisnis.

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
| Netlify usage quota | Mingguan pada bulan pertama | Seluruh meter di bawah hard limit |
| Broken links | Setiap content release | 0 |

Target halaman pertama Google untuk kueri `tauco` dalam 6-12 bulan adalah KPI
aspiratif, bukan jaminan atau acceptance criterion pada hari launch.

---

## 8. Evidence log

| Gate | Status | Owner | Tanggal | Evidence/link | Catatan |
| --- | --- | --- | --- | --- | --- |
| G0 Scope dan baseline | [x] | Pemilik project | 2026-07-27 | Percakapan project, GitHub PR #1 | Scope, origin, branch, dan release SHA final |
| G1 Bisnis dan privasi | [x] | Pemilik project | 2026-07-27 | Konfirmasi tertulis pemilik | Owner, inbox, retensi, copy, visual, omission, dan risiko dikonfirmasi |
| G2 Release candidate lokal | [x] | Technical QA | 2026-07-27 | `.lighthouseci`, local command log | Seluruh gate teknis lulus |
| G3 Konfigurasi Netlify | [x] | Pemilik project | 2026-07-27 | Production origin, PR #1, konfirmasi dashboard | Free plan, environment scope, form detection, dan preview dikonfirmasi |
| G4 Deploy Preview QA | [x] | Pemilik project | 2026-07-27 | PR #1, `playwright-report/g4/*`, request ID Netlify | 53 PASS, 7 intentional skip; verified form dan perangkat fisik dikonfirmasi |
| G5 Go/No-Go | [x] | Pemilik project dan bisnis | 2026-07-28 | Merge PR #1, production publish, konfirmasi owner | GO dilaksanakan dan prosedur rollback dipahami |
| G6 Production dan smoke | [x] | Technical QA | 2026-07-27 | `playwright-report/production`, `.lighthouseci-production` | 29/29 Playwright dan 12/12 Lighthouse lulus |
| G7 Forms dan Search Console | [x] | Pemilik inbox dan project | 2026-07-28 | Request ID form, Search Console, sitemap, konfirmasi owner | Lulus; backup notification ditunda dan diterima sebagai follow-up |
| G8 Dokumentasi dan closure | [x] | Technical QA dan pemilik project | 2026-07-28 | PRD, README, WALKTHROUGH, FACT_CHECK, plan, PDF, final gates | Complete; commit/push closeout dilakukan owner |

## 9. Final handoff record

| Field | Nilai |
| --- | --- |
| Production URL | `https://tauco-cap-badak.netlify.app` |
| Netlify project name | `tauco-cap-badak` |
| Netlify project ID | `5a64a9b3-659d-452e-95ea-dcc973b72d12` |
| Production deploy ID | `6a671c07dba5f22c9cfab616` |
| Production commit SHA | `2ce0a310075224b2cb8bb470d0e0ba4d0d301b98` |
| Published timestamp | 27 Juli 2026, 15:52:14 WIB |
| Owner Netlify | Pemilik project |
| Owner inbox | Pemilik project |
| Search Console owner | Pemilik project |
| Sitemap submitted at | 27 Juli 2026; status Sukses, 7 halaman ditemukan |
| Phase 1A closed at | 28 Juli 2026 |
| Closed by | Technical QA dan pemilik project |

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
- [Netlify Legacy pricing plans](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-legacy-plans/legacy-pricing-plans/)
- [Netlify Legacy billing FAQ](https://docs.netlify.com/manage/accounts-and-billing/billing/billing-for-legacy-plans/billing-faq-for-legacy-plans/)
- [Google Search Console property](https://support.google.com/webmasters/answer/34592)
- [Google sitemap documentation](https://developers.google.com/search/docs/crawling-indexing/sitemaps/build-sitemap)
- [Schema Markup Validator](https://validator.schema.org/)
- [Google Rich Results Test](https://search.google.com/test/rich-results)
- [Next.js July 2026 Security Release](https://nextjs.org/blog/july-2026-security-release)
