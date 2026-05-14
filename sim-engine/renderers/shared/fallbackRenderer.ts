import { PrimitiveBasedRenderer } from "./primitiveRenderer.js";
import type { RenderConfig, RenderContext, RenderState, SceneCategory } from "./types.js";

/**
 * FallbackRenderer 用于渲染器加载失败或领域未匹配时的错误展示。
 */
export class FallbackRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory;

  public constructor(
    domain: SceneCategory,
    private readonly reason = "当前场景渲染器不可用"
  ) {
    super();
    this.domain = domain;
  }

  public getTheme(): RenderConfig {
    return {
      title: "可视化加载失败",
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

  public override render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title);
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
    ctx.fillText(`category: ${state.category}`, 96, 280);
  }
}
