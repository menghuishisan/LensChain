// page.tsx
// 管理端镜像预拉取状态页。

import { ImagePullStatusPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminImagePullStatusPage 管理端镜像预拉取状态页面。 */
export default function AdminImagePullStatusPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ImagePullStatusPanel />
    </PermissionGate>
  );
}
