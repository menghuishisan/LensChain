"use client";

// useAssignmentAutosave.ts
// 模块03作业自动保存 hook，负责本地草稿、定时服务端草稿保存和离开确认。

import { useEffect } from "react";

import type { ID } from "@/types/api";
import type { AssignmentAnswersRequest } from "@/types/courseAssignment";

const ASSIGNMENT_AUTOSAVE_INTERVAL_MS = 60_000;

/**
 * 作业自动保存配置。
 */
export interface UseAssignmentAutosaveOptions {
  assignmentID: ID;
  answers: Record<ID, string>;
  hasUnsavedChanges: boolean;
  onSaveDraft: (payload: AssignmentAnswersRequest) => void;
  onAutosaved?: () => void;
  intervalMs?: number;
}

/**
 * buildAssignmentAnswersPayload 将页面答案草稿转换为模块03草稿/提交接口请求体。
 */
export function buildAssignmentAnswersPayload(answers: Record<ID, string>): AssignmentAnswersRequest {
  return {
    answers: Object.entries(answers).map(([questionID, answerContent]) => ({
      question_id: questionID,
      answer_content: answerContent,
    })),
  };
}

/**
 * persistAssignmentDraftLocal 将草稿写入 localStorage；网络异常时仍保留本地副本。
 */
export function persistAssignmentDraftLocal(assignmentID: ID, answers: Record<ID, string>) {
  localStorage.setItem(`assignment-draft:${assignmentID}`, JSON.stringify(answers));
}

/**
 * useAssignmentAutosave 每 60 秒自动保存未提交草稿，并在有未保存内容时阻止直接离开页面。
 */
export function useAssignmentAutosave({
  assignmentID,
  answers,
  hasUnsavedChanges,
  onSaveDraft,
  onAutosaved,
  intervalMs = ASSIGNMENT_AUTOSAVE_INTERVAL_MS,
}: UseAssignmentAutosaveOptions) {
  useEffect(() => {
    if (!hasUnsavedChanges || assignmentID.length === 0) {
      return undefined;
    }

    const timerID = window.setInterval(() => {
      persistAssignmentDraftLocal(assignmentID, answers);
      onSaveDraft(buildAssignmentAnswersPayload(answers));
      onAutosaved?.();
    }, intervalMs);

    return () => window.clearInterval(timerID);
  }, [answers, assignmentID, hasUnsavedChanges, intervalMs, onAutosaved, onSaveDraft]);

  useEffect(() => {
    if (!hasUnsavedChanges) {
      return undefined;
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "有未保存的作答，是否离开？";
      return event.returnValue;
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [hasUnsavedChanges]);
}
