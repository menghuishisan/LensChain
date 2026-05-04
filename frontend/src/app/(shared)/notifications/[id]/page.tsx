// page.tsx
// 通用消息详情页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";

/** NotificationDetailPage 消息详情页面。 */
export default function NotificationDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}><NotificationInboxPagePanel messageID={params.id} /></PermissionGate>;
}
