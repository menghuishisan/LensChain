"use client";

// AuthPanels.tsx
// 模块01认证页面组件：登录、SSO选择、SSO回调、强制改密和普通改密。

import { CheckCircle2, KeyRound, School, XCircle } from "lucide-react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useToast } from "@/components/ui/Toast";
import {
  getAuthHomePath,
  isForceChangeResponse,
  useForceChangePasswordMutation,
  useChangePasswordMutation,
  useLoginMutation,
  useSsoCallbackMutation,
  useSsoLoginUrl,
  useSsoSchools,
} from "@/hooks/useAuth";
import {
  getPasswordRules,
  validateChangePasswordForm,
  validateLoginForm,
} from "@/lib/auth-validation";
import { consumeAuthRedirectMessage } from "@/lib/auth-session";
import { useAuthStore } from "@/stores/authStore";
import type { LoginResponse } from "@/types/auth";

function getRedirectPath(response: LoginResponse) {
  return "user" in response ? getAuthHomePath(response.user.roles) : "/change-password";
}

/**
 * LoginForm 登录表单组件。
 */
export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { showToast } = useToast();
  const mutation = useLoginMutation();
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [authMessage, setAuthMessage] = useState<string | null>(null);

  useEffect(() => {
    setAuthMessage(consumeAuthRedirectMessage());
  }, []);

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const validation = validateLoginForm({ phone, password });
    setErrors(validation.errors);
    if (!validation.isValid) {
      return;
    }

    mutation.mutate(
      { phone, password },
      {
        onSuccess: (response) => {
          showToast({ title: isForceChangeResponse(response) ? "首次登录，请修改密码" : "登录成功", variant: "success" });
          const redirect = searchParams.get("redirect");
          router.push(isForceChangeResponse(response) ? "/change-password" : redirect ?? getRedirectPath(response));
        },
        onError: (error) => {
          showToast({ title: "登录失败", description: error.message, variant: "destructive" });
        },
      },
    );
  };

  return (
    <form className="space-y-4" onSubmit={handleSubmit}>
      {authMessage ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/8 px-4 py-3 text-sm text-destructive">{authMessage}</div>
      ) : null}
      {mutation.isError && mutation.error.message.includes("还剩") ? (
        <div className="rounded-xl border border-amber-500/20 bg-amber-500/8 px-4 py-3 text-sm text-amber-700">{mutation.error.message}</div>
      ) : null}
      <FormField id="phone" label="手机号" required error={errors.phone}>
        <Input id="phone" inputMode="numeric" value={phone} onChange={(event) => setPhone(event.target.value)} placeholder="请输入11位手机号" hasError={Boolean(errors.phone)} />
      </FormField>
      <FormField id="password" label="密码" required error={errors.password}>
        <Input id="password" type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="请输入密码" hasError={Boolean(errors.password)} />
      </FormField>
      <Button type="submit" fullWidth isLoading={mutation.isPending}>
        <KeyRound className="h-4 w-4" />
        登录
      </Button>
    </form>
  );
}

/**
 * SSOLoginButton SSO 登录入口按钮。
 */
export function SSOLoginButton() {
  return (
    <Link className={buttonClassName({ variant: "outline", fullWidth: true })} href="/sso">
      学校SSO登录
    </Link>
  );
}

/**
 * SsoSchoolSelector SSO 学校选择组件。
 */
export function SsoSchoolSelector() {
  const [keyword, setKeyword] = useState("");
  const { data, isLoading, isError, error } = useSsoSchools(keyword);
  const getSsoUrl = useSsoLoginUrl();

  return (
    <div className="mx-auto max-w-5xl space-y-6 px-4 py-10">
      <Link href="/login" className="text-sm font-semibold text-primary hover:underline">
        返回登录
      </Link>
      <Card>
        <CardHeader className="text-center">
          <CardTitle>请选择您的学校</CardTitle>
          <CardDescription>仅展示已配置 SSO 的学校，点击后将跳转到学校统一认证页面。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索学校名称" />
          {isLoading ? <LoadingState /> : null}
          {isError ? <ErrorState description={error.message} /> : null}
          {!isLoading && !isError && data?.list.length === 0 ? <EmptyState title="暂无SSO学校" description="请使用手机号密码登录，或联系学校管理员配置 SSO。" /> : null}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {data?.list.map((schoolItem) => (
              <a
                key={schoolItem.id}
                href={getSsoUrl(schoolItem.id)}
                className="group rounded-xl border border-border bg-card p-5 shadow-sm transition hover:-translate-y-1 hover:border-primary/40 hover:shadow-glow"
              >
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                  <School className="h-7 w-7" />
                </div>
                <p className="font-display text-xl font-semibold">{schoolItem.name}</p>
                <p className="mt-2 text-sm text-muted-foreground">点击进入统一认证</p>
              </a>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * SsoCallbackPanel SSO 回调处理组件。
 */
export function SsoCallbackPanel() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const mutation = useSsoCallbackMutation();
  const [hasStarted, setHasStarted] = useState(false);

  useEffect(() => {
    if (hasStarted) {
      return;
    }

    setHasStarted(true);
    const query: Record<string, string> = {};
    searchParams.forEach((value, key) => {
      query[key] = value;
    });

    mutation.mutate(query, {
      onSuccess: (response) => {
        router.replace(getRedirectPath(response));
      },
    });
  }, [hasStarted, mutation, router, searchParams]);

  if (mutation.isError) {
    return (
      <ErrorState
        className="mx-auto mt-16 max-w-xl"
        title="SSO认证失败"
        description={mutation.error.message.includes("未开通") ? "账号未开通，请联系管理员" : mutation.error.message}
        action={
          <Link className={buttonClassName({ variant: "primary" })} href="/login">
            返回登录
          </Link>
        }
      />
    );
  }

  return <LoadingState variant="spinner" className="mx-auto mt-16 max-w-xl" title="正在完成SSO登录" description="请稍候，链镜正在校验学校统一认证结果。" />;
}

/**
 * ForceChangePasswordForm 首次登录强制改密表单。
 */
export function ForceChangePasswordForm() {
  const router = useRouter();
  const { showToast } = useToast();
  const tempToken = useAuthStore((state) => state.tempToken);
  const tempTokenExpiresAt = useAuthStore((state) => state.tempTokenExpiresAt);
  const mutation = useForceChangePasswordMutation();
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const validation = validateChangePasswordForm({ newPassword, confirmPassword });
  const isExpired = tempTokenExpiresAt !== null && tempTokenExpiresAt < Date.now();

  if (tempToken === null || isExpired) {
    return (
      <ErrorState
        className="mx-auto mt-16 max-w-xl"
        title="临时凭证无效"
        description="请返回登录页重新登录后再修改初始密码。"
        action={
          <Link className={buttonClassName({ variant: "primary" })} href="/login">
            返回登录
          </Link>
        }
      />
    );
  }

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!validation.isValid) {
      return;
    }

    mutation.mutate(
      { tempToken, newPassword },
      {
        onSuccess: (response) => {
          showToast({ title: "密码修改成功", variant: "success" });
          router.replace(getRedirectPath(response));
        },
        onError: (error) => showToast({ title: "修改失败", description: error.message, variant: "destructive" }),
      },
    );
  };

  return (
    <AuthCard title="首次登录，请修改密码" description="新密码必须至少8位，并包含大小写字母和数字。">
      <form className="space-y-5" onSubmit={handleSubmit}>
        <PasswordInputs newPassword={newPassword} confirmPassword={confirmPassword} onNewPasswordChange={setNewPassword} onConfirmPasswordChange={setConfirmPassword} />
        <PasswordRuleList password={newPassword} />
        {validation.errors.confirmPassword ? <p className="text-sm text-destructive">{validation.errors.confirmPassword}</p> : null}
        <Button type="submit" fullWidth disabled={!validation.isValid} isLoading={mutation.isPending}>
          确认修改
        </Button>
      </form>
    </AuthCard>
  );
}

/**
 * ChangePasswordForm 已登录用户修改密码表单。
 */
export function ChangePasswordForm() {
  const { showToast } = useToast();
  const mutation = useChangePasswordMutation();
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const validation = validateChangePasswordForm({ oldPassword, newPassword, confirmPassword });

  return (
    <AuthCard title="修改密码" description="修改成功后当前会话保持有效。">
      <form
        className="space-y-5"
        onSubmit={(event) => {
          event.preventDefault();
          if (!validation.isValid) {
            return;
          }

          mutation.mutate(
            { old_password: oldPassword, new_password: newPassword },
            {
              onSuccess: () => showToast({ title: "密码修改成功", variant: "success" }),
              onError: (error) => showToast({ title: "修改失败", description: error.message, variant: "destructive" }),
            },
          );
        }}
      >
        <FormField id="old-password" label="旧密码" required error={validation.errors.oldPassword}>
          <Input id="old-password" type="password" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} hasError={Boolean(validation.errors.oldPassword)} />
        </FormField>
        <PasswordInputs newPassword={newPassword} confirmPassword={confirmPassword} onNewPasswordChange={setNewPassword} onConfirmPasswordChange={setConfirmPassword} errors={validation.errors} />
        <PasswordRuleList password={newPassword} />
        <Button type="submit" fullWidth disabled={!validation.isValid} isLoading={mutation.isPending}>
          保存新密码
        </Button>
      </form>
    </AuthCard>
  );
}

/**
 * AuthCard 认证表单卡片外壳。
 */
export function AuthCard({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <Card className="w-full max-w-xl">
        <CardHeader className="text-center">
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>{children}</CardContent>
      </Card>
    </main>
  );
}

function PasswordInputs({
  newPassword,
  confirmPassword,
  onNewPasswordChange,
  onConfirmPasswordChange,
  errors,
}: {
  newPassword: string;
  confirmPassword: string;
  onNewPasswordChange: (value: string) => void;
  onConfirmPasswordChange: (value: string) => void;
  errors?: Record<string, string | undefined>;
}) {
  return (
    <>
      <FormField id="new-password" label="新密码" required error={errors?.newPassword}>
        <Input id="new-password" type="password" value={newPassword} onChange={(event) => onNewPasswordChange(event.target.value)} hasError={Boolean(errors?.newPassword)} />
      </FormField>
      <FormField id="confirm-password" label="确认密码" required error={errors?.confirmPassword}>
        <Input id="confirm-password" type="password" value={confirmPassword} onChange={(event) => onConfirmPasswordChange(event.target.value)} hasError={Boolean(errors?.confirmPassword)} />
      </FormField>
    </>
  );
}

function PasswordRuleList({ password }: { password: string }) {
  return (
    <div className="grid gap-2 rounded-xl bg-muted/60 p-4">
      {getPasswordRules(password).map((rule) => (
        <div key={rule.key} className="flex items-center gap-2 text-sm">
          {rule.isPassed ? <CheckCircle2 className="h-4 w-4 text-emerald-600" /> : <XCircle className="h-4 w-4 text-muted-foreground" />}
          <span className={rule.isPassed ? "text-emerald-700 dark:text-emerald-300" : "text-muted-foreground"}>{rule.label}</span>
        </div>
      ))}
    </div>
  );
}
