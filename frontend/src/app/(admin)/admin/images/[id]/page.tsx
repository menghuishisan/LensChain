// page.tsx
// 超级管理员镜像详情/编辑页。

import { ExperimentImageDetailPanel } from "@/components/business/ExperimentTemplatePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminImageDetailPage 超级管理员镜像详情/编辑页面。 */
export default function AdminImageDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ExperimentImageDetailPanel imageID={params.id} />
    </PermissionGate>
  );
}
