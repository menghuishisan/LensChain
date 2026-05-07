// CtfCompetitionCard.tsx
// 模块05竞赛卡片，展示赛制、状态、报名人数、时间和进入动作。

import { CalendarClock, Shield, Swords, Trophy } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import { formatDateTime } from "@/lib/format";
import type { CtfCompetitionListItem } from "@/types/ctf";

/** CtfCompetitionCard 组件属性。 */
export interface CtfCompetitionCardProps {
  competition: CtfCompetitionListItem;
  onOpen: (competitionID: string) => void;
}

function getStatusVariant(status: number) {
  if (status === 3) {
    return "success" as const;
  }
  if (status === 4 || status === 5) {
    return "secondary" as const;
  }
  if (status === 2) {
    return "warning" as const;
  }
  return "warning" as const;
}

/** CtfCompetitionCard CTF竞赛卡片组件。 */
export function CtfCompetitionCard({ competition, onOpen }: CtfCompetitionCardProps) {
  const Icon = competition.competition_type === 2 ? Swords : Trophy;
  return (
    <Card className="group overflow-hidden border-amber-500/20 bg-gradient-to-br from-slate-950 via-zinc-950 to-amber-950 text-white shadow-[0_24px_70px_rgba(120,53,15,0.24)]">
      <div className="h-24 bg-[radial-gradient(circle_at_20%_20%,rgba(251,191,36,0.28),transparent_34%),linear-gradient(135deg,rgba(15,23,42,0.8),rgba(113,63,18,0.55))]" />
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="line-clamp-2 text-white">{competition.title}</CardTitle>
            <div className="mt-3 flex flex-wrap gap-2">
              <Badge variant={getStatusVariant(competition.status)}>{competition.status_text}</Badge>
              <Badge variant="outline" className="border-white/18 bg-white/8 text-white">{competition.competition_type_text}</Badge>
              <Badge variant="outline" className="border-white/18 bg-white/8 text-white">{competition.team_mode_text}</Badge>
            </div>
          </div>
          <Icon className="h-8 w-8 shrink-0 text-amber-200" />
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-3 gap-3">
          <div className="rounded-xl border border-white/10 bg-white/7 p-3">
            <p className="text-xs text-white/50">报名队伍</p>
            <p className="mt-1 text-sm font-semibold">{competition.registered_teams}/{competition.max_teams ?? "不限"}</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-3">
            <p className="text-xs text-white/50">题目数</p>
            <p className="mt-1 text-sm font-semibold">{competition.challenge_count}</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-3">
            <p className="text-xs text-white/50">团队上限</p>
            <p className="mt-1 text-sm font-semibold">{competition.max_team_size}</p>
          </div>
        </div>
        <p className="flex items-center gap-2 text-xs text-white/65">
          <CalendarClock className="h-4 w-4" />
          {formatDateTime(competition.start_at)} - {formatDateTime(competition.end_at)}
        </p>
      </CardContent>
      <CardFooter>
        <Button variant="secondary" onClick={() => onOpen(competition.id)}>
          <Shield className="h-4 w-4" />
          {competition.status === 3 ? "进入竞赛" : competition.status === 4 ? "查看结果" : "查看详情"}
        </Button>
      </CardFooter>
    </Card>
  );
}
