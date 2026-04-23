// page.tsx
// 学生端消息中心别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentNotificationsPage 学生消息中心页面。 */
export default function StudentNotificationsPage() {
  return <PermissionGate allowedRoles={["student"]}><NotificationInboxPagePanel /></PermissionGate>;
}
