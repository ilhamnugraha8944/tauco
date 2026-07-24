import { mkdir } from "node:fs/promises";
import { dirname, resolve } from "node:path";

import sharp from "sharp";

const assets = [
  {
    input: "assets/image-sources/tauco-hero-provisional.png",
    output: "public/images/tauco-hero-provisional",
  },
  {
    input: "assets/image-sources/tauco-fermentation-provisional.png",
    output: "public/images/tauco-fermentation-provisional",
  },
  {
    input: "assets/image-sources/tauco-dish-provisional.png",
    output: "public/images/tauco-dish-provisional",
  },
];

await Promise.all(
  assets.flatMap(({ input, output }) => {
    const inputPath = resolve(input);
    const outputPath = resolve(output);

    return [
      mkdir(dirname(outputPath), { recursive: true }).then(() =>
        sharp(inputPath)
          .rotate()
          .webp({ quality: 82, effort: 5 })
          .toFile(`${outputPath}.webp`),
      ),
      mkdir(dirname(outputPath), { recursive: true }).then(() =>
        sharp(inputPath)
          .rotate()
          .avif({ quality: 52, effort: 5 })
          .toFile(`${outputPath}.avif`),
      ),
    ];
  }),
);

console.log(`Optimized ${assets.length} source images to WebP and AVIF.`);
