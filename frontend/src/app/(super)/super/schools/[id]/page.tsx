// page.tsx
// 超管学校详情/编辑页，对应模块02 P-06。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolFormPanel } from "@/components/business/SchoolTable";

/**
 * SuperSchoolDetailPage 学校详情/编辑页。
 */
export default function SuperSchoolDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SchoolFormPanel schoolID={params.id} />
    </PermissionGate>
  );
}
