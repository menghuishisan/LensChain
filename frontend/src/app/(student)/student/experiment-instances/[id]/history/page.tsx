// page.tsx
// 学生端实验操作历史页。

import { ExperimentOperationHistoryPanel } from "@/components/business/ExperimentInstanceListPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentExperimentHistoryPage 学生实验操作历史页面。 */
export default function StudentExperimentHistoryPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <ExperimentOperationHistoryPanel instanceID={params.id} />
    </PermissionGate>
  );
}
