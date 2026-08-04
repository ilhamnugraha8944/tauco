import { notFound } from "next/navigation";

import { ContentPreview } from "@/features/admin/content-preview";
import type { AdminPageKey } from "@/features/admin/admin-api";
import { localContentSource } from "@/features/content";

export default async function Page({params,searchParams}:{params:Promise<{revisionId:string}>;searchParams:Promise<{key?:string}>}){const [{revisionId},{key},catalog]=await Promise.all([params,searchParams,localContentSource.listProducts()]);if(key!=="home"&&key!=="about")notFound();return <div className="admin-preview-page"><header><p className="admin-kicker">Authenticated preview</p><strong>Revision preview</strong><p>Preview memakai presentation component yang sama dengan website publik.</p></header><ContentPreview pageKey={key as AdminPageKey} revisionId={revisionId} catalog={catalog}/></div>}
