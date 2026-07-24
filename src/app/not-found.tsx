import { ButtonLink } from "@/components/button-link";
import { Container } from "@/components/container";

export default function NotFound() {
  return (
    <section className="not-found-page">
      <Container>
        <p className="eyebrow">404</p>
        <h1>Halaman tidak ditemukan</h1>
        <p>
          Alamat yang Anda buka tidak tersedia atau sudah tidak digunakan.
        </p>
        <ButtonLink href="/">Kembali ke beranda</ButtonLink>
      </Container>
    </section>
  );
}
