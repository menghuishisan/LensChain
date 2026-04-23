"use client";

// SchoolProfilePanels.tsx
// 模块02校管学校资料和授权状态组件。

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { SchoolStatusBadge } from "@/components/business/SchoolStatusBadge";
import { useSchoolLicense, useSchoolProfile, useUpdateSchoolProfileMutation } from "@/hooks/useSchools";
import { formatDateTime } from "@/lib/format";
import type { UpdateSchoolProfileRequest } from "@/types/school";

/**
 * SchoolProfilePanel 本校信息管理组件。
 */
export function SchoolProfilePanel() {
  const query = useSchoolProfile();
  const mutation = useUpdateSchoolProfileMutation();
  const { showToast } = useToast();
  const [form, setForm] = useState<UpdateSchoolProfileRequest>({ logo_url: "", address: "", website: "", description: "" });

  useEffect(() => {
    if (query.data !== undefined) {
      setForm({
        logo_url: query.data.logo_url,
        address: query.data.address,
        website: query.data.website,
        description: query.data.description,
      });
    }
  }, [query.data]);

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }
  if (query.data === undefined) {
    return <EmptyState title="暂无学校资料" description="请确认当前账号具有学校管理员权限。" />;
  }

  return (
    <Card>
      <CardHeader className="flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle>学校信息管理</CardTitle>
          <CardDescription>{query.data.name} · 编码：{query.data.code}（不可修改，需联系超管）</CardDescription>
        </div>
        <SchoolStatusBadge status={query.data.status} text={query.data.status_text} />
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            mutation.mutate(form, {
              onSuccess: () => showToast({ title: "学校资料已更新", variant: "success" }),
              onError: (error) => showToast({ title: "保存失败", description: error.message, variant: "destructive" }),
            });
          }}
        >
          <FormField label="LOGO URL">
            <Input value={form.logo_url ?? ""} onChange={(event) => setForm((current) => ({ ...current, logo_url: event.target.value }))} />
          </FormField>
          <FormField label="学校地址">
            <Input value={form.address ?? ""} onChange={(event) => setForm((current) => ({ ...current, address: event.target.value }))} />
          </FormField>
          <FormField label="学校官网">
            <Input value={form.website ?? ""} onChange={(event) => setForm((current) => ({ ...current, website: event.target.value }))} />
          </FormField>
          <FormField label="学校简介">
            <Textarea value={form.description ?? ""} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
          </FormField>
          <Button type="submit" isLoading={mutation.isPending}>保存修改</Button>
        </form>
      </CardContent>
    </Card>
  );
}

/**
 * SchoolLicensePanel 校管授权状态组件。
 */
export function SchoolLicensePanel() {
  const query = useSchoolLicense();

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }
  if (query.data === undefined) {
    return <EmptyState title="暂无授权状态" description="请联系平台管理员确认学校授权。" />;
  }

  const license = query.data;
  const percent = Math.max(0, Math.min(100, Math.round((license.remaining_days / 365) * 100)));

  return (
    <Card>
      <CardHeader>
        <CardTitle>授权状态</CardTitle>
        <CardDescription>如需续期，请联系平台管理员。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {license.is_expiring_soon ? <div className="rounded-xl bg-amber-500/12 p-4 text-amber-700">学校授权将在7天内到期，请尽快联系平台管理员。</div> : null}
        {license.status === 3 ? <div className="rounded-xl bg-destructive/10 p-4 text-destructive">学校处于缓冲期，缓冲期结束后将被冻结。</div> : null}
        {license.status === 4 ? <ErrorState title="学校已冻结" description="当前学校授权已冻结，功能访问将受到限制。" /> : null}
        <div className="rounded-[2rem] bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.18),transparent_24rem),linear-gradient(135deg,hsl(var(--card)),hsl(40_76%_95%))] p-8 shadow-panel">
          <SchoolStatusBadge status={license.status} text={license.status_text} />
          <p className="mt-6 text-sm text-muted-foreground">授权期限</p>
          <h2 className="mt-2 font-display text-3xl font-semibold">
            {formatDateTime(license.license_start_at)} 至 {formatDateTime(license.license_end_at)}
          </h2>
          <p className="mt-4 text-lg font-semibold">剩余 {license.remaining_days} 天</p>
          <div className="mt-5 h-4 overflow-hidden rounded-full bg-muted">
            <div className="h-full rounded-full bg-primary" style={{ width: `${percent}%` }} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
