import type { Metadata } from "next";

import { isLocalAdminAPIOrigin } from "@/features/admin/config";
import { MediaManager } from "@/features/admin/media-manager";

export const metadata: Metadata = { title: "Media CMS | Tauco Cap Badak" };

export default function AdminMediaPage() {
  return <MediaManager allowLegacyUpload={isLocalAdminAPIOrigin()} />;
}
