// page.tsx
// 管理端仿真场景审核页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SimScenarioLibraryPanel } from "@/components/business/ExperimentTemplatePanels";

/** AdminSimScenariosPage 管理端仿真场景页面。 */
export default function AdminSimScenariosPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SimScenarioLibraryPanel reviewMode />
    </PermissionGate>
  );
}
