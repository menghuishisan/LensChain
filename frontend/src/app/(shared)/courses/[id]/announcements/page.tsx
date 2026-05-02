// page.tsx
// 课程公告页，对应模块03 P-32。

import { AnnouncementPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * CourseAnnouncementsPage 课程公告页。
 */
export default function CourseAnnouncementsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student", "teacher"]}><AnnouncementPanel courseID={params.id} /></PermissionGate>;
}
