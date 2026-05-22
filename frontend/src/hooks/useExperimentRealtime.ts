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
  SimTeacherMonitorWSType,
  SimWSMessage,
} from "@/types/experiment";

const MAX_REALTIME_MESSAGES = 200;
const RECONNECT_DELAY_MS = 1800;

/** 实时消息泛型封装。 */
export interface RealtimeMessage<TType extends string = string> {
  type: TType;
  data?: JsonObject;
  content?: string;
  container?: string;
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
  sendRaw: (data: string) => boolean;
  reconnect: () => void;
  clearMessages: () => void;
}

// parseRealtimeMessage 将 WS 的 onmessage event.data 解析为业务消息。
//
// 协议契约（前后端唯一约定）：
//   - 二进制帧（ArrayBuffer）= K8s exec PTY 的 stdout 字节流，裹为
//     {type:"binary",data:{bytes:Uint8Array}}，下游 xterm.js 的 write(Uint8Array)
//     使用其内置有状态 UTF-8 解码器跨调用缓存半字符，中文/emoji 等多字节序列
//     即使被 WS 帧切断也能正确还原，且 ANSI 控制字符按字节透传不被损坏。
//   - 文本帧（string）= 后端 json.Marshal 出的业务 JSON，直接 JSON.parse。
//
// socket.binaryType="arraybuffer" 是走二进制分支的前提（默认 "blob" 是异步 API，
// 不能在同步路径里使用）。
function parseRealtimeMessage<TMessage>(data: ArrayBuffer | string): TMessage {
  if (data instanceof ArrayBuffer) {
    return { type: "binary", data: { bytes: new Uint8Array(data) } } as TMessage;
  }
  return JSON.parse(data) as TMessage;
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
  // connectGenerationRef：每次进入 connect() 自增，async URL 构造完成后比对，避免
  // 拨号期间被 closeSocket / 新一轮 connect 抢占后仍开出僵尸 socket。
  const connectGenerationRef = useRef(0);
  // WebSocket 连接不应因调用方传入的新对象字面量而反复重连，这里按 query 内容稳定依赖。
  const queryKey = JSON.stringify(query);
  const stableQuery = useMemo(() => JSON.parse(queryKey) as QueryParams, [queryKey]);

  // closeSocket 关闭当前 socket 并清理重连定时器。
  // 设计要点：每个 socket 的事件处理器通过闭包捕获自身引用，在 onclose 等回调里用
  // `socketRef.current !== capturedSocket` 判断"我是否已被替换"——单一数据源（socketRef）
  // 即活跃 socket，其余皆死。无需任何"manual-close"标志位，从根上消除 React 18 StrictMode
  // 双调用 / connect() 重连路径下共享标志位的竞态。
  const closeSocket = useCallback(() => {
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    // 失效任何 in-flight 的 buildWebSocketURL：拨号回来时 generation 已变，会被丢弃。
    connectGenerationRef.current += 1;
    const socket = socketRef.current;
    socketRef.current = null;
    socket?.close();
  }, []);

  const connect = useCallback(async () => {
    if (!enabled || typeof window === "undefined" || path.length === 0) {
      setStatus("idle");
      return;
    }

    closeSocket();
    setError(null);
    setStatus(reconnectCounterRef.current > 0 ? "reconnecting" : "connecting");

    // 拿 URL 是异步的（要先刷新 access_token），期间组件可能已卸载或重新调 connect。
    // 取一次"自家身份"，拨号完成后比对：不一致就说明已被新 connect / closeSocket 抢占，
    // 必须放弃本次拨号，否则会泄漏 socket、产生重复连接。
    const generation = ++connectGenerationRef.current;
    let url: string;
    try {
      url = await buildWebSocketURL(path, stableQuery);
    } catch (err) {
      if (connectGenerationRef.current !== generation) return;
      setError(err instanceof Error ? err.message : "WebSocket URL 构造失败");
      setStatus("error");
      return;
    }
    if (connectGenerationRef.current !== generation) return;
    if (url.length === 0) {
      // 未登录 / refresh 失败：进入"等待"，外层登录后会重置组件触发新一轮 connect。
      setStatus("closed");
      return;
    }

    const socket = new WebSocket(url);
    // PTY stdout 等二进制帧需以字节形式透传到消费者。binaryType 默认 "blob"
    // 在 onmessage 里拿到的是 Blob 对象，String(blob)==="[object Blob]"、blob.arrayBuffer()
    // 是异步的，都不适合在同步路径里使用。统一改 arraybuffer 后可与 parseRealtimeMessage
    // 里的 ArrayBuffer 分支直接衔接。文本帧不受影响（仍以 string 投递）。
    socket.binaryType = "arraybuffer";
    socketRef.current = socket;

    // 所有事件回调先做身份校验：若 socketRef 已被 closeSocket / 新一轮 connect 覆盖，
    // 说明本 socket 已被弃用，不应再触达任何状态变更或自动重连，直接 return。
    socket.onopen = () => {
      if (socketRef.current !== socket) return;
      reconnectCounterRef.current = 0;
      setStatus("open");
    };

    socket.onmessage = (event) => {
      if (socketRef.current !== socket) return;
      const message = parseRealtimeMessage<TMessage>(event.data);
      setMessages((current) => [...current.slice(-(MAX_REALTIME_MESSAGES - 1)), message]);
    };

    socket.onerror = () => {
      if (socketRef.current !== socket) return;
      setError("实时连接发生错误");
      setStatus("error");
    };

    socket.onclose = () => {
      if (socketRef.current !== socket) return;
      socketRef.current = null;
      if (!enabled || !reconnect) {
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

  const sendRaw = useCallback((data: string) => {
    const socket = socketRef.current;
    if (socket === null || socket.readyState !== WebSocket.OPEN) {
      return false;
    }
    socket.send(data);
    return true;
  }, []);

  return {
    status,
    messages,
    latestMessage: messages.at(-1) ?? null,
    error,
    sendJson,
    sendRaw,
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
 * useExperimentTerminal 连接学生 PTY 终端 WebSocket。
 * 后端走 K8s exec subresource 在选中容器内拉起真 PTY。
 */
export function useExperimentTerminal(instanceID: ID, container?: string, enabled = true) {
  const realtime = useExperimentRealtime<RealtimeMessage<ExperimentTerminalWSMessageType>>(`/experiment-instances/${instanceID}/terminal`, {
    enabled: enabled && instanceID.length > 0,
    query: container ? { container } : {},
  });
  return {
    ...realtime,
    /** 发送原始键击数据到 PTY。 */
    sendInput: (data: string) => realtime.sendRaw(data),
    /** 发送终端尺寸变更到后端（进而下发 SIGWINCH 到容器内 PTY）。 */
    sendResize: (cols: number, rows: number) => realtime.sendJson({ type: "resize", cols, rows }),
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
 * useSimEngineRealtime 连接 SimEngine 会话 WebSocket（API v3.1 §4.1）。
 */
export function useSimEngineRealtime(sessionID: ID, enabled = true) {
  const realtime = useExperimentRealtime<SimWSMessage & RealtimeMessage>(`/ws/sim-engine/${sessionID}`, { enabled: enabled && sessionID.length > 0 });
  return {
    ...realtime,
    sendControl: (sceneCode: string, command: string, value?: number) => realtime.sendJson({ type: "control", scene_code: sceneCode, timestamp: Date.now(), payload: value === undefined ? { command } : { command, value } }),
    sendAction: (sceneCode: string, actionCode: string, params: JsonObject) => realtime.sendJson({ type: "action", scene_code: sceneCode, timestamp: Date.now(), payload: { action_code: actionCode, params } }),
    stepBack: (sceneCode: string) => realtime.sendJson({ type: "step_back", scene_code: sceneCode, timestamp: Date.now(), payload: {} }),
  };
}

/**
 * useTeacherSimMonitor 连接教师 SimEngine 监控 WS（06.2 §9.8）。
 */
export function useTeacherSimMonitor(experimentID: ID, enabled = true) {
  return useExperimentRealtime<RealtimeMessage<SimTeacherMonitorWSType>>(`/ws/teacher-monitor/${experimentID}`, { enabled: enabled && experimentID.length > 0 });
}
