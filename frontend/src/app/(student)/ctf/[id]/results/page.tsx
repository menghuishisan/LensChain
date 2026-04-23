// page.tsx
// 学生端 CTF 竞赛结果页。

import { CtfResultsPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfResultsPage CTF竞赛结果页面。 */
export default function CtfResultsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CtfResultsPanel competitionID={params.id} /></PermissionGate>;
}
