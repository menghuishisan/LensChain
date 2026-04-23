// page.tsx
// 学校管理员消息中心别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminNotificationsPage 学校管理员消息中心页面。 */
export default function AdminNotificationsPage() {
  return <PermissionGate allowedRoles={["school_admin"]}><NotificationInboxPagePanel /></PermissionGate>;
}
