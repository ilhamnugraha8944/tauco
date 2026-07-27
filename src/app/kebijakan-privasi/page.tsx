import type { Metadata } from "next";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { Container } from "@/components/container";
import { createMetadata, staticPageMetadata } from "@/lib/seo";

export const metadata: Metadata = createMetadata(
  staticPageMetadata.privacy,
);

export default function PrivacyPage() {
  return (
    <section className="legal-page">
      <Container>
        <Breadcrumbs
          items={[
            { label: "Beranda", href: "/" },
            { label: "Kebijakan Privasi", href: "/kebijakan-privasi" },
          ]}
        />
        <article className="legal-content">
          <h1>Kebijakan Privasi</h1>
          <p className="legal-lead">
            Kebijakan ini menjelaskan data yang dikumpulkan ketika Anda
            mengirim pesan melalui formulir kontak.
          </p>

          <section>
            <h2>Data yang dikumpulkan</h2>
            <p>
              Formulir menyimpan nama, alamat email, nomor telepon bila diisi,
              topik, isi pesan, dan persetujuan privasi.
            </p>
          </section>

          <section>
            <h2>Tujuan penggunaan</h2>
            <p>
              Data digunakan untuk membaca, menindaklanjuti, dan menjawab
              pertanyaan yang Anda kirimkan. Website ini tidak menyediakan atau
              membuat akun pelanggan.
            </p>
          </section>

          <section>
            <h2>Penyimpanan dan retensi</h2>
            <p>
              Pesan diproses melalui Netlify Forms dan disimpan paling lama 12
              bulan sejak diterima. Setelah batas retensi berakhir, pesan
              dihapus.
            </p>
          </section>

          <section>
            <h2>Pihak yang dapat mengakses data</h2>
            <p>
              Pesan yang dikirim hanya dapat diakses oleh pengelola inbox yang
              ditunjuk pemilik Tauco Cap Badak untuk menanganinya. Netlify
              dapat memproses data sebagai penyedia layanan formulir.
            </p>
          </section>

          <section>
            <h2>Pembagian data</h2>
            <p>
              Data tidak dijual. Data dapat diproses oleh penyedia hosting dan
              layanan formulir yang digunakan untuk menjalankan website.
            </p>
          </section>

          <section>
            <h2>Permintaan terkait data</h2>
            <p>
              Untuk meminta akses, koreksi, atau penghapusan data, gunakan
              formulir kontak dan pilih topik pertanyaan umum. Cantumkan alamat
              email yang sama dengan pesan awal agar permintaan dapat
              ditelusuri.
            </p>
          </section>
        </article>
      </Container>
    </section>
  );
}
