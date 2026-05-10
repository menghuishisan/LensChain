import type { MicroStep, SpeedMultiplier, TimeControlMode } from "./types.js";

/**
 * MicroStepState 微步骤调度器当前快照。
 */
export interface MicroStepState {
  /** 当前活跃微步骤 ID（null 表示全部完成或未开始）。 */
  activeId: string | null;
  /** 当前微步骤在队列中的索引。 */
  index: number;
  /** 本步骤已经过时间（ms）。 */
  elapsed: number;
  /** 是否已全部播放完毕。 */
  finished: boolean;
}

/**
 * MicroStepScheduler 微步骤调度器。
 *
 * 负责根据 TimeControlMode 按序驱动 MicroStep 的推进：
 * - process:   等学生显式调用 advance() 或按 duration_ms 自动推进
 * - continuous: 按 duration_ms × speed 自动循环
 * - reactive:  操作触发后播完即停
 */
export class MicroStepScheduler {
  private steps: MicroStep[] = [];
  private mode: TimeControlMode = "process";
  private speed: SpeedMultiplier = 1;
  private index = 0;
  private elapsed = 0;
  private paused = true;
  private finished = false;
  private lastTimestamp = 0;

  /**
   * schedule 加载新的微步骤序列并重置调度状态。
   */
  public schedule(steps: MicroStep[], mode: TimeControlMode, speed: SpeedMultiplier = 1): void {
    this.steps = steps;
    this.mode = mode;
    this.speed = speed;
    this.index = 0;
    this.elapsed = 0;
    this.finished = steps.length === 0;
    this.lastTimestamp = 0;

    if (mode === "continuous") {
      this.paused = false;
    } else if (mode === "reactive") {
      this.paused = false;
    } else {
      this.paused = true;
    }
  }

  /**
   * setSpeed 调整速率倍率（运行中可动态调用）。
   */
  public setSpeed(speed: SpeedMultiplier): void {
    this.speed = speed;
  }

  /**
   * play 从暂停恢复自动推进。
   */
  public play(): void {
    if (this.finished) return;
    this.paused = false;
    this.lastTimestamp = 0;
  }

  /**
   * pause 暂停自动推进。
   */
  public pause(): void {
    this.paused = true;
  }

  /**
   * advance 手动推进一步（process 模式下使用）。
   * 返回 true 表示成功推进，false 表示已到末尾。
   */
  public advance(): boolean {
    if (this.finished) return false;
    this.index++;
    this.elapsed = 0;
    if (this.index >= this.steps.length) {
      this.handleSequenceEnd();
      return false;
    }
    return true;
  }

  /**
   * reset 重置到序列开头。
   */
  public reset(): void {
    this.index = 0;
    this.elapsed = 0;
    this.finished = this.steps.length === 0;
    this.lastTimestamp = 0;
  }

  /**
   * tick 由 requestAnimationFrame 驱动，传入当前帧时间戳（ms）。
   * 返回是否发生了步骤切换。
   */
  public tick(now: number): boolean {
    if (this.paused || this.finished || this.steps.length === 0) {
      this.lastTimestamp = now;
      return false;
    }

    if (this.lastTimestamp === 0) {
      this.lastTimestamp = now;
      return false;
    }

    const dt = (now - this.lastTimestamp) * this.speed;
    this.lastTimestamp = now;
    this.elapsed += dt;

    const current = this.steps[this.index];
    if (!current) {
      this.finished = true;
      return false;
    }

    if (this.elapsed >= current.duration_ms) {
      this.elapsed -= current.duration_ms;
      this.index++;
      if (this.index >= this.steps.length) {
        this.handleSequenceEnd();
        return true;
      }
      return true;
    }

    return false;
  }

  /**
   * getState 返回当前调度快照。
   */
  public getState(): MicroStepState {
    const current = this.steps[this.index];
    return {
      activeId: this.finished ? null : (current?.id ?? null),
      index: this.index,
      elapsed: this.elapsed,
      finished: this.finished
    };
  }

  /**
   * getActiveStep 返回当前活跃的微步骤定义。
   */
  public getActiveStep(): MicroStep | undefined {
    if (this.finished) return undefined;
    return this.steps[this.index];
  }

  /**
   * getHighlightIds 返回当前微步骤应高亮的原语 ID 集合。
   */
  public getHighlightIds(): string[] {
    const step = this.getActiveStep();
    return step?.highlight_ids ?? [];
  }

  /**
   * getFirePrimitives 返回当前微步骤应触发的原语 ID 集合。
   */
  public getFirePrimitives(): string[] {
    const step = this.getActiveStep();
    return step?.fire_primitives ?? [];
  }

  /**
   * getProgress 返回当前步骤的播放进度 [0, 1]。
   */
  public getProgress(): number {
    const step = this.getActiveStep();
    if (!step || step.duration_ms <= 0) return 0;
    return Math.min(1, this.elapsed / step.duration_ms);
  }

  /**
   * isLinkTrigger 当前步骤是否为联动触发点。
   */
  public isLinkTrigger(): boolean {
    return this.getActiveStep()?.is_link_trigger ?? false;
  }

  /**
   * getTotalSteps 返回总步骤数。
   */
  public getTotalSteps(): number {
    return this.steps.length;
  }

  private handleSequenceEnd(): void {
    if (this.mode === "continuous") {
      this.index = 0;
      this.elapsed = 0;
    } else {
      this.finished = true;
      this.paused = true;
    }
  }
}
