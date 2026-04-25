"use client";

// CtfChallengePanel.tsx
// 模块05解题赛题目面板，覆盖环境启动、Flag/攻击交易提交和明确成功失败反馈。

import { Copy, Play, RotateCcw, Send, Server, Trash2 } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Textarea } from "@/components/ui/Textarea";
import { useCtfCompetitionChallenges } from "@/hooks/useCtfCompetitions";
import { useCtfChallengeEnvironment, useCtfEnvironmentMutations, useCtfSubmissions, useSubmitCtfChallengeMutation } from "@/hooks/useCtfEnvironments";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";
import type { CtfCompetitionChallengeListItem } from "@/types/ctf";

/** CtfChallengePanel 组件属性。 */
export interface CtfChallengePanelProps {
  competitionID: ID;
  challengeID?: ID;
}

function getSubmitType(challenge: CtfCompetitionChallengeListItem) {
  if (challenge.challenge.flag_type === 3) {
    return 3 as const;
  }
  return challenge.challenge.flag_type === 2 ? 2 : 1;
}

/** CtfChallengePanel 解题赛题目与提交组件。 */
export function CtfChallengePanel({ competitionID, challengeID }: CtfChallengePanelProps) {
  const challengesQuery = useCtfCompetitionChallenges(competitionID);
  const selected = (challengesQuery.data?.list ?? []).find((item) => item.challenge.id === challengeID) ?? challengesQuery.data?.list[0];
  const environmentID = selected?.my_team_environment ?? "";
  const environmentQuery = useCtfChallengeEnvironment(environmentID);
  const environmentMutations = useCtfEnvironmentMutations(competitionID, selected?.challenge.id, environmentID);
  const submitMutation = useSubmitCtfChallengeMutation(competitionID);
  const submissionsQuery = useCtfSubmissions(competitionID, { page: 1, page_size: 10 });
  const [content, setContent] = useState("");

  if (!selected) {
    return <EmptyState title="暂无题目" description="竞赛开始后题目会在这里全量展示。" />;
  }

  const challenge = selected.challenge;
  const submitType = getSubmitType(selected);
  const submitResult = submitMutation.data;

  return (
    <div className="grid gap-5 xl:grid-cols-[320px_1fr]">
      <Card className="h-fit">
        <CardHeader>
          <CardTitle>题目列表</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {(challengesQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-3">
              <div className="flex items-center justify-between gap-2">
                <p className="font-semibold">{item.challenge.title}</p>
                <Badge variant={item.my_team_solved ? "success" : "outline"}>{item.my_team_solved ? "已解" : `${item.current_score ?? item.base_score}分`}</Badge>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">{item.challenge.category_text} · {item.challenge.difficulty_text} · 解出 {item.solve_count}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      <div className="space-y-5">
        <Card className="border-cyan-500/20 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 text-white">
          <CardHeader>
            <CardTitle className="text-white">{challenge.title}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Badge>{challenge.category_text}</Badge>
              <Badge variant="outline" className="border-white/18 bg-white/8 text-white">{challenge.difficulty_text}</Badge>
              <Badge variant="outline" className="border-white/18 bg-white/8 text-white">{challenge.flag_type_text}</Badge>
              {challenge.source_path_text ? <Badge variant="secondary">{challenge.source_path_text}</Badge> : null}
            </div>
            <p className="whitespace-pre-wrap text-sm leading-7 text-white/74">{challenge.description}</p>
            {challenge.chain_config?.fork ? (
              <div className="rounded-xl border border-amber-400/30 bg-amber-400/10 p-3 text-sm text-amber-100">
                当前题目使用固定链上快照：区块 {challenge.chain_config.fork.block_number}，链 ID {challenge.chain_config.fork.chain_id}。
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5 text-primary" />
              题目环境
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {environmentID ? (
              <div className="rounded-xl border border-border p-4">
                <p className="font-semibold">{environmentQuery.data?.status_text ?? "读取环境状态中"}</p>
                <p className="mt-1 text-sm text-muted-foreground">Namespace: {environmentQuery.data?.namespace ?? "-"}</p>
                {environmentQuery.data?.chain_rpc_url ? (
                  <p className="mt-2 flex items-center gap-2 rounded-lg bg-muted p-2 font-mono text-xs">
                    {environmentQuery.data.chain_rpc_url}
                    <Copy className="h-3.5 w-3.5" />
                  </p>
                ) : null}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">需要运行环境的题目请先启动环境，其他题目可直接提交答案。</p>
            )}
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => environmentMutations.start.mutate()} isLoading={environmentMutations.start.isPending}>
                <Play className="h-4 w-4" />
                启动环境
              </Button>
              <Button variant="outline" disabled={!environmentID} onClick={() => environmentMutations.reset.mutate()} isLoading={environmentMutations.reset.isPending}>
                <RotateCcw className="h-4 w-4" />
                重置
              </Button>
              <Button variant="destructive" disabled={!environmentID} onClick={() => environmentMutations.destroy.mutate()} isLoading={environmentMutations.destroy.isPending}>
                <Trash2 className="h-4 w-4" />
                销毁
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{submitType === 3 ? "提交攻击交易" : "提交 Flag"}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField label={submitType === 3 ? "攻击交易数据" : "Flag 内容"} description={submitType === 3 ? "系统会在题目环境中执行并返回结果。" : "提交后会立即返回是否正确。"}>
              <Textarea value={content} onChange={(event) => setContent(event.target.value)} rows={6} placeholder={submitType === 3 ? "0x6080..." : "flag{...}"} />
            </FormField>
            <Button
              disabled={content.trim().length === 0}
              onClick={() => submitMutation.mutate({ challenge_id: challenge.id, submission_type: submitType, content })}
              isLoading={submitMutation.isPending}
            >
              <Send className="h-4 w-4" />
              提交
            </Button>
            {submitResult ? (
              <div className={submitResult.is_correct ? "rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm text-emerald-700" : "rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-700"}>
                <p className="font-semibold">{submitResult.is_correct ? "提交正确" : "提交错误"}</p>
                {submitResult.is_correct ? <p>得分 +{submitResult.score_awarded ?? 0}，当前排名 #{submitResult.team_rank ?? "-"}</p> : <p>{submitResult.error_message ?? "本次提交未通过，请检查后重试"}</p>}
                {submitResult.cooldown_until ? <p>冷却至 {formatDateTime(submitResult.cooldown_until)}</p> : null}
              </div>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>提交记录</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {(submissionsQuery.data?.list ?? []).map((item) => (
              <div key={item.id} className="rounded-xl border border-border p-3 text-sm">
                <div className="flex items-center justify-between">
                  <span>{item.challenge_title}</span>
                  <Badge variant={item.is_correct ? "success" : "destructive"}>{item.is_correct ? "正确" : "错误"}</Badge>
                </div>
                <p className="mt-1 text-xs text-muted-foreground">{formatDateTime(item.created_at)} · {item.error_message ?? `+${item.score_awarded ?? 0}分`}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
