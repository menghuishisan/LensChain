// page.tsx
// 学校管理员用户管理列表页。

import { Upload } from "lucide-react";
import Link from "next/link";

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserListPanel } from "@/components/business/UserManagementPanels";
import { buttonClassName } from "@/components/ui/Button";

/** AdminUsersPage 学校管理员用户管理列表页。 */
export default function AdminUsersPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <UserListPanel
        basePath="/admin/users"
        headerActions={
          <>
            <Link className={buttonClassName({ variant: "outline" })} href="/admin/users/import"><Upload className="h-4 w-4" />导入用户</Link>
            <Link className={buttonClassName({ variant: "outline" })} href="/admin/logs/login">登录日志</Link>
            <Link className={buttonClassName({ variant: "outline" })} href="/admin/logs/operation">操作日志</Link>
          </>
        }
      />
    </PermissionGate>
  );
}
