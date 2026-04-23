// page.tsx
// 学生管理页，对应模块03 P-09。

import { CourseStudentsPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseStudentsPage 学生管理页。
 */
export default function TeacherCourseStudentsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><CourseStudentsPanel courseID={params.id} /></PermissionGate>;
}
