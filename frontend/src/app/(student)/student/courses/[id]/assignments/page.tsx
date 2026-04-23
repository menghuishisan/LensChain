// page.tsx
// 学生作业列表页，对应模块03 P-24。

import { AssignmentListPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentCourseAssignmentsPage 作业列表页。
 */
export default function StudentCourseAssignmentsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><AssignmentListPanel courseID={params.id} role="student" /></PermissionGate>;
}
