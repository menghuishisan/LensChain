// page.tsx
// 课程管理主页，对应模块03 P-03。

import { CourseOverviewPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseDetailPage 教师课程管理主页。
 */
export default function TeacherCourseDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><CourseOverviewPanel courseID={params.id} /></PermissionGate>;
}
