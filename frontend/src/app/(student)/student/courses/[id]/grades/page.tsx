// page.tsx
// 学生我的成绩页，对应模块03 P-26。

import { CourseGradesPanel } from "@/components/business/CourseGradesPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentCourseGradesPage 我的成绩页。
 */
export default function StudentCourseGradesPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CourseGradesPanel courseID={params.id} role="student" /></PermissionGate>;
}
