// page.tsx
// 超级管理员镜像审核详情页。

import { ExperimentImageReviewPanel } from "@/components/business/ExperimentAssetPanels";
import { PermissionGate } from "@/components/business/PermissionGate";
import React from "react";

/** SuperImageReviewPage 超级管理员镜像审核页面。 */
export default function SuperImageReviewPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <ExperimentImageReviewPanel imageID={params.id} />
    </PermissionGate>
  );
}
