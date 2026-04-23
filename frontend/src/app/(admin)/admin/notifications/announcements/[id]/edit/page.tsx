// page.tsx
// 管理端系统公告编辑页。

import { AdminAnnouncementPagePanel } from "@/components/business/NotificationPanels";

/** AdminAnnouncementEditPage 系统公告编辑页面。 */
export default function AdminAnnouncementEditPage({ params }: { params: { id: string } }) {
  return <AdminAnnouncementPagePanel announcementID={params.id} />;
}
