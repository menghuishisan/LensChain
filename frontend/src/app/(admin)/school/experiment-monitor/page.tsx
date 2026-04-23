// page.tsx
// 学校管理员本校实验监控页。

import { SchoolExperimentMonitorPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SchoolExperimentMonitorPage 学校管理员实验监控页面。 */
export default function SchoolExperimentMonitorPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <SchoolExperimentMonitorPanel />
    </PermissionGate>
  );
}
