import type { ReactNode } from "react";

import { AdminShell } from "@/features/admin/admin-shell";

export default function AdminShellLayout({ children }: { children: ReactNode }) {
  return <AdminShell>{children}</AdminShell>;
}
