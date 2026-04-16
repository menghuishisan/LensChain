import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * CRYPTOGRAPHY_ALGORITHMS 定义密码学领域支持的算法类型。
 */
export const CRYPTOGRAPHY_ALGORITHMS = [
  "sha256-hash",
  "keccak256-hash",
  "ecdsa-sign",
  "rsa-encrypt",
  "merkle-tree",
  "zkp-basic"
] as const;

/**
 * CryptographyRenderer 实现密码学运算领域渲染器。
 */
export class CryptographyRenderer extends BaseCanvasRenderer {
  public readonly domain = "cryptography" as const;
  public readonly supportedAlgorithms = CRYPTOGRAPHY_ALGORITHMS;

  /**
   * getRenderConfig 返回密码学领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "密码学运算",
      subtitle: "数据转换、密钥关系与证明流程",
      auxiliaryDomains: algorithmType === "merkle-tree" ? ["data_structure"] : [],
      theme: {
        background: "#130f1c",
        foreground: "#f7f0ff",
        accent: "#4dd5ff",
        success: "#52d6a4",
        warning: "#ffc857",
        danger: "#ff7a90",
        muted: "#b8a7d9",
        grid: "rgba(184, 167, 217, 0.16)"
      }
    };
  }

  /**
   * render 绘制密码学数据流与步骤摘要。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "哈希、签名与证明过程");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const steps = asArray<unknown>(asObject(state.renderData).steps).map((item) => asString(item, "步骤"));
    switch (state.algorithmType) {
      case "sha256-hash":
      case "keccak256-hash":
        this.renderHashPipeline(ctx, state, steps, context);
        break;
      case "ecdsa-sign":
      case "rsa-encrypt":
        this.renderKeyPairFlow(ctx, state);
        break;
      case "merkle-tree":
        this.renderMerkleTree(ctx, state, context);
        break;
      case "zkp-basic":
        this.renderZKP(ctx, state, steps);
        break;
      default:
        this.renderStepFlow(ctx, steps);
        break;
    }
    this.drawListPanel(
      ctx,
      "运算摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "步骤数", value: String(steps.length) },
        { label: "辅助领域", value: state.algorithmType === "merkle-tree" ? "data_structure" : "无" }
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
   * renderStepFlow 绘制通用步骤流。
   */
  private renderStepFlow(ctx: CanvasRenderingContext2D, steps: string[]): void {
    steps.forEach((step, index) => {
      const x = 40 + index * 120;
      ctx.fillStyle = "rgba(28, 21, 40, 0.92)";
      ctx.strokeStyle = "#4dd5ff";
      ctx.beginPath();
      ctx.roundRect(x, 220, 92, 72, 16);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#f7f0ff";
      ctx.font = "12px sans-serif";
      ctx.fillText(step, x + 12, 260);
      if (index < steps.length - 1) {
        ctx.strokeStyle = "#4dd5ff";
        ctx.beginPath();
        ctx.moveTo(x + 92, 256);
        ctx.lineTo(x + 120, 256);
        ctx.stroke();
      }
    });
  }

  /**
   * renderHashPipeline 绘制哈希压缩与雪崩对比。
   */
  private renderHashPipeline(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    steps: string[],
    context: RenderContext
  ): void {
    this.renderStepFlow(ctx, steps.length > 0 ? steps : ["分块", "填充", "轮函数", "输出"]);
    this.drawComparisonPanel(
      ctx,
      36,
      318,
      420,
      110,
      "原始输入",
      "扰动后输出",
      [
        { label: "消息", value: asString(asObject(state.renderData).input, "abc") },
        { label: "轮次", value: String(asNumber(asObject(state.renderData).round, 64)) }
      ],
      [
        { label: "哈希", value: asString(asObject(state.renderData).hash, "0x...") },
        { label: "差异位", value: String(asNumber(asObject(state.renderData).changed_bits, 32)) }
      ]
    );
    this.renderOverlaySVG(
      context,
      `<rect x="486" y="204" width="132" height="132" rx="16" fill="rgba(77,213,255,0.08)" stroke="rgba(77,213,255,0.45)" />
       <text x="520" y="272" fill="#f7f0ff" font-size="16">状态矩阵</text>`
    );
  }

  /**
   * renderKeyPairFlow 绘制密钥对、签名和验签/解密关系。
   */
  private renderKeyPairFlow(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const leftTitle = state.algorithmType === "ecdsa-sign" ? "签名生成" : "加密过程";
    const rightTitle = state.algorithmType === "ecdsa-sign" ? "验签结果" : "解密结果";
    this.drawComparisonPanel(
      ctx,
      40,
      184,
      520,
      154,
      leftTitle,
      rightTitle,
      [
        { label: "公钥", value: asString(asObject(state.renderData).public_key, "pub-key") },
        { label: "私钥", value: asString(asObject(state.renderData).private_key, "priv-key") },
        { label: "消息", value: asString(asObject(state.renderData).message, "payload") }
      ],
      [
        { label: "签名/密文", value: asString(asObject(state.renderData).signature, "sig") },
        { label: "验证", value: asString(asObject(state.renderData).result, "通过") },
        { label: "椭圆点/模幂", value: asString(asObject(state.renderData).operation, "计算中") }
      ]
    );
  }

  /**
   * renderMerkleTree 绘制 Merkle 树路径与验证节点。
   */
  private renderMerkleTree(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const levels = asArray<unknown>(asObject(state.renderData).levels);
    levels.slice(0, 4).forEach((level, levelIndex) => {
      const values = asArray<unknown>(level);
      values.slice(0, 4).forEach((item, itemIndex) => {
        const x = 54 + itemIndex * 120 + levelIndex * 24;
        const y = 340 - levelIndex * 68;
        ctx.fillStyle = levelIndex === 0 ? "#4dd5ff" : "#7b65ff";
        ctx.beginPath();
        ctx.roundRect(x, y, 88, 30, 12);
        ctx.fill();
        ctx.fillStyle = "#f7f0ff";
        ctx.font = "11px sans-serif";
        ctx.fillText(asString(item, "hash"), x + 8, y + 19);
      });
    });
    this.drawProcessStages(ctx, ["叶子哈希", "逐层合并", "根验证"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 54, 168, 260);
    this.renderOverlaySVG(
      context,
      `<path d="M98 322 L158 270 L220 214" fill="none" stroke="rgba(255,200,87,0.65)" stroke-width="4" />`
    );
  }

  /**
   * renderZKP 绘制证明者与验证者三步交互过程。
   */
  private renderZKP(ctx: CanvasRenderingContext2D, state: RenderState, steps: string[]): void {
    this.renderStepFlow(ctx, steps.length > 0 ? steps : ["承诺", "挑战", "响应"]);
    this.drawComparisonPanel(
      ctx,
      36,
      326,
      420,
      96,
      "证明者",
      "验证者",
      [
        { label: "秘密", value: asString(asObject(state.renderData).secret, "隐藏") },
        { label: "承诺", value: asString(asObject(state.renderData).commitment, "已提交") }
      ],
      [
        { label: "挑战", value: asString(asObject(state.renderData).challenge, "随机挑战") },
        { label: "结论", value: asString(asObject(state.renderData).result, "验证通过") }
      ]
    );
  }
}
