'use client';

// ExperimentTerminal.tsx
// 实验终端组件
// 基于 xterm.js 提供真 PTY 终端体验，支持多容器切换

import { useRef, useEffect, useState } from 'react';
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
 * 使用 xterm.js 提供完整的终端仿真，支持 ANSI 转义、光标控制、多容器切换
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
  const termRef = useRef<XTermTerminalHandle>(null);
  const prevMsgCountRef = useRef(0);

  const writableTerminal = useExperimentTerminal(instanceID, activeContainer, !readOnly);
  const readonlyTerminal = useTeacherTerminalStream(instanceID, readOnly);
  const realtime = readOnly ? readonlyTerminal : writableTerminal;

  // 将 WS 消息写入 xterm
  useEffect(() => {
    const msgs = realtime.messages;
    if (msgs.length <= prevMsgCountRef.current) {
      prevMsgCountRef.current = msgs.length;
      return;
    }

    const newMessages = msgs.slice(prevMsgCountRef.current);
    prevMsgCountRef.current = msgs.length;

    for (const msg of newMessages) {
      let output = '';
      if (msg.stdout) output += msg.stdout;
      if (msg.stderr) output += msg.stderr;
      if (typeof msg.data?.value === 'string') output += msg.data.value;
      if (!output && typeof msg.data === 'object' && msg.data !== null) {
        output = JSON.stringify(msg.data, null, 2);
      }
      if (output) {
        termRef.current?.write(output);
      }
    }
  }, [realtime.messages.length]);

  // 切换容器时清空终端
  useEffect(() => {
    termRef.current?.clear();
    prevMsgCountRef.current = 0;
  }, [activeContainer]);

  const handleTerminalData = (data: string) => {
    if (readOnly) return;
    if ('sendCommand' in writableTerminal && writableTerminal.sendCommand) {
      writableTerminal.sendCommand(data);
    }
  };

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
        className="flex-1 px-1 pb-1"
      />
    </div>
  );
}
