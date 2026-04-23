"use client";

// useUserImport.ts
// 模块01用户导入 hook，封装模板下载、预览、执行导入和失败明细下载。

import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
  downloadUserImportFailures,
  downloadUserImportTemplate,
  executeUserImport,
  previewUserImport,
} from "@/services/auth";
import type { UserImportType } from "@/types/auth";

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

/**
 * useDownloadUserImportTemplateMutation 下载用户导入模板。
 */
export function useDownloadUserImportTemplateMutation() {
  return useMutation({
    mutationFn: downloadUserImportTemplate,
    onSuccess: (result, type: UserImportType) => {
      downloadBlob(result.blob, result.filename ?? `${type}-import-template.xlsx`);
    },
  });
}

/**
 * usePreviewUserImportMutation 调用 POST /user-imports/preview。
 */
export function usePreviewUserImportMutation() {
  return useMutation({
    mutationFn: previewUserImport,
  });
}

/**
 * useExecuteUserImportMutation 调用 POST /user-imports/execute，成功后刷新用户列表。
 */
export function useExecuteUserImportMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: executeUserImport,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useDownloadUserImportFailuresMutation 下载导入失败明细。
 */
export function useDownloadUserImportFailuresMutation() {
  return useMutation({
    mutationFn: downloadUserImportFailures,
    onSuccess: (result, importID) => {
      downloadBlob(result.blob, result.filename ?? `${importID}-failures.xlsx`);
    },
  });
}
