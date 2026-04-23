"use client";

// SchoolApplicationQueryPanel.tsx
// 模块02申请状态查询组件，支持发送验证码、查询状态和拒绝后重新申请入口。

import Link from "next/link";
import { useState } from "react";

import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useToast } from "@/components/ui/Toast";
import { SchoolApplicationStatusBadge } from "@/components/business/SchoolStatusBadge";
import { useQuerySchoolApplications, useSendSchoolApplicationSmsMutation } from "@/hooks/useSchoolApplications";
import { formatDateTime } from "@/lib/format";
import { validateSchoolQueryForm } from "@/lib/school-validation";
import type { QuerySchoolApplicationsParams } from "@/types/school";

const REAPPLY_AUTH_STORAGE_KEY = "lenschain-school-reapply-auth";

/**
 * writeSchoolReapplyAuth 保存重新申请预填详情所需的手机号和验证码。
 */
export function writeSchoolReapplyAuth(phone: string, smsCode: string) {
  sessionStorage.setItem(REAPPLY_AUTH_STORAGE_KEY, JSON.stringify({ phone, sms_code: smsCode }));
}

/**
 * SchoolApplicationQueryPanel 入驻申请状态查询组件。
 */
export function SchoolApplicationQueryPanel() {
  const [phone, setPhone] = useState("");
  const [smsCode, setSmsCode] = useState("");
  const [queryParams, setQueryParams] = useState<QuerySchoolApplicationsParams | null>(null);
  const smsMutation = useSendSchoolApplicationSmsMutation();
  const query = useQuerySchoolApplications(queryParams);
  const { showToast } = useToast();
  const validation = validateSchoolQueryForm({ phone, sms_code: smsCode });

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <Card className="w-full max-w-4xl">
        <CardHeader className="text-center">
          <CardTitle>查询入驻申请进度</CardTitle>
          <CardDescription>输入联系人手机号和短信验证码，查看该手机号关联的所有申请记录。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <form
            className="grid gap-4 md:grid-cols-[1fr_1fr_auto]"
            onSubmit={(event) => {
              event.preventDefault();
              if (validation.isValid) {
                setQueryParams({ phone, sms_code: smsCode });
              }
            }}
          >
            <FormField label="手机号" required error={validation.errors.phone}>
              <Input value={phone} onChange={(event) => setPhone(event.target.value)} hasError={Boolean(validation.errors.phone)} />
            </FormField>
            <FormField label="验证码" required error={validation.errors.sms_code}>
              <div className="flex gap-2">
                <Input value={smsCode} onChange={(event) => setSmsCode(event.target.value)} hasError={Boolean(validation.errors.sms_code)} />
                <Button
                  type="button"
                  variant="outline"
                  isLoading={smsMutation.isPending}
                  onClick={() =>
                    smsMutation.mutate(
                      { phone },
                      {
                        onSuccess: () => showToast({ title: "验证码已发送", variant: "success" }),
                        onError: (error) => showToast({ title: "发送失败", description: error.message, variant: "destructive" }),
                      },
                    )
                  }
                >
                  获取验证码
                </Button>
              </div>
            </FormField>
            <div className="flex items-end">
              <Button type="submit" disabled={!validation.isValid}>查询</Button>
            </div>
          </form>

          {query.isLoading ? <LoadingState /> : null}
          {query.isError ? <ErrorState description={query.error.message} /> : null}
          {query.data?.applications.length === 0 ? <EmptyState title="暂无申请记录" description="该手机号下没有可查询的入驻申请。" /> : null}
          <div className="grid gap-4">
            {query.data?.applications.map((item) => (
              <div key={item.application_id} className="rounded-xl border border-border bg-card p-5 shadow-sm">
                <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                  <div>
                    <h3 className="font-display text-xl font-semibold">{item.school_name}</h3>
                    <p className="mt-1 text-sm text-muted-foreground">
                      申请时间：{formatDateTime(item.created_at)} {item.reviewed_at ? ` · 审核时间：${formatDateTime(item.reviewed_at)}` : ""}
                    </p>
                  </div>
                  <SchoolApplicationStatusBadge status={item.status} text={item.status_text} />
                </div>
                {item.reject_reason ? <p className="mt-3 rounded-lg bg-destructive/8 p-3 text-sm text-destructive">拒绝原因：{item.reject_reason}</p> : null}
                {item.status === 3 ? (
                  <Link
                    className={buttonClassName({ variant: "outline", className: "mt-4" })}
                    href={`/apply?reapply=${item.application_id}`}
                    onClick={() => writeSchoolReapplyAuth(phone, smsCode)}
                  >
                    修改后重新申请
                  </Link>
                ) : null}
              </div>
            ))}
          </div>
          <Link className={buttonClassName({ variant: "ghost" })} href="/apply">返回入驻申请</Link>
        </CardContent>
      </Card>
    </main>
  );
}
