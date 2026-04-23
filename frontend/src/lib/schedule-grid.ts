// schedule-grid.ts
// 模块03课程表周视图工具，将课程表项按星期分组后供页面渲染。

import type { MyScheduleResponse } from "@/types/course";

const WEEKDAY_LABELS = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"] as const;

/**
 * buildWeeklyScheduleGrid 将课程表按周一到周日分组，并按开始时间排序。
 */
export function buildWeeklyScheduleGrid(schedules: MyScheduleResponse["schedules"]) {
  return WEEKDAY_LABELS.map((dayLabel, index) => {
    const dayOfWeek = index + 1;
    return {
      dayOfWeek,
      dayLabel,
      items: schedules
        .filter((item) => item.day_of_week === dayOfWeek)
        .sort((left, right) => left.start_time.localeCompare(right.start_time)),
    };
  });
}
