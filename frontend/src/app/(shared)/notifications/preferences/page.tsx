// page.tsx
// 通用通知偏好设置页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { NotificationPreferencePagePanel } from "@/components/business/NotificationPanels";

/** NotificationPreferencesPage 通知偏好设置页面。 */
export default function NotificationPreferencesPage() {
  return <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}><NotificationPreferencePagePanel /></PermissionGate>;
}
