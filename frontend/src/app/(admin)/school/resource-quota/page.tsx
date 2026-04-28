// page.tsx
// 学校管理员资源配额页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { ResourceQuotaPanel } from "@/components/business/ExperimentAdminPanels";

/** SchoolResourceQuotaPage 学校管理员资源配额页面。 */
export default function SchoolResourceQuotaPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <ResourceQuotaPanel readOnly />
    </PermissionGate>
  );
}
