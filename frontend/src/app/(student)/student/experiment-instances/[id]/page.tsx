// page.tsx
// 学生端实验实例详情页。

import { ExperimentInstancePanel } from "@/components/business/ExperimentInstancePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentExperimentInstancePage 学生实验实例详情页面。 */
export default function StudentExperimentInstancePage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <ExperimentInstancePanel instanceID={params.id} />
    </PermissionGate>
  );
}
