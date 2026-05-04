// page.tsx
// 超级管理员系统公告管理别名页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { AdminAnnouncementPagePanel } from "@/components/business/NotificationPanels";

/** SuperAnnouncementsPage 超级管理员系统公告页面。 */
export default function SuperAnnouncementsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><AdminAnnouncementPagePanel /></PermissionGate>;
}
