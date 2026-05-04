// page.tsx
// 超级管理员 CTF 竞赛监控页。

import { CtfCompetitionMonitorPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperCtfCompetitionMonitorPage 超级管理员CTF竞赛监控页面。 */
export default function SuperCtfCompetitionMonitorPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfCompetitionMonitorPanel competitionID={params.id} /></PermissionGate>;
}
