import { ProductEditor } from "@/features/admin/product-manager";
export default async function AdminProductPage({params}:{params:Promise<{id:string}>}){return <ProductEditor productId={(await params).id}/>;}
