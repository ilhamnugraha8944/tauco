"use client";

import { useEffect, useState } from "react";

import { adminAPI, type AdminPageKey } from "@/features/admin/admin-api";
import { AboutPresentation } from "@/features/content/about-presentation";
import { HomePresentation } from "@/features/content/home-presentation";
import { aboutContentSchema, homeContentSchema } from "@/features/content/schemas";
import type { ProductCatalogContent } from "@/features/content/types";

export function ContentPreview({pageKey,revisionId,catalog}:{pageKey:AdminPageKey;revisionId:string;catalog:ProductCatalogContent}){const [content,setContent]=useState<unknown>();const [error,setError]=useState("");useEffect(()=>{let active=true;adminAPI.getPageRevision(pageKey,revisionId).then((response)=>{if(active)setContent(response.data.content)},(reason)=>{if(active)setError(reason instanceof Error?reason.message:"Preview gagal dimuat.")});return()=>{active=false}},[pageKey,revisionId]);if(error)return <p className="admin-form-error">{error}</p>;if(!content)return <p>Memuat preview...</p>;const parsed=(pageKey==="home"?homeContentSchema:aboutContentSchema).safeParse(content);if(!parsed.success)return <p className="admin-form-error">Revision tidak memenuhi schema tampilan.</p>;return <div className="admin-public-preview">{pageKey==="home"?<HomePresentation content={homeContentSchema.parse(content)} catalog={catalog} adminPreview/>:<AboutPresentation content={aboutContentSchema.parse(content)} adminPreview/>}</div>}
