// page.tsx
// 管理端发送定向通知页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { DirectNotificationPanel } from "@/components/business/NotificationPanels";

/** AdminSendNotificationPage 发送通知页面。 */
export default function AdminSendNotificationPage() {
  return <PermissionGate allowedRoles={["school_admin"]}><DirectNotificationPanel /></PermissionGate>;
}
