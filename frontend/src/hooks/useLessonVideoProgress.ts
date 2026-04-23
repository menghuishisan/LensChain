"use client";

// useLessonVideoProgress.ts
// 模块03视频课时进度工具，统一 30 秒上报节流和 95% 完成判定。

import type { UpdateProgressRequest } from "@/types/course";

/**
 * shouldReportVideoProgress 判断当前播放秒数是否达到下一次 30 秒上报窗口。
 */
export function shouldReportVideoProgress(currentSeconds: number, lastReportedSeconds: number) {
  return currentSeconds > 0 && currentSeconds - lastReportedSeconds >= 30;
}

/**
 * buildLessonProgressPayload 根据播放位置生成学习进度上报请求体。
 */
export function buildLessonProgressPayload(currentSeconds: number, durationSeconds?: number | null): UpdateProgressRequest {
  const isCompleted = durationSeconds !== undefined && durationSeconds !== null && durationSeconds > 0 && currentSeconds / durationSeconds >= 0.95;

  return {
    status: isCompleted ? 3 : 2,
    video_progress: currentSeconds,
    study_duration_increment: 30,
  };
}

/**
 * buildLessonUnloadProgressPayload 在页面离开时补报尚未上报的最新播放位置。
 */
export function buildLessonUnloadProgressPayload(currentSeconds: number, lastReportedSeconds: number, durationSeconds?: number | null) {
  if (currentSeconds <= 0 || currentSeconds <= lastReportedSeconds) {
    return null;
  }

  const payload = buildLessonProgressPayload(currentSeconds, durationSeconds);
  return {
    ...payload,
    study_duration_increment: 0,
  } satisfies UpdateProgressRequest;
}

/**
 * getLessonResumeSecond 计算课时再次进入时的视频续播位置。
 */
export function getLessonResumeSecond(videoProgress?: number | null, durationSeconds?: number | null) {
  if (videoProgress === undefined || videoProgress === null || videoProgress <= 0) {
    return 0;
  }
  if (durationSeconds === undefined || durationSeconds === null || durationSeconds <= 0) {
    return videoProgress;
  }
  return Math.min(videoProgress, durationSeconds);
}
