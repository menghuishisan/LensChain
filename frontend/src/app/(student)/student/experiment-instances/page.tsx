// page.tsx
// 学生端实验实例列表页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { StudentExperimentListPanel } from "@/components/business/ExperimentInstanceListPanels";

/** StudentExperimentInstancesPage 学生实验实例列表页面。 */
export default function StudentExperimentInstancesPage() {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <StudentExperimentListPanel />
    </PermissionGate>
  );
}
