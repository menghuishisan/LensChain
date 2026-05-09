import type { SimAction, TimeControlCommand, WebSocketMessage } from "./types.js";

/**
 * WSClient 封装 SimEngine 数据通道的连接与消息收发。
 *
 * 设计要点（与 frontend/src/hooks/useExperimentRealtime.ts 同构）：
 * 每个 socket 的事件监听器通过闭包捕获自身引用（localSocket），所有回调首行做
 * `this.socket !== localSocket` 的身份校验。`this.socket` 即活跃 socket，其余皆死。
 * 不再使用 closedManually 共享标志位，从根上消除 React StrictMode 双调用 / 任意
 * connect() 重连路径下的状态竞态：旧 socket 的异步 close 事件无法误触发重连去杀掉
 * 已经成功建立的新 socket（之前症状："WebSocket is closed before the connection is
 * established"无限循环，DevTools WS 列表看似为空）。
 */
export class WSClient {
  private socket: WebSocket | undefined;
  private readonly listeners = new Set<(message: WebSocketMessage) => void>();
  private readonly statusListeners = new Set<(connected: boolean) => void>();
  private reconnectTimer: number | undefined;
  private reconnectAttempts = 0;

  /**
   * constructor 初始化数据通道客户端。
   */
  public constructor(private readonly url: string) {}

  /**
   * connect 建立数据通道连接，并在断开后自动重连。
   */
  public connect(): void {
    this.cancelReconnect();
    const previous = this.socket;
    this.socket = undefined;
    previous?.close();

    const localSocket = new WebSocket(this.url);
    this.socket = localSocket;

    localSocket.addEventListener("message", (event) => {
      if (this.socket !== localSocket) return;
      const message = JSON.parse(String(event.data)) as WebSocketMessage;
      for (const listener of this.listeners) {
        listener(message);
      }
    });
    localSocket.addEventListener("open", () => {
      if (this.socket !== localSocket) return;
      this.reconnectAttempts = 0;
      this.notifyStatus(true);
    });
    localSocket.addEventListener("close", () => {
      if (this.socket !== localSocket) return;
      this.socket = undefined;
      this.notifyStatus(false);
      this.scheduleReconnect();
    });
  }

  /**
   * disconnect 主动关闭数据通道连接。
   * 通过先清空 this.socket 再关闭，使被弃 socket 的异步 close 事件因身份校验失败而失效，
   * 不会触发自动重连。
   */
  public disconnect(): void {
    this.cancelReconnect();
    const previous = this.socket;
    this.socket = undefined;
    previous?.close();
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
   * subscribeStatus 订阅连接状态变化。
   */
  public subscribeStatus(listener: (connected: boolean) => void): () => void {
    this.statusListeners.add(listener);
    return () => {
      this.statusListeners.delete(listener);
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

  /**
   * notifyStatus 广播连接状态变化。
   */
  private notifyStatus(connected: boolean): void {
    for (const listener of this.statusListeners) {
      listener(connected);
    }
  }
}
