// page.tsx
// 学生管理页，对应模块03 P-09。

import { CourseStudentsPanel } from "@/components/business/CourseStudentPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseStudentsPage 学生管理页。
 */
export default function TeacherCourseStudentsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><CourseStudentsPanel courseID={params.id} /></PermissionGate>;
}
