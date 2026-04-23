// page.tsx
// 教师端实验评分页。

import { ExperimentInstancePanel } from "@/components/business/ExperimentInstancePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherExperimentGradePage 教师实验评分页面。 */
export default function TeacherExperimentGradePage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <ExperimentInstancePanel instanceID={params.id} canGrade terminalReadOnly />
    </PermissionGate>
  );
}
