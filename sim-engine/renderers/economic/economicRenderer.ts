import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * ECONOMIC_ALGORITHMS 定义经济模型领域支持的算法类型。
 */
export const ECONOMIC_ALGORITHMS = [
  "token-economics",
  "pos-staking",
  "governance-voting",
  "defi-liquidity",
  "gas-market"
] as const;

/**
 * EconomicRenderer 实现经济模型领域渲染器。
 */
export class EconomicRenderer extends BaseCanvasRenderer {
  public readonly domain = "economic" as const;
  public readonly supportedAlgorithms = ECONOMIC_ALGORITHMS;

  /**
   * getRenderConfig 返回经济模型领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "经济模型",
      subtitle: "供应量、质押权重与治理流程",
      theme: {
        background: "#101712",
        foreground: "#f4fff6",
        accent: "#74d680",
        success: "#74d680",
        warning: "#ffd166",
        danger: "#ff7b7b",
        muted: "#bad1bc",
        grid: "rgba(186, 209, 188, 0.16)"
      }
    };
  }

  /**
   * render 绘制权重分布与奖励时间线。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "质押、供应量与治理奖惩");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const actors = asArray<unknown>(asObject(state.renderData).actors).map((item, index) =>
      asString(asObject(item).label, `参与者${index + 1}`)
    );
    switch (state.algorithmType) {
      case "token-economics":
        this.renderTokenEconomics(ctx, state, context);
        break;
      case "pos-staking":
        this.renderPoSStaking(ctx, state, actors);
        break;
      case "governance-voting":
        this.renderGovernance(ctx, state);
        break;
      case "defi-liquidity":
        this.renderLiquidity(ctx, state, context);
        break;
      case "gas-market":
        this.renderGasMarket(ctx, state);
        break;
      default:
        this.renderActorBars(ctx, actors);
        break;
    }
    this.drawListPanel(
      ctx,
      "经济摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "参与者", value: String(actors.length) },
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
   * renderActorBars 绘制通用参与者柱状图。
   */
  private renderActorBars(ctx: CanvasRenderingContext2D, actors: string[]): void {
    actors.forEach((actor, index) => {
      const x = 52 + index * 110;
      const height = 40 + index * 14;
      ctx.fillStyle = "#74d680";
      ctx.fillRect(x, 320 - height, 54, height);
      ctx.fillStyle = "#f4fff6";
      ctx.font = "11px sans-serif";
      ctx.fillText(actor, x - 4, 338);
    });
  }

  /**
   * renderTokenEconomics 绘制供应量曲线和分配比例。
   */
  private renderTokenEconomics(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const supply = asArray<unknown>(asObject(state.renderData).supply_curve);
    ctx.strokeStyle = "#74d680";
    ctx.beginPath();
    supply.slice(0, 12).forEach((item, index) => {
      const x = 60 + index * 36;
      const y = 330 - asNumber(item, 20 + index * 4);
      if (index === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    });
    ctx.stroke();
    this.drawComparisonPanel(ctx, 54, 178, 420, 100, "通胀", "分配", [
      { label: "模型", value: asString(asObject(state.renderData).inflation, "线性释放") },
      { label: "供应量", value: asString(asObject(state.renderData).supply, "100M") }
    ], [
      { label: "团队", value: asString(asObject(state.renderData).team_share, "20%") },
      { label: "社区", value: asString(asObject(state.renderData).community_share, "80%") }
    ]);
    this.renderOverlaySVG(context, `<path d="M60 330 C160 260 260 280 440 210" fill="none" stroke="rgba(116,214,128,0.35)" stroke-width="8" />`);
  }

  /**
   * renderPoSStaking 绘制质押量、收益率和 Slash 事件。
   */
  private renderPoSStaking(ctx: CanvasRenderingContext2D, state: RenderState, actors: string[]): void {
    this.renderActorBars(ctx, actors);
    this.drawProcessStages(ctx, ["质押", "验证", "奖励/Slash"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 52, 382, 320);
    this.drawListPanel(ctx, "质押收益", [
      { label: "APR", value: asString(asObject(state.renderData).apr, "5.2%") },
      { label: "Slash", value: asString(asObject(state.renderData).slash, "0") },
      { label: "验证者", value: String(actors.length) }
    ], 382, 188, 220, 100);
  }

  /**
   * renderGovernance 绘制治理提案、投票权重和法定人数。
   */
  private renderGovernance(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const votes = asArray<unknown>(asObject(state.renderData).votes);
    votes.slice(0, 4).forEach((item, index) => {
      const vote = asObject(item);
      const x = 62;
      const y = 186 + index * 38;
      const weight = asNumber(vote.weight, 20 + index * 10);
      ctx.fillStyle = index % 2 === 0 ? "#74d680" : "#ffd166";
      ctx.fillRect(x, y, weight * 4, 22);
      ctx.fillStyle = "#101712";
      ctx.font = "11px sans-serif";
      ctx.fillText(`${asString(vote.label, `选项${index + 1}`)} ${weight}%`, x + 8, y + 15);
    });
    this.drawProcessStages(ctx, ["提案", "投票", "法定人数", "执行"], Math.min(3, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 62, 360, 350);
  }

  /**
   * renderLiquidity 绘制 AMM 曲线和滑点变化。
   */
  private renderLiquidity(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    ctx.strokeStyle = "#74d680";
    ctx.beginPath();
    for (let index = 1; index <= 12; index += 1) {
      const x = 70 + index * 28;
      const y = 330 - 900 / (index + 4);
      if (index === 1) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    }
    ctx.stroke();
    this.drawComparisonPanel(ctx, 384, 188, 240, 116, "交易前", "交易后", [
      { label: "x*y", value: asString(asObject(state.renderData).before_k, "k") },
      { label: "价格", value: asString(asObject(state.renderData).before_price, "1.0") }
    ], [
      { label: "滑点", value: asString(asObject(state.renderData).slippage, "0.4%") },
      { label: "无常损失", value: asString(asObject(state.renderData).impermanent_loss, "0.1%") }
    ]);
    this.renderOverlaySVG(context, `<circle cx="236" cy="262" r="8" fill="rgba(255,209,102,0.8)" />`);
  }

  /**
   * renderGasMarket 绘制 EIP-1559 基础费和区块利用率。
   */
  private renderGasMarket(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const fees = asArray<unknown>(asObject(state.renderData).base_fees);
    fees.slice(0, 12).forEach((item, index) => {
      const x = 56 + index * 34;
      const value = asNumber(item, 20 + index * 2);
      ctx.fillStyle = value > 40 ? "#ff7b7b" : "#74d680";
      ctx.fillRect(x, 330 - value * 3, 22, value * 3);
    });
    this.drawProcessStages(ctx, ["区块利用率", "基础费调整", "燃烧统计"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 58, 376, 320);
    this.drawListPanel(ctx, "费用市场", [
      { label: "燃烧量", value: asString(asObject(state.renderData).burned, "0 ETH") },
      { label: "小费", value: asString(asObject(state.renderData).tips, "0 ETH") },
      { label: "利用率", value: asString(asObject(state.renderData).utilization, "50%") }
    ], 402, 188, 220, 100);
  }
}
