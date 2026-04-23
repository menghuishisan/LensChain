// page.tsx
// 超管学校管理列表页，对应模块02 P-05。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolTable } from "@/components/business/SchoolTable";

/**
 * AdminSchoolsPage 学校管理列表页。
 */
export default function AdminSchoolsPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SchoolTable />
    </PermissionGate>
  );
}
