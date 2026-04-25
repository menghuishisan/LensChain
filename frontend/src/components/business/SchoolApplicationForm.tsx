"use client";

// SchoolApplicationForm.tsx
// 模块02公开入驻申请表单，支持首次申请和拒绝后重新申请。

import { Building2, Send } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useReapplySchoolApplicationMutation, useSchoolApplicationReapplyDetail, useSubmitSchoolApplicationMutation } from "@/hooks/useSchoolApplications";
import { validateSchoolApplicationForm } from "@/lib/school-validation";
import type { QuerySchoolApplicationsParams, SchoolApplicationRequest } from "@/types/school";

const REAPPLY_AUTH_STORAGE_KEY = "lenschain-school-reapply-auth";

const EMPTY_APPLICATION: SchoolApplicationRequest = {
  school_name: "",
  school_code: "",
  school_address: "",
  school_website: "",
  school_logo_url: "",
  contact_name: "",
  contact_phone: "",
  contact_email: "",
  contact_title: "",
};

/**
 * SchoolApplicationForm 学校入驻申请表单组件。
 */
export function SchoolApplicationForm() {
  const searchParams = useSearchParams();
  const reapplyID = searchParams.get("reapply");
  const [reapplyAuth, setReapplyAuth] = useState<QuerySchoolApplicationsParams | null>(null);
  const [form, setForm] = useState<SchoolApplicationRequest>(EMPTY_APPLICATION);
  const [smsCode, setSmsCode] = useState("");
  const [submitResult, setSubmitResult] = useState<string | null>(null);
  const { showToast } = useToast();
  const submitMutation = useSubmitSchoolApplicationMutation();
  const reapplyMutation = useReapplySchoolApplicationMutation(reapplyID ?? "");
  const reapplyDetailQuery = useSchoolApplicationReapplyDetail(reapplyID, reapplyAuth);
  const validation = validateSchoolApplicationForm(form);
  const isReapply = reapplyID !== null;

  useEffect(() => {
    if (!isReapply) {
      return;
    }
    const raw = sessionStorage.getItem(REAPPLY_AUTH_STORAGE_KEY);
    if (raw === null) {
      return;
    }
    const parsed = JSON.parse(raw) as QuerySchoolApplicationsParams;
    setReapplyAuth(parsed);
    setSmsCode(parsed.sms_code);
  }, [isReapply]);

  useEffect(() => {
    if (reapplyDetailQuery.data === undefined) {
      return;
    }
    setForm({
      school_name: reapplyDetailQuery.data.school_name,
      school_code: reapplyDetailQuery.data.school_code,
      school_address: reapplyDetailQuery.data.school_address,
      school_website: reapplyDetailQuery.data.school_website,
      school_logo_url: reapplyDetailQuery.data.school_logo_url,
      contact_name: reapplyDetailQuery.data.contact_name,
      contact_phone: reapplyDetailQuery.data.contact_phone,
      contact_email: reapplyDetailQuery.data.contact_email,
      contact_title: reapplyDetailQuery.data.contact_title,
    });
  }, [reapplyDetailQuery.data]);

  return (
    <main className="grid min-h-screen bg-slate-950 lg:grid-cols-[0.9fr_1.1fr]">
      <section className="hidden bg-[radial-gradient(circle_at_20%_20%,rgba(20,184,166,.38),transparent_28rem),linear-gradient(135deg,#062f2c,#101827_62%,#3b2414)] p-10 text-white lg:flex lg:flex-col lg:justify-between">
        <div>
          <div className="mb-10 flex h-14 w-14 items-center justify-center rounded-2xl bg-white/12">
            <Building2 className="h-8 w-8" />
          </div>
          <h1 className="font-display text-5xl font-semibold leading-tight">让学校的区块链教学快速接入链镜。</h1>
          <p className="mt-6 text-lg leading-8 text-white/70">课程、实验、竞赛一体化支持，学校可按需接入统一认证。</p>
        </div>
        <div className="grid gap-3 text-sm text-white/70">
          <span>多链全覆盖</span>
          <span>动态实验环境</span>
          <span>CTF竞赛实战</span>
          <span>教学实践一体化</span>
        </div>
      </section>

      <section className="flex items-center justify-center px-4 py-10">
        <Card className="w-full max-w-3xl">
          <CardHeader>
            <CardTitle>{isReapply ? "修改后重新申请" : "学校入驻申请"}</CardTitle>
            <CardDescription>提交后会生成申请编号，申请人可使用联系人手机号和短信验证码查询审核进度。</CardDescription>
          </CardHeader>
          <CardContent>
            {submitResult ? (
              <div className="mb-5 rounded-xl border border-primary/20 bg-primary/8 p-4">
                <p className="font-semibold text-primary">申请提交成功：{submitResult}</p>
                <p className="mt-1 text-sm text-muted-foreground">请使用联系人手机号查询审核进度。</p>
                <Link className={buttonClassName({ variant: "outline", className: "mt-4" })} href="/apply/query">
                  查询申请进度
                </Link>
              </div>
            ) : null}
            {isReapply && reapplyAuth === null ? (
              <div className="mb-5 rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-700">
                未找到查询验证码上下文，请从申请状态查询页点击“修改后重新申请”进入，或手动填写完整信息并输入验证码。
              </div>
            ) : null}
            <form
              className="grid gap-4 md:grid-cols-2"
              onSubmit={(event) => {
                event.preventDefault();
                if (!validation.isValid) {
                  return;
                }
                const onSuccess = (result: { application_id: string }) => {
                  setSubmitResult(result.application_id);
                  showToast({ title: "申请提交成功", variant: "success" });
                };
                const onError = (error: Error) => showToast({ title: "提交失败", description: error.message, variant: "destructive" });

                if (isReapply) {
                  reapplyMutation.mutate({ ...form, sms_code: smsCode }, { onSuccess, onError });
                  return;
                }

                submitMutation.mutate(form, { onSuccess, onError });
              }}
            >
              <ApplicationTextField label="学校名称" value={form.school_name} onChange={(school_name) => setForm((current) => ({ ...current, school_name }))} error={validation.errors.school_name} required />
              <ApplicationTextField label="学校编码" value={form.school_code} onChange={(school_code) => setForm((current) => ({ ...current, school_code }))} error={validation.errors.school_code} required />
              <ApplicationTextField label="学校地址" value={form.school_address ?? ""} onChange={(school_address) => setForm((current) => ({ ...current, school_address }))} />
              <ApplicationTextField label="学校官网" value={form.school_website ?? ""} onChange={(school_website) => setForm((current) => ({ ...current, school_website }))} error={validation.errors.school_website} />
              <ApplicationTextField label="学校LOGO URL" value={form.school_logo_url ?? ""} onChange={(school_logo_url) => setForm((current) => ({ ...current, school_logo_url }))} />
              <ApplicationTextField label="联系人姓名" value={form.contact_name} onChange={(contact_name) => setForm((current) => ({ ...current, contact_name }))} error={validation.errors.contact_name} required />
              <ApplicationTextField label="联系人手机号" value={form.contact_phone} onChange={(contact_phone) => setForm((current) => ({ ...current, contact_phone }))} error={validation.errors.contact_phone} required />
              <ApplicationTextField label="联系人邮箱" value={form.contact_email ?? ""} onChange={(contact_email) => setForm((current) => ({ ...current, contact_email }))} error={validation.errors.contact_email} />
              <ApplicationTextField label="职务/部门" value={form.contact_title ?? ""} onChange={(contact_title) => setForm((current) => ({ ...current, contact_title }))} />
              {isReapply ? <ApplicationTextField label="短信验证码" value={smsCode} onChange={setSmsCode} required /> : null}
              <div className="flex gap-3 md:col-span-2">
                <Button type="submit" disabled={!validation.isValid || (isReapply && smsCode.length === 0)} isLoading={submitMutation.isPending || reapplyMutation.isPending}>
                  <Send className="h-4 w-4" />
                  {isReapply ? "重新提交申请" : "提交申请"}
                </Button>
                <Link className={buttonClassName({ variant: "outline" })} href="/apply/query">
                  查询申请进度
                </Link>
                <Link className={buttonClassName({ variant: "ghost" })} href="/login">
                  去登录
                </Link>
              </div>
            </form>
          </CardContent>
        </Card>
      </section>
    </main>
  );
}

function ApplicationTextField({
  label,
  value,
  onChange,
  error,
  required = false,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  required?: boolean;
}) {
  return (
    <FormField label={label} required={required} error={error}>
      <Input value={value} onChange={(event) => onChange(event.target.value)} hasError={Boolean(error)} />
    </FormField>
  );
}
