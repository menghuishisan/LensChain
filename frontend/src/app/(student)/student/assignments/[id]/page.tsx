// page.tsx
// 学生作业作答页，对应模块03 P-25。

import { AssignmentAnswerPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentAssignmentPage 作业作答页。
 */
export default function StudentAssignmentPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><AssignmentAnswerPanel assignmentID={params.id} /></PermissionGate>;
}
