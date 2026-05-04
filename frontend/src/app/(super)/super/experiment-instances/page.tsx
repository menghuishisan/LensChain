// page.tsx
// 超级管理员实验实例治理页。

import { AdminExperimentInstancesPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperExperimentInstancesPage 超级管理员实验实例页面。 */
export default function SuperExperimentInstancesPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <AdminExperimentInstancesPanel />
    </PermissionGate>
  );
}
