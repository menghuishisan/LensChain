// page.tsx
// 学生端通知偏好别名页。

import { NotificationPreferencePagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentNotificationPreferencesPage 学生通知偏好页面。 */
export default function StudentNotificationPreferencesPage() {
  return <PermissionGate allowedRoles={["student"]}><NotificationPreferencePagePanel /></PermissionGate>;
}
