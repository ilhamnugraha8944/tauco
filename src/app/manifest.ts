import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Tauco Cap Badak Cianjur",
    short_name: "Tauco Cap Badak",
    description:
      "Informasi produk Tauco Cap Badak dari Cianjur dan panduan mengenal tauco.",
    start_url: "/",
    display: "standalone",
    background_color: "#F2F4F1",
    theme_color: "#2F654E",
    lang: "id-ID",
    icons: [
      {
        src: "/icon",
        sizes: "32x32",
        type: "image/png",
      },
    ],
  };
}
