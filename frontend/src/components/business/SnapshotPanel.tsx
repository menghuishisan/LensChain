"use client";

// SnapshotPanel.tsx
// 模块04快照面板，提供快照创建、恢复和删除操作。

import { History, RotateCcw, Trash2 } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useSnapshotMutations, useSnapshots } from "@/hooks/useExperimentInstances";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * SnapshotPanel 组件属性。
 */
export interface SnapshotPanelProps {
  instanceID: ID;
}

/**
 * SnapshotPanel 实验快照管理组件。
 */
export function SnapshotPanel({ instanceID }: SnapshotPanelProps) {
  const snapshotsQuery = useSnapshots(instanceID);
  const mutations = useSnapshotMutations(instanceID);
  const [description, setDescription] = useState("");

  if (snapshotsQuery.isLoading) {
    return <LoadingState variant="list" title="正在加载快照" description="读取实例快照列表。" />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <History className="h-5 w-5 text-primary" />
          快照管理
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-[1fr_auto]">
          <FormField label="快照说明" description="快照会保存容器状态和 SimEngine 状态，便于回滚。">
            <Input value={description} onChange={(event) => setDescription(event.target.value)} placeholder="例如：完成初始化后的稳定状态" />
          </FormField>
          <Button className="self-end" onClick={() => mutations.create.mutate(description)} isLoading={mutations.create.isPending}>
            创建快照
          </Button>
        </div>
        <div className="space-y-3">
          {(snapshotsQuery.data ?? []).length === 0 ? (
            <div className="rounded-xl border border-dashed border-border p-6 text-sm text-muted-foreground">还没有快照。</div>
          ) : null}
          {(snapshotsQuery.data ?? []).map((snapshot) => (
            <div key={snapshot.id} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-background/70 p-4">
              <div>
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{snapshot.snapshot_type_text}</Badge>
                  <p className="font-semibold">{snapshot.description ?? "未填写说明"}</p>
                </div>
                <p className="mt-1 text-xs text-muted-foreground">{formatDateTime(snapshot.created_at)}</p>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => mutations.restore.mutate(snapshot.id)} isLoading={mutations.restore.isPending}>
                  <RotateCcw className="h-4 w-4" />
                  恢复
                </Button>
                <ConfirmDialog
                  title="删除快照"
                  description="删除后快照将无法恢复，确定继续吗？"
                  confirmText="删除"
                  onConfirm={() => mutations.remove.mutate(snapshot.id)}
                  trigger={<Button variant="destructive" size="sm"><Trash2 className="h-4 w-4" />删除</Button>}
                />
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
