"use client";

// CheckpointPanel.tsx
// 模块04检查点面板，提供自动验证结果展示和教师手动评分入口。

import { CheckCircle2, CircleAlert, Gauge } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useCheckpointMutations, useExperimentInstance } from "@/hooks/useExperimentInstances";
import type { ID } from "@/types/api";

/**
 * CheckpointPanel 组件属性。
 */
export interface CheckpointPanelProps {
  instanceID: ID;
  canGrade?: boolean;
}

/**
 * CheckpointPanel 实验检查点验证与评分组件。
 */
export function CheckpointPanel({ instanceID, canGrade = false }: CheckpointPanelProps) {
  const instanceQuery = useExperimentInstance(instanceID);
  const mutations = useCheckpointMutations(instanceID);
  const [scoreDraft, setScoreDraft] = useState<Record<ID, string>>({});

  if (instanceQuery.isLoading) {
    return <LoadingState title="正在加载检查点" description="正在整理当前实验的检查结果。" />;
  }

  const checkpoints = instanceQuery.data?.checkpoints ?? [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <Gauge className="h-5 w-5 text-primary" />
          检查点验证
        </CardTitle>
        <Button size="sm" onClick={() => mutations.verify.mutate(undefined)} isLoading={mutations.verify.isPending} disabled={checkpoints.length === 0}>
          验证全部
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {checkpoints.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border p-6 text-sm text-muted-foreground">当前实验还没有设置检查点。</div>
        ) : null}
        {checkpoints.map((checkpoint) => {
          const isPassed = checkpoint.result?.is_passed ?? false;
          const resultID = checkpoint.result?.id ?? "";
          return (
            <div key={checkpoint.checkpoint_id} className="rounded-xl border border-border bg-background/70 p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    {isPassed ? <CheckCircle2 className="h-4 w-4 text-emerald-500" /> : <CircleAlert className="h-4 w-4 text-amber-500" />}
                    <p className="font-semibold">{checkpoint.title}</p>
                    <Badge variant={isPassed ? "success" : "outline"}>{isPassed ? "已通过" : "未通过"}</Badge>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">满分 {checkpoint.score}，当前得分 {checkpoint.result?.score ?? 0}</p>
                </div>
                <Button variant="outline" size="sm" onClick={() => mutations.verify.mutate(checkpoint.checkpoint_id)} isLoading={mutations.verify.isPending}>
                  单项验证
                </Button>
              </div>
              {canGrade ? (
                <div className="mt-4 grid gap-3 md:grid-cols-[1fr_auto]">
                  <FormField label="教师手动评分" description="手动评分会同步更新当前实验的总分和检查结果。">
                    <Input
                      type="number"
                      min={0}
                      max={checkpoint.score}
                      value={scoreDraft[checkpoint.checkpoint_id] ?? ""}
                      onChange={(event) => setScoreDraft((current) => ({ ...current, [checkpoint.checkpoint_id]: event.target.value }))}
                      placeholder={`0-${checkpoint.score}`}
                    />
                  </FormField>
                  <Button
                    className="self-end"
                    disabled={resultID.length === 0}
                    onClick={() =>
                      mutations.gradeCheckpoint.mutate({
                        resultID,
                        score: Number(scoreDraft[checkpoint.checkpoint_id] ?? "0"),
                      })
                    }
                    isLoading={mutations.gradeCheckpoint.isPending}
                  >
                    保存评分
                  </Button>
                </div>
              ) : null}
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
