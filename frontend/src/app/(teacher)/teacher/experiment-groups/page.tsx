// page.tsx
// 教师端实验分组管理页。

import { ExperimentGroupManagementPanel } from "@/components/business/ExperimentInstanceListPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherExperimentGroupsPage 教师实验分组页面。 */
export default function TeacherExperimentGroupsPage() {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <ExperimentGroupManagementPanel />
    </PermissionGate>
  );
}
