/**
 * useSimMode.ts — SimEngine 4 模式判定与配速常量（06.2 §2.1 / §四 / §3.4）。
 *
 * 不带任何渲染状态；纯派生函数 + 一个 hook 包装。
 */

import { useMemo } from 'react';
import type { ExperimentType, SimLayoutMode, SimMode, SimTimeControlMode } from '@/types/experiment';

/** 速率选项（06.2 §4.3）。 */
export const SPEED_OPTIONS: readonly { value: number; label: string }[] = [
  { value: 0.5, label: '0.5×' },
  { value: 1, label: '1×' },
  { value: 1.5, label: '1.5×' },
  { value: 2, label: '2×' },
];

/** 默认速率值（1×）。 */
export const DEFAULT_SPEED = 1;

/** 单步回退最大次数（06.2 §4.2 协议要求）。 */
export const STEP_BACK_BUFFER_LIMIT = 20;

/**
 * 判定 SimEngine 4 模式（06.2 §2.2）。
 *
 * 优先级：experiment_type=3 → hybrid（不论场景数 / 联动）；否则按场景数与联动组组合。
 */
export function detectMode(
  experimentType: ExperimentType,
  sceneCount: number,
  hasActiveLinkGroup: boolean,
): SimMode {
  if (experimentType === 3) return 'hybrid';
  if (sceneCount === 1) return 'single';
  if (hasActiveLinkGroup) return 'linkage';
  return 'comparison';
}

/**
 * 默认主区布局（06.2 §3.4）：场景数 ≤4 且视口 ≥1280 才用 grid，否则降级 focus。
 * carousel 由学生在 ≥5 场景时显式选择。
 */
export function defaultLayout(sceneCount: number, viewportWidth: number): SimLayoutMode {
  if (sceneCount <= 4 && viewportWidth >= 1280) return 'grid';
  return 'focus';
}

/**
 * 判定 ⏮ 单步回退按钮在当前模式下的可用性（06.2 §4.2）。
 *
 * 仅 A 单仿真 + process 时间模式启用；B/C/D 全部置灰；reactive/continuous 不显示。
 */
export interface StepBackVisibility {
  /** 是否在 UI 上呈现按钮（false 表示直接不渲染）。 */
  visible: boolean;
  /** 是否可点击（visible=true 但 enabled=false 表示置灰）。 */
  enabled: boolean;
  /** 置灰 / 隐藏原因（用于 tooltip）。 */
  reason?: string;
}

export function stepBackVisibility(mode: SimMode, timeMode: SimTimeControlMode): StepBackVisibility {
  if (timeMode !== 'process') return { visible: false, enabled: false };
  switch (mode) {
    case 'single':
      return { visible: true, enabled: true };
    case 'comparison':
      return { visible: true, enabled: false, reason: '对照模式不支持单步回退' };
    case 'linkage':
      return { visible: true, enabled: false, reason: '联动模式不支持单步回退，请重置' };
    case 'hybrid':
      return { visible: true, enabled: false, reason: '混合实验不支持单步回退' };
  }
}

/**
 * useSimMode 把判定逻辑包装为 React hook，结果随输入变化记忆化。
 */
export function useSimMode(input: {
  experimentType: ExperimentType;
  sceneCount: number;
  hasActiveLinkGroup: boolean;
}): SimMode {
  return useMemo(
    () => detectMode(input.experimentType, input.sceneCount, input.hasActiveLinkGroup),
    [input.experimentType, input.sceneCount, input.hasActiveLinkGroup],
  );
}
