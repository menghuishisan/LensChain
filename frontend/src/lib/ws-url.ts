// ws-url.ts
// 统一构造 WebSocket 连接 URL：读取 NEXT_PUBLIC_WS_BASE_URL，拼接路径与鉴权 Token。
// 所有实时连接 hook（CTF / 实验 / 通知 / SimEngine / 终端）必须通过此工具构造 URL，
// 不得各自从 API 地址推导 WS 协议，避免 WS 独立域名 / 反代场景下连接错误。

import { getStoredAuthSession } from "@/lib/auth-session";

/**
 * WS URL 查询参数值类型。
 */
export type WSQueryValue = string | number | boolean | null | undefined;

/**
 * WS URL 查询参数集合。
 */
export type WSQueryParams = Record<string, WSQueryValue | readonly WSQueryValue[]>;

/**
 * buildWebSocketURL 构造完整 WebSocket 连接 URL。
 * - 基址来自 NEXT_PUBLIC_WS_BASE_URL 环境变量，必须显式配置
 * - 自动注入 Authorization Token（从 localStorage 读取）到 query 的 token 参数
 * - 自动按基址协议转换为 ws:// 或 wss://
 */
export function buildWebSocketURL(path: string, query: WSQueryParams = {}): string {
  const rawBase = process.env.NEXT_PUBLIC_WS_BASE_URL ?? "";
  if (rawBase.length === 0) {
    throw new Error("NEXT_PUBLIC_WS_BASE_URL 未配置，无法建立 WebSocket 连接");
  }

  const baseURL = rawBase.replace(/\/$/, "");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const rawURL = `${baseURL}${normalizedPath}`;
  const fallbackOrigin = typeof window === "undefined" ? "http://localhost" : window.location.origin;
  const url = new URL(rawURL, fallbackOrigin);

  const session = getStoredAuthSession();
  if (session.accessToken !== null) {
    url.searchParams.set("token", session.accessToken);
  }

  Object.entries(query).forEach(([key, value]) => {
    if (value === null || value === undefined) {
      return;
    }
    if (Array.isArray(value)) {
      value.forEach((item) => {
        if (item !== null && item !== undefined) {
          url.searchParams.append(key, String(item));
        }
      });
      return;
    }
    url.searchParams.set(key, String(value));
  });

  if (url.protocol === "https:") {
    url.protocol = "wss:";
  } else if (url.protocol === "http:") {
    url.protocol = "ws:";
  }

  return url.toString();
}
