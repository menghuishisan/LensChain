// page.tsx
// 课程成绩管理页，对应模块03 P-10。

import { CourseGradesPanel } from "@/components/business/CourseGradesPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseGradesPage 课程成绩管理页。
 */
export default function TeacherCourseGradesPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><CourseGradesPanel courseID={params.id} role="teacher" /></PermissionGate>;
}
