// page.tsx
// 模块08全局配置页面。

import { SystemConfigPanel } from "@/components/business/SystemConfigPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemConfigsPage 全局配置页面。
 */
export default function SuperSystemConfigsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><SystemConfigPanel /></PermissionGate>;
}
