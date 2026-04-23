// page.tsx
// 教师端通知偏好别名页。

import { NotificationPreferencePagePanel } from "@/components/business/NotificationPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherNotificationPreferencesPage 教师通知偏好页面。 */
export default function TeacherNotificationPreferencesPage() {
  return <PermissionGate allowedRoles={["teacher"]}><NotificationPreferencePagePanel /></PermissionGate>;
}
