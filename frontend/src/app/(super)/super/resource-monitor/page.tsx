// page.tsx
// 管理端资源监控页。

import { AdminExperimentDashboardPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminResourceMonitorPage 管理端资源监控页面。 */
export default function AdminResourceMonitorPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <AdminExperimentDashboardPanel />
    </PermissionGate>
  );
}
