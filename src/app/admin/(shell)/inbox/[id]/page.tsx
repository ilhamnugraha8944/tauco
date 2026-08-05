import { InboxDetail } from "@/features/admin/inbox-manager";
export default async function AdminInboxDetailPage({params}:{params:Promise<{id:string}>}){return <InboxDetail id={(await params).id}/>}
