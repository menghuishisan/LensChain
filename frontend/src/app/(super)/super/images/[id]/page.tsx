// page.tsx
// 超级管理员镜像详情/编辑页。

import { ExperimentImageDetailPanel } from "@/components/business/ExperimentAssetPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperImageDetailPage 超级管理员镜像详情/编辑页面。 */
export default function SuperImageDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ExperimentImageDetailPanel imageID={params.id} />
    </PermissionGate>
  );
}
