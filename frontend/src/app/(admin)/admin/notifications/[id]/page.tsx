// page.tsx
// 学校管理员消息详情别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminNotificationDetailPage 学校管理员消息详情页面。 */
export default function AdminNotificationDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["school_admin"]}><NotificationInboxPagePanel messageID={params.id} /></PermissionGate>;
}
