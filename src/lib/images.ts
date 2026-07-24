import type { StaticImageData } from "next/image";

import dishImage from "../../public/images/tauco-dish-provisional.webp";
import fermentationImage from "../../public/images/tauco-fermentation-provisional.webp";
import heroImage from "../../public/images/tauco-hero-provisional.webp";

const imageAssets: Record<string, StaticImageData> = {
  "/images/tauco-hero-provisional.webp": heroImage,
  "/images/tauco-fermentation-provisional.webp": fermentationImage,
  "/images/tauco-dish-provisional.webp": dishImage,
};

export function getImageAsset(src: string): StaticImageData {
  const image = imageAssets[src];

  if (!image) {
    throw new Error(`Aset gambar belum terdaftar: ${src}`);
  }

  return image;
}
