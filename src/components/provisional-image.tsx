import Image from "next/image";

import { getImageAsset } from "@/lib/images";

type ProvisionalImageProps = {
  src: string;
  alt: string;
  className?: string;
  imageClassName?: string;
  caption?: string;
  preload?: boolean;
  sizes: string;
  adminPreview?: boolean;
};

export function ProvisionalImage({
  src,
  alt,
  className = "",
  imageClassName = "",
  caption = "Ilustrasi penyajian",
  preload = false,
  sizes,
  adminPreview = false,
}: ProvisionalImageProps) {
  const previewSource = adminPreview && src.startsWith("/api/v1/media/")
    ? src.replace("/api/v1/media/", "/admin-api/public-media/")
    : null;
  return (
    <figure className={className}>
      <div className="image-frame">
        <Image
          src={previewSource ?? getImageAsset(src)}
          alt={alt}
          className={`h-full w-full object-cover ${imageClassName}`}
          placeholder={previewSource ? "empty" : "blur"}
          fill={previewSource ? true : undefined}
          unoptimized={Boolean(previewSource)}
          preload={preload}
          fetchPriority={preload ? "high" : "low"}
          decoding="async"
          quality={preload ? 50 : 75}
          sizes={sizes}
        />
      </div>
      {alt && caption ? <figcaption>{caption}</figcaption> : null}
    </figure>
  );
}
