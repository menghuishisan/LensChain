// ws-url.ts
// 统一构造 WebSocket 连接 URL：读取 NEXT_PUBLIC_WS_BASE_URL，拼接路径与鉴权 Token。
// 所有实时连接 hook（CTF / 实验 / 通知 / SimEngine / 终端）必须通过此工具构造 URL，
// 不得各自从 API 地址推导 WS 协议，避免 WS 独立域名 / 反代场景下连接错误。
//
// **必须 async**：URL 里嵌的 access_token 是个有 30 分钟有效期的短令牌，每次 WS 拨号 /
// 重连前都要先走 `apiClient.ensureFreshAccessToken()` 走双 token 无感刷新拿当前未过期
// 的 token。这是和 HTTP 客户端共用同一套刷新单例，并发安全。早期版本是同步函数直接读
// localStorage，导致 access_token 过期后 WS 死循环 401。

import { apiClient } from "@/lib/api-client";

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
 *
 * - 基址来自 NEXT_PUBLIC_WS_BASE_URL 环境变量，必须显式配置
 * - 先 await `apiClient.ensureFreshAccessToken()`：如 access_token 距过期 < 5 分钟，
 *   走双 token 无感刷新拿新 token；与 HTTP 客户端共用同一 refresh 单例，并发去重
 * - 自动按基址协议转换为 ws:// 或 wss://
 *
 * 返回为空字符串表示未登录或 refresh 失败，调用方应跳过 `new WebSocket(url)` 并进入
 * 重连退避，避免空 URL 抛错。
 */
export async function buildWebSocketURL(path: string, query: WSQueryParams = {}): Promise<string> {
  const rawBase = process.env.NEXT_PUBLIC_WS_BASE_URL ?? "";
  if (rawBase.length === 0) {
    throw new Error("NEXT_PUBLIC_WS_BASE_URL 未配置，无法建立 WebSocket 连接");
  }

  const accessToken = await apiClient.ensureFreshAccessToken();
  if (accessToken === null) {
    return "";
  }

  const baseURL = rawBase.replace(/\/$/, "");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  const rawURL = `${baseURL}${normalizedPath}`;
  const fallbackOrigin = typeof window === "undefined" ? "http://localhost" : window.location.origin;
  const url = new URL(rawURL, fallbackOrigin);

  url.searchParams.set("token", accessToken);

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
