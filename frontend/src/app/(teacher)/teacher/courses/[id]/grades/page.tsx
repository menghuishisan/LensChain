// page.tsx
// 课程成绩管理页，对应模块03 P-10。

import { GradePanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseGradesPage 课程成绩管理页。
 */
export default function TeacherCourseGradesPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><GradePanel courseID={params.id} role="teacher" /></PermissionGate>;
}
