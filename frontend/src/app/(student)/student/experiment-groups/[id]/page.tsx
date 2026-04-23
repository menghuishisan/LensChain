// page.tsx
// 学生端实验分组协作页。

import { ExperimentGroupPanel } from "@/components/business/ExperimentGroupPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentExperimentGroupPage 学生实验分组页面。 */
export default function StudentExperimentGroupPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <ExperimentGroupPanel groupID={params.id} />
    </PermissionGate>
  );
}
