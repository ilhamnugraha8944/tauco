import type { Metadata } from "next";

import { AdminAuthFrame } from "@/features/admin/admin-auth-frame";
import { LoginForm } from "@/features/admin/login-form";

export const metadata: Metadata = { title: "Masuk Admin | Tauco Cap Badak" };

export default function AdminLoginPage() {
  return <AdminAuthFrame><LoginForm /></AdminAuthFrame>;
}
