// page.tsx
// 学生端实验结果查看页。

import { ExperimentResultPanel } from "@/components/business/ExperimentInstancePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentExperimentResultPage 学生实验结果查看页面。 */
export default function StudentExperimentResultPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <ExperimentResultPanel instanceID={params.id} />
    </PermissionGate>
  );
}
