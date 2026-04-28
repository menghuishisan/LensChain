"use client";

// ExperimentGroupPanel.tsx
// 模块04实验分组面板，展示角色分配、组内进度、组级检查点和组员终端查看。

import { CheckCircle, Eye, ExternalLink, MessageSquare, RefreshCcw, Send, Terminal, Users, XCircle } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { XTermTerminal, type XTermTerminalHandle } from "@/components/business/XTermTerminal";
import { useAuth } from "@/hooks/useAuth";
import { useExperimentGroup, useExperimentGroupMessages, useExperimentGroupMutations, useExperimentGroupProgress } from "@/hooks/useExperimentGroups";
import { useExperimentGroupChatRealtime, useGroupMemberTerminalStream } from "@/hooks/useExperimentRealtime";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * ExperimentGroupPanel 组件属性。
 */
export interface ExperimentGroupPanelProps {
  groupID: ID;
}

/**
 * ExperimentGroupPanel 实验分组协作组件。
 * 包含组员概览、组内进度、组级检查点、组员终端只读查看和组内消息。
 */
export function ExperimentGroupPanel({ groupID }: ExperimentGroupPanelProps) {
  const router = useRouter();
  const { user } = useAuth();
  const groupQuery = useExperimentGroup(groupID);
  const progressQuery = useExperimentGroupProgress(groupID);
  const messagesQuery = useExperimentGroupMessages(groupID, { page: 1, page_size: 20 });
  const realtime = useExperimentGroupChatRealtime(groupID);
  const mutations = useExperimentGroupMutations(groupID);
  const [message, setMessage] = useState("");
  const [viewingMemberID, setViewingMemberID] = useState("");

  if (groupQuery.isLoading) {
    return <LoadingState title="正在加载分组" description="读取组员、角色和协作进度。" />;
  }

  const group = groupQuery.data;
  const progress = progressQuery.data;
  const historyMessages = messagesQuery.data?.list ?? [];
  const groupCheckpoints = progress?.group_checkpoints ?? [];

  const sendMessage = () => {
    const trimmed = message.trim();
    if (trimmed.length === 0) {
      return;
    }
    const sent = realtime.sendMessage(trimmed);
    if (!sent) {
      mutations.sendMessage.mutate(trimmed);
    }
    setMessage("");
  };

  return (
    <div className="space-y-5">
      <Card className="border-teal-500/20 bg-gradient-to-br from-teal-950 via-slate-950 to-slate-900 text-white">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-white">
            <Users className="h-5 w-5 text-teal-200" />
            {group?.group_name ?? "实验分组"}
          </CardTitle>
          <Badge variant="outline" className="border-white/18 bg-white/8 text-white">
            {group?.status_text ?? "未知状态"}
          </Badge>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          {(group?.members ?? []).map((member) => (
            <div key={member.id} className="rounded-xl border border-white/10 bg-white/7 p-4">
              <p className="font-semibold">{member.student_name}</p>
              <p className="mt-1 text-xs text-white/55">{member.student_no}</p>
              <p className="mt-3 text-sm text-teal-100">{member.role_name ?? "未分配角色"}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Tabs defaultValue="progress">
        <TabsList>
          <TabsTrigger value="progress">组内进度</TabsTrigger>
          <TabsTrigger value="checkpoints">组级检查点</TabsTrigger>
          <TabsTrigger value="terminal">组员终端</TabsTrigger>
          <TabsTrigger value="chat">组内消息</TabsTrigger>
        </TabsList>

        <TabsContent value="progress">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>组内进度</CardTitle>
              <Button variant="outline" size="sm" onClick={() => progressQuery.refetch()}>
                <RefreshCcw className="h-4 w-4" />
                刷新
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              {(progress?.members ?? []).map((member) => (
                <div key={member.student_id} className="rounded-xl border border-border p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-semibold">{member.student_name}</p>
                      <p className="text-sm text-muted-foreground">{member.role_name}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={member.instance_status === 7 ? "success" : "outline"}>{member.instance_status_text ?? "未开始"}</Badge>
                      {member.instance_id ? (
                        <Button variant="ghost" size="sm" onClick={() => setViewingMemberID(member.student_id)}>
                          <Eye className="h-4 w-4" />
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  <div className="mt-3 h-2 rounded-full bg-muted">
                    <div
                      className="h-2 rounded-full bg-primary"
                      style={{ width: `${member.checkpoints_total === 0 ? 0 : Math.round((member.checkpoints_passed / member.checkpoints_total) * 100)}%` }}
                    />
                  </div>
                  <p className="mt-2 text-xs text-muted-foreground">
                    {member.checkpoints_passed}/{member.checkpoints_total} 检查点，通过分 {member.personal_score ?? 0}
                  </p>
                </div>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="checkpoints">
          <Card>
            <CardHeader>
              <CardTitle>组级检查点</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {groupCheckpoints.length === 0 ? (
                <p className="text-sm text-muted-foreground">暂无组级检查点。</p>
              ) : null}
              {groupCheckpoints.map((checkpoint) => (
                <div key={checkpoint.checkpoint_id} className="flex items-center justify-between rounded-xl border border-border p-4">
                  <div className="flex items-center gap-3">
                    {checkpoint.is_passed ? (
                      <CheckCircle className="h-5 w-5 shrink-0 text-emerald-500" />
                    ) : (
                      <XCircle className="h-5 w-5 shrink-0 text-muted-foreground/40" />
                    )}
                    <div>
                      <p className="font-semibold">{checkpoint.title}</p>
                      <p className="text-sm text-muted-foreground">
                        {checkpoint.scope === 2 ? "组级" : "个人"} · {checkpoint.checked_at ? formatDateTime(checkpoint.checked_at) : "未验证"}
                      </p>
                    </div>
                  </div>
                  <Badge variant={checkpoint.is_passed ? "success" : "outline"}>
                    {checkpoint.is_passed ? "通过" : "未通过"}
                  </Badge>
                </div>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="terminal">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Terminal className="h-5 w-5 text-primary" />
                组员终端（只读）
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex flex-wrap gap-2">
                {(progress?.members ?? []).filter((m) => m.instance_id).map((member) => (
                  <Button
                    key={member.student_id}
                    variant={viewingMemberID === member.student_id ? "primary" : "outline"}
                    size="sm"
                    onClick={() => setViewingMemberID(member.student_id)}
                  >
                    {member.student_name}
                  </Button>
                ))}
              </div>
              {viewingMemberID ? (
                <MemberTerminalViewer groupID={groupID} studentID={viewingMemberID} />
              ) : (
                <p className="text-sm text-muted-foreground">选择一个组员查看其终端输出。</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="chat">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MessageSquare className="h-5 w-5 text-primary" />
                组内消息
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="max-h-72 space-y-3 overflow-y-auto rounded-xl border border-border bg-muted/25 p-3">
                {historyMessages.map((item) => (
                  <div key={item.id} className="rounded-lg bg-background p-3 text-sm">
                    <p className="font-semibold">{item.sender_name}</p>
                    <p className="mt-1 text-muted-foreground">{item.content}</p>
                  </div>
                ))}
                {realtime.messages.map((item, index) => (
                  <div key={`${item.type}-${index}`} className="rounded-lg border border-primary/20 bg-primary/8 p-3 text-sm">
                    <p className="font-semibold">{String(item.data?.sender_name ?? "实时消息")}</p>
                    <p className="mt-1 text-muted-foreground">{String(item.content ?? item.data?.content ?? "")}</p>
                  </div>
                ))}
              </div>
              <div className="flex gap-2">
                <Input value={message} onChange={(event) => setMessage(event.target.value)} placeholder="输入组内消息" />
                <Button onClick={sendMessage}>
                  <Send className="h-4 w-4" />
                  发送
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 进入我的实验环境 */}
      {(() => {
        const myMember = (progress?.members ?? []).find((m) => m.student_id === user?.id);
        if (!myMember?.instance_id) return null;
        return (
          <Button onClick={() => router.push(`/student/experiment-instances/${myMember.instance_id}`)}>
            <ExternalLink className="h-4 w-4" />
            进入我的实验环境
          </Button>
        );
      })()}
    </div>
  );
}

/**
 * MemberTerminalViewer 组员终端只读查看子组件。
 * 通过 useGroupMemberTerminalStream 接入组员终端 WS 流。
 */
function MemberTerminalViewer({ groupID, studentID }: { groupID: ID; studentID: ID }) {
  const termRef = useRef<XTermTerminalHandle>(null);
  const terminal = useGroupMemberTerminalStream(groupID, studentID);

  useEffect(() => {
    if (!terminal.messages.length) return;
    const lastMsg = terminal.messages[terminal.messages.length - 1];
    const output = (lastMsg?.data as Record<string, unknown> | undefined)?.output;
    if (typeof output === "string") {
      termRef.current?.write(output);
    }
  }, [terminal.messages.length]);

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant={terminal.status === "open" ? "secondary" : "destructive"}>
          {terminal.status === "open" ? "已连接" : "未连接"}
        </Badge>
        <span className="text-xs text-muted-foreground">只读模式 · 查看组员终端输出</span>
      </div>
      <XTermTerminal ref={termRef} readOnly className="h-[350px] rounded-md overflow-hidden border" />
    </div>
  );
}
