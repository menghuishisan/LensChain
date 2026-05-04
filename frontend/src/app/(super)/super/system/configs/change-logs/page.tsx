// page.tsx
// 模块08配置变更记录页面。

import { SystemConfigPanel } from "@/components/business/SystemConfigPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemConfigChangeLogsPage 配置变更记录页面。
 */
export default function SuperSystemConfigChangeLogsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><SystemConfigPanel mode="change-logs" /></PermissionGate>;
}
