// page.tsx
// 模块08数据备份页面。

import { BackupPanel } from "@/components/business/BackupPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemBackupsPage 数据备份页面。
 */
export default function SuperSystemBackupsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><BackupPanel /></PermissionGate>;
}
