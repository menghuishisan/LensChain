// page.tsx
// 通用消息中心页，展示站内信和置顶系统公告。

import { PermissionGate } from "@/components/business/PermissionGate";
import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";

/**
 * NotificationsPage 消息中心页。
 */
export default function NotificationsPage() {
  return <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}><NotificationInboxPagePanel /></PermissionGate>;
}
