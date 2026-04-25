// page.tsx
// 教师端学生实验协助页。

import { ExperimentInstancePanel } from "@/components/business/ExperimentInstancePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherExperimentAssistPage 教师实验协助页面。 */
export default function TeacherExperimentAssistPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <ExperimentInstancePanel instanceID={params.id} mode="assist" />
    </PermissionGate>
  );
}
