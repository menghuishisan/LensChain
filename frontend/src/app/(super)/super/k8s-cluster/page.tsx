// page.tsx
// 管理端 K8s 集群状态页。

import { K8sClusterStatusPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminK8sClusterPage 管理端 K8s 集群页面。 */
export default function AdminK8sClusterPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <K8sClusterStatusPanel />
    </PermissionGate>
  );
}
