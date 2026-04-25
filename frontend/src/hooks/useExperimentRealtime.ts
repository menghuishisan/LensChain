"use client";

// useExperimentRealtime.ts
// 模块04 WebSocket hook：实例状态、组内消息、教师监控、终端和 SimEngine 实时通道。

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { buildWebSocketURL } from "@/lib/ws-url";
import type { ID, QueryParams } from "@/types/api";
import type {
  ExperimentGroupWSMessageType,
  ExperimentInstanceWSMessageType,
  ExperimentMonitorWSMessageType,
  ExperimentTerminalWSMessageType,
  JsonObject,
  RealtimeStatus,
  SimEngineMessage,
} from "@/types/experiment";

const MAX_REALTIME_MESSAGES = 200;
const RECONNECT_DELAY_MS = 1800;

/** 实时消息泛型封装。 */
export interface RealtimeMessage<TType extends string = string> {
  type: TType;
  data?: JsonObject;
  content?: string;
  container?: string;
  command?: string;
  stdout?: string;
  stderr?: string;
  exit_code?: number;
  timestamp?: string | number;
}

/** 实时连接配置。 */
export interface UseExperimentRealtimeOptions {
  enabled?: boolean;
  query?: QueryParams;
  reconnect?: boolean;
}

/** 实时连接返回值。 */
export interface UseExperimentRealtimeResult<TMessage> {
  status: RealtimeStatus;
  messages: TMessage[];
  latestMessage: TMessage | null;
  error: string | null;
  sendJson: (message: JsonObject) => boolean;
  reconnect: () => void;
  clearMessages: () => void;
}

function parseRealtimeMessage<TMessage>(data: unknown): TMessage {
  if (typeof data !== "string") {
    return { type: "raw", data: { value: String(data) } } as TMessage;
  }

  try {
    return JSON.parse(data) as TMessage;
  } catch {
    return { type: "raw", data: { value: data } } as TMessage;
  }
}

/**
 * useExperimentRealtime 建立统一 WebSocket 连接，页面和业务组件不得直接创建 WebSocket。
 */
export function useExperimentRealtime<TMessage extends RealtimeMessage>(path: string, options: UseExperimentRealtimeOptions = {}): UseExperimentRealtimeResult<TMessage> {
  const { enabled = true, query = {}, reconnect = true } = options;
  const [status, setStatus] = useState<RealtimeStatus>("idle");
  const [messages, setMessages] = useState<TMessage[]>([]);
  const [error, setError] = useState<string | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectCounterRef = useRef(0);
  const isManualCloseRef = useRef(false);
  // WebSocket 连接不应因调用方传入的新对象字面量而反复重连，这里按 query 内容稳定依赖。
  const queryKey = JSON.stringify(query);
  const stableQuery = useMemo(() => JSON.parse(queryKey) as QueryParams, [queryKey]);

  const closeSocket = useCallback(() => {
    isManualCloseRef.current = true;
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    socketRef.current?.close();
    socketRef.current = null;
  }, []);

  const connect = useCallback(() => {
    if (!enabled || typeof window === "undefined" || path.length === 0) {
      setStatus("idle");
      return;
    }

    closeSocket();
    setError(null);
    setStatus(reconnectCounterRef.current > 0 ? "reconnecting" : "connecting");

    isManualCloseRef.current = false;
    const socket = new WebSocket(buildWebSocketURL(path, stableQuery));
    socketRef.current = socket;

    socket.onopen = () => {
      reconnectCounterRef.current = 0;
      setStatus("open");
    };

    socket.onmessage = (event) => {
      const message = parseRealtimeMessage<TMessage>(event.data);
      setMessages((current) => [...current.slice(-(MAX_REALTIME_MESSAGES - 1)), message]);
    };

    socket.onerror = () => {
      setError("实时连接发生错误");
      setStatus("error");
    };

    socket.onclose = () => {
      socketRef.current = null;
      if (isManualCloseRef.current) {
        isManualCloseRef.current = false;
        setStatus("closed");
        return;
      }
      if (!enabled) {
        setStatus("closed");
        return;
      }
      if (!reconnect) {
        setStatus("closed");
        return;
      }

      reconnectCounterRef.current += 1;
      setStatus("reconnecting");
      reconnectTimerRef.current = window.setTimeout(connect, RECONNECT_DELAY_MS);
    };
  }, [closeSocket, enabled, path, reconnect, stableQuery]);

  useEffect(() => {
    connect();
    return closeSocket;
  }, [closeSocket, connect]);

  const sendJson = useCallback((message: JsonObject) => {
    const socket = socketRef.current;
    if (socket === null || socket.readyState !== WebSocket.OPEN) {
      return false;
    }
    socket.send(JSON.stringify(message));
    return true;
  }, []);

  return {
    status,
    messages,
    latestMessage: messages.at(-1) ?? null,
    error,
    sendJson,
    reconnect: connect,
    clearMessages: () => setMessages([]),
  };
}

/**
 * useExperimentInstanceRealtime 连接实例状态 WebSocket。
 */
export function useExperimentInstanceRealtime(instanceID: ID, enabled = true) {
  return useExperimentRealtime<RealtimeMessage<ExperimentInstanceWSMessageType>>(`/ws/experiment-instances/${instanceID}`, { enabled: enabled && instanceID.length > 0 });
}

/**
 * useExperimentGroupChatRealtime 连接组内实时消息 WebSocket。
 */
export function useExperimentGroupChatRealtime(groupID: ID, enabled = true) {
  const realtime = useExperimentRealtime<RealtimeMessage<ExperimentGroupWSMessageType>>(`/ws/experiment-groups/${groupID}/chat`, { enabled: enabled && groupID.length > 0 });
  return {
    ...realtime,
    sendMessage: (content: string) => realtime.sendJson({ type: "chat_message", content }),
  };
}

/**
 * useCourseExperimentMonitorRealtime 连接教师监控实时推送 WebSocket。
 */
export function useCourseExperimentMonitorRealtime(courseID: ID, templateID?: ID, enabled = true) {
  return useExperimentRealtime<RealtimeMessage<ExperimentMonitorWSMessageType>>(`/ws/courses/${courseID}/experiment-monitor`, {
    enabled: enabled && courseID.length > 0,
    query: templateID ? { template_id: templateID } : {},
  });
}

/**
 * useExperimentTerminal 连接学生可执行终端 WebSocket。
 */
export function useExperimentTerminal(instanceID: ID, container?: string, enabled = true) {
  const realtime = useExperimentRealtime<RealtimeMessage<ExperimentTerminalWSMessageType>>(`/experiment-instances/${instanceID}/terminal`, {
    enabled: enabled && instanceID.length > 0,
    query: container ? { container } : {},
  });
  return {
    ...realtime,
    sendCommand: (command: string) => realtime.sendJson({ type: "terminal_command", container: container ?? "", command }),
  };
}

/**
 * useTeacherTerminalStream 连接教师只读终端流 WebSocket。
 */
export function useTeacherTerminalStream(instanceID: ID, enabled = true) {
  return useExperimentRealtime<RealtimeMessage<ExperimentTerminalWSMessageType>>(`/experiment-instances/${instanceID}/terminal-stream`, { enabled: enabled && instanceID.length > 0 });
}

/**
 * useGroupMemberTerminalStream 连接组员只读终端流 WebSocket。
 */
export function useGroupMemberTerminalStream(groupID: ID, studentID: ID, enabled = true) {
  return useExperimentRealtime<RealtimeMessage<ExperimentTerminalWSMessageType>>(`/experiment-groups/${groupID}/members/${studentID}/terminal-stream`, { enabled: enabled && groupID.length > 0 && studentID.length > 0 });
}

/**
 * useSimEngineRealtime 连接 SimEngine 会话 WebSocket。
 */
export function useSimEngineRealtime(sessionID: ID, enabled = true) {
  const realtime = useExperimentRealtime<SimEngineMessage & RealtimeMessage>(`/ws/sim-engine/${sessionID}`, { enabled: enabled && sessionID.length > 0 });
  return {
    ...realtime,
    sendControl: (sceneCode: string, command: string, value?: number) => realtime.sendJson({ type: "control", scene_code: sceneCode, timestamp: Date.now(), payload: value === undefined ? { command } : { command, value } }),
    sendAction: (sceneCode: string, actionCode: string, params: JsonObject) => realtime.sendJson({ type: "action", scene_code: sceneCode, timestamp: Date.now(), payload: { action_code: actionCode, params } }),
    rewindTo: (sceneCode: string, targetTick: number) => realtime.sendJson({ type: "rewind_to", scene_code: sceneCode, timestamp: Date.now(), payload: { target_tick: targetTick } }),
  };
}
