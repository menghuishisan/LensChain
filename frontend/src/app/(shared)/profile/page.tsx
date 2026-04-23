// page.tsx
// 个人中心页，对应模块01 P-09。

import { ProfilePanel } from "@/components/business/ProfilePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * ProfilePage 个人中心页。
 */
export default function ProfilePage() {
  return (
    <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}>
      <ProfilePanel />
    </PermissionGate>
  );
}
