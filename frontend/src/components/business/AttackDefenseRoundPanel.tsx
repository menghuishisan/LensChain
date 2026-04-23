"use client";

// AttackDefenseRoundPanel.tsx
// 模块05攻防赛回合面板，展示回合阶段、攻击/防守提交、攻击结果、补丁验证和 Token 状态。

import { ShieldCheck, Swords, Timer } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { useAdAttacks, useAdBattleMutations, useAdDefenses, useAdGroups, useCurrentAdRound } from "@/hooks/useCtfBattle";
import { useCtfRealtime } from "@/hooks/useCtfRealtime";
import { useCtfLeaderboard } from "@/hooks/useCtfCompetitions";
import type { ID } from "@/types/api";

/** AttackDefenseRoundPanel 组件属性。 */
export interface AttackDefenseRoundPanelProps {
  competitionID: ID;
  groupID?: ID;
}

/** AttackDefenseRoundPanel 攻防赛回合控制组件。 */
export function AttackDefenseRoundPanel({ competitionID, groupID }: AttackDefenseRoundPanelProps) {
  const groupsQuery = useAdGroups(competitionID);
  const [selectedGroupID, setSelectedGroupID] = useState(groupID ?? "");
  const activeGroupID = selectedGroupID || groupID || groupsQuery.data?.list[0]?.id || "";
  const roundQuery = useCurrentAdRound(activeGroupID);
  const roundID = roundQuery.data?.round_id ?? "";
  const battleMutations = useAdBattleMutations(roundID, competitionID);
  const attacksQuery = useAdAttacks(roundID, { page: 1, page_size: 10 });
  const defensesQuery = useAdDefenses(roundID, { page: 1, page_size: 10 });
  const realtime = useCtfRealtime(competitionID, competitionID.length > 0);
  const leaderboardQuery = useCtfLeaderboard(competitionID, activeGroupID ? { group_id: activeGroupID, top: 10 } : {});
  const [targetTeamID, setTargetTeamID] = useState("");
  const [challengeID, setChallengeID] = useState("");
  const [attackTxData, setAttackTxData] = useState("");
  const [patchSourceCode, setPatchSourceCode] = useState("");

  const current = roundQuery.data;
  const attackResult = battleMutations.attack.data;
  const defenseResult = battleMutations.defense.data;

  return (
    <div className="space-y-5">
      <Card className="border-red-500/20 bg-gradient-to-br from-slate-950 via-red-950 to-zinc-950 text-white">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-white">
            <Timer className="h-5 w-5 text-red-200" />
            攻防赛回合状态
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-4">
          <div className="md:col-span-4">
            <FormField label="分组">
              <Select value={activeGroupID} onValueChange={setSelectedGroupID}>
                <SelectTrigger className="border-white/15 bg-white/5 text-white"><SelectValue placeholder="选择分组" /></SelectTrigger>
                <SelectContent>
                  {(groupsQuery.data?.list ?? []).map((group) => (
                    <SelectItem key={group.id} value={group.id}>{group.group_name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
          </div>
          <Metric title="回合" value={current ? `${current.round_number}/${current.total_rounds}` : "-"} />
          <Metric title="阶段" value={current?.phase_text ?? "-"} />
          <Metric title="剩余秒数" value={current?.remaining_seconds ?? "-"} />
          <Metric title="我的 Token" value={current?.my_team?.token_balance ?? "-"} />
          <div className="md:col-span-4 flex flex-wrap gap-2">
            <Badge variant={realtime.status === "open" ? "success" : "outline"}>WS {realtime.status}</Badge>
            <Badge variant={current?.phase === 1 ? "destructive" : current?.phase === 2 ? "success" : "secondary"}>{current?.phase_text ?? "未开始"}</Badge>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-5 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Swords className="h-5 w-5 text-red-500" />
              提交攻击
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField label="目标队伍 ID">
              <Input value={targetTeamID} onChange={(event) => setTargetTeamID(event.target.value)} />
            </FormField>
            <FormField label="漏洞题目 ID">
              <Input value={challengeID} onChange={(event) => setChallengeID(event.target.value)} />
            </FormField>
            <FormField label="攻击交易数据">
              <Textarea value={attackTxData} onChange={(event) => setAttackTxData(event.target.value)} rows={7} className="font-mono" />
            </FormField>
            <Button disabled={current?.phase !== 1 || !targetTeamID || !challengeID || !attackTxData} onClick={() => battleMutations.attack.mutate({ target_team_id: targetTeamID, challenge_id: challengeID, attack_tx_data: attackTxData })} isLoading={battleMutations.attack.isPending}>
              提交攻击
            </Button>
            {attackResult ? (
              <ResultBox ok={attackResult.is_successful} okText={`攻击成功，奖励 ${attackResult.token_reward ?? 0} Token`} failText={attackResult.error_message ?? "攻击失败，后端未返回原因"} />
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-emerald-500" />
              提交补丁
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField label="漏洞题目 ID">
              <Input value={challengeID} onChange={(event) => setChallengeID(event.target.value)} />
            </FormField>
            <FormField label="补丁合约源码">
              <Textarea value={patchSourceCode} onChange={(event) => setPatchSourceCode(event.target.value)} rows={11} className="font-mono" />
            </FormField>
            <Button disabled={current?.phase !== 2 || !challengeID || !patchSourceCode} onClick={() => battleMutations.defense.mutate({ challenge_id: challengeID, patch_source_code: patchSourceCode })} isLoading={battleMutations.defense.isPending}>
              提交补丁
            </Button>
            {defenseResult ? (
              <ResultBox ok={defenseResult.is_accepted} okText={`补丁通过，奖励 ${defenseResult.token_reward ?? 0} Token`} failText={defenseResult.rejection_reason ?? "补丁未通过，后端未返回原因"} />
            ) : null}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-5 xl:grid-cols-2">
        <RecordCard title="战场总览" items={(leaderboardQuery.data?.rankings ?? []).map((item) => `#${item.rank} ${item.team_name} Token ${item.token_balance ?? 0} 攻${item.attacks_successful ?? 0} 防${item.defenses_successful ?? 0} 补${item.patches_accepted ?? 0}`)} />
        <RecordCard title="攻击记录" items={(attacksQuery.data?.list ?? []).map((item) => `${item.attacker_team_name} -> ${item.target_team_name} ${item.is_successful ? "成功" : "失败"} ${item.token_reward ?? 0}`)} />
        <RecordCard title="防守记录" items={(defensesQuery.data?.list ?? []).map((item) => `${item.team_name} ${item.challenge_title} ${item.is_accepted ? "通过" : "拒绝"} ${item.token_reward ?? 0}`)} />
      </div>
      <RecordCard title="实时攻击动态" items={realtime.messages.filter((message) => message.channel === "attacks").map((message) => JSON.stringify(message.data))} />
    </div>
  );
}

function Metric({ title, value }: { title: string; value: string | number }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/7 p-4">
      <p className="text-xs text-white/50">{title}</p>
      <p className="mt-1 text-xl font-semibold">{value}</p>
    </div>
  );
}

function ResultBox({ ok, okText, failText }: { ok: boolean; okText: string; failText: string }) {
  return <div className={ok ? "rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm text-emerald-700" : "rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-700"}>{ok ? okText : failText}</div>;
}

function RecordCard({ title, items }: { title: string; items: string[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {items.length === 0 ? <p className="text-sm text-muted-foreground">暂无记录。</p> : null}
        {items.map((item) => <div key={item} className="rounded-lg border border-border p-3 text-sm">{item}</div>)}
      </CardContent>
    </Card>
  );
}
