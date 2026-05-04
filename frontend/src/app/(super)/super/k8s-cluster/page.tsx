// page.tsx
// 超级管理员 K8s 集群状态页。

import { K8sClusterStatusPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperK8sClusterPage 超级管理员 K8s 集群页面。 */
export default function SuperK8sClusterPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <K8sClusterStatusPanel />
    </PermissionGate>
  );
}
