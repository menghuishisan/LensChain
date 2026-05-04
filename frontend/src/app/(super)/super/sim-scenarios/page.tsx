// page.tsx
// 超级管理员仿真场景审核页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SimScenarioLibraryPanel } from "@/components/business/ExperimentAssetPanels";

/** SuperSimScenariosPage 超级管理员仿真场景页面。 */
export default function SuperSimScenariosPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SimScenarioLibraryPanel reviewMode />
    </PermissionGate>
  );
}
