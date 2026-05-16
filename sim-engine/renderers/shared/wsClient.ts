/**
 * wsClient.ts — SimEngine WebSocket 客户端。
 *
 * 职责：
 *   • 维护单条 WS 连接；urlProvider 在每次 (重)连之前调用以确保 token 新鲜。
 *   • 收到消息 → JSON.parse → 派发到 onMessage。
 *   • send() 序列化 WebSocketMessage 发送。
 *   • 断线自动重连（指数退避，上限 30s）。
 *
 * 不写兑底：解析失败 / 未知 type → 通过 onError 上抛，不静默丢弃。
 */

import type { WebSocketMessage } from "./types.js";

export interface WSClientOptions {
  /** 每次 (重)连前调用，返回完整 ws:// URL（含 token）。 */
  urlProvider: () => string | Promise<string>;
  onOpen?: () => void;
  onMessage: (msg: WebSocketMessage) => void;
  onClose?: (code: number, reason: string) => void;
  onError: (err: Error) => void;
  /** 最大重连次数；0 = 不重连。默认无限。 */
  maxReconnects?: number;
}

export class WSClient {
  private socket: WebSocket | null = null;
  private opts: WSClientOptions;
  private reconnectCount = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closedByUser = false;

  constructor(opts: WSClientOptions) {
    this.opts = opts;
  }

  async connect(): Promise<void> {
    this.closedByUser = false;
    const url = await this.opts.urlProvider();
    if (!url || typeof url !== "string") {
      throw new Error("WSClient.connect: urlProvider 必须返回非空 URL 字符串");
    }
    const ws = new WebSocket(url);
    this.socket = ws;
    ws.addEventListener("open", () => {
      this.reconnectCount = 0;
      this.opts.onOpen?.();
    });
    ws.addEventListener("message", (ev) => {
      let parsed: unknown;
      try {
        parsed = JSON.parse(typeof ev.data === "string" ? ev.data : "");
      } catch (e) {
        this.opts.onError(new Error(`WSClient: JSON 解析失败 ${(e as Error).message}`));
        return;
      }
      if (!isWebSocketMessage(parsed)) {
        this.opts.onError(new Error(`WSClient: 消息格式不符合 WebSocketMessage`));
        return;
      }
      this.opts.onMessage(parsed);
    });
    ws.addEventListener("close", (ev) => {
      this.socket = null;
      this.opts.onClose?.(ev.code, ev.reason);
      if (!this.closedByUser) this.scheduleReconnect();
    });
    ws.addEventListener("error", () => {
      // 浏览器 WS error 事件不带具体信息；关闭事件会随后到来。
      this.opts.onError(new Error("WSClient: WebSocket 错误（详见 close 事件）"));
    });
  }

  send(msg: WebSocketMessage): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      throw new Error("WSClient.send: 连接未就绪");
    }
    this.socket.send(JSON.stringify(msg));
  }

  close(): void {
    this.closedByUser = true;
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null; }
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }

  isConnected(): boolean {
    return this.socket?.readyState === WebSocket.OPEN;
  }

  private scheduleReconnect(): void {
    const max = this.opts.maxReconnects ?? Number.POSITIVE_INFINITY;
    if (this.reconnectCount >= max) return;
    const delay = Math.min(30_000, 500 * 2 ** Math.min(6, this.reconnectCount));
    this.reconnectCount++;
    this.reconnectTimer = setTimeout(() => {
      this.connect().catch(err => this.opts.onError(err as Error));
    }, delay);
  }
}

function isWebSocketMessage(v: unknown): v is WebSocketMessage {
  if (!v || typeof v !== "object") return false;
  const o = v as Record<string, unknown>;
  return typeof o.type === "string"
    && typeof o.tick === "number"
    && typeof o.timestamp === "number"
    && typeof o.payload === "object" && o.payload !== null;
}
