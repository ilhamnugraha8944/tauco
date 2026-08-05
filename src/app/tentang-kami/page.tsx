import type { Metadata } from "next";

import { localContentSource } from "@/features/content";
import { AboutPresentation } from "@/features/content/about-presentation";
import { createMetadata } from "@/lib/seo";

export async function generateMetadata(): Promise<Metadata> {
  const content = await localContentSource.getAbout();
  return createMetadata(content.metadata);
}

export default async function AboutPage() {
  const content = await localContentSource.getAbout();

  return <AboutPresentation content={content} />;
}
