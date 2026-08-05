import type { Metadata } from "next";

import { AdminAuthFrame } from "@/features/admin/admin-auth-frame";
import { TOTPSetup } from "@/features/admin/totp-setup";

export const metadata: Metadata = { title: "Setup TOTP | Tauco Cap Badak" };

export default function AdminSetupTOTPPage() {
  return <AdminAuthFrame><TOTPSetup /></AdminAuthFrame>;
}
