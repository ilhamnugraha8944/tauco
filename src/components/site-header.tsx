import { Container } from "@/components/container";

const navigation = [
  { href: "/", label: "Beranda" },
  { href: "/tauco", label: "Mengenal Tauco" },
  { href: "/tentang-kami", label: "Tentang Kami" },
  { href: "/produk", label: "Produk" },
  { href: "/kontak", label: "Kontak" },
] as const;

export function SiteHeader() {
  return (
    <header className="site-header">
      <Container className="flex h-[72px] items-center justify-between gap-8">
        <a
          href="/"
          className="wordmark shrink-0"
          aria-label="Tauco Cap Badak, beranda"
        >
          Tauco Cap Badak
        </a>

        <nav aria-label="Navigasi utama" className="hidden lg:block">
          <ul className="flex items-center gap-7">
            {navigation.map((item) => (
              <li key={item.href}>
                <a className="nav-link" href={item.href}>
                  {item.label}
                </a>
              </li>
            ))}
          </ul>
        </nav>

        <details className="mobile-menu lg:hidden">
          <summary>Menu</summary>
          <nav aria-label="Navigasi seluler">
            <ul>
              {navigation.map((item) => (
                <li key={item.href}>
                  <a href={item.href}>{item.label}</a>
                </li>
              ))}
            </ul>
          </nav>
        </details>
      </Container>
    </header>
  );
}
