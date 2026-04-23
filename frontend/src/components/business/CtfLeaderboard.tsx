"use client";

// CtfLeaderboard.tsx
// 模块05排行榜组件，展示解题赛分数榜或攻防赛 Token 榜，并显示 WebSocket 同步状态。

import { RefreshCcw, Trophy } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { useCtfLeaderboard } from "@/hooks/useCtfCompetitions";
import { useCtfRealtime } from "@/hooks/useCtfRealtime";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/** CtfLeaderboard 组件属性。 */
export interface CtfLeaderboardProps {
  competitionID: ID;
  groupID?: ID;
}

/** CtfLeaderboard CTF实时排行榜组件。 */
export function CtfLeaderboard({ competitionID, groupID }: CtfLeaderboardProps) {
  const leaderboardQuery = useCtfLeaderboard(competitionID, groupID ? { group_id: groupID, top: 50 } : { top: 50 });
  const realtime = useCtfRealtime(competitionID, competitionID.length > 0);
  const leaderboard = leaderboardQuery.data;

  return (
    <Card className="overflow-hidden border-amber-500/20">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <Trophy className="h-5 w-5 text-amber-500" />
          实时排行榜
        </CardTitle>
        <div className="flex items-center gap-2">
          <Badge variant={leaderboard?.is_frozen ? "destructive" : "success"}>{leaderboard?.is_frozen ? "已冻结" : "实时"}</Badge>
          <Badge variant={realtime.status === "open" ? "success" : "outline"}>{realtime.status === "open" ? "WS已连接" : realtime.status}</Badge>
          <Button size="sm" variant="outline" onClick={() => {
            realtime.reconnect();
            void leaderboardQuery.refetch();
          }}>
            <RefreshCcw className="h-4 w-4" />
            同步
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {leaderboard?.is_frozen ? <p className="rounded-lg bg-amber-500/10 p-3 text-sm text-amber-700">排行榜已于 {formatDateTime(leaderboard.frozen_at)} 冻结，最终排名将在竞赛结束后揭晓。</p> : null}
        <TableContainer>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>排名</TableHead>
                <TableHead>团队</TableHead>
                <TableHead>{leaderboard?.competition_type === 2 ? "Token" : "分数"}</TableHead>
                <TableHead>{leaderboard?.competition_type === 2 ? "攻防统计" : "解题数"}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(leaderboard?.rankings ?? []).map((item) => (
                <TableRow key={item.team_id}>
                  <TableCell className="font-semibold">#{item.rank}</TableCell>
                  <TableCell>{item.team_name}</TableCell>
                  <TableCell>{leaderboard?.competition_type === 2 ? item.token_balance ?? 0 : item.score ?? 0}</TableCell>
                  <TableCell>
                    {leaderboard?.competition_type === 2
                      ? `攻 ${item.attacks_successful ?? 0} / 防 ${item.defenses_successful ?? 0} / 补 ${item.patches_accepted ?? 0}`
                      : `${item.solve_count ?? 0} 题`}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
        <p className="text-xs text-muted-foreground">最新快照：{formatDateTime(leaderboard?.updated_at)}；WebSocket 快照同步：{realtime.hasSnapshotSynced ? "已同步" : "等待同步"}</p>
      </CardContent>
    </Card>
  );
}
