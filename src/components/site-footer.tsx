import { Container } from "@/components/container";

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <Container>
        <div className="footer-grid">
          <div>
            <p className="wordmark">Tauco Cap Badak</p>
            <p className="mt-4 max-w-md text-sm leading-6 text-muted">
              Informasi produk tauco dari Cianjur dan panduan mengenal tauco.
            </p>
          </div>
          <nav aria-label="Navigasi footer">
            <ul className="footer-links">
              <li>
                <a href="/tauco">Mengenal Tauco</a>
              </li>
              <li>
                <a href="/tentang-kami">Tentang Kami</a>
              </li>
              <li>
                <a href="/produk">Produk</a>
              </li>
              <li>
                <a href="/kontak">Kontak</a>
              </li>
              <li>
                <a href="/kebijakan-privasi">Kebijakan Privasi</a>
              </li>
            </ul>
          </nav>
        </div>
        <div className="footer-base">
          <p>© {new Date().getFullYear()} Tauco Cap Badak.</p>
          <p>Cianjur, Jawa Barat</p>
        </div>
      </Container>
    </footer>
  );
}
