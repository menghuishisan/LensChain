// page.tsx
// 学生端指定模板实验启动页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { ExperimentLaunchPanel } from "@/components/business/ExperimentInstanceListPanels";

/** StudentExperimentLaunchPage 学生指定模板启动页面。 */
export default function StudentExperimentLaunchPage({ params }: { params: { template_id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <ExperimentLaunchPanel templateID={params.template_id} />
    </PermissionGate>
  );
}
