// page.tsx
// 超管入驻申请审核列表页，对应模块02 P-03。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolApplicationReviewList } from "@/components/business/SchoolApplicationAdminPanels";

/**
 * AdminSchoolApplicationsPage 入驻申请管理列表页。
 */
export default function AdminSchoolApplicationsPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SchoolApplicationReviewList />
    </PermissionGate>
  );
}
