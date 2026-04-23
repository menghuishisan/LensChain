// page.tsx
// 学生端 CTF 攻防赛攻击页。

import { AttackDefenseRoundPanel } from "@/components/business/AttackDefenseRoundPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfAttackPage CTF攻防赛攻击页面。 */
export default function CtfAttackPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><AttackDefenseRoundPanel competitionID={params.id} /></PermissionGate>;
}
