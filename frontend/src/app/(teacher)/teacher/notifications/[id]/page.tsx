// page.tsx
// 教师端消息详情别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherNotificationDetailPage 教师消息详情页面。 */
export default function TeacherNotificationDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><NotificationInboxPagePanel messageID={params.id} /></PermissionGate>;
}
