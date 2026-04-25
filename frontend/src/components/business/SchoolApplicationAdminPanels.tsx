"use client";

// SchoolApplicationAdminPanels.tsx
// 模块02超管入驻申请审核组件，覆盖申请列表、详情、审核通过和拒绝。

import Link from "next/link";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/Dialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { SchoolApplicationStatusBadge } from "@/components/business/SchoolStatusBadge";
import {
  useApproveSchoolApplicationMutation,
  useRejectSchoolApplicationMutation,
  useSchoolApplication,
  useSchoolApplications,
} from "@/hooks/useSchoolApplications";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";
import type { AdminSchoolApplicationListParams, SchoolApplicationStatus } from "@/types/school";

/**
 * SchoolApplicationReviewList 入驻申请审核列表组件。
 */
export function SchoolApplicationReviewList() {
  const [params, setParams] = useState<AdminSchoolApplicationListParams>({ page: 1, page_size: 20, status: 1 });
  const query = useSchoolApplications(params);
  const list = query.data?.list ?? [];
  const pendingCount = list.filter((item) => item.status === 1).length;
  const approvedCount = list.filter((item) => item.status === 2).length;
  const rejectedCount = list.filter((item) => item.status === 3).length;

  return (
    <Card>
      <CardHeader>
        <CardTitle>入驻申请管理</CardTitle>
        <CardDescription>查看所有学校入驻申请，并对待审核申请执行审核。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap gap-3">
          <Badge variant={params.status === 1 ? "default" : "outline"}>待审核 {pendingCount}</Badge>
          <Badge variant={params.status === 2 ? "default" : "outline"}>已通过 {approvedCount}</Badge>
          <Badge variant={params.status === 3 ? "default" : "outline"}>已拒绝 {rejectedCount}</Badge>
          <Badge variant={params.status === undefined ? "default" : "outline"}>全部 {list.length}</Badge>
        </div>
        <div className="grid gap-3 md:grid-cols-[1fr_12rem]">
          <Input placeholder="搜索学校、联系人或手机号" value={params.keyword ?? ""} onChange={(event) => setParams((current) => ({ ...current, keyword: event.target.value, page: 1 }))} />
          <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={params.status ?? ""} onChange={(event) => setParams((current) => ({ ...current, status: event.target.value ? (Number(event.target.value) as SchoolApplicationStatus) : undefined, page: 1 }))}>
            <option value="">全部</option>
            <option value="1">待审核</option>
            <option value="2">已通过</option>
            <option value="3">已拒绝</option>
          </select>
        </div>
        {query.isLoading ? <LoadingState /> : null}
        {query.isError ? <ErrorState description={query.error.message} /> : null}
        {!query.isLoading && !query.isError && list.length === 0 ? <EmptyState title="暂无入驻申请" description="当前筛选条件下没有申请记录。" /> : null}
        {list.length > 0 ? (
          <TableContainer>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>学校</TableHead>
                  <TableHead>编码</TableHead>
                  <TableHead>联系人</TableHead>
                  <TableHead>手机号</TableHead>
                  <TableHead>申请时间</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {list.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="font-semibold">{item.school_name}</TableCell>
                    <TableCell>{item.school_code}</TableCell>
                    <TableCell>{item.contact_name}</TableCell>
                    <TableCell>{item.contact_phone}</TableCell>
                    <TableCell>{formatDateTime(item.created_at)}</TableCell>
                    <TableCell><SchoolApplicationStatusBadge status={item.status} text={item.status_text} /></TableCell>
                    <TableCell>
                      <Link className={buttonClassName({ variant: "outline", size: "sm" })} href={`/admin/school-applications/${item.id}`}>
                        {item.status === 1 ? "审核" : "详情"}
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : null}
        {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={(page) => setParams((current) => ({ ...current, page }))} /> : null}
      </CardContent>
    </Card>
  );
}

/**
 * SchoolApplicationReviewDetail 入驻申请审核详情组件。
 */
export function SchoolApplicationReviewDetail({ applicationID }: { applicationID: ID }) {
  const query = useSchoolApplication(applicationID);
  const approveMutation = useApproveSchoolApplicationMutation(applicationID);
  const rejectMutation = useRejectSchoolApplicationMutation(applicationID);
  const { showToast } = useToast();
  const [licenseEndAt, setLicenseEndAt] = useState("");
  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectReason, setRejectReason] = useState("");

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }
  if (query.data === undefined) {
    return <EmptyState title="申请不存在" description="该申请可能已被处理或无权查看。" />;
  }

  const application = query.data;

  return (
    <div className="space-y-5">
      <Link className={buttonClassName({ variant: "ghost" })} href="/admin/school-applications">返回列表</Link>
      <Card>
        <CardHeader className="flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle>入驻申请审核</CardTitle>
            <CardDescription>{application.school_name} · {application.school_code}</CardDescription>
          </div>
          <SchoolApplicationStatusBadge status={application.status} text={application.status_text} />
        </CardHeader>
        <CardContent className="space-y-5">
          <InfoGrid
            title="学校信息"
            items={[
              ["学校名称", application.school_name],
              ["学校编码", application.school_code],
              ["学校地址", application.school_address ?? "—"],
              ["学校官网", application.school_website ?? "—"],
              ["LOGO", application.school_logo_url ?? "—"],
            ]}
          />
          <InfoGrid
            title="联系人信息"
            items={[
              ["姓名", application.contact_name],
              ["手机号", application.contact_phone],
              ["邮箱", application.contact_email ?? "—"],
              ["职务/部门", application.contact_title ?? "—"],
            ]}
          />
          <InfoGrid
            title="审核记录"
            items={[
              ["申请时间", formatDateTime(application.created_at)],
              ["审核时间", application.reviewed_at ? formatDateTime(application.reviewed_at) : "尚未审核"],
              ["上一条申请", application.previous_application_id ?? "无"],
              ["当前状态", application.status_text],
            ]}
          />
          {application.reject_reason ? <ErrorState title="拒绝原因" description={application.reject_reason} /> : null}
          {application.status === 1 ? (
            <div className="rounded-xl border border-border bg-muted/50 p-4">
              <FormField label="授权有效期" required description="审核通过后学校状态为已激活，并自动创建首个校管账号。">
                <Input type="datetime-local" value={licenseEndAt} onChange={(event) => setLicenseEndAt(event.target.value)} />
              </FormField>
              <div className="mt-4 flex justify-between">
                <Button type="button" variant="destructive" onClick={() => setRejectOpen(true)}>拒绝</Button>
                <Button
                  type="button"
                  disabled={!licenseEndAt}
                  isLoading={approveMutation.isPending}
                  onClick={() =>
                    approveMutation.mutate(
                      { license_end_at: new Date(licenseEndAt).toISOString() },
                      {
                        onSuccess: () => showToast({ title: "审核通过", variant: "success" }),
                        onError: (error) => showToast({ title: "审核失败", description: error.message, variant: "destructive" }),
                      },
                    )
                  }
                >
                  审核通过
                </Button>
              </div>
            </div>
          ) : null}
        </CardContent>
      </Card>
      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>拒绝入驻申请</DialogTitle>
            <DialogDescription>拒绝原因不能为空，并会通过短信通知申请人。</DialogDescription>
          </DialogHeader>
          <Textarea value={rejectReason} onChange={(event) => setRejectReason(event.target.value)} placeholder="请输入拒绝原因" />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectOpen(false)}>取消</Button>
            <Button
              variant="destructive"
              disabled={!rejectReason.trim()}
              isLoading={rejectMutation.isPending}
              onClick={() =>
                rejectMutation.mutate(
                  { reject_reason: rejectReason },
                  {
                    onSuccess: () => {
                      setRejectOpen(false);
                      showToast({ title: "已拒绝申请", variant: "success" });
                    },
                    onError: (error) => showToast({ title: "拒绝失败", description: error.message, variant: "destructive" }),
                  },
                )
              }
            >
              确认拒绝
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function InfoGrid({ title, items }: { title: string; items: [string, string][] }) {
  return (
    <section className="rounded-xl border border-border bg-card p-4">
      <h3 className="mb-3 font-display text-xl font-semibold">{title}</h3>
      <div className="grid gap-3 md:grid-cols-2">
        {items.map(([label, value]) => (
          <div key={label} className="rounded-lg bg-muted/60 p-3">
            <p className="text-xs text-muted-foreground">{label}</p>
            <p className="mt-1 text-sm font-semibold">{value}</p>
          </div>
        ))}
      </div>
    </section>
  );
}
