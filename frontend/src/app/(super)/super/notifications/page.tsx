// page.tsx
// 超级管理员消息中心别名页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperNotificationsPage 超级管理员消息中心页面。 */
export default function SuperNotificationsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><NotificationInboxPagePanel /></PermissionGate>;
}
