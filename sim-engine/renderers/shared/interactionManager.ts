import type { DomainRenderer } from "./domainRenderer.js";
import type { RenderState, SimAction } from "./types.js";

/**
 * InteractionManager 将浏览器交互事件转换为标准操作请求。
 */
export class InteractionManager {
  /**
   * bind 将交互监听绑定到画布。
   */
  public bind(
    canvas: HTMLCanvasElement,
    renderer: DomainRenderer,
    stateProvider: () => RenderState,
    onAction: (action: SimAction) => void
  ): () => void {
    let dragging = false;
    let lastX = 0;
    let lastY = 0;

    /**
     * emit 将浏览器事件转换为统一交互事件并回调给外部。
     */
    const emit = (input: {
      type: "click" | "drag" | "hover";
      x: number;
      y: number;
      deltaX?: number;
      deltaY?: number;
    }) => {
      const state = stateProvider();
      const event = {
        sceneCode: state.sceneCode,
        algorithmType: state.algorithmType,
        type: input.type,
        x: input.x,
        y: input.y
      } as const;
      const action = renderer.handleInteraction(
        input.deltaX === undefined && input.deltaY === undefined
          ? event
          : {
              ...event,
              deltaX: input.deltaX ?? 0,
              deltaY: input.deltaY ?? 0
            },
        state
      );
      if (action) {
        onAction(action);
      }
    };

    const clickHandler = (event: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      emit({ type: "click", x: event.clientX - rect.left, y: event.clientY - rect.top });
    };
    const pointerDownHandler = (event: PointerEvent) => {
      const rect = canvas.getBoundingClientRect();
      dragging = true;
      lastX = event.clientX - rect.left;
      lastY = event.clientY - rect.top;
    };
    const pointerMoveHandler = (event: PointerEvent) => {
      const rect = canvas.getBoundingClientRect();
      const nextX = event.clientX - rect.left;
      const nextY = event.clientY - rect.top;
      if (!dragging) {
        emit({ type: "hover", x: nextX, y: nextY });
        return;
      }
      emit({
        type: "drag",
        x: nextX,
        y: nextY,
        deltaX: nextX - lastX,
        deltaY: nextY - lastY
      });
      lastX = nextX;
      lastY = nextY;
    };
    const pointerUpHandler = () => {
      dragging = false;
    };
    canvas.addEventListener("click", clickHandler);
    canvas.addEventListener("pointerdown", pointerDownHandler);
    canvas.addEventListener("pointermove", pointerMoveHandler);
    canvas.addEventListener("pointerup", pointerUpHandler);
    canvas.addEventListener("pointerleave", pointerUpHandler);
    return () => {
      canvas.removeEventListener("click", clickHandler);
      canvas.removeEventListener("pointerdown", pointerDownHandler);
      canvas.removeEventListener("pointermove", pointerMoveHandler);
      canvas.removeEventListener("pointerup", pointerUpHandler);
      canvas.removeEventListener("pointerleave", pointerUpHandler);
    };
  }
}
