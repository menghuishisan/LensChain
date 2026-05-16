/**
 * microStepScheduler.ts — 微动画步骤推进。
 *
 * 职责（依据 06.md §5.3）：
 *   • 接收 MicroStep[]，按 duration_ms 顺序推进。
 *   • 暴露 activeStepId / activeHighlightIds / activeFireIds 供 primitiveRenderer 取用。
 *   • 支持 play / pause / reset / setSpeed（速率影响推进步长）。
 *
 * 不写兑底：duration_ms ≤ 0 直接抛错；fire_primitives 引用不存在的 id 由 drawer 端兜底
 * （drawer 检查 anchor / id 时已会抛错）。
 */

import type { MicroStep } from "./types.js";

export interface MicroStepStatus {
  activeStepId: string | null;
  activeHighlightIds: ReadonlySet<string>;
  activeFireIds: ReadonlySet<string>;
  currentIndex: number;
  total: number;
  progressInStep: number; // 0~1
}

export class MicroStepScheduler {
  private steps: readonly MicroStep[] = [];
  private currentIndex = 0;
  private stepStartedAt = 0;
  private accumulated = 0; // 暂停时累计已用 ms
  private running = false;
  private speed = 1;

  load(steps: readonly MicroStep[]): void {
    for (const s of steps) {
      if (s.duration_ms <= 0) {
        throw new Error(`MicroStep "${s.id}": duration_ms 必须 > 0`);
      }
    }
    this.steps = steps;
    this.reset();
  }

  play(now: number): void {
    if (this.running || this.steps.length === 0) return;
    this.running = true;
    this.stepStartedAt = now - this.accumulated;
  }

  pause(now: number): void {
    if (!this.running) return;
    this.accumulated = now - this.stepStartedAt;
    this.running = false;
  }

  reset(): void {
    this.currentIndex = 0;
    this.accumulated = 0;
    this.stepStartedAt = 0;
    this.running = false;
  }

  setSpeed(multiplier: number): void {
    if (multiplier <= 0) {
      throw new Error(`MicroStepScheduler.setSpeed: 倍率必须 > 0，得到 ${multiplier}`);
    }
    this.speed = multiplier;
  }

  /** 调用方每帧调用，推进到 now 对应步骤；返回当前状态。 */
  advance(now: number): MicroStepStatus {
    if (this.running && this.steps.length > 0) {
      while (this.currentIndex < this.steps.length) {
        const step = this.steps[this.currentIndex]!;
        const elapsed = (now - this.stepStartedAt) * this.speed;
        if (elapsed < step.duration_ms) break;
        this.currentIndex++;
        this.stepStartedAt = now;
        this.accumulated = 0;
      }
      if (this.currentIndex >= this.steps.length) {
        this.running = false;
      }
    }
    return this.snapshot(now);
  }

  /** 立即跳到指定 index（不播放动画过渡）。 */
  jumpTo(index: number): void {
    if (index < 0 || index >= this.steps.length) {
      throw new Error(`MicroStepScheduler.jumpTo: index ${index} 越界 [0,${this.steps.length})`);
    }
    this.currentIndex = index;
    this.accumulated = 0;
    this.stepStartedAt = 0;
  }

  private snapshot(now: number): MicroStepStatus {
    const total = this.steps.length;
    if (total === 0 || this.currentIndex >= total) {
      return {
        activeStepId: null,
        activeHighlightIds: new Set(),
        activeFireIds: new Set(),
        currentIndex: this.currentIndex,
        total,
        progressInStep: 0,
      };
    }
    const step = this.steps[this.currentIndex]!;
    const elapsed = this.running ? (now - this.stepStartedAt) * this.speed : this.accumulated * this.speed;
    const progress = Math.min(1, elapsed / step.duration_ms);
    return {
      activeStepId: step.id,
      activeHighlightIds: new Set(step.highlight_ids ?? []),
      activeFireIds: new Set(step.fire_primitives ?? []),
      currentIndex: this.currentIndex,
      total,
      progressInStep: progress,
    };
  }
}
