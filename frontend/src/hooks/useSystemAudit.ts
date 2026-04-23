"use client";

// useSystemAudit.ts
// 模块08统一审计 hook，封装审计分页查询与导出能力。

import { useMutation, useQuery } from "@tanstack/react-query";

import { exportAuditLogs, listAuditLogs } from "@/services/system";
import type { SystemAuditExportParams, SystemAuditListParams } from "@/types/system";

/**
 * systemAuditQueryKey 审计日志 Query key。
 */
export function systemAuditQueryKey(params: SystemAuditListParams) {
  return ["system", "audit", "logs", params] as const;
}

/**
 * useSystemAudit 查询统一审计日志列表。
 */
export function useSystemAudit(params: SystemAuditListParams, enabled = true) {
  return useQuery({
    queryKey: systemAuditQueryKey(params),
    queryFn: () => listAuditLogs(params),
    enabled,
  });
}

/**
 * useSystemAuditExport 导出统一审计日志。
 */
export function useSystemAuditExport() {
  return useMutation({
    mutationFn: (params: SystemAuditExportParams) => exportAuditLogs(params),
  });
}
