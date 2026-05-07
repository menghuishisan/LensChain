// page.tsx
// 学生端 CTF 攻防赛主页。

import { AttackDefenseRoundPanel } from "@/components/business/AttackDefenseRoundPanel";
import { CtfLeaderboard } from "@/components/business/CtfLeaderboard";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfAttackDefensePage CTF攻防赛主页。 */
export default function CtfAttackDefensePage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["student"]}>
      <div className="space-y-5">
        <AttackDefenseRoundPanel competitionID={params.id} />
        <CtfLeaderboard competitionID={params.id} />
      </div>
    </PermissionGate>
  );
}
