// useSimMode.ts
// SimEngine 模式判定与布局计算 Hook（06.2 §二 / §三）。
// 纯计算逻辑，不发起 API 请求，不渲染 JSX。

import { useMemo } from "react";

import type { ExperimentType, SimLayoutMode, SimMode, SimTimeControlMode } from "@/types/experiment";

/** detectMode 根据三字段组合判定 SimEngine 模式（06.2 §2.2）。 */
export function detectMode(
  experimentType: ExperimentType,
  sceneCount: number,
  hasActiveLinkGroup: boolean,
): SimMode {
  if (experimentType === 3) return "hybrid";
  if (sceneCount === 1) return "single";
  if (hasActiveLinkGroup) return "linkage";
  return "comparison";
}

/** defaultLayout 根据场景数与视口宽度判定默认布局（06.2 §3.4）。 */
export function defaultLayout(sceneCount: number, viewportWidth: number): SimLayoutMode {
  if (sceneCount <= 4 && viewportWidth >= 1280) return "grid";
  return "focus";
}

/** 回退按钮是否可用（06.2 §4.2）。 */
export function isStepBackEnabled(mode: SimMode, timeControlMode: SimTimeControlMode): boolean {
  return mode === "single" && timeControlMode === "process";
}

/** 回退按钮 tooltip 文本（06.2 §4.2）。 */
export function stepBackDisabledReason(mode: SimMode, timeControlMode: SimTimeControlMode): string | null {
  if (mode === "single" && timeControlMode === "process") return null;
  if (mode === "comparison") return "对照模式不支持单步回退";
  if (mode === "linkage") return "联动模式不支持单步回退，请重置";
  if (mode === "hybrid") return "混合实验不支持单步回退";
  return null;
}

/** 速度档位（06.2 §4.3）。 */
export const SPEED_OPTIONS = [0.5, 1, 1.5, 2] as const;

/** useSimMode 参数。 */
export interface UseSimModeOptions {
  experimentType: ExperimentType;
  sceneCount: number;
  hasActiveLinkGroup: boolean;
  viewportWidth: number;
  timeControlMode: SimTimeControlMode;
}

/** useSimMode 返回值。 */
export interface UseSimModeReturn {
  mode: SimMode;
  layout: SimLayoutMode;
  canStepBack: boolean;
  stepBackTooltip: string | null;
  showTimeControl: boolean;
  showSharedStatePanel: boolean;
  forceSyncClock: boolean;
}

/**
 * useSimMode 组合计算 SimEngine 模式、布局和控件显隐。
 * 页面和组件从本 hook 获取当前 SimEngine 的显示规则，不自行判断。
 */
export function useSimMode(options: UseSimModeOptions): UseSimModeReturn {
  const { experimentType, sceneCount, hasActiveLinkGroup, viewportWidth, timeControlMode } = options;

  return useMemo(() => {
    const mode = detectMode(experimentType, sceneCount, hasActiveLinkGroup);
    const layout = defaultLayout(sceneCount, viewportWidth);
    const canStepBack = isStepBackEnabled(mode, timeControlMode);
    const stepBackTooltip = stepBackDisabledReason(mode, timeControlMode);
    const showTimeControl = timeControlMode !== "reactive";
    const showSharedStatePanel = mode === "linkage";
    const forceSyncClock = mode === "linkage";

    return { mode, layout, canStepBack, stepBackTooltip, showTimeControl, showSharedStatePanel, forceSyncClock };
  }, [experimentType, sceneCount, hasActiveLinkGroup, viewportWidth, timeControlMode]);
}
