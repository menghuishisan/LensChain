import type { ControlDescriptor, TimeControlMode } from "./types.js";

/**
 * TimelineControl 为不同时间模式生成控制按钮描述。
 */
export class TimelineControl {
  private static readonly SPEED_OPTIONS = [0.5, 1, 1.5, 2] as const;

  /**
   * getControls 返回某种时间模式下允许显示的控件。
   */
  public getControls(mode: TimeControlMode): ControlDescriptor[] {
    if (mode === "reactive") {
      return [];
    }
    if (mode === "continuous") {
      return [
        { command: "pause", label: "暂停", enabled: true },
        { command: "resume", label: "恢复", enabled: true },
        {
          command: "set_speed",
          label: "变速",
          enabled: true,
          valueOptions: [...TimelineControl.SPEED_OPTIONS]
        }
      ];
    }
    return [
      { command: "play", label: "播放", enabled: true },
      { command: "pause", label: "暂停", enabled: true },
      { command: "step", label: "单步", enabled: true },
      {
        command: "set_speed",
        label: "变速",
        enabled: true,
        valueOptions: [...TimelineControl.SPEED_OPTIONS]
      },
      { command: "reset", label: "重置", enabled: true }
    ];
  }
}
