// page.tsx
// 创建/编辑作业页，对应模块03 P-07。

import { AssignmentEditor } from "@/components/business/AssignmentEditor";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherAssignmentEditPage 作业编辑页。
 */
export default function TeacherAssignmentEditPage({ params, searchParams }: { params: { id: string }; searchParams: { course_id?: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><AssignmentEditor courseID={searchParams.course_id ?? ""} assignmentID={params.id === "new" ? undefined : params.id} /></PermissionGate>;
}
