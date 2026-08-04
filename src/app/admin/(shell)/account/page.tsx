import type { Metadata } from "next";

import { AccountPanel } from "@/features/admin/account-panel";

export const metadata: Metadata = { title: "Akun Admin | Tauco Cap Badak" };

export default function AdminAccountPage() {
  return <AccountPanel />;
}
