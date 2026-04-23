// page.tsx
// 超级管理员消息详情别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperNotificationDetailPage 超级管理员消息详情页面。 */
export default function SuperNotificationDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["super_admin"]}><NotificationInboxPagePanel messageID={params.id} /></PermissionGate>;
}
