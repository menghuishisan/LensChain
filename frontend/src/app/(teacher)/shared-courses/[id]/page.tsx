// page.tsx
// 共享课程详情页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SharedCourseDetailPanel } from "@/components/business/CourseListPanels";

/**
 * SharedCourseDetailPage 共享课程详情页。
 */
export default function SharedCourseDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><SharedCourseDetailPanel courseID={params.id} /></PermissionGate>;
}
