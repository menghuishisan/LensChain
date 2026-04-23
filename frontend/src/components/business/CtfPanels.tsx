"use client";

// CtfPanels.tsx
// 模块05 CTF页面级业务面板，组合竞赛、题目、团队、排行榜、攻防、监控和结果组件。

import { Activity, ChevronDown, ChevronUp, Plus, Send, ShieldCheck, Swords, Trophy } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

import { AttackDefenseRoundPanel } from "@/components/business/AttackDefenseRoundPanel";
import { ChallengeVerificationPanel } from "@/components/business/ChallengeVerificationPanel";
import { CtfChallengePanel } from "@/components/business/CtfChallengePanel";
import { CtfCompetitionCard } from "@/components/business/CtfCompetitionCard";
import { CtfLeaderboard } from "@/components/business/CtfLeaderboard";
import { VulnerabilityConvertPanel } from "@/components/business/VulnerabilityConvertPanel";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { Textarea } from "@/components/ui/Textarea";
import { useCtfChallenge, useCtfChallengeAssetMutations, useCtfChallengeAssets, useCtfChallengeMutations, useCtfChallenges, useCtfVerification, usePendingCtfChallengeReviews } from "@/hooks/useCtfChallenges";
import { useCtfAdminOverview, useCtfAnnouncementMutations, useCtfAnnouncements, useCtfCompetition, useCtfCompetitionChallenges, useCtfCompetitionMonitor, useCtfCompetitionMutations, useCtfCompetitionResults, useCtfCompetitions, useCtfCompetitionStatistics, useCtfLeaderboardHistory, useCtfResourceQuota, useCtfResourceQuotaMutation } from "@/hooks/useCtfCompetitions";
import { useCtfTeam, useCtfTeamMutations, useCtfTeams, useMyCtfRegistration } from "@/hooks/useCtfTeams";
import { useCtfRealtime } from "@/hooks/useCtfRealtime";
import { formatDateTime, formatScore } from "@/lib/format";
import type { ID } from "@/types/api";
import type { CreateCtfChallengeRequest, CreateCtfCompetitionRequest, CtfChallengeCategory, CtfChallengeStatus } from "@/types/ctf";

const DEFAULT_COMPETITION: CreateCtfCompetitionRequest = {
  title: "",
  competition_type: 1,
  scope: 1,
  team_mode: 2,
  max_team_size: 4,
  min_team_size: 1,
  max_teams: 100,
  jeopardy_config: { scoring: { decay_factor: 0.95, min_score_ratio: 0.2, first_blood_bonus: 0.1 }, submission_limit: { max_per_minute: 5, cooldown_threshold: 10, cooldown_minutes: 5 } },
  ad_config: { total_rounds: 10, attack_duration_minutes: 10, defense_duration_minutes: 10, initial_token: 10000, attack_bonus_ratio: 0.05, defense_reward_per_round: 50, first_patch_bonus: 200, first_blood_bonus_ratio: 0.1, vulnerability_decay_factor: 0.8, max_teams_per_group: 8, judge_chain_image: "judge-service:latest", team_chain_image: "ganache:latest" },
};

/** CtfHallPanel 学生竞赛大厅页面。 */
export function CtfHallPanel() {
  const router = useRouter();
  const competitionsQuery = useCtfCompetitions({ page: 1, page_size: 20 });
  const competitions = competitionsQuery.data?.list ?? [];

  return (
    <div className="space-y-6">
      <div className="overflow-hidden rounded-3xl border border-amber-500/20 bg-gradient-to-br from-slate-950 via-zinc-950 to-amber-950 p-8 text-white">
        <p className="font-display text-4xl font-semibold">链镜 CTF 竞赛平台</p>
        <p className="mt-3 max-w-2xl text-white/65">区块链安全解题赛与链上攻防对抗赛，所有判题结果以后端验证为准。</p>
      </div>
      {competitions.length === 0 ? <EmptyState title="暂无可见竞赛" description="报名中或进行中的竞赛会显示在这里。" /> : null}
      <div className="grid gap-5 xl:grid-cols-2">
        {competitions.map((competition) => (
          <CtfCompetitionCard key={competition.id} competition={competition} onOpen={(id) => router.push(`/ctf/${id}`)} />
        ))}
      </div>
    </div>
  );
}

/** CtfCompetitionDetailPanel 学生竞赛详情与报名页面。 */
export function CtfCompetitionDetailPanel({ competitionID }: { competitionID: ID }) {
  const router = useRouter();
  const competitionQuery = useCtfCompetition(competitionID);
  const registrationQuery = useMyCtfRegistration(competitionID);
  const teamsQuery = useCtfTeams(competitionID);
  const announcementsQuery = useCtfAnnouncements(competitionID);
  const realtime = useCtfRealtime(competitionID, competitionID.length > 0);
  const teamMutations = useCtfTeamMutations(competitionID);
  const [teamName, setTeamName] = useState("BlockSec");
  const [inviteCode, setInviteCode] = useState("");

  if (competitionQuery.isLoading) {
    return <LoadingState title="正在加载竞赛" description="读取竞赛信息、报名状态和组队规则。" />;
  }

  if (!competitionQuery.data) {
    return <ErrorState title="竞赛不存在" />;
  }

  const competition = competitionQuery.data;
  const targetPath = competition.competition_type === 2 ? `/ctf/${competitionID}/attack-defense` : `/ctf/${competitionID}/jeopardy`;
  const myTeam = teamsQuery.data?.list.find((team) => team.id === registrationQuery.data?.team_id);

  return (
    <div className="space-y-5">
      <Card className="border-amber-500/20 bg-gradient-to-br from-slate-950 via-zinc-950 to-amber-950 text-white">
        <div className="h-36 bg-[radial-gradient(circle_at_20%_30%,rgba(251,191,36,0.28),transparent_28%),linear-gradient(135deg,rgba(15,23,42,0.88),rgba(120,53,15,0.56))]" />
        <CardHeader>
          <CardTitle className="text-white">{competition.title}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Badge>{competition.competition_type_text}</Badge>
            <Badge variant="outline" className="border-white/18 bg-white/8 text-white">{competition.team_mode_text}</Badge>
            <Badge variant="secondary">{competition.status_text}</Badge>
          </div>
          <div className="grid gap-3 md:grid-cols-4">
            <MetricCard title="报名截止" value={formatDateTime(competition.registration_end_at)} />
            <MetricCard title="竞赛时间" value={formatDateTime(competition.start_at)} />
            <MetricCard title="已报名" value={`${competition.registered_teams}/${competition.max_teams ?? "不限"}`} />
            <MetricCard title="题目数" value={competition.challenge_count} />
          </div>
          <Tabs defaultValue="intro">
            <TabsList className="bg-white/10">
              <TabsTrigger value="intro">竞赛介绍</TabsTrigger>
              <TabsTrigger value="rules">竞赛规则</TabsTrigger>
              <TabsTrigger value="team">我的团队</TabsTrigger>
            </TabsList>
            <TabsContent value="intro">
              <div className="rounded-xl border border-white/10 bg-white/7 p-4 text-sm leading-7 text-white/75 whitespace-pre-wrap">{competition.description ?? "暂无介绍"}</div>
            </TabsContent>
            <TabsContent value="rules">
              <div className="rounded-xl border border-white/10 bg-white/7 p-4 text-sm leading-7 text-white/75 whitespace-pre-wrap">{competition.rules ?? "暂无规则说明"}</div>
            </TabsContent>
            <TabsContent value="team">
              {myTeam ? (
                <div className="rounded-xl border border-white/10 bg-white/7 p-4 text-sm text-white/75">
                  当前团队：{myTeam.name} · 成员 {myTeam.member_count} · 状态 {myTeam.status_text}
                </div>
              ) : (
                <div className="rounded-xl border border-dashed border-white/18 bg-white/7 p-4 text-sm text-white/65">您尚未加入任何团队。</div>
              )}
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>报名与组队</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {registrationQuery.data?.is_registered ? (
            <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm text-emerald-700">已报名：{registrationQuery.data.team_name}</div>
          ) : (
            <div className="grid gap-3 md:grid-cols-[1fr_auto_1fr_auto]">
              <Input value={teamName} onChange={(event) => setTeamName(event.target.value)} placeholder="团队名称" />
              <Button onClick={() => teamMutations.createTeam.mutate(teamName)} isLoading={teamMutations.createTeam.isPending}>创建团队</Button>
              <Input value={inviteCode} onChange={(event) => setInviteCode(event.target.value)} placeholder="邀请码" />
              <Button variant="outline" onClick={() => teamMutations.joinTeam.mutate(inviteCode)} isLoading={teamMutations.joinTeam.isPending}>加入团队</Button>
            </div>
          )}
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => teamMutations.register.mutate(registrationQuery.data?.team_id)} isLoading={teamMutations.register.isPending}>{competition.team_mode === 1 ? "报名参赛" : "团队报名"}</Button>
            <Button variant="outline" onClick={() => router.push(targetPath)}>进入竞赛</Button>
            <Button variant="ghost" onClick={() => router.push(`/ctf/${competitionID}/team`)}>我的团队</Button>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>公告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(announcementsQuery.data?.list ?? []).length === 0 ? <p className="text-sm text-muted-foreground">暂无公告。</p> : null}
          {(announcementsQuery.data?.list ?? []).map((announcement) => (
            <div key={announcement.id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <p className="font-semibold">{announcement.title}</p>
                <Badge>{announcement.announcement_type_text}</Badge>
              </div>
              <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">{announcement.content}</p>
            </div>
          ))}
          {realtime.messages.filter((message) => message.channel === "announcement").length > 0 ? <p className="text-xs text-muted-foreground">实时公告通道已同步 {realtime.messages.filter((message) => message.channel === "announcement").length} 条消息。</p> : null}
        </CardContent>
      </Card>
    </div>
  );
}

/** CtfTeamPanel 学生团队管理页面。 */
export function CtfTeamPanel({ competitionID }: { competitionID: ID }) {
  const teamsQuery = useCtfTeams(competitionID);
  const registrationQuery = useMyCtfRegistration(competitionID);
  const teamMutations = useCtfTeamMutations(competitionID);
  const [teamName, setTeamName] = useState("");
  const myTeam = teamsQuery.data?.list.find((team) => team.id === registrationQuery.data?.team_id);
  const myTeamDetailQuery = useCtfTeam(registrationQuery.data?.team_id ?? "");

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">团队管理</h1>
      {myTeam ? (
        <Card>
          <CardHeader>
            <CardTitle>{myTeam.name}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 text-sm text-muted-foreground">
            <p>队长：{myTeam.captain_name}</p>
            <p>成员：{myTeam.member_count}</p>
            <p>报名：{myTeam.registered ? "已报名" : "未报名"}</p>
            <p>邀请码：{myTeamDetailQuery.data?.invite_code ?? "无"}</p>
            <div className="space-y-2">
              {(myTeamDetailQuery.data?.members ?? []).map((member) => (
                <div key={member.student_id} className="rounded-lg border border-border p-3">
                  {member.name} · {member.role_text}
                </div>
              ))}
            </div>
            <div className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
              <Input value={teamName} onChange={(event) => setTeamName(event.target.value)} placeholder="新团队名称" />
              <Button variant="outline" onClick={() => teamMutations.updateTeam.mutate({ teamID: myTeam.id, name: teamName })} isLoading={teamMutations.updateTeam.isPending}>修改队名</Button>
              <Button variant="destructive" onClick={() => teamMutations.disbandTeam.mutate(myTeam.id)} isLoading={teamMutations.disbandTeam.isPending}>解散团队</Button>
            </div>
            <Button variant="ghost" onClick={() => teamMutations.leaveTeam.mutate(myTeam.id)} isLoading={teamMutations.leaveTeam.isPending}>退出团队</Button>
          </CardContent>
        </Card>
      ) : (
        <EmptyState title="尚未加入团队" description="请在竞赛详情页创建团队或通过邀请码加入。" />
      )}
    </div>
  );
}

/** CtfJeopardyPanel 解题赛主页。 */
export function CtfJeopardyPanel({ competitionID }: { competitionID: ID }) {
  const announcementsQuery = useCtfAnnouncements(competitionID);
  return (
    <div className="grid gap-5 xl:grid-cols-[1fr_380px]">
      <div className="space-y-5">
        {(announcementsQuery.data?.list ?? []).length > 0 ? (
          <Card>
            <CardHeader>
              <CardTitle>竞赛公告</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {(announcementsQuery.data?.list ?? []).slice(0, 3).map((item) => (
                <div key={item.id} className="rounded-xl border border-border p-3 text-sm">
                  <p className="font-semibold">{item.title}</p>
                  <p className="mt-1 text-muted-foreground">{item.challenge_title ?? "全局公告"}</p>
                </div>
              ))}
            </CardContent>
          </Card>
        ) : null}
        <CtfChallengePanel competitionID={competitionID} />
      </div>
      <CtfLeaderboard competitionID={competitionID} />
    </div>
  );
}

/** CtfLeaderboardPagePanel 排行榜页面。 */
export function CtfLeaderboardPagePanel({ competitionID }: { competitionID: ID }) {
  const historyQuery = useCtfLeaderboardHistory(competitionID, { page: 1, page_size: 10 });
  return (
    <div className="space-y-5">
      <CtfLeaderboard competitionID={competitionID} />
      <Card>
        <CardHeader>
          <CardTitle>历史快照</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(historyQuery.data?.list ?? []).map((snapshot) => (
            <div key={snapshot.snapshot_at} className="rounded-xl border border-border p-4">
              <p className="font-semibold">{formatDateTime(snapshot.snapshot_at)}</p>
              <p className="mt-1 text-sm text-muted-foreground">{snapshot.rankings.slice(0, 3).map((item) => `#${item.rank} ${item.team_name}`).join(" · ")}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/** CtfResultsPanel 竞赛结果页面。 */
export function CtfResultsPanel({ competitionID }: { competitionID: ID }) {
  const resultsQuery = useCtfCompetitionResults(competitionID);
  const statisticsQuery = useCtfCompetitionStatistics(competitionID);

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">竞赛结果</h1>
      <div className="grid gap-3 md:grid-cols-4">
        <MetricCard title="队伍" value={resultsQuery.data?.summary.total_teams ?? 0} />
        <MetricCard title="提交" value={resultsQuery.data?.summary.total_submissions ?? 0} />
        <MetricCard title="正确率" value={`${Math.round((resultsQuery.data?.summary.overall_solve_rate ?? 0) * 100)}%`} />
        <MetricCard title="最高分" value={resultsQuery.data?.summary.highest_score ?? 0} />
      </div>
      <TableContainer>
        <Table>
          <TableHeader><TableRow><TableHead>排名</TableHead><TableHead>团队</TableHead><TableHead>分数/Token</TableHead><TableHead>解题/攻防</TableHead></TableRow></TableHeader>
          <TableBody>
            {(resultsQuery.data?.rankings ?? []).map((item) => (
              <TableRow key={item.team_id}>
                <TableCell>#{item.rank}</TableCell>
                <TableCell>{item.team_name}</TableCell>
                <TableCell>{item.score ?? item.token_balance ?? 0}</TableCell>
                <TableCell>{item.solve_count ?? item.attacks_successful ?? 0}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <Card>
        <CardHeader><CardTitle>分数分布</CardTitle></CardHeader>
        <CardContent className="space-y-2">
          {(statisticsQuery.data?.score_distribution.ranges ?? []).map((range) => (
            <div key={range.label} className="flex items-center gap-3 text-sm">
              <span className="w-20">{range.label}</span>
              <div className="h-2 flex-1 rounded-full bg-muted"><div className="h-2 rounded-full bg-primary" style={{ width: `${Math.min(100, range.count * 5)}%` }} /></div>
              <span>{range.count}</span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/** CtfAdminCompetitionListPanel 管理端竞赛列表页面。 */
export function CtfAdminCompetitionListPanel() {
  const router = useRouter();
  const competitionsQuery = useCtfCompetitions({ page: 1, page_size: 30 });
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between"><h1 className="font-display text-3xl font-semibold">CTF竞赛管理</h1><Button onClick={() => router.push("/admin/ctf/competitions/create")}><Plus className="h-4 w-4" />创建竞赛</Button></div>
      <TableContainer><Table><TableHeader><TableRow><TableHead>竞赛</TableHead><TableHead>类型</TableHead><TableHead>状态</TableHead><TableHead>队伍</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>
        {(competitionsQuery.data?.list ?? []).map((competition) => <TableRow key={competition.id}><TableCell>{competition.title}</TableCell><TableCell>{competition.competition_type_text}</TableCell><TableCell><Badge>{competition.status_text}</Badge></TableCell><TableCell>{competition.registered_teams}/{competition.max_teams ?? "不限"}</TableCell><TableCell><Button size="sm" variant="outline" onClick={() => router.push(`/admin/ctf/competitions/${competition.id}/monitor`)}>监控</Button></TableCell></TableRow>)}
      </TableBody></Table></TableContainer>
    </div>
  );
}

/** CtfCompetitionEditorPanel 管理端竞赛创建/编辑页面。 */
export function CtfCompetitionEditorPanel() {
  const [step, setStep] = useState<1 | 2 | 3>(1);
  const [competitionID, setCompetitionID] = useState<ID>("");
  const [form, setForm] = useState<CreateCtfCompetitionRequest>(DEFAULT_COMPETITION);
  const [challengeKeyword, setChallengeKeyword] = useState("");
  const [selectedChallengeIDs, setSelectedChallengeIDs] = useState<ID[]>([]);
  const mutations = useCtfCompetitionMutations(competitionID);
  const challengeLibraryQuery = useCtfChallenges({ page: 1, page_size: 30, status: 3, keyword: challengeKeyword });
  const selectedChallengesQuery = useCtfCompetitionChallenges(competitionID);

  const saveDraft = () => {
    if (competitionID.length > 0) {
      mutations.update.mutate(form, { onSuccess: () => setStep(2) });
      return;
    }
    mutations.create.mutate(form, {
      onSuccess: (created) => {
        setCompetitionID(created.id);
        setStep(2);
      },
    });
  };

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">创建竞赛</h1>
      <div className="flex items-center gap-3 text-sm font-semibold">
        <Badge variant={step === 1 ? "default" : "outline"}>Step 1 基本信息</Badge>
        <Badge variant={step === 2 ? "default" : "outline"}>Step 2 题目配置</Badge>
        <Badge variant={step === 3 ? "default" : "outline"}>Step 3 确认</Badge>
      </div>
      {step === 1 ? (
        <Card><CardContent className="grid gap-4 p-5 md:grid-cols-2">
          <FormField label="竞赛名称" required><Input value={form.title} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} /></FormField>
          <FormField label="竞赛类型"><Select value={String(form.competition_type)} onValueChange={(value) => setForm((current) => ({ ...current, competition_type: Number(value) as 1 | 2 }))}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="1">解题赛</SelectItem><SelectItem value="2">攻防对抗赛</SelectItem></SelectContent></Select></FormField>
          <FormField label="竞赛范围"><Select value={String(form.scope)} onValueChange={(value) => setForm((current) => ({ ...current, scope: Number(value) as 1 | 2 }))}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="1">平台级</SelectItem><SelectItem value="2">校级</SelectItem></SelectContent></Select></FormField>
          <FormField label="参赛模式"><Select value={String(form.team_mode)} onValueChange={(value) => setForm((current) => {
            const mode = Number(value) as 1 | 2 | 3;
            return mode === 1 ? { ...current, team_mode: mode, min_team_size: 1, max_team_size: 1 } : { ...current, team_mode: mode };
          })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="1">个人赛</SelectItem><SelectItem value="2">自由组队</SelectItem><SelectItem value="3">指定组队</SelectItem></SelectContent></Select></FormField>
          <FormField label="团队最小人数"><Input type="number" value={form.min_team_size} disabled={form.team_mode === 1} onChange={(event) => setForm((current) => ({ ...current, min_team_size: Number(event.target.value) }))} /></FormField>
          <FormField label="团队最大人数"><Input type="number" value={form.max_team_size} disabled={form.team_mode === 1} onChange={(event) => setForm((current) => ({ ...current, max_team_size: Number(event.target.value) }))} /></FormField>
          <FormField label="队伍上限"><Input type="number" value={form.max_teams ?? ""} onChange={(event) => setForm((current) => ({ ...current, max_teams: Number(event.target.value) }))} /></FormField>
          <FormField label="冻结时间"><Input type="datetime-local" value={toDateTimeLocal(form.freeze_at)} onChange={(event) => setForm((current) => ({ ...current, freeze_at: fromDateTimeLocal(event.target.value) }))} /></FormField>
          <FormField label="报名开始"><Input type="datetime-local" value={toDateTimeLocal(form.registration_start_at)} onChange={(event) => setForm((current) => ({ ...current, registration_start_at: fromDateTimeLocal(event.target.value) }))} /></FormField>
          <FormField label="报名结束"><Input type="datetime-local" value={toDateTimeLocal(form.registration_end_at)} onChange={(event) => setForm((current) => ({ ...current, registration_end_at: fromDateTimeLocal(event.target.value) }))} /></FormField>
          <FormField label="竞赛开始"><Input type="datetime-local" value={toDateTimeLocal(form.start_at)} onChange={(event) => setForm((current) => ({ ...current, start_at: fromDateTimeLocal(event.target.value) }))} /></FormField>
          <FormField label="竞赛结束"><Input type="datetime-local" value={toDateTimeLocal(form.end_at)} onChange={(event) => setForm((current) => ({ ...current, end_at: fromDateTimeLocal(event.target.value) }))} /></FormField>
          <FormField label="竞赛描述" className="md:col-span-2"><Textarea value={form.description ?? ""} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} rows={6} /></FormField>
          <FormField label="竞赛规则" className="md:col-span-2"><Textarea value={form.rules ?? ""} onChange={(event) => setForm((current) => ({ ...current, rules: event.target.value }))} rows={6} /></FormField>
          {form.competition_type === 1 ? (
            <div className="grid gap-4 md:col-span-2 md:grid-cols-3 rounded-xl border border-border p-4">
              <FormField label="衰减因子"><Input type="number" step="0.01" value={form.jeopardy_config?.scoring.decay_factor ?? 0.95} onChange={(event) => setForm((current) => ({ ...current, jeopardy_config: { ...(current.jeopardy_config ?? DEFAULT_COMPETITION.jeopardy_config!), scoring: { ...(current.jeopardy_config?.scoring ?? DEFAULT_COMPETITION.jeopardy_config!.scoring), decay_factor: Number(event.target.value) } } }))} /></FormField>
              <FormField label="最低分比例"><Input type="number" step="0.01" value={form.jeopardy_config?.scoring.min_score_ratio ?? 0.2} onChange={(event) => setForm((current) => ({ ...current, jeopardy_config: { ...(current.jeopardy_config ?? DEFAULT_COMPETITION.jeopardy_config!), scoring: { ...(current.jeopardy_config?.scoring ?? DEFAULT_COMPETITION.jeopardy_config!.scoring), min_score_ratio: Number(event.target.value) } } }))} /></FormField>
              <FormField label="First Blood"><Input type="number" step="0.01" value={form.jeopardy_config?.scoring.first_blood_bonus ?? 0.1} onChange={(event) => setForm((current) => ({ ...current, jeopardy_config: { ...(current.jeopardy_config ?? DEFAULT_COMPETITION.jeopardy_config!), scoring: { ...(current.jeopardy_config?.scoring ?? DEFAULT_COMPETITION.jeopardy_config!.scoring), first_blood_bonus: Number(event.target.value) } } }))} /></FormField>
            </div>
          ) : (
            <div className="grid gap-4 md:col-span-2 md:grid-cols-3 rounded-xl border border-border p-4">
              <FormField label="总回合数"><Input type="number" value={form.ad_config?.total_rounds ?? 10} onChange={(event) => setForm((current) => ({ ...current, ad_config: { ...(current.ad_config ?? DEFAULT_COMPETITION.ad_config!), total_rounds: Number(event.target.value) } }))} /></FormField>
              <FormField label="攻击时长"><Input type="number" value={form.ad_config?.attack_duration_minutes ?? 10} onChange={(event) => setForm((current) => ({ ...current, ad_config: { ...(current.ad_config ?? DEFAULT_COMPETITION.ad_config!), attack_duration_minutes: Number(event.target.value) } }))} /></FormField>
              <FormField label="防守时长"><Input type="number" value={form.ad_config?.defense_duration_minutes ?? 10} onChange={(event) => setForm((current) => ({ ...current, ad_config: { ...(current.ad_config ?? DEFAULT_COMPETITION.ad_config!), defense_duration_minutes: Number(event.target.value) } }))} /></FormField>
            </div>
          )}
          <div className="flex gap-2 md:col-span-2">
            <Button disabled={!form.title} onClick={saveDraft} isLoading={mutations.create.isPending || mutations.update.isPending}>保存草稿</Button>
            <Button variant="outline" disabled={!form.title} onClick={saveDraft}>下一步</Button>
          </div>
        </CardContent></Card>
      ) : null}
      {step === 2 ? (
        <Card><CardContent className="space-y-4 p-5">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-3">
              <p className="font-semibold">已选题目</p>
              {(selectedChallengesQuery.data?.list ?? []).map((item, index) => (
                <div key={item.id} className="rounded-xl border border-border p-3">
                  <div className="flex items-center justify-between gap-2">
                    <span>{index + 1}. {item.challenge.title}</span>
                    <div className="flex gap-2">
                      <Button size="sm" variant="ghost" disabled={index === 0} onClick={() => mutations.sortChallenges.mutate(reorderCompetitionItems(selectedChallengesQuery.data?.list ?? [], item.id, -1))}><ChevronUp className="h-4 w-4" /></Button>
                      <Button size="sm" variant="ghost" disabled={index === (selectedChallengesQuery.data?.list.length ?? 1) - 1} onClick={() => mutations.sortChallenges.mutate(reorderCompetitionItems(selectedChallengesQuery.data?.list ?? [], item.id, 1))}><ChevronDown className="h-4 w-4" /></Button>
                      <Button size="sm" variant="destructive" onClick={() => mutations.removeChallenge.mutate(item.id)}>移除</Button>
                    </div>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">{item.challenge.category_text} · {item.challenge.difficulty_text} · 基础分 {item.base_score}</p>
                </div>
              ))}
            </div>
            <div className="space-y-3">
              <Input value={challengeKeyword} onChange={(event) => setChallengeKeyword(event.target.value)} placeholder="搜索题库" />
              {(challengeLibraryQuery.data?.list ?? []).map((challenge) => (
                <label key={challenge.id} className="flex items-center justify-between gap-3 rounded-xl border border-border p-3">
                  <div>
                    <p className="font-semibold">{challenge.title}</p>
                    <p className="text-xs text-muted-foreground">{challenge.category_text} · {challenge.difficulty_text} · {challenge.base_score}</p>
                  </div>
                  <input type="checkbox" checked={selectedChallengeIDs.includes(challenge.id)} onChange={(event) => setSelectedChallengeIDs((current) => event.target.checked ? [...new Set([...current, challenge.id])] : current.filter((id) => id !== challenge.id))} />
                </label>
              ))}
              <Button disabled={selectedChallengeIDs.length === 0} onClick={() => mutations.addChallenges.mutate(selectedChallengeIDs)} isLoading={mutations.addChallenges.isPending}>添加选中题目</Button>
            </div>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setStep(1)}>上一步</Button>
            <Button onClick={() => setStep(3)}>下一步</Button>
          </div>
        </CardContent></Card>
      ) : null}
      {step === 3 ? (
        <Card><CardContent className="space-y-4 p-5">
          <p className="font-semibold">竞赛确认</p>
          <div className="grid gap-3 md:grid-cols-4">
            <MetricCard title="赛制" value={form.competition_type === 2 ? "攻防赛" : "解题赛"} />
            <MetricCard title="范围" value={form.scope === 2 ? "校级" : "平台级"} />
            <MetricCard title="队伍模式" value={form.team_mode === 1 ? "个人赛" : form.team_mode === 2 ? "自由组队" : "指定组队"} />
            <MetricCard title="题目数" value={selectedChallengesQuery.data?.list.length ?? 0} />
          </div>
          <div className="rounded-xl border border-border p-4 text-sm text-muted-foreground whitespace-pre-wrap">{form.description || "暂无描述"}</div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setStep(2)}>上一步</Button>
            <Button onClick={() => mutations.publish.mutate()} isLoading={mutations.publish.isPending} disabled={competitionID.length === 0}>发布竞赛</Button>
          </div>
        </CardContent></Card>
      ) : null}
    </div>
  );
}

/** CtfCompetitionMonitorPanel 管理端竞赛监控页面。 */
export function CtfCompetitionMonitorPanel({ competitionID }: { competitionID: ID }) {
  const monitorQuery = useCtfCompetitionMonitor(competitionID);
  const announcementMutation = useCtfAnnouncementMutations(competitionID);
  const competitionMutations = useCtfCompetitionMutations(competitionID);
  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">竞赛监控</h1>
      <div className="grid gap-3 md:grid-cols-4"><MetricCard title="参赛队伍" value={monitorQuery.data?.overview.registered_teams ?? 0} /><MetricCard title="总提交" value={monitorQuery.data?.overview.total_submissions ?? 0} /><MetricCard title="正确提交" value={monitorQuery.data?.overview.correct_submissions ?? 0} /><MetricCard title="运行环境" value={monitorQuery.data?.overview.running_environments ?? 0} /></div>
      <Card><CardHeader><CardTitle>资源使用</CardTitle></CardHeader><CardContent className="space-y-2">
        <MetricBar label="CPU" used={monitorQuery.data?.resource_usage.cpu_used ?? "0"} total={monitorQuery.data?.resource_usage.cpu_max ?? "0"} ratio={percentFromText(monitorQuery.data?.resource_usage.cpu_used, monitorQuery.data?.resource_usage.cpu_max)} />
        <MetricBar label="内存" used={monitorQuery.data?.resource_usage.memory_used ?? "0"} total={monitorQuery.data?.resource_usage.memory_max ?? "0"} ratio={percentFromText(monitorQuery.data?.resource_usage.memory_used, monitorQuery.data?.resource_usage.memory_max)} />
        <MetricBar label="Namespace" used={String(monitorQuery.data?.resource_usage.namespaces_used ?? 0)} total={String(monitorQuery.data?.resource_usage.namespaces_max ?? 0)} ratio={percentFromNumber(monitorQuery.data?.resource_usage.namespaces_used ?? 0, monitorQuery.data?.resource_usage.namespaces_max ?? 0)} />
      </CardContent></Card>
      <Card><CardHeader><CardTitle>题目统计</CardTitle></CardHeader><CardContent className="space-y-2">{(monitorQuery.data?.challenge_stats ?? []).map((item) => <div key={item.challenge_id} className="rounded-xl border border-border p-3 text-sm">{item.title} · 解题 {item.solve_count} · 正确率 {Math.round(item.solve_rate * 100)}% · 当前分 {item.current_score ?? "-"} · 环境 {item.environments_running}</div>)}</CardContent></Card>
      <Card><CardHeader><CardTitle>最近提交</CardTitle></CardHeader><CardContent className="space-y-2">{(monitorQuery.data?.recent_submissions ?? []).map((item) => <div key={`${item.team_name}-${item.submitted_at}`} className="rounded-xl border border-border p-3 text-sm">{item.team_name} → {item.challenge_title} {item.is_correct ? "正确" : "错误"} · {formatDateTime(item.submitted_at)}</div>)}</CardContent></Card>
      <div className="flex gap-2"><Button onClick={() => announcementMutation.mutate({ title: "竞赛公告", content: "请关注竞赛规则更新", announcement_type: 1 })} isLoading={announcementMutation.isPending}><Send className="h-4 w-4" />发布公告</Button><Button variant="destructive" onClick={() => competitionMutations.terminate.mutate("管理员终止竞赛")} isLoading={competitionMutations.terminate.isPending}>强制终止竞赛</Button></div>
    </div>
  );
}

/** CtfChallengeManagementPanel 教师题目管理页面。 */
export function CtfChallengeManagementPanel() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [category, setCategory] = useState<CtfChallengeCategory | "all">("all");
  const [status, setStatus] = useState("all");
  const challengesQuery = useCtfChallenges({ page: 1, page_size: 30, keyword, category: category === "all" ? undefined : category, status: status === "all" ? undefined : (Number(status) as CtfChallengeStatus) });
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between"><h1 className="font-display text-3xl font-semibold">我的CTF题目</h1><div className="flex gap-2"><Button onClick={() => router.push("/teacher/ctf/challenges/create")}>创建题目</Button><Button variant="outline" onClick={() => router.push("/teacher/ctf/challenges/import")}>漏洞转化</Button></div></div>
      <div className="grid gap-3 md:grid-cols-3">
        <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索题目名称" />
        <Select value={category} onValueChange={(value) => setCategory(value as CtfChallengeCategory | "all")}><SelectTrigger><SelectValue placeholder="全部类型" /></SelectTrigger><SelectContent><SelectItem value="all">全部类型</SelectItem><SelectItem value="contract">智能合约</SelectItem><SelectItem value="blockchain">链上分析</SelectItem><SelectItem value="crypto">密码学</SelectItem></SelectContent></Select>
        <Select value={status} onValueChange={setStatus}><SelectTrigger><SelectValue placeholder="全部状态" /></SelectTrigger><SelectContent><SelectItem value="all">全部状态</SelectItem><SelectItem value="1">草稿</SelectItem><SelectItem value="2">待审核</SelectItem><SelectItem value="3">已通过</SelectItem><SelectItem value="4">已拒绝</SelectItem></SelectContent></Select>
      </div>
      <TableContainer><Table><TableHeader><TableRow><TableHead>题目</TableHead><TableHead>类型</TableHead><TableHead>运行时</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>
        {(challengesQuery.data?.list ?? []).map((challenge) => <TableRow key={challenge.id}><TableCell>{challenge.title}</TableCell><TableCell>{challenge.category_text}</TableCell><TableCell>{challenge.runtime_mode_text ?? "-"}</TableCell><TableCell><Badge>{challenge.status_text}</Badge></TableCell><TableCell className="space-x-2"><Button size="sm" variant="outline" onClick={() => router.push(`/teacher/ctf/challenges/${challenge.id}/verify`)}>验证</Button><Button size="sm" variant="ghost" onClick={() => router.push(`/teacher/ctf/challenges/create?challenge_id=${challenge.id}`)}>编辑</Button></TableCell></TableRow>)}
      </TableBody></Table></TableContainer>
    </div>
  );
}

/** CtfChallengeEditorPanel 教师题目创建页面。 */
export function CtfChallengeEditorPanel() {
  const searchParams = useSearchParams();
  const initialChallengeID = searchParams.get("challenge_id") ?? "";
  const [challengeID, setChallengeID] = useState<ID>(initialChallengeID);
  const challengeQuery = useCtfChallenge(challengeID);
  const assetsQuery = useCtfChallengeAssets(challengeID);
  const mutations = useCtfChallengeMutations(challengeID);
  const assetMutations = useCtfChallengeAssetMutations(challengeID);
  const [form, setForm] = useState<CreateCtfChallengeRequest>({ title: "", description: "", category: "contract", difficulty: 2, base_score: 300, flag_type: 3, runtime_mode: 1, setup_transactions: [], attachment_urls: [], chain_config: { chain_type: "evm", chain_version: "london", block_number: 0, accounts: [{ name: "deployer", balance: "100 ether" }, { name: "attacker", balance: "10 ether" }] } });
  const [contractName, setContractName] = useState("VulnerableBank");
  const [contractSource, setContractSource] = useState("pragma solidity ^0.8.20;\ncontract VulnerableBank {\n  bool public solved;\n  function attack() external { solved = true; }\n}");
  const [assertionTarget, setAssertionTarget] = useState("solved");

  const currentChallenge = challengeQuery.data;
  const hasDraft = challengeID.length > 0;

  const saveDraft = () => {
    if (hasDraft) {
      mutations.update.mutate({
        title: form.title || currentChallenge?.title,
        description: form.description || currentChallenge?.description,
        category: form.category || currentChallenge?.category,
        difficulty: form.difficulty ?? currentChallenge?.difficulty,
        base_score: form.base_score ?? currentChallenge?.base_score,
        flag_type: form.flag_type ?? currentChallenge?.flag_type,
        runtime_mode: form.runtime_mode ?? currentChallenge?.runtime_mode ?? 1,
        chain_config: form.chain_config ?? currentChallenge?.chain_config,
        setup_transactions: form.setup_transactions.length > 0 ? form.setup_transactions : currentChallenge?.setup_transactions ?? [],
        attachment_urls: form.attachment_urls.length > 0 ? form.attachment_urls : currentChallenge?.attachment_urls ?? [],
      });
      return;
    }
    mutations.create.mutate(form, {
      onSuccess: (created) => setChallengeID(created.id),
    });
  };

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">{hasDraft ? "编辑题目" : "创建题目"}</h1>
      <Tabs defaultValue="basic">
        <TabsList>
          <TabsTrigger value="basic">基本信息</TabsTrigger>
          <TabsTrigger value="assets" disabled={!hasDraft}>合约与断言</TabsTrigger>
          <TabsTrigger value="actions" disabled={!hasDraft}>验证与审核</TabsTrigger>
        </TabsList>
        <TabsContent value="basic">
          <Card><CardContent className="grid gap-4 p-5 md:grid-cols-2">
            <FormField label="题目名称"><Input value={form.title || currentChallenge?.title || ""} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} /></FormField>
            <FormField label="运行时模式"><Select value={String(form.runtime_mode ?? currentChallenge?.runtime_mode ?? 1)} onValueChange={(value) => setForm((current) => ({ ...current, runtime_mode: Number(value) as 1 | 2 }))}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="1">isolated 独立链</SelectItem><SelectItem value="2">forked 固定区块 Fork</SelectItem></SelectContent></Select></FormField>
            <FormField label="分类"><Input value={form.category || currentChallenge?.category || "contract"} onChange={(event) => setForm((current) => ({ ...current, category: event.target.value as CreateCtfChallengeRequest["category"] }))} /></FormField>
            <FormField label="基础分"><Input type="number" value={form.base_score || currentChallenge?.base_score || 300} onChange={(event) => setForm((current) => ({ ...current, base_score: Number(event.target.value) }))} /></FormField>
            <FormField label="描述" className="md:col-span-2"><Textarea value={form.description || currentChallenge?.description || ""} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} rows={8} /></FormField>
            <Button className="md:col-span-2" disabled={!((form.title || currentChallenge?.title) && (form.description || currentChallenge?.description))} onClick={saveDraft} isLoading={mutations.create.isPending || mutations.update.isPending}>{hasDraft ? "保存修改" : "保存题目草稿"}</Button>
          </CardContent></Card>
        </TabsContent>
        <TabsContent value="assets">
          <div className="grid gap-5 xl:grid-cols-2">
            <Card><CardHeader><CardTitle>添加合约</CardTitle></CardHeader><CardContent className="space-y-4">
              <FormField label="合约名称"><Input value={contractName} onChange={(event) => setContractName(event.target.value)} /></FormField>
              <FormField label="源码"><Textarea value={contractSource} onChange={(event) => setContractSource(event.target.value)} rows={14} className="font-mono" /></FormField>
              <Button onClick={() => assetMutations.createContract.mutate({ name: contractName, source_code: contractSource, abi: [], bytecode: "0x00", constructor_args: [], deploy_order: 1 })} isLoading={assetMutations.createContract.isPending}>添加合约</Button>
            </CardContent></Card>
            <Card><CardHeader><CardTitle>添加断言</CardTitle></CardHeader><CardContent className="space-y-4">
              <FormField label="断言目标"><Input value={assertionTarget} onChange={(event) => setAssertionTarget(event.target.value)} /></FormField>
              <Button onClick={() => assetMutations.createAssertion.mutate({ assertion_type: "storage_check", target: assertionTarget, operator: "eq", expected_value: "true", description: "漏洞利用后应设置 solved=true", extra_params: {}, sort_order: 1 })} isLoading={assetMutations.createAssertion.isPending}>添加断言</Button>
              <div className="space-y-2">
                {(assetsQuery.contracts.data?.list ?? []).map((item) => <div key={item.id} className="rounded-xl border border-border p-3 text-sm">{item.name} · 部署顺序 {item.deploy_order}</div>)}
                {(assetsQuery.assertions.data?.list ?? []).map((item) => <div key={item.id} className="rounded-xl border border-border p-3 text-sm">{item.assertion_type} · {item.target} {item.operator} {item.expected_value}</div>)}
              </div>
            </CardContent></Card>
          </div>
        </TabsContent>
        <TabsContent value="actions">
          <Card><CardHeader><CardTitle>验证与审核</CardTitle></CardHeader><CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => window.location.assign(`/teacher/ctf/challenges/${challengeID}/verify`)}>发起预验证</Button>
              <Button variant="outline" onClick={() => mutations.submitReview.mutate()} isLoading={mutations.submitReview.isPending}>提交审核</Button>
            </div>
            <p className="text-sm text-muted-foreground">链上验证题目必须完成预验证后才能审核通过。当前状态：{currentChallenge?.status_text ?? "草稿"}</p>
          </CardContent></Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

/** CtfChallengeReviewPanel 超级管理员题目审核页面。 */
export function CtfChallengeReviewPanel() {
  const reviewsQuery = usePendingCtfChallengeReviews();
  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">题目审核</h1>
      {(reviewsQuery.data?.list ?? []).map((item) => <Card key={item.id}><CardContent className="flex items-center justify-between p-5"><div><p className="font-semibold">{item.title}</p><p className="text-sm text-muted-foreground">{item.category_text} · {item.difficulty_text} · {item.author_name}</p></div><Badge>待审核</Badge></CardContent></Card>)}
    </div>
  );
}

/** CtfChallengeReviewDetailPanel 超级管理员题目审核详情页面。 */
export function CtfChallengeReviewDetailPanel({ challengeID }: { challengeID: ID }) {
  const challengeQuery = useCtfChallenge(challengeID);
  const mutations = useCtfChallengeMutations(challengeID);
  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">题目审核详情</h1>
      <Card>
        <CardHeader><CardTitle>{challengeQuery.data?.title ?? challengeID}</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2"><Badge>{challengeQuery.data?.category_text ?? "-"}</Badge><Badge>{challengeQuery.data?.difficulty_text ?? "-"}</Badge><Badge>{challengeQuery.data?.runtime_mode_text ?? "非链上运行时"}</Badge></div>
          <p className="whitespace-pre-wrap text-sm leading-7 text-muted-foreground">{challengeQuery.data?.description}</p>
          <Tabs defaultValue="contracts"><TabsList><TabsTrigger value="contracts">合约</TabsTrigger><TabsTrigger value="assertions">断言</TabsTrigger><TabsTrigger value="verification">预验证</TabsTrigger></TabsList>
            <TabsContent value="contracts">{(challengeQuery.data?.contracts ?? []).map((item) => <pre key={item.id} className="overflow-auto rounded-xl bg-muted p-4 text-xs">{item.source_code}</pre>)}</TabsContent>
            <TabsContent value="assertions">{(challengeQuery.data?.assertions ?? []).map((item) => <div key={item.id} className="rounded-xl border border-border p-3 text-sm">{item.assertion_type} · {item.target} {item.operator} {item.expected_value}</div>)}</TabsContent>
            <TabsContent value="verification"><Badge variant={challengeQuery.data?.latest_verification?.status === 2 ? "success" : "outline"}>{challengeQuery.data?.latest_verification?.status_text ?? "无预验证"}</Badge></TabsContent>
          </Tabs>
          <div className="flex gap-2"><Button onClick={() => mutations.review.mutate({ action: 1, comment: "审核通过" })}>通过</Button><Button variant="destructive" onClick={() => mutations.review.mutate({ action: 2, comment: "请修正预验证或题面" })}>拒绝</Button></div>
        </CardContent>
      </Card>
    </div>
  );
}

/** CtfResourceQuotaPanel 超级管理员竞赛资源配额管理页面。 */
export function CtfResourceQuotaPanel() {
  const [competitionID, setCompetitionID] = useState("");
  const quotaQuery = useCtfResourceQuota(competitionID);
  const quotaMutation = useCtfResourceQuotaMutation(competitionID);
  const [maxCPU, setMaxCPU] = useState("32");
  const [maxMemory, setMaxMemory] = useState("64Gi");
  const [maxStorage, setMaxStorage] = useState("100Gi");
  const [maxNamespaces, setMaxNamespaces] = useState("200");
  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">CTF资源配额</h1>
      <Card><CardContent className="grid gap-3 p-5 md:grid-cols-5">
        <FormField label="竞赛 ID"><Input value={competitionID} onChange={(event) => setCompetitionID(event.target.value)} /></FormField>
        <FormField label="CPU"><Input value={maxCPU} onChange={(event) => setMaxCPU(event.target.value)} /></FormField>
        <FormField label="内存"><Input value={maxMemory} onChange={(event) => setMaxMemory(event.target.value)} /></FormField>
        <FormField label="存储"><Input value={maxStorage} onChange={(event) => setMaxStorage(event.target.value)} /></FormField>
        <FormField label="Namespace"><Input type="number" value={maxNamespaces} onChange={(event) => setMaxNamespaces(event.target.value)} /></FormField>
        <Button className="md:col-span-5" disabled={!competitionID} onClick={() => quotaMutation.mutate({ max_cpu: maxCPU, max_memory: maxMemory, max_storage: maxStorage, max_namespaces: Number(maxNamespaces) })} isLoading={quotaMutation.isPending}>保存配额</Button>
      </CardContent></Card>
      {quotaQuery.data ? <Card><CardContent className="grid gap-3 p-5 md:grid-cols-4"><MetricCard title="CPU" value={`${quotaQuery.data.used_cpu}/${quotaQuery.data.max_cpu}`} /><MetricCard title="内存" value={`${quotaQuery.data.used_memory}/${quotaQuery.data.max_memory}`} /><MetricCard title="存储" value={`${quotaQuery.data.used_storage}/${quotaQuery.data.max_storage}`} /><MetricCard title="Namespace" value={`${quotaQuery.data.current_namespaces}/${quotaQuery.data.max_namespaces}`} /></CardContent></Card> : null}
    </div>
  );
}

/** CtfAdminOverviewPanel 超级管理员 CTF 总览页面。 */
export function CtfAdminOverviewPanel() {
  const overviewQuery = useCtfAdminOverview();
  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">CTF竞赛概览</h1>
      <div className="grid gap-3 md:grid-cols-4"><MetricCard title="竞赛总数" value={overviewQuery.data?.total_competitions ?? 0} /><MetricCard title="进行中" value={overviewQuery.data?.running_competitions ?? 0} /><MetricCard title="参赛人数" value={overviewQuery.data?.total_participants ?? 0} /><MetricCard title="活跃NS" value={overviewQuery.data?.total_resource_usage.namespaces_active ?? 0} /></div>
      <Card><CardHeader><CardTitle>告警</CardTitle></CardHeader><CardContent>{(overviewQuery.data?.alerts ?? []).map((alert) => <p key={`${alert.competition_id}-${alert.created_at}`} className="text-sm text-muted-foreground">{alert.message} · {formatDateTime(alert.created_at)}</p>)}</CardContent></Card>
    </div>
  );
}

function MetricCard({ title, value }: { title: string; value: string | number }) {
  return <Card><CardContent className="p-4"><p className="text-sm text-muted-foreground">{title}</p><p className="mt-1 font-display text-2xl font-semibold">{value}</p></CardContent></Card>;
}

function MetricBar({ label, used, total, ratio }: { label: string; used: string; total: string; ratio: number }) {
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-sm">
        <span>{label}</span>
        <span>{used}/{total}</span>
      </div>
      <div className="h-2 rounded-full bg-muted">
        <div className="h-2 rounded-full bg-primary" style={{ width: `${ratio}%` }} />
      </div>
    </div>
  );
}

function percentFromText(used?: string | null, total?: string | null) {
  const usedValue = Number.parseFloat((used ?? "0").replace(/[^\d.]/g, ""));
  const totalValue = Number.parseFloat((total ?? "0").replace(/[^\d.]/g, ""));
  return percentFromNumber(usedValue, totalValue);
}

function percentFromNumber(used: number, total: number) {
  if (!Number.isFinite(used) || !Number.isFinite(total) || total <= 0) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round((used / total) * 100)));
}

function toDateTimeLocal(value?: string | null) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const offset = date.getTimezoneOffset();
  const local = new Date(date.getTime() - offset * 60_000);
  return local.toISOString().slice(0, 16);
}

function fromDateTimeLocal(value: string) {
  if (!value) {
    return null;
  }
  return new Date(value).toISOString();
}

function reorderCompetitionItems(items: Array<{ id: ID }>, targetID: ID, direction: -1 | 1) {
  const currentIndex = items.findIndex((item) => item.id === targetID);
  const targetIndex = currentIndex + direction;
  if (currentIndex < 0 || targetIndex < 0 || targetIndex >= items.length) {
    return items.map((item, index) => ({ id: item.id, sort_order: index + 1 }));
  }
  const next = [...items];
  const [moved] = next.splice(currentIndex, 1);
  next.splice(targetIndex, 0, moved);
  return next.map((item, index) => ({ id: item.id, sort_order: index + 1 }));
}
