"use client";

// ChallengeVerificationPanel.tsx
// 模块05题目预验证面板，展示 isolated/forked 六步验证进度、断言结果和失败原因。

import { CheckCircle2, CircleAlert, FlaskConical } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { useCtfChallenge, useCtfVerification, useCtfVerifications, useVerifyCtfChallengeMutation } from "@/hooks/useCtfChallenges";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/** ChallengeVerificationPanel 组件属性。 */
export interface ChallengeVerificationPanelProps {
  challengeID: ID;
}

/** ChallengeVerificationPanel 题目预验证组件。 */
export function ChallengeVerificationPanel({ challengeID }: ChallengeVerificationPanelProps) {
  const challengeQuery = useCtfChallenge(challengeID);
  const verificationsQuery = useCtfVerifications(challengeID);
  const verifyMutation = useVerifyCtfChallengeMutation(challengeID);
  const [pocLanguage, setPocLanguage] = useState<"solidity" | "javascript" | "python">("javascript");
  const [pocContent, setPocContent] = useState("async function exploit(provider) {\n  // 提交官方 PoC\n}");
  const [selectedVerificationID, setSelectedVerificationID] = useState("");

  const latestVerification = verifyMutation.data;
  const challenge = challengeQuery.data;
  const effectiveVerificationID = selectedVerificationID || latestVerification?.verification_id || verificationsQuery.data?.list[0]?.id || "";
  const verificationDetailQuery = useCtfVerification(effectiveVerificationID);

  return (
    <div className="space-y-5">
      <Card className="border-cyan-500/20 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 text-white">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-white">
            <FlaskConical className="h-5 w-5 text-cyan-200" />
            题目预验证：{challenge?.title ?? challengeID}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-white/72">
          <p>运行时模式：{challenge?.runtime_mode_text ?? "未配置"}；Fork 模式必须使用固定历史区块，预验证会重置到同一 Fork 快照执行反向验证。</p>
          {challenge?.chain_config?.fork ? <p>Fork 区块：{challenge.chain_config.fork.block_number}，RPC：{challenge.chain_config.fork.rpc_url}</p> : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>提交官方 PoC</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="PoC 语言">
            <Select value={pocLanguage} onValueChange={(value) => setPocLanguage(value as "solidity" | "javascript" | "python")}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="javascript">JavaScript</SelectItem>
                <SelectItem value="python">Python</SelectItem>
                <SelectItem value="solidity">Solidity</SelectItem>
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="PoC 内容">
            <Textarea value={pocContent} onChange={(event) => setPocContent(event.target.value)} rows={10} className="font-mono" />
          </FormField>
          <Button onClick={() => verifyMutation.mutate({ poc_content: pocContent, poc_language: pocLanguage })} isLoading={verifyMutation.isPending}>
            发起预验证
          </Button>
        </CardContent>
      </Card>

      {latestVerification ? (
        <Card>
          <CardHeader>
            <CardTitle>当前验证任务</CardTitle>
          </CardHeader>
          <CardContent>
            <Badge variant={latestVerification.status === 2 ? "success" : "outline"}>{latestVerification.status_text}</Badge>
            <p className="mt-2 text-sm text-muted-foreground">任务 ID：{latestVerification.verification_id}，开始于 {formatDateTime(latestVerification.started_at)}</p>
          </CardContent>
        </Card>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle>历史验证记录</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(verificationsQuery.data?.list ?? []).map((item) => (
            <Button key={item.id} variant="ghost" className="h-auto w-full justify-start rounded-xl border border-border p-4 text-left" onClick={() => setSelectedVerificationID(item.id)}>
              <div className="w-full">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {item.status === 2 ? <CheckCircle2 className="h-4 w-4 text-emerald-500" /> : <CircleAlert className="h-4 w-4 text-amber-500" />}
                    <span className="font-semibold">{item.status_text}</span>
                  </div>
                  <span className="text-xs text-muted-foreground">{formatDateTime(item.started_at)}</span>
                </div>
                {item.error_message ? <p className="mt-2 text-sm text-destructive">{item.error_message}</p> : null}
              </div>
            </Button>
          ))}
        </CardContent>
      </Card>

      {verificationDetailQuery.data ? (
        <Card>
          <CardHeader>
            <CardTitle>验证进度详情</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {verificationDetailQuery.data.step_results.map((step) => (
              <div key={`${step.step}-${step.name}`} className="rounded-xl border border-border p-4">
                <div className="flex items-center justify-between gap-3">
                  <p className="font-semibold">Step {step.step} · {step.name}</p>
                  <Badge variant={step.status === "passed" ? "success" : step.status === "failed" ? "destructive" : "outline"}>{step.status}</Badge>
                </div>
                <p className="mt-2 text-sm text-muted-foreground">{step.detail}</p>
                {(step.assertions ?? []).length > 0 ? (
                  <div className="mt-3 space-y-2">
                    {step.assertions?.map((assertion, index) => (
                      <div key={`${assertion.type}-${index}`} className="rounded-lg bg-muted p-3 text-xs">
                        {assertion.type}: {assertion.actual} vs {assertion.expected} · {assertion.passed ? "通过" : "未通过"}
                      </div>
                    ))}
                  </div>
                ) : null}
              </div>
            ))}
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
