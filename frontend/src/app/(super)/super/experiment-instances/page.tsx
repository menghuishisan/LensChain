// page.tsx
// 管理端实验实例治理页。

import { AdminExperimentInstancesPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminExperimentInstancesPage 管理端实验实例页面。 */
export default function AdminExperimentInstancesPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <AdminExperimentInstancesPanel />
    </PermissionGate>
  );
}
