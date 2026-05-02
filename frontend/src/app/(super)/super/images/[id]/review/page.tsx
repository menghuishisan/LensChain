// page.tsx
// 管理端镜像审核详情页。

import { ExperimentImageReviewPanel } from "@/components/business/ExperimentAssetPanels";
import { PermissionGate } from "@/components/business/PermissionGate";
import React from "react";

/** AdminImageReviewPage 管理端镜像审核页面。 */
export default function AdminImageReviewPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ExperimentImageReviewPanel imageID={params.id} />
    </PermissionGate>
  );
}
