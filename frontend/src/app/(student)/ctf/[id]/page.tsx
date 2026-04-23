// page.tsx
// 学生端 CTF 竞赛详情和报名页。

import { CtfCompetitionDetailPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfCompetitionDetailPage CTF竞赛详情页面。 */
export default function CtfCompetitionDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CtfCompetitionDetailPanel competitionID={params.id} /></PermissionGate>;
}
