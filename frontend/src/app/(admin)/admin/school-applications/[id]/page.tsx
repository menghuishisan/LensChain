// page.tsx
// 超管入驻申请审核详情页，对应模块02 P-04。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolApplicationReviewDetail } from "@/components/business/SchoolApplicationAdminPanels";

/**
 * AdminSchoolApplicationDetailPage 入驻申请审核详情页。
 */
export default function AdminSchoolApplicationDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SchoolApplicationReviewDetail applicationID={params.id} />
    </PermissionGate>
  );
}
