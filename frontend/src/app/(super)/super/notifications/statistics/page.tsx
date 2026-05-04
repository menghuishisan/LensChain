// page.tsx
// 超级管理员消息统计别名页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { NotificationStatisticsPanel } from "@/components/business/NotificationPanels";

/** SuperNotificationStatisticsPage 超级管理员消息统计页面。 */
export default function SuperNotificationStatisticsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><NotificationStatisticsPanel /></PermissionGate>;
}
