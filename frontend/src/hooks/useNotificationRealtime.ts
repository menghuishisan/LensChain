"use client";

// useNotificationRealtime.ts
// 模块07通知 WebSocket hook：未读数、最新消息、公告提醒，含断线重连和初始同步。

import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { buildWebSocketURL } from "@/lib/ws-url";
import type { NotificationRealtimeMessage } from "@/types/notification";
import type { RealtimeStatus } from "@/types/experiment";

const PING_INTERVAL_MS = 30_000;
const RECONNECT_DELAY_MS = 1_500;

/** 通知实时连接返回值。 */
export interface UseNotificationRealtimeResult {
  status: RealtimeStatus;
  messages: NotificationRealtimeMessage[];
  latestMessage: NotificationRealtimeMessage | null;
  error: string | null;
  reconnect: () => void;
}

function parseNotificationMessage(raw: string) {
  try {
    return JSON.parse(raw) as NotificationRealtimeMessage;
  } catch {
    return { type: "new_notification", data: { raw } } as NotificationRealtimeMessage;
  }
}

/**
 * useNotificationRealtime 建立通知 WebSocket 连接，页面和铃铛组件不得直接创建 WebSocket。
 */
export function useNotificationRealtime(enabled = true): UseNotificationRealtimeResult {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<RealtimeStatus>("idle");
  const [messages, setMessages] = useState<NotificationRealtimeMessage[]>([]);
  const [error, setError] = useState<string | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const pingTimerRef = useRef<number | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);

  const clearTimers = useCallback(() => {
    if (pingTimerRef.current !== null) {
      window.clearInterval(pingTimerRef.current);
      pingTimerRef.current = null;
    }
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
  }, []);

  // 设计参见 useExperimentRealtime：身份校验取代 manual-close 标志位，避免 StrictMode 下竞态。
  const closeSocket = useCallback(() => {
    clearTimers();
    const socket = socketRef.current;
    socketRef.current = null;
    socket?.close();
  }, [clearTimers]);

  const connect = useCallback(() => {
    if (!enabled || typeof window === "undefined") {
      setStatus("idle");
      return;
    }
    closeSocket();
    setStatus("connecting");
    setError(null);
    const socket = new WebSocket(buildWebSocketURL("/ws/notifications"));
    socketRef.current = socket;

    socket.onopen = () => {
      if (socketRef.current !== socket) return;
      setStatus("open");
      pingTimerRef.current = window.setInterval(() => socket.send(JSON.stringify({ type: "ping" })), PING_INTERVAL_MS);
    };

    socket.onmessage = (event) => {
      if (socketRef.current !== socket) return;
      const message = parseNotificationMessage(String(event.data));
      setMessages((current) => [...current.slice(-49), message]);
      if (message.type === "unread_count_update" || message.type === "new_notification") {
        void queryClient.invalidateQueries({ queryKey: ["notification", "unread-count"] });
        void queryClient.invalidateQueries({ queryKey: ["notification", "inbox"] });
      }
      if (message.type === "new_announcement") {
        void queryClient.invalidateQueries({ queryKey: ["notification", "announcements"] });
      }
    };

    socket.onerror = () => {
      if (socketRef.current !== socket) return;
      setStatus("error");
      setError("通知实时连接发生错误");
    };

    socket.onclose = () => {
      if (socketRef.current !== socket) return;
      clearTimers();
      socketRef.current = null;
      if (!enabled) {
        setStatus("closed");
        return;
      }
      setStatus("reconnecting");
      reconnectTimerRef.current = window.setTimeout(connect, RECONNECT_DELAY_MS);
    };
  }, [clearTimers, closeSocket, enabled, queryClient]);

  useEffect(() => {
    connect();
    return closeSocket;
  }, [closeSocket, connect]);

  return {
    status,
    messages,
    latestMessage: messages.at(-1) ?? null,
    error,
    reconnect: connect,
  };
}
