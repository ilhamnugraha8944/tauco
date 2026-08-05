import type { Metadata } from "next";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";

import { isAdminCMSEnabled } from "@/features/admin/config";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Admin CMS | Tauco Cap Badak",
  robots: { index: false, follow: false, nocache: true },
};

export default function AdminLayout({ children }: { children: ReactNode }) {
  if (!isAdminCMSEnabled()) {
    notFound();
  }

  return children;
}
