// experimentToolProxy.ts
// 模块04 — 实验工具反代 cookie 签发服务。
//
// iframe 加载 code-server / blockscout / VNC / monitor 等工具页面前必须先调本端点拿
// HttpOnly cookie：cookie 路径作用域绑定 /instance/<id>/<kind>/，浏览器自动在 iframe
// 主资源与所有子资源（CSS/JS/WS upgrade）请求中携带，无需 URL token，避免 token 进
// referer / 浏览器历史 / 后端 access log。
//
// 关键点：
//   1) 该端点的响应 Set-Cookie 必须被浏览器接收，因此使用 fetch credentials: "include"，
//      不能走默认 ApiClient（默认 same-origin，跨端口 / 跨子域时浏览器会丢弃 cookie）。
//   2) 必须在 iframe 渲染之前 await 完成，否则 iframe 第一次请求没有 cookie → 401。
//   3) cookie 默认 30 分钟过期，前端用 useToolProxyCookie 每 25 分钟刷新一次。

import type { ID } from "@/types/api";
import { getStoredAuthSession } from "@/lib/auth-session";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

/** 反代 cookie 端点响应。 */
export interface ToolProxyCookie {
  /** 形如 /instance/<id>/<kind>/ 的反代访问路径，前端拼当前 origin 或外部域名后用作 iframe src。 */
  proxy_path: string;
  /** cookie 有效期（秒），前端据此决定刷新节奏。 */
  expires_in: number;
}

/**
 * 调 POST /api/v1/experiment-instances/:id/tools/:kind/proxy-cookie 签发反代 cookie。
 *
 * 失败抛错；成功返回 proxy_path + expires_in，cookie 已被浏览器存入。
 */
export async function ensureToolProxyCookie(instanceID: ID, toolKind: string): Promise<ToolProxyCookie> {
  if (!instanceID || !toolKind) {
    throw new Error("ensureToolProxyCookie 缺少 instanceID / toolKind");
  }
  const session = getStoredAuthSession();
  if (!session.accessToken) {
    throw new Error("未登录，无法签发反代凭证");
  }
  const url = `${API_BASE_URL.replace(/\/$/, "")}/experiment-instances/${instanceID}/tools/${encodeURIComponent(toolKind)}/proxy-cookie`;
  const res = await fetch(url, {
    method: "POST",
    headers: { Authorization: `${session.tokenType} ${session.accessToken}` },
    credentials: "include",
  });
  if (!res.ok) {
    let msg = `proxy-cookie HTTP ${res.status}`;
    try {
      const body = (await res.json()) as { message?: string };
      if (body?.message) {
        msg = body.message;
      }
    } catch {
      // ignore
    }
    throw new Error(msg);
  }
  const json = (await res.json()) as { code: number; message: string; data: ToolProxyCookie };
  if (json.code !== 200 || !json.data) {
    throw new Error(json.message || "反代凭证签发失败");
  }
  return json.data;
}
