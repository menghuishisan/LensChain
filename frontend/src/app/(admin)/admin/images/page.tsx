// page.tsx
// 管理端镜像库治理页。

import { ExperimentImageLibraryPanel } from "@/components/business/ExperimentTemplatePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminImagesPage 管理端镜像库页面。 */
export default function AdminImagesPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ExperimentImageLibraryPanel reviewMode />
    </PermissionGate>
  );
}
