// page.tsx
// 超级管理员资源配额页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { ResourceQuotaPanel } from "@/components/business/ExperimentAdminPanels";

/** SuperResourceQuotasPage 超级管理员资源配额页面。 */
export default function SuperResourceQuotasPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ResourceQuotaPanel />
    </PermissionGate>
  );
}
