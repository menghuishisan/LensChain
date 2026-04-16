import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * TRANSACTION_ALGORITHMS 定义交易生命周期领域支持的算法类型。
 */
export const TRANSACTION_ALGORITHMS = [
  "tx-lifecycle",
  "gas-calculation",
  "token-transfer",
  "tx-ordering-mev",
  "cross-chain-bridge"
] as const;

/**
 * TransactionRenderer 实现交易生命周期领域渲染器。
 */
export class TransactionRenderer extends BaseCanvasRenderer {
  public readonly domain = "transaction" as const;
  public readonly supportedAlgorithms = TRANSACTION_ALGORITHMS;

  /**
   * getRenderConfig 返回交易生命周期领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "交易生命周期",
      subtitle: "创建、广播、入池、打包与确认流程",
      theme: {
        background: "#13130d",
        foreground: "#fff8eb",
        accent: "#ffb84d",
        success: "#52d273",
        warning: "#ffd166",
        danger: "#ff6f61",
        muted: "#c7b691",
        grid: "rgba(199, 182, 145, 0.16)"
      }
    };
  }

  /**
   * render 绘制交易流水线与关键费用指标。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "交易流水线与 Gas 消耗");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const stages = asArray<unknown>(asObject(state.renderData).stages).map((item) => asString(item, "阶段"));
    switch (state.algorithmType) {
      case "tx-lifecycle":
        this.renderLifecycle(ctx, state, stages, context);
        break;
      case "gas-calculation":
        this.renderGasWaterfall(ctx, state);
        break;
      case "token-transfer":
        this.renderTokenTransfer(ctx, state, context);
        break;
      case "tx-ordering-mev":
        this.renderMEV(ctx, state);
        break;
      case "cross-chain-bridge":
        this.renderCrossChain(ctx, state, context);
        break;
      default:
        this.renderPipeline(ctx, stages);
        break;
    }
    this.drawListPanel(
      ctx,
      "交易摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "流程阶段", value: String(stages.length) },
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
   * renderPipeline 绘制通用交易流水线。
   */
  private renderPipeline(ctx: CanvasRenderingContext2D, stages: string[]): void {
    stages.forEach((stage, index) => {
      const x = 38 + index * 120;
      ctx.fillStyle = "rgba(42, 31, 20, 0.94)";
      ctx.strokeStyle = "#ffb84d";
      ctx.beginPath();
      ctx.roundRect(x, 220, 96, 68, 16);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#fff8eb";
      ctx.font = "12px sans-serif";
      ctx.fillText(stage, x + 18, 258);
      if (index < stages.length - 1) {
        ctx.strokeStyle = "#ffb84d";
        ctx.beginPath();
        ctx.moveTo(x + 96, 254);
        ctx.lineTo(x + 120, 254);
        ctx.stroke();
      }
    });
  }

  /**
   * renderLifecycle 绘制交易创建到确认的过程化阶段。
   */
  private renderLifecycle(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    stages: string[],
    context: RenderContext
  ): void {
    this.renderPipeline(ctx, stages);
    if (stages.length > 0) {
      this.drawProcessStages(
        ctx,
        stages,
        Math.min(stages.length - 1, asNumber(asObject(state.renderData).phase_index, 0)),
        asNumber(asObject(state.renderData).progress, 0),
        48,
        338,
        500
      );
    }
    this.renderOverlaySVG(context, `<path d="M76 254 C190 180 330 320 560 254" fill="none" stroke="rgba(255,184,77,0.35)" stroke-width="5" />`);
  }

  /**
   * renderGasWaterfall 绘制 Gas 消耗瀑布图。
   */
  private renderGasWaterfall(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const operations = asArray<unknown>(asObject(state.renderData).operations);
    let baseline = 330;
    operations.slice(0, 8).forEach((item, index) => {
      const op = asObject(item);
      const gas = asNumber(op.gas, 5 + index * 3);
      const height = gas * 4;
      const x = 52 + index * 58;
      ctx.fillStyle = gas > 20 ? "#ff6f61" : "#ffb84d";
      ctx.fillRect(x, baseline - height, 38, height);
      ctx.fillStyle = "#fff8eb";
      ctx.font = "10px sans-serif";
      ctx.fillText(asString(op.label, `op${index}`), x - 2, baseline + 16);
    });
    this.drawComparisonPanel(ctx, 52, 374, 410, 82, "Gas Limit", "Gas Used", [
      { label: "限制", value: String(asNumber(asObject(state.renderData).gas_limit, 21000)) }
    ], [
      { label: "已用", value: String(asNumber(asObject(state.renderData).gas_used, 16800)) }
    ]);
  }

  /**
   * renderTokenTransfer 绘制账户余额变化和 ERC-20 事件。
   */
  private renderTokenTransfer(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const accounts = asArray<unknown>(asObject(state.renderData).accounts);
    accounts.slice(0, 4).forEach((item, index) => {
      const account = asObject(item);
      const x = 58 + index * 132;
      const balance = asNumber(account.balance, 100 - index * 20);
      ctx.fillStyle = "#52d273";
      ctx.beginPath();
      ctx.roundRect(x, 210, 86, 48, 16);
      ctx.fill();
      ctx.fillStyle = "#13130d";
      ctx.font = "12px sans-serif";
      ctx.fillText(asString(account.label, `账户${index + 1}`), x + 14, 232);
      ctx.fillText(String(balance), x + 24, 248);
    });
    this.renderOverlaySVG(context, `<path d="M140 234 L218 234 L218 228 L232 238 L218 248 L218 242 L140 242Z" fill="rgba(255,184,77,0.65)" />`);
    this.drawProcessStages(ctx, ["扣减余额", "事件日志", "接收确认"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 58, 330, 300);
  }

  /**
   * renderMEV 绘制内存池排序和三明治攻击路径。
   */
  private renderMEV(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const txs = asArray<unknown>(asObject(state.renderData).mempool);
    txs.slice(0, 8).forEach((item, index) => {
      const tx = asObject(item);
      const y = 176 + index * 28;
      ctx.fillStyle = asString(tx.kind, "") === "attack" ? "#ff6f61" : "#ffb84d";
      ctx.beginPath();
      ctx.roundRect(54, y, 300, 20, 10);
      ctx.fill();
      ctx.fillStyle = "#13130d";
      ctx.font = "11px sans-serif";
      ctx.fillText(`${index + 1}. ${asString(tx.label, `tx-${index}`)} · ${asString(tx.kind, "normal")}`, 66, y + 14);
    });
    this.drawComparisonPanel(ctx, 382, 184, 260, 128, "用户交易", "矿工排序", [
      { label: "原始滑点", value: asString(asObject(state.renderData).slippage, "0.3%") },
      { label: "风险", value: "低" }
    ], [
      { label: "MEV", value: asString(asObject(state.renderData).mev, "可提取") },
      { label: "攻击", value: "三明治" }
    ]);
  }

  /**
   * renderCrossChain 绘制双链锁定、铸造和中继消息。
   */
  private renderCrossChain(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    this.drawComparisonPanel(ctx, 54, 188, 460, 136, "源链", "目标链", [
      { label: "锁定", value: asString(asObject(state.renderData).locked, "待确认") },
      { label: "确认数", value: String(asNumber(asObject(state.renderData).source_confirmations, 0)) }
    ], [
      { label: "铸造", value: asString(asObject(state.renderData).minted, "等待中") },
      { label: "中继", value: asString(asObject(state.renderData).relay, "传递中") }
    ]);
    this.drawProcessStages(ctx, ["锁定", "证明", "中继", "铸造"], Math.min(3, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 80, 360, 360);
    this.renderOverlaySVG(context, `<path d="M254 252 C310 210 360 210 414 252" fill="none" stroke="rgba(255,184,77,0.6)" stroke-width="4" marker-end="url(#arrow)" />`);
  }
}
