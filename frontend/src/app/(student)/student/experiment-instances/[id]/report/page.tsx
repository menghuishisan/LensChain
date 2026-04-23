// page.tsx
// 学生端实验报告页。

import { ExperimentInstancePanel } from "@/components/business/ExperimentInstancePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** StudentExperimentReportPage 学生实验报告页面。 */
export default function StudentExperimentReportPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <ExperimentInstancePanel instanceID={params.id} />
    </PermissionGate>
  );
}
