"use client";

// LogPanels.tsx
// 模块01日志查询组件，覆盖登录日志和操作日志筛选表格。

import { useState } from "react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { useAuth } from "@/hooks/useAuth";
import { useLoginLogs, useOperationLogs } from "@/hooks/useLogs";
import { formatDateTime } from "@/lib/format";
import type { LoginLogParams, OperationLogParams } from "@/types/auth";

/**
 * LoginLogPanel 登录日志查询组件。
 */
export function LoginLogPanel() {
  const { roles } = useAuth("school_admin");
  const isSuperAdmin = roles.includes("super_admin");
  const [params, setParams] = useState<LoginLogParams>({ page: 1, page_size: 20 });
  const query = useLoginLogs(params);
  const list = query.data?.list ?? [];

  return (
    <LogShell title="登录日志" description="查询认证相关登录、登出、失败和锁定记录。">
      <LogFilters
        userID={params.user_id ?? ""}
        action={params.action?.toString() ?? ""}
        onChange={(next) =>
          setParams({
            ...params,
            user_id: typeof next.user_id === "string" ? next.user_id : undefined,
            action: typeof next.action === "string" && next.action.length > 0 ? Number(next.action) : undefined,
            page: 1,
          })
        }
      />
      {query.isLoading ? <LoadingState /> : null}
      {query.isError ? <ErrorState description={query.error.message} /> : null}
      {!query.isLoading && !query.isError && list.length === 0 ? <EmptyState title="暂无登录日志" description="当前筛选条件下没有日志记录。" /> : null}
      {list.length > 0 ? (
        <TableContainer>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户</TableHead>
                <TableHead>手机号</TableHead>
                {isSuperAdmin && <TableHead>学校</TableHead>}
                <TableHead>动作</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>{item.user_name ?? item.user_id}</TableCell>
                  <TableCell>{item.login_method_text ?? item.login_method ?? "—"}</TableCell>
                  {isSuperAdmin && <TableCell>{item.school_name}</TableCell>}
                  <TableCell>{item.action_text ?? item.action}</TableCell>
                  <TableCell>{item.ip ?? "—"}</TableCell>
                  <TableCell>{item.fail_reason ?? "—"}</TableCell>
                  <TableCell>{formatDateTime(item.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ) : null}
      {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={(page) => setParams((current) => ({ ...current, page }))} /> : null}
    </LogShell>
  );
}

/**
 * OperationLogPanel 操作日志查询组件。
 */
export function OperationLogPanel() {
  const { roles } = useAuth("school_admin");
  const isSuperAdmin = roles.includes("super_admin");
  const [params, setParams] = useState<OperationLogParams>({ page: 1, page_size: 20 });
  const query = useOperationLogs(params);
  const list = query.data?.list ?? [];

  return (
    <LogShell title="操作日志" description="查询用户管理、密码重置、状态变更等操作审计记录。">
      <LogFilters
        userID={typeof params.operator_id === "string" ? params.operator_id : ""}
        action={typeof params.action === "string" ? params.action : ""}
        onChange={(next) =>
          setParams({
            ...params,
            operator_id: typeof next.user_id === "string" ? next.user_id : undefined,
            action: typeof next.action === "string" ? next.action : undefined,
            page: 1,
          })
        }
      />
      {query.isLoading ? <LoadingState /> : null}
      {query.isError ? <ErrorState description={query.error.message} /> : null}
      {!query.isLoading && !query.isError && list.length === 0 ? <EmptyState title="暂无操作日志" description="当前筛选条件下没有日志记录。" /> : null}
      {list.length > 0 ? (
        <TableContainer>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户</TableHead>
                {isSuperAdmin && <TableHead>学校</TableHead>}
                <TableHead>操作</TableHead>
                <TableHead>资源</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>详情</TableHead>
                <TableHead>时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>{item.operator_name ?? item.operator_id}</TableCell>
                  {isSuperAdmin && <TableCell>{item.school_name}</TableCell>}
                  <TableCell>{item.action}</TableCell>
                  <TableCell>{item.target_type ?? "—"} {item.target_id ?? ""}</TableCell>
                  <TableCell>{item.ip ?? "—"}</TableCell>
                  <TableCell>{item.detail ?? "—"}</TableCell>
                  <TableCell>{formatDateTime(item.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ) : null}
      {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={(page) => setParams((current) => ({ ...current, page }))} /> : null}
    </LogShell>
  );
}

function LogShell({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">{children}</CardContent>
    </Card>
  );
}

function LogFilters({ userID, action, onChange }: { userID: string; action: string; onChange: (value: { user_id?: string; action?: string | number }) => void }) {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      <Input placeholder="用户ID" value={userID} onChange={(event) => onChange({ user_id: event.target.value || undefined, action })} />
      <Input placeholder="操作类型" value={action} onChange={(event) => onChange({ user_id: userID || undefined, action: event.target.value || undefined })} />
    </div>
  );
}
