import { ProductPreview } from "@/features/admin/product-preview";
export default async function AdminProductPreviewPage({params,searchParams}:{params:Promise<{revisionId:string}>;searchParams:Promise<{id?:string}>}){const[{revisionId},{id}]=await Promise.all([params,searchParams]);return id?<ProductPreview productId={id} revisionId={revisionId}/>:<p>Product ID tidak tersedia.</p>;}
