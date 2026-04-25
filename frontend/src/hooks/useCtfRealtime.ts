"use client";

// useCtfRealtime.ts
// 模块05 CTF WebSocket hook：排行榜、公告、回合、攻击事件实时推送，含心跳、断线重连和快照同步状态。

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { ID } from "@/types/api";
import type { CtfRealtimeMessage } from "@/types/ctf";
import type { RealtimeStatus } from "@/types/experiment";
import { buildWebSocketURL } from "@/lib/ws-url";

const MAX_MESSAGES = 200;
const PING_INTERVAL_MS = 30_000;
const BASE_RECONNECT_DELAY_MS = 1200;
const MAX_RECONNECT_DELAY_MS = 30_000;

/** CTF 实时连接返回值。 */
export interface UseCtfRealtimeResult {
  status: RealtimeStatus;
  messages: CtfRealtimeMessage[];
  latestMessage: CtfRealtimeMessage | null;
  error: string | null;
  hasSnapshotSynced: boolean;
  reconnect: () => void;
  subscribe: (channel: string, params?: Record<string, string>) => boolean;
}

function parseMessage(data: string): CtfRealtimeMessage {
  try {
    return JSON.parse(data) as CtfRealtimeMessage;
  } catch {
    return { type: "message", channel: "announcement", data: { raw: data }, timestamp: new Date().toISOString() };
  }
}

/**
 * useCtfRealtime 建立 CTF WebSocket，页面和业务组件不得直接创建 WebSocket。
 */
export function useCtfRealtime(competitionID: ID, enabled = true): UseCtfRealtimeResult {
  const [status, setStatus] = useState<RealtimeStatus>("idle");
  const [messages, setMessages] = useState<CtfRealtimeMessage[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [hasSnapshotSynced, setHasSnapshotSynced] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const pingTimerRef = useRef<number | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectCountRef = useRef(0);
  const isManualCloseRef = useRef(false);
  const stableCompetitionID = useMemo(() => competitionID, [competitionID]);

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

  const closeSocket = useCallback(() => {
    isManualCloseRef.current = true;
    clearTimers();
    socketRef.current?.close();
    socketRef.current = null;
  }, [clearTimers]);

  const connect = useCallback(() => {
    if (!enabled || stableCompetitionID.length === 0 || typeof window === "undefined") {
      setStatus("idle");
      return;
    }
    closeSocket();
    isManualCloseRef.current = false;
    setStatus(reconnectCountRef.current > 0 ? "reconnecting" : "connecting");
    setError(null);

    const socket = new WebSocket(buildWebSocketURL("/ctf/ws", { competition_id: stableCompetitionID }));
    socketRef.current = socket;

    socket.onopen = () => {
      reconnectCountRef.current = 0;
      setStatus("open");
      setHasSnapshotSynced(false);
      pingTimerRef.current = window.setInterval(() => socket.send(JSON.stringify({ type: "ping" })), PING_INTERVAL_MS);
      socket.send(JSON.stringify({ type: "subscribe", channel: "leaderboard", params: { competition_id: stableCompetitionID } }));
      socket.send(JSON.stringify({ type: "subscribe", channel: "announcement", params: { competition_id: stableCompetitionID } }));
    };

    socket.onmessage = (event) => {
      const message = parseMessage(String(event.data));
      if (message.type === "snapshot" || message.channel === "leaderboard") {
        setHasSnapshotSynced(true);
      }
      setMessages((current) => [...current.slice(-(MAX_MESSAGES - 1)), message]);
    };

    socket.onerror = () => {
      setStatus("error");
      setError("CTF 实时连接发生错误");
    };

    socket.onclose = () => {
      clearTimers();
      socketRef.current = null;
      if (isManualCloseRef.current || !enabled) {
        setStatus("closed");
        isManualCloseRef.current = false;
        return;
      }
      reconnectCountRef.current += 1;
      const delay = Math.min(MAX_RECONNECT_DELAY_MS, BASE_RECONNECT_DELAY_MS * 2 ** reconnectCountRef.current);
      setStatus("reconnecting");
      reconnectTimerRef.current = window.setTimeout(connect, delay);
    };
  }, [clearTimers, closeSocket, enabled, stableCompetitionID]);

  useEffect(() => {
    connect();
    return closeSocket;
  }, [closeSocket, connect]);

  const subscribe = useCallback((channel: string, params: Record<string, string> = {}) => {
    const socket = socketRef.current;
    if (socket === null || socket.readyState !== WebSocket.OPEN) {
      return false;
    }
    socket.send(JSON.stringify({ type: "subscribe", channel, params: { competition_id: stableCompetitionID, ...params } }));
    return true;
  }, [stableCompetitionID]);

  return {
    status,
    messages,
    latestMessage: messages.at(-1) ?? null,
    error,
    hasSnapshotSynced,
    reconnect: connect,
    subscribe,
  };
}
