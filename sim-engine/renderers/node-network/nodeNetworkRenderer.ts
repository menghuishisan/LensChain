import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * NODE_NETWORK_ALGORITHMS 定义节点与网络领域支持的算法类型。
 */
export const NODE_NETWORK_ALGORITHMS = [
  "p2p-discovery",
  "gossip-propagation",
  "network-partition",
  "tx-broadcast",
  "block-sync",
  "node-load-balance"
] as const;

/**
 * NodeNetworkRenderer 实现节点与网络领域渲染器。
 */
export class NodeNetworkRenderer extends BaseCanvasRenderer {
  public readonly domain = "node_network" as const;
  public readonly supportedAlgorithms = NODE_NETWORK_ALGORITHMS;

  /**
   * getRenderConfig 返回节点与网络领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "节点与网络",
      subtitle: "节点拓扑、消息传播与网络状态",
      theme: {
        background: "#09111f",
        foreground: "#e5f2ff",
        accent: "#2fb3ff",
        success: "#25c48a",
        warning: "#ffb020",
        danger: "#ff5d73",
        muted: "#8fa3bf",
        grid: "rgba(143, 163, 191, 0.18)"
      }
    };
  }

  /**
   * render 绘制节点、连线和传播状态。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "节点拓扑图与消息传播");
    this.clearOverlaySVG(context);
    const nodes = this.nodePoints(state);
    const totalNodes = asArray<unknown>(asObject(state.renderData).nodes).length;
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    if (state.linked && state.linkGroupName) {
      this.drawBadge(ctx, `联动组：${state.linkGroupName}`);
    }
    this.drawAggregationNotice(ctx, "节点", totalNodes, nodes.length, context.width - 252, 206);
    switch (state.algorithmType) {
      case "p2p-discovery":
        this.renderP2PDiscovery(ctx, state, nodes, context);
        break;
      case "gossip-propagation":
        this.renderGossipPropagation(ctx, state, nodes, context);
        break;
      case "network-partition":
        this.renderNetworkPartition(ctx, state, nodes, context);
        break;
      case "tx-broadcast":
        this.renderBroadcastFlow(ctx, state, nodes, context, "交易广播路径");
        break;
      case "block-sync":
        this.renderBlockSync(ctx, state, nodes, context);
        break;
      case "node-load-balance":
        this.renderLoadBalance(ctx, state, nodes, context);
        break;
      default:
        this.renderTopology(ctx, nodes, context, false);
        break;
    }
    this.drawListPanel(
      ctx,
      "网络摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "节点数", value: String(nodes.length) },
        { label: "变化项", value: String(state.changedKeys?.length ?? 0) }
      ],
      context.width - 240,
      72,
      220,
      122
    );
    this.drawChangedKeys(ctx, state.changedKeys ?? [], context.width);
    this.drawAnnotations(ctx, state);
    this.drawTooltip(ctx, state.tooltip ?? [], 24, context.height - 150);
    if ((state.timeline?.length ?? 0) > 0) {
      this.drawTimeline(ctx, context.width, context.height, state.timeline ?? []);
    }
  }

  /**
   * renderTopology 绘制通用网络拓扑。
   */
  private renderTopology(
    ctx: CanvasRenderingContext2D,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext,
    emphasizeFlow: boolean
  ): void {
    ctx.strokeStyle = emphasizeFlow ? "rgba(255, 176, 32, 0.55)" : "rgba(47, 179, 255, 0.45)";
    ctx.lineWidth = 2;
    for (let index = 0; index < nodes.length - 1; index += 1) {
      const current = nodes[index];
      const next = nodes[index + 1];
      if (!current || !next) {
        continue;
      }
      ctx.beginPath();
      ctx.moveTo(current.x, current.y);
      ctx.lineTo(next.x, next.y);
      ctx.stroke();
      if (emphasizeFlow) {
        const pulseX = current.x + (next.x - current.x) * 0.55;
        const pulseY = current.y + (next.y - current.y) * 0.55;
        ctx.fillStyle = "#ffb020";
        ctx.beginPath();
        ctx.arc(pulseX, pulseY, 5, 0, Math.PI * 2);
        ctx.fill();
      }
    }
    nodes.forEach((node) => {
      ctx.fillStyle = node.status === "partitioned" ? "#ffb020" : node.status === "fault" ? "#ff5d73" : "#2fb3ff";
      ctx.beginPath();
      ctx.arc(node.x, node.y, 24, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = "#e5f2ff";
      ctx.font = "12px sans-serif";
      ctx.fillText(node.label, node.x - 18, node.y + 42);
    });
    this.renderOverlaySVG(
      context,
      nodes
        .map((node) => `<circle cx="${node.x}" cy="${node.y}" r="34" fill="none" stroke="rgba(47,179,255,0.18)" stroke-width="2" />`)
        .join("")
    );
  }

  /**
   * renderP2PDiscovery 绘制节点发现与路由探索过程。
   */
  private renderP2PDiscovery(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext
  ): void {
    this.renderTopology(ctx, nodes, context, true);
    const routes = asArray<unknown>(asObject(state.renderData).routing_table)
      .slice(0, 4)
      .map((item, index) => ({
        label: asString(asObject(item).source, `节点${index + 1}`),
        value: asString(asObject(item).target, "最近邻")
      }));
    this.drawListPanel(ctx, "路由表", routes, 36, 318, 220, 110);
  }

  /**
   * renderGossipPropagation 绘制消息扩散波纹与覆盖率。
   */
  private renderGossipPropagation(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext
  ): void {
    this.renderTopology(ctx, nodes, context, true);
    const progress = asNumber(asObject(state.renderData).coverage, 0);
    nodes.forEach((node, index) => {
      const radius = 30 + ((index + state.tick) % 3) * 8;
      ctx.strokeStyle = "rgba(47, 179, 255, 0.25)";
      ctx.beginPath();
      ctx.arc(node.x, node.y, radius, 0, Math.PI * 2);
      ctx.stroke();
    });
    this.drawProcessStages(ctx, ["源节点", "邻居扩散", "全网覆盖"], Math.min(2, Math.floor(progress * 3)), progress, 44, 182, 280);
  }

  /**
   * renderNetworkPartition 绘制网络分区边界与恢复过程。
   */
  private renderNetworkPartition(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext
  ): void {
    this.renderTopology(ctx, nodes, context, false);
    const partitionX = asNumber(asObject(state.renderData).partition_x, context.width / 2);
    ctx.fillStyle = "rgba(255, 93, 115, 0.12)";
    ctx.fillRect(partitionX, 150, 20, 220);
    ctx.fillStyle = "#ff5d73";
    ctx.font = "600 12px sans-serif";
    ctx.fillText("分区边界", partitionX - 8, 142);
    this.drawProcessStages(ctx, ["正常", "分区隔离", "恢复同步"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 48, 392, 320);
  }

  /**
   * renderBroadcastFlow 绘制交易或消息广播路径。
   */
  private renderBroadcastFlow(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext,
    title: string
  ): void {
    this.renderTopology(ctx, nodes, context, true);
    const messages = asArray<unknown>(asObject(state.renderData).messages);
    this.drawAggregationNotice(ctx, "消息", messages.length, BaseCanvasRenderer.MAX_VISIBLE_FLOW_ITEMS, 36, 392);
    messages.slice(0, BaseCanvasRenderer.MAX_VISIBLE_FLOW_ITEMS).forEach((item, index) => {
      const message = asObject(item);
      const x = asNumber(message.x, 80 + (index % 6) * 64);
      const y = asNumber(message.y, 170 + Math.floor(index / 6) * 22);
      ctx.fillStyle = "#ffd166";
      ctx.beginPath();
      ctx.arc(x, y, 4, 0, Math.PI * 2);
      ctx.fill();
    });
    this.drawListPanel(ctx, title, [
      { label: "广播条目", value: String(messages.length) },
      { label: "活跃节点", value: String(nodes.length) },
      { label: "当前 Tick", value: String(state.tick) }
    ], 36, 318, 220, 96);
  }

  /**
   * renderBlockSync 绘制区块同步与链高度对比。
   */
  private renderBlockSync(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext
  ): void {
    this.renderTopology(ctx, nodes, context, true);
    const heights = asArray<unknown>(asObject(state.renderData).heights);
    heights.slice(0, 6).forEach((item, index) => {
      const value = asNumber(item, index + 1);
      const x = 36 + index * 52;
      const h = value * 10;
      ctx.fillStyle = "#25c48a";
      ctx.fillRect(x, 370 - h, 26, h);
      ctx.fillStyle = "#e5f2ff";
      ctx.font = "10px sans-serif";
      ctx.fillText(`H${value}`, x - 2, 386);
    });
    this.drawProcessStages(ctx, ["出块", "传播", "追平"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 356, 392, 220);
  }

  /**
   * renderLoadBalance 绘制节点负载热力图与迁移结果。
   */
  private renderLoadBalance(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<NodeNetworkRenderer["nodePoints"]>,
    context: RenderContext
  ): void {
    this.renderTopology(ctx, nodes, context, false);
    nodes.forEach((node, index) => {
      const load = asNumber(asObject(asArray<unknown>(asObject(state.renderData).loads)[index]).value, (index + 1) * 10);
      ctx.fillStyle = load > 70 ? "#ff5d73" : load > 40 ? "#ffb020" : "#25c48a";
      ctx.fillRect(node.x - 16, node.y + 52, 32, Math.max(8, load * 0.6));
    });
    this.drawListPanel(ctx, "负载迁移", [
      { label: "模式", value: "持续运行式" },
      { label: "节点数", value: String(nodes.length) },
      { label: "迁移批次", value: String(asArray<unknown>(asObject(state.renderData).migrations).length) }
    ], 36, 318, 220, 96);
  }
}
