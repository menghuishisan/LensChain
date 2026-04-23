// page.tsx
// 学生端消息详情别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentNotificationDetailPage 学生消息详情页面。 */
export default function StudentNotificationDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><NotificationInboxPagePanel messageID={params.id} /></PermissionGate>;
}
