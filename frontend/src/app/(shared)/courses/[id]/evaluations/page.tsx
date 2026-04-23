// page.tsx
// 课程评价页，对应模块03课程评价功能。

import { EvaluationPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * CourseEvaluationsPage 课程评价页。
 */
export default function CourseEvaluationsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}><EvaluationPanel courseID={params.id} role="teacher" /></PermissionGate>;
}
