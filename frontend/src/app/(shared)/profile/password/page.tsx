// page.tsx
// 修改密码页，对应模块01 P-10。

import { ChangePasswordForm } from "@/components/business/AuthPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * ProfilePasswordPage 修改密码页。
 */
export default function ProfilePasswordPage() {
  return (
    <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}>
      <ChangePasswordForm />
    </PermissionGate>
  );
}
