"use client";

// ExperimentTerminal.tsx
// 模块04实验终端组件，支持学生命令执行和教师只读终端流。

import { RefreshCcw, Send, Terminal } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { useExperimentTerminal, useTeacherTerminalStream } from "@/hooks/useExperimentRealtime";
import { cn } from "@/lib/utils";
import type { ID } from "@/types/api";

/**
 * ExperimentTerminal 组件属性。
 */
export interface ExperimentTerminalProps {
  instanceID: ID;
  container?: string;
  readOnly?: boolean;
  className?: string;
}

function renderConnectionText(status: string) {
  if (status === "open") {
    return "实时连接中";
  }
  if (status === "connecting") {
    return "正在连接终端";
  }
  if (status === "reconnecting") {
    return "连接中断，正在重连";
  }
  if (status === "error") {
    return "连接异常";
  }
  return "连接已关闭";
}

/**
 * ExperimentTerminal 实验 WebSocket 终端组件。
 */
export function ExperimentTerminal({ instanceID, container, readOnly = false, className }: ExperimentTerminalProps) {
  const writableTerminal = useExperimentTerminal(instanceID, container, !readOnly);
  const readonlyTerminal = useTeacherTerminalStream(instanceID, readOnly);
  const realtime = readOnly ? readonlyTerminal : writableTerminal;
  const [command, setCommand] = useState("");

  const output = realtime.messages
    .map((message) => {
      if (message.stdout || message.stderr) {
        return [`$ ${message.command ?? ""}`, message.stdout ?? "", message.stderr ?? ""].filter(Boolean).join("\n");
      }
      if (typeof message.data?.value === "string") {
        return message.data.value;
      }
      if (typeof message.data === "object") {
        return JSON.stringify(message.data, null, 2);
      }
      return "";
    })
    .filter(Boolean)
    .join("\n\n");

  const submitCommand = () => {
    const trimmed = command.trim();
    if (trimmed.length === 0 || readOnly) {
      return;
    }
    const sent = writableTerminal.sendCommand(trimmed);
    if (sent) {
      setCommand("");
    }
  };

  return (
    <Card className={cn("overflow-hidden border-slate-800 bg-slate-950 text-slate-100", className)}>
      <CardHeader className="border-b border-white/10">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <CardTitle className="flex items-center gap-2 text-slate-50">
            <Terminal className="h-5 w-5 text-cyan-300" />
            {readOnly ? "教师只读终端流" : "学生实验终端"}
          </CardTitle>
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <span className={cn("h-2 w-2 rounded-full", realtime.status === "open" ? "bg-emerald-400" : "bg-amber-400")} />
            {renderConnectionText(realtime.status)}
            <Button variant="ghost" size="sm" className="text-slate-200 hover:bg-white/10" onClick={realtime.reconnect}>
              <RefreshCcw className="h-4 w-4" />
              重连
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 p-4">
        {realtime.error ? (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-100">{realtime.error}</div>
        ) : null}
        <pre className="min-h-72 overflow-auto rounded-xl border border-white/10 bg-black/70 p-4 font-mono text-xs leading-6 text-emerald-100">
          {output || (realtime.status === "open" ? "等待终端输出..." : "终端尚未连接，连接成功后将显示最新输出。")}
        </pre>
        {!readOnly ? (
          <div className="flex gap-2">
            <Input
              className="border-white/10 bg-white/8 text-white placeholder:text-slate-500"
              placeholder="输入容器内命令，例如 geth attach http://127.0.0.1:8545"
              value={command}
              onChange={(event) => setCommand(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  submitCommand();
                }
              }}
            />
            <Button onClick={submitCommand} disabled={realtime.status !== "open" || command.trim().length === 0}>
              <Send className="h-4 w-4" />
              执行
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
