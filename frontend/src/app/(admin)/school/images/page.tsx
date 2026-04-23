// page.tsx
// 学校管理员本校镜像页。

import { SchoolImageLibraryPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SchoolImagesPage 学校管理员本校镜像页面。 */
export default function SchoolImagesPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <SchoolImageLibraryPanel />
    </PermissionGate>
  );
}
