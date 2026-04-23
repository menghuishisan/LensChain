"use client";

// useLogs.ts
// 模块01日志 hook，封装登录日志与操作日志分页查询。

import { useQuery } from "@tanstack/react-query";

import { listLoginLogs, listOperationLogs } from "@/services/auth";
import type { LoginLogParams, OperationLogParams } from "@/types/auth";

/**
 * useLoginLogs 查询 GET /login-logs。
 */
export function useLoginLogs(params: LoginLogParams) {
  return useQuery({
    queryKey: ["auth", "login-logs", params],
    queryFn: () => listLoginLogs(params),
  });
}

/**
 * useOperationLogs 查询 GET /operation-logs。
 */
export function useOperationLogs(params: OperationLogParams) {
  return useQuery({
    queryKey: ["auth", "operation-logs", params],
    queryFn: () => listOperationLogs(params),
  });
}
