// page.tsx
// 学校管理员通知偏好别名页。

import { NotificationPreferencePagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminNotificationPreferencesPage 学校管理员通知偏好页面。 */
export default function AdminNotificationPreferencesPage() {
  return <PermissionGate allowedRoles={["school_admin"]}><NotificationPreferencePagePanel /></PermissionGate>;
}
