"use client";

// SchoolSSOConfigForm.tsx
// 模块02校管 SSO 配置表单，敏感字段按文档脱敏展示，不明文回显未允许字段。

import { Eye, EyeOff, PlugZap } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useToast } from "@/components/ui/Toast";
import {
  useEnableSchoolSsoMutation,
  useSchoolSsoConfig,
  useTestSchoolSsoConfigMutation,
  useUpdateSchoolSsoConfigMutation,
} from "@/hooks/useSchoolSSO";
import { formatDateTime } from "@/lib/format";
import { validateSsoConfigForm } from "@/lib/school-validation";
import type { CasSsoConfig, OAuth2SsoConfig, SsoProvider, UpdateSchoolSsoConfigRequest } from "@/types/school";

const DEFAULT_OAUTH2_CONFIG: OAuth2SsoConfig = {
  authorize_url: "",
  token_url: "",
  userinfo_url: "",
  client_id: "",
  client_secret: "",
  redirect_uri: "",
  scope: "openid profile",
  user_id_attribute: "student_id",
};

const DEFAULT_CAS_CONFIG: CasSsoConfig = {
  cas_server_url: "",
  cas_service_url: "",
  cas_version: "3.0",
  user_id_attribute: "student_id",
};

/**
 * SchoolSSOConfigForm SSO配置表单组件。
 */
export function SchoolSSOConfigForm() {
  const query = useSchoolSsoConfig();
  const saveMutation = useUpdateSchoolSsoConfigMutation();
  const testMutation = useTestSchoolSsoConfigMutation();
  const enableMutation = useEnableSchoolSsoMutation();
  const { showToast } = useToast();
  const [provider, setProvider] = useState<SsoProvider>("oauth2");
  const [oauth2Config, setOauth2Config] = useState<OAuth2SsoConfig>(DEFAULT_OAUTH2_CONFIG);
  const [casConfig, setCasConfig] = useState<CasSsoConfig>(DEFAULT_CAS_CONFIG);
  const [shouldShowSecret, setShouldShowSecret] = useState(false);

  useEffect(() => {
    if (query.data === undefined) {
      return;
    }
    setProvider(query.data.provider);
    if (query.data.provider === "oauth2") {
      setOauth2Config({ ...DEFAULT_OAUTH2_CONFIG, ...(query.data.config as OAuth2SsoConfig) });
    } else {
      setCasConfig({ ...DEFAULT_CAS_CONFIG, ...(query.data.config as CasSsoConfig) });
    }
  }, [query.data]);

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }

  const payload: UpdateSchoolSsoConfigRequest = {
    provider,
    config: provider === "oauth2" ? oauth2Config : casConfig,
  };
  const validation = validateSsoConfigForm(payload);
  const isEnabled = query.data?.is_enabled ?? false;
  const isTested = query.data?.is_tested ?? false;

  return (
    <Card>
      <CardHeader>
        <CardTitle>SSO接入配置</CardTitle>
        <CardDescription>
          当前状态：{isEnabled ? "已启用" : "未启用"} · {isTested ? "已通过测试" : "未通过测试或需重新测试"}
          {query.data?.tested_at ? ` · 测试时间：${formatDateTime(query.data.tested_at)}` : ""}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            if (!validation.isValid) {
              return;
            }
            saveMutation.mutate(payload, {
              onSuccess: () => showToast({ title: "SSO配置已保存，请重新测试连接", variant: "success" }),
              onError: (error) => showToast({ title: "保存失败", description: error.message, variant: "destructive" }),
            });
          }}
        >
          <div className="flex gap-3">
            <label className="flex items-center gap-2 rounded-lg border border-border px-4 py-3 text-sm font-semibold">
              <input type="radio" checked={provider === "cas"} onChange={() => setProvider("cas")} />
              CAS
            </label>
            <label className="flex items-center gap-2 rounded-lg border border-border px-4 py-3 text-sm font-semibold">
              <input type="radio" checked={provider === "oauth2"} onChange={() => setProvider("oauth2")} />
              OAuth2.0
            </label>
          </div>
          {provider === "oauth2" ? (
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="授权端点" value={oauth2Config.authorize_url ?? ""} error={validation.errors.authorize_url} onChange={(authorize_url) => setOauth2Config((current) => ({ ...current, authorize_url }))} />
              <Field label="Token端点" value={oauth2Config.token_url ?? ""} error={validation.errors.token_url} onChange={(token_url) => setOauth2Config((current) => ({ ...current, token_url }))} />
              <Field label="用户信息端点" value={oauth2Config.userinfo_url ?? ""} error={validation.errors.userinfo_url} onChange={(userinfo_url) => setOauth2Config((current) => ({ ...current, userinfo_url }))} />
              <Field label="Client ID" value={oauth2Config.client_id ?? ""} error={validation.errors.client_id} onChange={(client_id) => setOauth2Config((current) => ({ ...current, client_id }))} />
              <FormField label="Client Secret" required error={validation.errors.client_secret} description="后端返回 ****** 时表示已配置密钥；保存新密钥时请输入明文，前端不会强制明文回显旧密钥。">
                <div className="flex gap-2">
                  <Input
                    type={shouldShowSecret ? "text" : "password"}
                    value={oauth2Config.client_secret ?? ""}
                    onChange={(event) => setOauth2Config((current) => ({ ...current, client_secret: event.target.value }))}
                    hasError={Boolean(validation.errors.client_secret)}
                  />
                  <Button type="button" variant="outline" size="icon" aria-label={shouldShowSecret ? "隐藏密钥" : "显示密钥"} onClick={() => setShouldShowSecret((current) => !current)}>
                    {shouldShowSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                </div>
              </FormField>
              <Field label="回调URL" value={oauth2Config.redirect_uri ?? ""} error={validation.errors.redirect_uri} onChange={(redirect_uri) => setOauth2Config((current) => ({ ...current, redirect_uri }))} />
              <Field label="Scope" value={oauth2Config.scope ?? ""} onChange={(scope) => setOauth2Config((current) => ({ ...current, scope }))} />
              <Field label="用户ID属性" value={oauth2Config.user_id_attribute ?? ""} error={validation.errors.user_id_attribute} onChange={(user_id_attribute) => setOauth2Config((current) => ({ ...current, user_id_attribute }))} />
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="CAS服务端URL" value={casConfig.cas_server_url ?? ""} error={validation.errors.cas_server_url} onChange={(cas_server_url) => setCasConfig((current) => ({ ...current, cas_server_url }))} />
              <Field label="CAS回调URL" value={casConfig.cas_service_url ?? ""} error={validation.errors.cas_service_url} onChange={(cas_service_url) => setCasConfig((current) => ({ ...current, cas_service_url }))} />
              <Field label="CAS版本" value={casConfig.cas_version ?? ""} onChange={(cas_version) => setCasConfig((current) => ({ ...current, cas_version: cas_version === "2.0" ? "2.0" : "3.0" }))} />
              <Field label="用户ID属性" value={casConfig.user_id_attribute ?? ""} error={validation.errors.user_id_attribute} onChange={(user_id_attribute) => setCasConfig((current) => ({ ...current, user_id_attribute }))} />
            </div>
          )}
          {testMutation.data ? (
            <div className={testMutation.data.is_tested ? "rounded-xl bg-emerald-500/10 p-4 text-emerald-700" : "rounded-xl bg-destructive/10 p-4 text-destructive"}>
              <p className="font-semibold">{testMutation.data.is_tested ? "连接成功" : "连接失败"}</p>
              <p className="mt-1 text-sm">{testMutation.data.test_detail ?? testMutation.data.error_detail}</p>
            </div>
          ) : null}
          {!isTested && isEnabled ? (
            <div className="rounded-xl bg-amber-500/10 p-4 text-sm text-amber-700">当前配置尚未通过测试，建议先测试连接再保持启用状态。</div>
          ) : null}
          <div className="flex flex-wrap gap-3">
            <Button type="submit" disabled={!validation.isValid} isLoading={saveMutation.isPending}>保存配置</Button>
            <Button type="button" variant="outline" isLoading={testMutation.isPending} onClick={() => testMutation.mutate(undefined, { onSuccess: () => showToast({ title: "SSO连接测试完成", variant: "success" }), onError: (error) => showToast({ title: "测试失败", description: error.message, variant: "destructive" }) })}>
              <PlugZap className="h-4 w-4" />
              测试连接
            </Button>
            <Button type="button" variant={isEnabled ? "destructive" : "secondary"} disabled={!isEnabled && !isTested} isLoading={enableMutation.isPending} onClick={() => enableMutation.mutate(!isEnabled, { onSuccess: () => showToast({ title: isEnabled ? "SSO已停用" : "SSO已启用", variant: "success" }) })}>
              {isEnabled ? "停用SSO" : "启用SSO"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function Field({ label, value, error, onChange }: { label: string; value: string; error?: string; onChange: (value: string) => void }) {
  return (
    <FormField label={label} required error={error}>
      <Input value={value} onChange={(event) => onChange(event.target.value)} hasError={Boolean(error)} />
    </FormField>
  );
}
