// page.tsx
// 课程内容管理页，对应模块03 P-04。

import { CourseContentManagerPanel } from "@/components/business/CourseContentPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseContentPage 内容管理页。
 */
export default function TeacherCourseContentPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><CourseContentManagerPanel courseID={params.id} /></PermissionGate>;
}
