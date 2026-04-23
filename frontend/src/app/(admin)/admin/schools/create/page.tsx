// page.tsx
// 超管创建学校页，对应模块02 P-07。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolFormPanel } from "@/components/business/SchoolTable";

/**
 * AdminSchoolCreatePage 创建学校页。
 */
export default function AdminSchoolCreatePage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SchoolFormPanel />
    </PermissionGate>
  );
}
