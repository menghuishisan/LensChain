import type {
  InteractionEvent,
  RenderConfig,
  RenderContext,
  RenderState,
  SceneCategory,
  SimAction
} from "./types.js";

/**
 * DomainRenderer 描述领域渲染器的公开接口。
 * 渲染完全由 primitives 驱动，领域渲染器仅提供主题配色和可选的 drawer 覆盖。
 */
export interface DomainRenderer {
  readonly domain: SceneCategory;
  getTheme(): RenderConfig;
  render(state: RenderState, context: RenderContext): void;
  handleInteraction(event: InteractionEvent, state: RenderState): SimAction | null;
  interpolate(previous: RenderState, next: RenderState, progress: number): RenderState;
}
