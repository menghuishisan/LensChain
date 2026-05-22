'use client';

// ExperimentTerminal.tsx
// 实验终端组件
// 后端通过 K8s Pod exec subresource 在选中容器内直接拉起 PTY，
// 与 xterm.js 进行 WebSocket 双向字节流桥接。实例内任意 Running
// 容器都可作为终端目标，工具（redis-cli / psql / geth attach 等）
// 按镜像自然就绪，无需额外边车容器。

import { useRef, useEffect, useState, useCallback } from 'react';
import { XTermTerminal, type XTermTerminalHandle } from './XTermTerminal';
import { useExperimentTerminal, useTeacherTerminalStream } from '@/hooks/useExperimentRealtime';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select';
import { cn } from '@/lib/utils';
import { Terminal, Wifi, WifiOff, RefreshCcw } from 'lucide-react';
import type { ID } from '@/types/api';

interface ContainerInfo {
  container_name: string;
  display_name?: string;
}

/**
 * ExperimentTerminal 组件属性
 */
export interface ExperimentTerminalProps {
  instanceID: ID;
  containers?: ContainerInfo[];
  container?: string;
  readOnly?: boolean;
  className?: string;
}

/**
 * 实验 Web 终端组件
 * 后端走 K8s exec subresource 提供真 PTY，支持 Tab 补全、vim、信号处理、
 * ANSI 转义等完整终端功能。容器选择器列出实例内所有 Running 容器，
 * 工具按镜像自然就绪。
 */
export function ExperimentTerminal({
  instanceID,
  containers = [],
  container,
  readOnly = false,
  className,
}: ExperimentTerminalProps) {
  const defaultContainer = container ?? containers[0]?.container_name ?? '';
  const [activeContainer, setActiveContainer] = useState(defaultContainer);
  const [ready, setReady] = useState(false);
  const termRef = useRef<XTermTerminalHandle>(null);
  const prevMsgCountRef = useRef(0);

  const writableTerminal = useExperimentTerminal(instanceID, activeContainer, !readOnly);
  const readonlyTerminal = useTeacherTerminalStream(instanceID, readOnly);
  const realtime = readOnly ? readonlyTerminal : writableTerminal;

  // 处理 WS 消息写入 xterm
  useEffect(() => {
    const msgs = realtime.messages;
    if (msgs.length <= prevMsgCountRef.current) {
      prevMsgCountRef.current = msgs.length;
      return;
    }

    const newMessages = msgs.slice(prevMsgCountRef.current);
    prevMsgCountRef.current = msgs.length;

    for (const msg of newMessages) {
      // PTY 字节流（K8s exec stdout）：以 Uint8Array 形式由 useExperimentRealtime 透传，
      // 直接交给 xterm.js 的 write(Uint8Array) 重载——它内部的有状态 UTF-8 解码器跨
      // 调用缓存半字符，正确还原中文 / emoji 等多字节序列，并按字节透传 ANSI 控制序
      // 列，不会损坏二进制 PTY 流。
      if (msg.type === 'binary') {
        termRef.current?.write(msg.data!.bytes as Uint8Array);
        continue;
      }

      // 连接初始化 / 错误消息（JSON 文本帧）。
      if (msg.type === 'terminal_init') {
        const mode = msg.data!.mode as string;
        if (mode === 'pty') {
          setReady(true);
          continue;
        }
        // mode === 'error'：后端透传的上游错误（K8s exec 失败原因、目标命名空间 /
        // Pod / 容器名等）渲染到 xterm，便于运维定位是 Pod 未起、/bin/sh 不存在
        // 还是 RBAC 被拒。
        const message = msg.data!.message as string;
        const upstreamTarget = msg.data!.upstream_target as string | undefined;
        const upstreamReason = msg.data!.upstream_reason as string | undefined;
        let detail = `\r\n\x1b[31m✗ ${message}\x1b[0m\r\n`;
        if (upstreamTarget) {
          detail += `\x1b[90m  目标: ${upstreamTarget}\x1b[0m\r\n`;
        }
        if (upstreamReason) {
          detail += `\x1b[90m  原因: ${upstreamReason}\x1b[0m\r\n`;
        }
        termRef.current?.write(detail);
      }
    }
  }, [realtime.messages.length]);

  // 切换容器时重置状态
  useEffect(() => {
    termRef.current?.clear();
    prevMsgCountRef.current = 0;
    setReady(false);
  }, [activeContainer]);

  // 终端 resize 事件转发到后端（进而下发 SIGWINCH 到容器内 PTY）
  const handleResize = useCallback((cols: number, rows: number) => {
    if (readOnly || !ready) return;
    if ('sendResize' in writableTerminal) {
      writableTerminal.sendResize(cols, rows);
    }
  }, [readOnly, ready, writableTerminal]);

  // 用户键击直接转发到容器内 PTY 的 stdin
  const handleTerminalData = useCallback((data: string) => {
    if (readOnly || !ready) return;
    if ('sendInput' in writableTerminal) {
      writableTerminal.sendInput(data);
    }
  }, [readOnly, ready, writableTerminal]);

  return (
    <div className={cn('flex flex-col gap-2 rounded-lg border border-slate-800 bg-slate-950 overflow-hidden', className)}>
      {/* 顶部状态栏 */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-white/10">
        <div className="flex items-center gap-2">
          <Terminal className="h-4 w-4 text-cyan-300" />
          <span className="text-sm font-medium text-slate-200">
            {readOnly ? '只读终端' : '实验终端'}
          </span>
          {containers.length > 1 && (
            <Select value={activeContainer} onValueChange={setActiveContainer}>
              <SelectTrigger className="h-7 w-auto text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {containers.map(c => (
                  <SelectItem key={c.container_name} value={c.container_name}>
                    {c.display_name ?? c.container_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          {readOnly && <Badge variant="outline" className="text-xs">只读</Badge>}
        </div>
        <div className="flex items-center gap-2">
          <Badge
            variant={realtime.status === 'open' ? 'default' : 'destructive'}
            className="text-xs gap-1"
          >
            {realtime.status === 'open' ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
            {realtime.status === 'open' ? '已连接' : '未连接'}
          </Badge>
          <Button
            variant="ghost"
            size="sm"
            onClick={realtime.reconnect}
            className="h-7 text-slate-300 hover:bg-white/10"
          >
            <RefreshCcw className="h-3 w-3" />
          </Button>
        </div>
      </div>

      {/* 错误提示 */}
      {realtime.error && (
        <div className="mx-3 rounded border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-xs text-red-200">
          {realtime.error}
        </div>
      )}

      {/* xterm.js 终端 */}
      <XTermTerminal
        ref={termRef}
        readOnly={readOnly}
        onData={handleTerminalData}
        onResize={handleResize}
        className="flex-1 px-1 pb-1"
      />
    </div>
  );
}
