// page.tsx
// 超级管理员用户管理列表页。

import Link from "next/link";

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserListPanel } from "@/components/business/UserManagementPanels";
import { buttonClassName } from "@/components/ui/Button";

/** SuperUsersPage 超级管理员用户管理列表页。 */
export default function SuperUsersPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <UserListPanel
        basePath="/super/users"
        showSchoolColumn
        headerActions={
          <>
            <Link className={buttonClassName({ variant: "outline" })} href="/super/security">安全策略</Link>
            <Link className={buttonClassName({ variant: "outline" })} href="/super/logs/login">登录日志</Link>
            <Link className={buttonClassName({ variant: "outline" })} href="/super/logs/operation">操作日志</Link>
          </>
        }
      />
    </PermissionGate>
  );
}
