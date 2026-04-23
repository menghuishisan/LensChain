// page.tsx
// 教师端消息中心别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherNotificationsPage 教师消息中心页面。 */
export default function TeacherNotificationsPage() {
  return <PermissionGate allowedRoles={["teacher"]}><NotificationInboxPagePanel /></PermissionGate>;
}
