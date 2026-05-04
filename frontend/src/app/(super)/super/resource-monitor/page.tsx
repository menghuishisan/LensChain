// page.tsx
// 超级管理员资源监控页。

import { AdminExperimentDashboardPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperResourceMonitorPage 超级管理员资源监控页面。 */
export default function SuperResourceMonitorPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <AdminExperimentDashboardPanel />
    </PermissionGate>
  );
}
