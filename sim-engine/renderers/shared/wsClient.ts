import type { SimAction, TimeControlCommand, WebSocketMessage } from "./types.js";

/**
 * WSClient 封装 SimEngine 数据通道的连接与消息收发。
 */
export class WSClient {
  private socket: WebSocket | undefined;
  private readonly listeners = new Set<(message: WebSocketMessage) => void>();
  private reconnectTimer: number | undefined;
  private reconnectAttempts = 0;
  private closedManually = false;

  /**
   * constructor 初始化数据通道客户端。
   */
  public constructor(private readonly url: string) {}

  /**
   * connect 建立数据通道连接，并在断开后自动重连。
   */
  public connect(): void {
    this.closedManually = false;
    this.cancelReconnect();
    this.socket?.close();
    this.socket = new WebSocket(this.url);
    this.socket.addEventListener("message", (event) => {
      const message = JSON.parse(String(event.data)) as WebSocketMessage;
      for (const listener of this.listeners) {
        listener(message);
      }
    });
    this.socket.addEventListener("open", () => {
      this.reconnectAttempts = 0;
    });
    this.socket.addEventListener("close", () => {
      this.socket = undefined;
      if (!this.closedManually) {
        this.scheduleReconnect();
      }
    });
  }

  /**
   * disconnect 主动关闭数据通道连接。
   */
  public disconnect(): void {
    this.closedManually = true;
    this.cancelReconnect();
    this.socket?.close();
    this.socket = undefined;
  }

  /**
   * subscribe 订阅来自 Core 的消息。
   */
  public subscribe(listener: (message: WebSocketMessage) => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  /**
   * sendAction 发送场景交互操作。
   */
  public sendAction(action: SimAction): void {
    this.send({
      type: "action",
      tick: 0,
      timestamp: Date.now(),
      scene_code: action.sceneCode,
      payload: {
        action_code: action.actionCode,
        actor_id: action.actorId ?? "",
        role_key: action.roleKey ?? "",
        params: action.params
      }
    });
  }

  /**
   * sendControl 发送时间控制命令。
   */
  public sendControl(command: TimeControlCommand, value?: number): void {
    this.send({
      type: "control",
      tick: 0,
      timestamp: Date.now(),
      payload: value === undefined ? { command } : { command, value }
    });
  }

  /**
   * sendRewind 发送定点回退指令。
   */
  public sendRewind(targetTick: number): void {
    this.send({
      type: "rewind_to",
      tick: 0,
      timestamp: Date.now(),
      payload: { target_tick: targetTick }
    });
  }

  /**
   * send 执行底层消息发送。
   */
  private send(message: WebSocketMessage): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      throw new Error("SimEngine 数据通道尚未连接");
    }
    this.socket.send(JSON.stringify(message));
  }

  /**
   * scheduleReconnect 按指数退避安排下一次重连，最大间隔 30 秒。
   */
  private scheduleReconnect(): void {
    const delay = Math.min(30000, 1500 * Math.pow(2, this.reconnectAttempts));
    this.reconnectAttempts += 1;
    this.reconnectTimer = window.setTimeout(() => this.connect(), delay);
  }

  /**
   * cancelReconnect 清理尚未执行的重连定时器。
   */
  private cancelReconnect(): void {
    if (this.reconnectTimer !== undefined) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
  }
}
