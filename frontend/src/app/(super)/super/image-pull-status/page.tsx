// page.tsx
// 超级管理员镜像预拉取状态页。

import { ImagePullStatusPanel } from "@/components/business/ExperimentAdminPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperImagePullStatusPage 超级管理员镜像预拉取状态页面。 */
export default function SuperImagePullStatusPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ImagePullStatusPanel />
    </PermissionGate>
  );
}
