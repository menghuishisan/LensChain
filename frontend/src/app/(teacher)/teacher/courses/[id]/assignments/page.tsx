// page.tsx
// 教师作业管理页，对应模块03 P-06。

import { AssignmentListPanel } from "@/components/business/AssignmentListPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseAssignmentsPage 作业管理页。
 */
export default function TeacherCourseAssignmentsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><AssignmentListPanel courseID={params.id} role="teacher" /></PermissionGate>;
}
