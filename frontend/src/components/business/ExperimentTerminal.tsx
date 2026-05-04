'use client';

// ExperimentTerminal.tsx
// 实验终端组件
// 通过 xterm-server PTY 代理提供真终端交互体验
// 后端 WebSocket 代理自动连接实例中的 xterm-server 工具容器

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
 * 通过 xterm-server 提供真 PTY 终端，支持 Tab 补全、vim、信号处理、ANSI 转义等完整终端功能。
 * 后端将 WebSocket 连接代理到实例 Pod 中的 xterm-server 工具容器。
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
      // 处理初始化消息
      if (msg.type === 'terminal_init') {
        const mode = (msg.data?.mode as string) ?? null;
        if (mode === 'pty') {
          setReady(true);
        }
        if (mode === 'error') {
          termRef.current?.write(`\r\n\x1b[31m${(msg.data?.message as string) ?? '终端连接失败'}\x1b[0m\r\n`);
        }
        continue;
      }

      // PTY 输出：原始终端数据直接写入 xterm
      const raw = typeof msg.data?.value === 'string' ? msg.data.value : '';
      if (raw) {
        termRef.current?.write(raw);
      }
    }
  }, [realtime.messages.length]);

  // 切换容器时重置状态
  useEffect(() => {
    termRef.current?.clear();
    prevMsgCountRef.current = 0;
    setReady(false);
  }, [activeContainer]);

  // 终端 resize 事件转发到 xterm-server
  const handleResize = useCallback((cols: number, rows: number) => {
    if (readOnly || !ready) return;
    if ('sendResize' in writableTerminal) {
      writableTerminal.sendResize(cols, rows);
    }
  }, [readOnly, ready, writableTerminal]);

  // 用户键击直接发送到 xterm-server PTY
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
