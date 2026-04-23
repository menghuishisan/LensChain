// page.tsx
// 模块08配置变更记录页面。

import { SystemConfigPanel } from "@/components/business/SystemConfigPanel";

/**
 * SuperSystemConfigChangeLogsPage 配置变更记录页面。
 */
export default function SuperSystemConfigChangeLogsPage() {
  return <SystemConfigPanel mode="change-logs" />;
}
