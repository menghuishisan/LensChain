// page.tsx
// 学生端 CTF 排行榜页。

import { CtfLeaderboardPagePanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfLeaderboardPage CTF排行榜页面。 */
export default function CtfLeaderboardPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CtfLeaderboardPagePanel competitionID={params.id} /></PermissionGate>;
}
