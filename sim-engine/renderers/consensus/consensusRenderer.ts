import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * CONSENSUS_ALGORITHMS 定义共识过程领域支持的算法类型。
 */
export const CONSENSUS_ALGORITHMS = [
  "pow-mining",
  "pos-validator",
  "pbft-consensus",
  "raft-election",
  "dpos-voting"
] as const;

/**
 * ConsensusRenderer 实现共识过程领域渲染器。
 */
export class ConsensusRenderer extends BaseCanvasRenderer {
  public readonly domain = "consensus" as const;
  public readonly supportedAlgorithms = CONSENSUS_ALGORITHMS;

  /**
   * getRenderConfig 返回共识过程领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "共识过程",
      subtitle: "投票轮次、视图切换与领导者状态",
      theme: {
        background: "#0d1222",
        foreground: "#f4f7fb",
        accent: "#63d2ff",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.16)"
      }
    };
  }

  /**
   * render 绘制投票矩阵与轮次摘要。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "多阶段投票时序与节点状态矩阵");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const stages = asArray<unknown>(asObject(state.renderData).stages).map((value) => asString(value, "阶段"));
    const nodes = this.nodePoints(state);
    switch (state.algorithmType) {
      case "pow-mining":
        this.renderPowMining(ctx, state, nodes, context);
        break;
      case "pos-validator":
        this.renderPoSValidator(ctx, state, nodes);
        break;
      case "pbft-consensus":
        this.renderPBFT(ctx, state, nodes, stages);
        break;
      case "raft-election":
        this.renderRaft(ctx, state, nodes);
        break;
      case "dpos-voting":
        this.renderDPoS(ctx, state);
        break;
      default:
        this.renderStageMatrix(ctx, nodes, stages);
        break;
    }
    this.drawListPanel(
      ctx,
      "轮次摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "阶段数", value: String(stages.length) },
        { label: "节点数", value: String(nodes.length) }
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
   * renderStageMatrix 绘制通用阶段矩阵。
   */
  private renderStageMatrix(
    ctx: CanvasRenderingContext2D,
    nodes: ReturnType<ConsensusRenderer["nodePoints"]>,
    stages: string[]
  ): void {
    stages.forEach((stage, stageIndex) => {
      const x = 40 + stageIndex * 120;
      ctx.fillStyle = "rgba(20, 34, 58, 0.94)";
      ctx.strokeStyle = "#63d2ff";
      ctx.beginPath();
      ctx.roundRect(x, 180, 96, 180, 16);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#f4f7fb";
      ctx.font = "600 13px sans-serif";
      ctx.fillText(stage, x + 20, 206);
      nodes.forEach((node, nodeIndex) => {
        const y = 238 + nodeIndex * 28;
        ctx.fillStyle = node.status === "leader" ? "#47c98b" : node.status === "fault" ? "#ff6b6b" : "#93a4bb";
        ctx.fillRect(x + 16, y, 64, 16);
        ctx.fillStyle = "#f4f7fb";
        ctx.font = "11px sans-serif";
        ctx.fillText(node.label, x + 18, y + 12);
      });
    });
  }

  /**
   * renderPowMining 绘制 PoW 挖矿竞争与出块竞赛。
   */
  private renderPowMining(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<ConsensusRenderer["nodePoints"]>,
    context: RenderContext
  ): void {
    const hashRates = asArray<unknown>(asObject(state.renderData).hashrates);
    nodes.slice(0, 6).forEach((node, index) => {
      const rate = asNumber(asObject(hashRates[index]).value, 10 + index * 10);
      const barHeight = Math.min(180, rate * 2);
      const x = 48 + index * 84;
      ctx.fillStyle = node.status === "leader" ? "#47c98b" : "#63d2ff";
      ctx.fillRect(x, 360 - barHeight, 38, barHeight);
      ctx.fillStyle = "#f4f7fb";
      ctx.font = "11px sans-serif";
      ctx.fillText(node.label, x - 4, 382);
      ctx.fillText(`${rate}H`, x - 2, 396);
    });
    this.drawProcessStages(ctx, ["Nonce 搜索", "命中目标", "新区块广播"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.6), 356, 214, 240);
    this.renderOverlaySVG(
      context,
      `<line x1="356" y1="274" x2="596" y2="274" stroke="rgba(255,200,87,0.5)" stroke-width="3" stroke-dasharray="6 6" />`
    );
  }

  /**
   * renderPoSValidator 绘制验证者选举与质押权重。
   */
  private renderPoSValidator(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<ConsensusRenderer["nodePoints"]>
  ): void {
    const totalStake = Math.max(1, asNumber(asObject(state.renderData).total_stake, nodes.length * 10));
    let startAngle = -Math.PI / 2;
    nodes.slice(0, 6).forEach((node, index) => {
      const stake = asNumber(asObject(asArray<unknown>(asObject(state.renderData).stakes)[index]).value, 10);
      const angle = (stake / totalStake) * Math.PI * 2;
      ctx.fillStyle = index % 2 === 0 ? "#63d2ff" : "#47c98b";
      ctx.beginPath();
      ctx.moveTo(180, 270);
      ctx.arc(180, 270, 90, startAngle, startAngle + angle);
      ctx.closePath();
      ctx.fill();
      startAngle += angle;
      ctx.fillStyle = "#f4f7fb";
      ctx.font = "11px sans-serif";
      ctx.fillText(`${node.label}:${stake}`, 44, 168 + index * 18);
    });
    this.drawProcessStages(ctx, ["质押权重", "随机选举", "Epoch 轮转"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.4), 336, 214, 260);
  }

  /**
   * renderPBFT 绘制 PBFT 三阶段消息矩阵与主节点切换。
   */
  private renderPBFT(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<ConsensusRenderer["nodePoints"]>,
    stages: string[]
  ): void {
    this.renderStageMatrix(ctx, nodes, stages.length > 0 ? stages : ["Pre-prepare", "Prepare", "Commit"]);
    this.drawProcessStages(ctx, ["Pre-prepare", "Prepare", "Commit"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 392, 392, 200);
  }

  /**
   * renderRaft 绘制任期、选票和日志复制状态。
   */
  private renderRaft(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    nodes: ReturnType<ConsensusRenderer["nodePoints"]>
  ): void {
    const terms = asArray<unknown>(asObject(state.renderData).terms);
    nodes.slice(0, 5).forEach((node, index) => {
      const y = 190 + index * 42;
      ctx.fillStyle = node.status === "leader" ? "#47c98b" : "#2b3d5d";
      ctx.beginPath();
      ctx.roundRect(44, y, 180, 28, 12);
      ctx.fill();
      ctx.fillStyle = "#f4f7fb";
      ctx.font = "12px sans-serif";
      ctx.fillText(`${node.label} | term ${asNumber(terms[index], index + 1)}`, 58, y + 18);
    });
    this.drawProcessStages(ctx, ["超时", "拉票", "Leader 产生", "日志复制"], Math.min(3, asNumber(asObject(state.renderData).phase_index, 2)), asNumber(asObject(state.renderData).progress, 0.5), 286, 250, 300);
  }

  /**
   * renderDPoS 绘制投票流向与超级节点排名。
   */
  private renderDPoS(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const delegates = asArray<unknown>(asObject(state.renderData).delegates);
    delegates.slice(0, 6).forEach((item, index) => {
      const delegate = asObject(item);
      const x = 56;
      const y = 188 + index * 34;
      const votes = asNumber(delegate.votes, 100 - index * 10);
      ctx.fillStyle = index < 3 ? "#47c98b" : "#63d2ff";
      ctx.fillRect(x, y, Math.max(80, votes * 2), 18);
      ctx.fillStyle = "#f4f7fb";
      ctx.font = "11px sans-serif";
      ctx.fillText(`${asString(delegate.label, `代表${index + 1}`)} · ${votes}`, x + 8, y + 13);
    });
    this.drawProcessStages(ctx, ["委托投票", "排名收敛", "轮次出块"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 342, 214, 250);
  }
}
