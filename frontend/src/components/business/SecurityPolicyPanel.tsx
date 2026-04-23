"use client";

// SecurityPolicyPanel.tsx
// 模块01安全策略配置组件，覆盖登录锁定、密码策略和 Token 策略。

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useToast } from "@/components/ui/Toast";
import { useSecurityPolicy, useUpdateSecurityPolicyMutation } from "@/hooks/useSecurity";
import type { SecurityPolicy } from "@/types/auth";

const EMPTY_POLICY: SecurityPolicy = {
  login_fail_max_count: 5,
  login_lock_duration_minutes: 15,
  password_min_length: 8,
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_digit: true,
  password_require_special_char: false,
  access_token_expire_minutes: 30,
  refresh_token_expire_days: 7,
};

/**
 * SecurityPolicyPanel 安全策略配置组件。
 */
export function SecurityPolicyPanel() {
  const query = useSecurityPolicy();
  const mutation = useUpdateSecurityPolicyMutation();
  const { showToast } = useToast();
  const [form, setForm] = useState<SecurityPolicy>(EMPTY_POLICY);

  useEffect(() => {
    if (query.data !== undefined) {
      setForm(query.data);
    }
  }, [query.data]);

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>安全策略配置</CardTitle>
        <CardDescription>超级管理员可配置登录锁定、密码复杂度和 Token 有效期。</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="space-y-6"
          onSubmit={(event) => {
            event.preventDefault();
            mutation.mutate(form, {
              onSuccess: () => showToast({ title: "安全策略已保存", variant: "success" }),
              onError: (error) => showToast({ title: "保存失败", description: error.message, variant: "destructive" }),
            });
          }}
        >
          <PolicySection title="登录锁定策略">
            <NumberField label="最大失败次数" value={form.login_fail_max_count} onChange={(value) => setForm((current) => ({ ...current, login_fail_max_count: value }))} suffix="次" />
            <NumberField label="锁定时长" value={form.login_lock_duration_minutes} onChange={(value) => setForm((current) => ({ ...current, login_lock_duration_minutes: value }))} suffix="分钟" />
          </PolicySection>
          <PolicySection title="密码策略">
            <NumberField label="最小密码长度" value={form.password_min_length} onChange={(value) => setForm((current) => ({ ...current, password_min_length: value }))} suffix="位" />
            <CheckboxField label="要求大写字母" checked={form.password_require_uppercase} onChange={(checked) => setForm((current) => ({ ...current, password_require_uppercase: checked }))} />
            <CheckboxField label="要求小写字母" checked={form.password_require_lowercase} onChange={(checked) => setForm((current) => ({ ...current, password_require_lowercase: checked }))} />
            <CheckboxField label="要求数字" checked={form.password_require_digit} onChange={(checked) => setForm((current) => ({ ...current, password_require_digit: checked }))} />
            <CheckboxField label="要求特殊字符" checked={form.password_require_special_char} onChange={(checked) => setForm((current) => ({ ...current, password_require_special_char: checked }))} />
          </PolicySection>
          <PolicySection title="Token策略">
            <NumberField label="Access Token有效期" value={form.access_token_expire_minutes} onChange={(value) => setForm((current) => ({ ...current, access_token_expire_minutes: value }))} suffix="分钟" />
            <NumberField label="Refresh Token有效期" value={form.refresh_token_expire_days} onChange={(value) => setForm((current) => ({ ...current, refresh_token_expire_days: value }))} suffix="天" />
          </PolicySection>
          <Button type="submit" isLoading={mutation.isPending}>保存配置</Button>
        </form>
      </CardContent>
    </Card>
  );
}

function PolicySection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-xl border border-border bg-muted/40 p-4">
      <h3 className="mb-4 font-display text-xl font-semibold">{title}</h3>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{children}</div>
    </section>
  );
}

function NumberField({ label, value, suffix, onChange }: { label: string; value: number; suffix: string; onChange: (value: number) => void }) {
  return (
    <FormField label={label} required>
      <div className="flex items-center gap-2">
        <Input type="number" min={1} value={value} onChange={(event) => onChange(Number(event.target.value))} />
        <span className="text-sm text-muted-foreground">{suffix}</span>
      </div>
    </FormField>
  );
}

function CheckboxField({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-center gap-3 rounded-lg border border-border bg-card px-3 py-3 text-sm font-semibold">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      {label}
    </label>
  );
}
