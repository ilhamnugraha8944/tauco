import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ContentEditor } from "@/features/admin/content-editor";
import type { AdminPageKey } from "@/features/admin/admin-api";

export const metadata: Metadata = { title: "Editor Konten CMS | Tauco Cap Badak" };
export default async function Page({params}:{params:Promise<{key:string}>}){const {key}=await params;if(key!=="home"&&key!=="about")notFound();return <ContentEditor pageKey={key as AdminPageKey}/>;}
