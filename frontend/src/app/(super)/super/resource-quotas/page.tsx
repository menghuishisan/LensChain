// page.tsx
// 管理端资源配额页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { ResourceQuotaPanel } from "@/components/business/ExperimentAdminPanels";

/** AdminResourceQuotasPage 管理端资源配额页面。 */
export default function AdminResourceQuotasPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ResourceQuotaPanel />
    </PermissionGate>
  );
}
