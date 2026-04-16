import type {
  InteractionEvent,
  RenderConfig,
  RenderContext,
  RenderState,
  SceneCategory,
  SimAction
} from "./types.js";

/**
 * DomainRenderer 定义领域渲染器统一接口。
 * domain 既可以是平台内置领域，也可以是教师扩展出的自定义领域编码。
 */
export interface DomainRenderer {
  readonly domain: SceneCategory;
  readonly supportedAlgorithms: readonly string[];
  getRenderConfig(algorithmType: string): RenderConfig;
  render(state: RenderState, context: RenderContext): void;
  handleInteraction(event: InteractionEvent, state: RenderState): SimAction | null;
  interpolate(previous: RenderState, next: RenderState, progress: number): RenderState;
}
