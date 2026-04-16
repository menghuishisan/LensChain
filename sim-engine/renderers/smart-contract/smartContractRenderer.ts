import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * SMART_CONTRACT_ALGORITHMS 定义智能合约领域支持的算法类型。
 */
export const SMART_CONTRACT_ALGORITHMS = [
  "contract-state-machine",
  "evm-execution",
  "contract-interaction",
  "contract-deployment",
  "state-channel"
] as const;

/**
 * SmartContractRenderer 实现智能合约领域渲染器。
 */
export class SmartContractRenderer extends BaseCanvasRenderer {
  public readonly domain = "smart_contract" as const;
  public readonly supportedAlgorithms = SMART_CONTRACT_ALGORITHMS;

  /**
   * getRenderConfig 返回智能合约领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "智能合约",
      subtitle: "调用栈、状态槽与执行流程",
      theme: {
        background: "#11101a",
        foreground: "#f2f0ff",
        accent: "#9d7dff",
        success: "#52d6a4",
        warning: "#ffcf5a",
        danger: "#ff6b8b",
        muted: "#b8aed8",
        grid: "rgba(184, 174, 216, 0.16)"
      }
    };
  }

  /**
   * render 绘制调用栈与存储槽摘要。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "调用栈、存储槽与执行步进");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const stack = asArray<unknown>(asObject(state.renderData).call_stack).map((item, index) =>
      asString(item, `frame-${index + 1}`)
    );
    switch (state.algorithmType) {
      case "contract-state-machine":
        this.renderStateMachine(ctx, state, context);
        break;
      case "evm-execution":
        this.renderEVM(ctx, state, stack);
        break;
      case "contract-interaction":
        this.renderContractInteraction(ctx, state, context);
        break;
      case "contract-deployment":
        this.renderDeployment(ctx, state);
        break;
      case "state-channel":
        this.renderStateChannel(ctx, state, context);
        break;
      default:
        this.renderCallStack(ctx, stack);
        break;
    }
    this.drawListPanel(
      ctx,
      "合约摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "调用栈", value: String(stack.length) },
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
   * renderCallStack 绘制通用调用栈。
   */
  private renderCallStack(ctx: CanvasRenderingContext2D, stack: string[]): void {
    stack.forEach((frame, index) => {
      const y = 210 + index * 38;
      ctx.fillStyle = "rgba(31, 27, 49, 0.94)";
      ctx.strokeStyle = "#9d7dff";
      ctx.beginPath();
      ctx.roundRect(48, y, 200, 28, 12);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#f2f0ff";
      ctx.font = "12px sans-serif";
      ctx.fillText(frame, 62, y + 18);
    });
  }

  /**
   * renderStateMachine 绘制合约状态转换图。
   */
  private renderStateMachine(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const states = asArray<unknown>(asObject(state.renderData).states);
    states.slice(0, 5).forEach((item, index) => {
      const label = asString(asObject(item).label, `状态${index + 1}`);
      const x = 58 + index * 108;
      ctx.fillStyle = asString(asObject(item).status, "") === "active" ? "#52d6a4" : "#9d7dff";
      ctx.beginPath();
      ctx.roundRect(x, 226, 82, 42, 18);
      ctx.fill();
      ctx.fillStyle = "#11101a";
      ctx.font = "12px sans-serif";
      ctx.fillText(label, x + 16, 252);
    });
    this.renderOverlaySVG(context, `<path d="M140 246 L496 246" stroke="rgba(157,125,255,0.45)" stroke-width="4" stroke-dasharray="8 8" />`);
    this.drawProcessStages(ctx, ["事件触发", "状态转换", "存储更新"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 72, 338, 330);
  }

  /**
   * renderEVM 绘制 EVM 操作码步进和三面板。
   */
  private renderEVM(ctx: CanvasRenderingContext2D, state: RenderState, stack: string[]): void {
    this.renderCallStack(ctx, stack.length > 0 ? stack : ["PUSH1", "SLOAD", "CALL", "SSTORE"]);
    this.drawComparisonPanel(ctx, 286, 190, 360, 150, "内存/栈", "存储槽", [
      { label: "PC", value: String(asNumber(asObject(state.renderData).pc, 0)) },
      { label: "Opcode", value: asString(asObject(state.renderData).opcode, "PUSH1") },
      { label: "Gas", value: String(asNumber(asObject(state.renderData).gas, 0)) }
    ], [
      { label: "slot0", value: asString(asObject(state.renderData).slot0, "0x00") },
      { label: "slot1", value: asString(asObject(state.renderData).slot1, "0x00") },
      { label: "变更", value: String((state.changedKeys ?? []).length) }
    ]);
    this.drawProcessStages(ctx, ["取指", "执行", "写回"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 54, 382, 240);
  }

  /**
   * renderContractInteraction 绘制合约调用图和上下文切换。
   */
  private renderContractInteraction(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const contracts = asArray<unknown>(asObject(state.renderData).contracts);
    contracts.slice(0, 4).forEach((item, index) => {
      const contract = asObject(item);
      const x = 70 + index * 128;
      ctx.fillStyle = "#9d7dff";
      ctx.beginPath();
      ctx.roundRect(x, 214, 92, 46, 16);
      ctx.fill();
      ctx.fillStyle = "#11101a";
      ctx.font = "12px sans-serif";
      ctx.fillText(asString(contract.label, `合约${index + 1}`), x + 16, 242);
    });
    this.renderOverlaySVG(context, `<path d="M162 238 C220 180 290 296 354 238" fill="none" stroke="rgba(82,214,164,0.6)" stroke-width="4" />`);
    this.drawListPanel(ctx, "调用类型", [
      { label: "call", value: asString(asObject(state.renderData).call, "执行中") },
      { label: "delegatecall", value: asString(asObject(state.renderData).delegatecall, "未触发") },
      { label: "返回值", value: asString(asObject(state.renderData).return_value, "等待") }
    ], 52, 318, 220, 100);
  }

  /**
   * renderDeployment 绘制合约部署流程。
   */
  private renderDeployment(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const stages = ["字节码", "构造函数", "存储初始化", "地址计算"];
    this.drawProcessStages(ctx, stages, Math.min(3, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 66, 224, 420);
    this.drawComparisonPanel(ctx, 66, 282, 420, 118, "部署输入", "链上结果", [
      { label: "sender", value: asString(asObject(state.renderData).sender, "0x...") },
      { label: "nonce", value: String(asNumber(asObject(state.renderData).nonce, 1)) }
    ], [
      { label: "address", value: asString(asObject(state.renderData).address, "0xcontract") },
      { label: "status", value: asString(asObject(state.renderData).status, "已部署") }
    ]);
  }

  /**
   * renderStateChannel 绘制链上开通、链下交易和争议解决。
   */
  private renderStateChannel(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    this.drawComparisonPanel(ctx, 58, 198, 460, 126, "链上状态", "链下序列", [
      { label: "通道", value: asString(asObject(state.renderData).channel, "open") },
      { label: "保证金", value: asString(asObject(state.renderData).deposit, "10 ETH") }
    ], [
      { label: "序号", value: String(asNumber(asObject(state.renderData).sequence, 1)) },
      { label: "签名", value: asString(asObject(state.renderData).signature, "双方已签名") }
    ]);
    this.drawProcessStages(ctx, ["开通", "链下更新", "争议期", "关闭"], Math.min(3, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 76, 360, 380);
    this.renderOverlaySVG(context, `<line x1="180" y1="180" x2="430" y2="180" stroke="rgba(157,125,255,0.45)" stroke-width="5" />`);
  }
}
