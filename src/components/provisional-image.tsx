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
};

export function ProvisionalImage({
  src,
  alt,
  className = "",
  imageClassName = "",
  caption = "Ilustrasi penyajian",
  preload = false,
  sizes,
}: ProvisionalImageProps) {
  return (
    <figure className={className}>
      <div className="image-frame">
        <Image
          src={getImageAsset(src)}
          alt={alt}
          className={`h-full w-full object-cover ${imageClassName}`}
          placeholder="blur"
          preload={preload}
          fetchPriority={preload ? "high" : "low"}
          decoding="async"
          quality={preload ? 50 : 75}
          sizes={sizes}
        />
      </div>
      {caption ? <figcaption>{caption}</figcaption> : null}
    </figure>
  );
}
