import type { Annotation } from "./types.js";
import { randomID } from "./utils.js";

/**
 * AnnotationStore 管理用户在画布上的文本标注。
 */
export class AnnotationStore {
  private annotations: Annotation[] = [];

  /**
   * list 返回全部标注。
   */
  public list(): Annotation[] {
    return [...this.annotations];
  }

  /**
   * add 新增一个文本标注。
   */
  public add(text: string, x: number, y: number, color = "#ffb020"): Annotation {
    const annotation: Annotation = {
      id: randomID("annotation"),
      text,
      x,
      y,
      color,
      createdAt: Date.now()
    };
    this.annotations = [...this.annotations, annotation];
    return annotation;
  }

  /**
   * remove 删除一个标注。
   */
  public remove(id: string): void {
    this.annotations = this.annotations.filter((item) => item.id !== id);
  }

  /**
   * clear 清空当前画布上的全部标注。
   */
  public clear(): void {
    this.annotations = [];
  }
}

/**
 * captureCanvas 将当前画布截图导出为 PNG 数据地址。
 */
export function captureCanvas(canvas: HTMLCanvasElement): string {
  return canvas.toDataURL("image/png");
}

/**
 * startCanvasRecording 启动画布录制。
 */
export function startCanvasRecording(canvas: HTMLCanvasElement, fps = 30): MediaRecorder {
  const stream = canvas.captureStream(fps);
  return new MediaRecorder(stream, { mimeType: "video/webm" });
}
