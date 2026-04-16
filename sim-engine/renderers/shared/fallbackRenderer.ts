import { BaseCanvasRenderer } from "./baseCanvasRenderer.js";
import type { RenderConfig, RenderContext, RenderState, SceneCategory } from "./types.js";

/**
 * FallbackRenderer 用于渲染器加载失败或领域未匹配时的错误展示。
 */
export class FallbackRenderer extends BaseCanvasRenderer {
  public readonly domain: SceneCategory;
  public readonly supportedAlgorithms = ["fallback-renderer"] as const;

  /**
   * constructor 记录故障场景所属领域和错误信息。
   */
  public constructor(
    domain: SceneCategory,
    private readonly reason = "当前场景渲染器不可用"
  ) {
    super();
    this.domain = domain;
  }

  /**
   * getRenderConfig 返回故障展示渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "可视化加载失败",
      subtitle: this.reason,
      theme: {
        background: "#160f10",
        foreground: "#fff2f4",
        accent: "#ff7a90",
        success: "#52d6a4",
        warning: "#ffcf5a",
        danger: "#ff5d73",
        muted: "#d9b0b8",
        grid: "rgba(217, 176, 184, 0.16)"
      }
    };
  }

  /**
   * render 绘制加载失败提示。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "显示场景加载失败提示，不影响其他场景");
    ctx.fillStyle = "rgba(55, 22, 28, 0.96)";
    ctx.strokeStyle = "#ff5d73";
    ctx.beginPath();
    ctx.roundRect(60, 120, context.width - 120, 180, 20);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = "#fff2f4";
    ctx.font = "600 22px sans-serif";
    ctx.fillText("当前场景渲染失败", 96, 184);
    ctx.font = "14px sans-serif";
    ctx.fillText(this.reason, 96, 220);
    ctx.fillText(`scene_code: ${state.sceneCode}`, 96, 252);
    ctx.fillText(`algorithm_type: ${state.algorithmType}`, 96, 280);
    ctx.fillText(`category: ${state.category}`, 96, 308);
  }
}
