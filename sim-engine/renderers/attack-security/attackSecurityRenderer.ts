import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * ATTACK_SECURITY_ALGORITHMS 定义攻击与安全领域支持的算法类型。
 */
export const ATTACK_SECURITY_ALGORITHMS = [
  "51-percent-attack",
  "double-spend",
  "pbft-byzantine",
  "reentrancy-attack",
  "integer-overflow",
  "selfish-mining"
] as const;

/**
 * AttackSecurityRenderer 实现攻击与安全领域渲染器。
 */
export class AttackSecurityRenderer extends BaseCanvasRenderer {
  public readonly domain = "attack_security" as const;
  public readonly supportedAlgorithms = ATTACK_SECURITY_ALGORITHMS;

  /**
   * getRenderConfig 返回攻击与安全领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "攻击与安全",
      subtitle: "攻击路径、防御状态与攻防对比",
      theme: {
        background: "#170f11",
        foreground: "#fff2f4",
        accent: "#ff7a90",
        success: "#52d6a4",
        warning: "#ffcf5a",
        danger: "#ff5d73",
        muted: "#d7aeb7",
        grid: "rgba(215, 174, 183, 0.16)"
      }
    };
  }

  /**
   * render 绘制攻击链路与风险摘要。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "攻击路径、异常节点与防御状态");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const vectors = asArray<unknown>(asObject(state.renderData).vectors).map((item) => asString(item, "向量"));
    switch (state.algorithmType) {
      case "51-percent-attack":
      case "selfish-mining":
        this.renderChainRace(ctx, state, context);
        break;
      case "double-spend":
        this.renderDoubleSpend(ctx, state, context);
        break;
      case "pbft-byzantine":
        this.renderByzantine(ctx, state);
        break;
      case "reentrancy-attack":
        this.renderReentrancy(ctx, state);
        break;
      case "integer-overflow":
        this.renderIntegerOverflow(ctx, state);
        break;
      default:
        this.renderVectors(ctx, vectors);
        break;
    }
    this.drawListPanel(
      ctx,
      "安全摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "攻击向量", value: String(vectors.length) },
        { label: "联动状态", value: state.linked ? "联动中" : "独立" }
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
   * renderVectors 绘制通用攻击向量链。
   */
  private renderVectors(ctx: CanvasRenderingContext2D, vectors: string[]): void {
    vectors.forEach((vector, index) => {
      const x = 48 + index * 116;
      ctx.fillStyle = "rgba(46, 20, 26, 0.94)";
      ctx.strokeStyle = index % 2 === 0 ? "#ff5d73" : "#ffcf5a";
      ctx.beginPath();
      ctx.roundRect(x, 230, 92, 64, 14);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#fff2f4";
      ctx.font = "12px sans-serif";
      ctx.fillText(vector, x + 12, 266);
    });
  }

  /**
   * renderChainRace 绘制诚实链与攻击链双轨对比。
   */
  private renderChainRace(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const honest = asNumber(asObject(state.renderData).honest_blocks, 4);
    const attacker = asNumber(asObject(state.renderData).attacker_blocks, 3);
    this.drawComparisonPanel(ctx, 48, 184, 450, 132, "诚实链", "攻击链", [
      { label: "区块数", value: String(honest) },
      { label: "算力", value: asString(asObject(state.renderData).honest_hashrate, "55%") }
    ], [
      { label: "区块数", value: String(attacker) },
      { label: "算力", value: asString(asObject(state.renderData).attacker_hashrate, "45%") }
    ]);
    this.drawProcessStages(ctx, ["私链积累", "追平", "链重组"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 72, 366, 320);
    this.renderOverlaySVG(context, `<path d="M90 332 L420 332" stroke="rgba(255,93,115,0.55)" stroke-width="5" /><path d="M90 348 L360 348" stroke="rgba(82,214,164,0.55)" stroke-width="5" />`);
  }

  /**
   * renderDoubleSpend 绘制冲突交易路径和确认数竞赛。
   */
  private renderDoubleSpend(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    this.drawComparisonPanel(ctx, 54, 192, 450, 126, "商家交易", "攻击交易", [
      { label: "tx", value: asString(asObject(state.renderData).merchant_tx, "tx-A") },
      { label: "确认", value: String(asNumber(asObject(state.renderData).merchant_confirmations, 1)) }
    ], [
      { label: "tx", value: asString(asObject(state.renderData).attacker_tx, "tx-B") },
      { label: "确认", value: String(asNumber(asObject(state.renderData).attacker_confirmations, 0)) }
    ]);
    this.drawProcessStages(ctx, ["支付商家", "私链挖矿", "公开替换"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 72, 362, 320);
    this.renderOverlaySVG(context, `<path d="M120 260 C220 180 320 340 440 260" fill="none" stroke="rgba(255,207,90,0.55)" stroke-width="4" />`);
  }

  /**
   * renderByzantine 绘制异常节点、投票阈值和消息偏差。
   */
  private renderByzantine(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const nodes = this.nodePoints(state);
    nodes.slice(0, 8).forEach((node, index) => {
      const x = 60 + (index % 4) * 100;
      const y = 190 + Math.floor(index / 4) * 82;
      ctx.fillStyle = node.status === "fault" || node.status === "byzantine" ? "#ff5d73" : "#52d6a4";
      ctx.beginPath();
      ctx.arc(x, y, 22, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = "#fff2f4";
      ctx.font = "11px sans-serif";
      ctx.fillText(node.label, x - 18, y + 40);
    });
    this.drawProcessStages(ctx, ["异常注入", "消息偏差", "视图切换"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 70, 384, 300);
  }

  /**
   * renderReentrancy 绘制递归调用栈和余额变化。
   */
  private renderReentrancy(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const stack = asArray<unknown>(asObject(state.renderData).call_stack);
    stack.slice(0, 6).forEach((item, index) => {
      const y = 178 + index * 34;
      ctx.fillStyle = index % 2 === 0 ? "#ff5d73" : "#ffcf5a";
      ctx.beginPath();
      ctx.roundRect(58 + index * 18, y, 240, 24, 12);
      ctx.fill();
      ctx.fillStyle = "#170f11";
      ctx.font = "11px sans-serif";
      ctx.fillText(asString(item, `withdraw#${index + 1}`), 72 + index * 18, y + 16);
    });
    this.drawComparisonPanel(ctx, 350, 190, 280, 126, "攻击前", "攻击后", [
      { label: "合约余额", value: asString(asObject(state.renderData).before_balance, "10 ETH") }
    ], [
      { label: "合约余额", value: asString(asObject(state.renderData).after_balance, "0 ETH") }
    ]);
  }

  /**
   * renderIntegerOverflow 绘制数值时间线和 SafeMath 对比。
   */
  private renderIntegerOverflow(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const values = asArray<unknown>(asObject(state.renderData).values);
    values.slice(0, 8).forEach((item, index) => {
      const x = 58 + index * 52;
      const value = asNumber(item, index * 32);
      ctx.fillStyle = value > 240 ? "#ff5d73" : "#ffcf5a";
      ctx.fillRect(x, 330 - value * 0.45, 26, value * 0.45);
      ctx.fillStyle = "#fff2f4";
      ctx.font = "10px sans-serif";
      ctx.fillText(String(value), x - 2, 350);
    });
    this.drawComparisonPanel(ctx, 52, 376, 420, 82, "未保护计算", "SafeMath", [
      { label: "结果", value: asString(asObject(state.renderData).unsafe_result, "回绕") }
    ], [
      { label: "结果", value: asString(asObject(state.renderData).safe_result, "拒绝") }
    ]);
  }
}
