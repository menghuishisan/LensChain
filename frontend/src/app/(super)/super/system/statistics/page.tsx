// page.tsx
// 模块08平台统计页面。

import { SystemStatisticsPanel } from "@/components/business/SystemStatisticsPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemStatisticsPage 平台统计页面。
 */
export default function SuperSystemStatisticsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><SystemStatisticsPanel /></PermissionGate>;
}
