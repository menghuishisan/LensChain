// page.tsx
// 超级管理员镜像库治理页。

import { ExperimentImageLibraryPanel } from "@/components/business/ExperimentAssetPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperImagesPage 超级管理员镜像库页面。 */
export default function SuperImagesPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ExperimentImageLibraryPanel reviewMode />
    </PermissionGate>
  );
}
