// page.tsx
// 超级管理员系统公告编辑别名页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { AdminAnnouncementPagePanel } from "@/components/business/NotificationPanels";

/** SuperAnnouncementEditPage 超级管理员系统公告编辑页面。 */
export default function SuperAnnouncementEditPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["super_admin"]}><AdminAnnouncementPagePanel announcementID={params.id} /></PermissionGate>;
}
