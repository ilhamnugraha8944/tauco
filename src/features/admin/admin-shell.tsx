"use client";

import {
  Article,
  ArrowSquareOut,
  ClockCounterClockwise,
  EnvelopeSimple,
  Images,
  Package,
  SignOut,
  UserCircle,
} from "@phosphor-icons/react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { type ReactNode, useEffect, useState } from "react";

import { adminAPI, type AdminUser } from "@/features/admin/admin-api";

const modules = [
  { label: "Konten", href: "/admin/content", icon: Article, available: true },
  { label: "Produk", icon: Package, available: false },
  { label: "Media", href: "/admin/media", icon: Images, available: true },
  { label: "Inbox", icon: EnvelopeSimple, available: false },
  { label: "Aktivitas", icon: ClockCounterClockwise, available: false },
  { label: "Akun", href: "/admin/account", icon: UserCircle, available: true },
] as const;

export function AdminShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<AdminUser>();

  useEffect(() => {
    let active = true;
    adminAPI.me().then(
      (response) => {
        if (!active) return;
        if (!response.data.mfaEnabled) {
          router.replace("/admin/setup-totp");
          return;
        }
        setUser(response.data);
      },
      () => { if (active) router.replace("/admin/login"); },
    );
    return () => { active = false; };
  }, [router]);

  async function logout() {
    try {
      await adminAPI.logout();
    } finally {
      router.replace("/admin/login");
      router.refresh();
    }
  }

  if (!user) {
    return (
      <section className="admin-root admin-shell-loading" aria-live="polite">
        <p>Memeriksa session admin...</p>
      </section>
    );
  }

  return (
    <section className="admin-root admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-sidebar-head">
          <Link href="/admin/content" className="admin-wordmark">Tauco Cap Badak</Link>
          <p>Admin CMS</p>
        </div>

        <nav aria-label="Navigasi CMS">
          <ul>
            {modules.map((item) => {
              const Icon = item.icon;
              return (
                <li key={item.label}>
                  {item.available && item.href ? (
                    <Link href={item.href} className="admin-nav-link" aria-current={pathname.startsWith(item.href) ? "page" : undefined}>
                      <Icon size={20} aria-hidden="true" />
                      {item.label}
                    </Link>
                  ) : (
                    <span className="admin-nav-link" aria-disabled="true">
                      <Icon size={20} aria-hidden="true" />
                      {item.label}
                      <small>Segera</small>
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="admin-sidebar-footer">
          <a href="/" className="admin-nav-link" target="_blank" rel="noreferrer">
            <ArrowSquareOut size={20} aria-hidden="true" />
            Website publik
          </a>
          <button type="button" className="admin-nav-link" onClick={logout}>
            <SignOut size={20} aria-hidden="true" />
            Keluar
          </button>
          <p title={user.email}>{user.email}</p>
        </div>
      </aside>

      <div className="admin-workspace">{children}</div>
    </section>
  );
}
