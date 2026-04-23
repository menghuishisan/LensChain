// page.tsx
// 超级管理员通知偏好别名页。

import { NotificationPreferencePagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperNotificationPreferencesPage 超级管理员通知偏好页面。 */
export default function SuperNotificationPreferencesPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><NotificationPreferencePagePanel /></PermissionGate>;
}
