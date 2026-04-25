"use client";

// SchoolTable.tsx
// 模块02超管学校管理组件，包含学校列表、学校详情/编辑、授权有效期和生命周期操作。

import Link from "next/link";
import { useEffect, useState } from "react";

import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
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
import { SchoolStatusBadge } from "@/components/business/SchoolStatusBadge";
import {
  useCreateSchoolMutation,
  useSchool,
  useSchoolLifecycleMutations,
  useSchools,
  useUpdateSchoolLicenseMutation,
  useUpdateSchoolMutation,
} from "@/hooks/useSchools";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";
import type { CreateSchoolRequest, SchoolListParams, SchoolStatus, UpdateSchoolRequest } from "@/types/school";

const EMPTY_SCHOOL: CreateSchoolRequest = {
  name: "",
  code: "",
  address: "",
  website: "",
  logo_url: "",
  description: "",
  license_start_at: "",
  license_end_at: "",
  contact_name: "",
  contact_phone: "",
  contact_email: "",
  contact_title: "",
};

function maskSchoolPhone(phone: string | null | undefined) {
  if (!phone || phone.length < 7) {
    return phone ?? "—";
  }
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
}

/**
 * SchoolTable 学校管理列表组件。
 */
export function SchoolTable() {
  const [params, setParams] = useState<SchoolListParams>({ page: 1, page_size: 20 });
  const [licenseTarget, setLicenseTarget] = useState<ID | null>(null);
  const [licenseEndAt, setLicenseEndAt] = useState("");
  const [freezeTarget, setFreezeTarget] = useState<ID | null>(null);
  const [freezeReason, setFreezeReason] = useState("");
  const query = useSchools(params);
  const lifecycle = useSchoolLifecycleMutations();
  const licenseMutation = useUpdateSchoolLicenseMutation(licenseTarget ?? "");
  const { showToast } = useToast();
  const list = query.data?.list ?? [];

  return (
    <Card>
      <CardHeader className="flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle>学校管理</CardTitle>
          <CardDescription>查看、搜索和筛选学校信息，管理服务有效期与当前状态。</CardDescription>
        </div>
        <Link className={buttonClassName({ variant: "primary" })} href="/admin/schools/create">创建学校</Link>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-[1fr_12rem_auto]">
          <Input placeholder="搜索学校名称或编码" value={params.keyword ?? ""} onChange={(event) => setParams((current) => ({ ...current, keyword: event.target.value, page: 1 }))} />
          <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={params.status ?? ""} onChange={(event) => setParams((current) => ({ ...current, status: event.target.value ? (Number(event.target.value) as SchoolStatus) : undefined, page: 1 }))}>
            <option value="">全部状态</option>
            <option value="2">已激活</option>
            <option value="3">缓冲期</option>
            <option value="4">已冻结</option>
            <option value="5">已注销</option>
          </select>
          <label className="flex items-center gap-2 rounded-lg border border-border px-3 text-sm">
            <input type="checkbox" checked={Boolean(params.license_expiring)} onChange={(event) => setParams((current) => ({ ...current, license_expiring: event.target.checked || undefined, page: 1 }))} />
            即将到期
          </label>
        </div>
        {query.isLoading ? <LoadingState /> : null}
        {query.isError ? <ErrorState description={query.error.message} /> : null}
        {!query.isLoading && !query.isError && list.length === 0 ? <EmptyState title="暂无学校" description="可通过创建学校或入驻申请审核通过生成学校记录。" /> : null}
        {list.length > 0 ? (
          <TableContainer>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>学校</TableHead>
                  <TableHead>编码</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>有效期至</TableHead>
                  <TableHead>剩余天数</TableHead>
                  <TableHead>联系人</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {list.map((school) => (
                  <TableRow key={school.id}>
                    <TableCell className="font-semibold">{school.name}</TableCell>
                    <TableCell>{school.code}</TableCell>
                    <TableCell><SchoolStatusBadge status={school.status} text={school.status_text} /></TableCell>
                    <TableCell>{formatDateTime(school.license_end_at)}</TableCell>
                    <TableCell>{school.license_remaining_days === null ? "—" : `${school.license_remaining_days}天`}</TableCell>
                    <TableCell>{school.contact_name} · {maskSchoolPhone(school.contact_phone)}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-2">
                        <Link className={buttonClassName({ variant: "outline", size: "sm" })} href={`/admin/schools/${school.id}`}>详情/编辑</Link>
                        <Button size="sm" variant="outline" onClick={() => setLicenseTarget(school.id)}>有效期</Button>
                        {school.status === 4 ? (
                          <Button size="sm" variant="outline" onClick={() => lifecycle.unfreeze.mutate(school.id, { onSuccess: () => showToast({ title: "学校已解冻", variant: "success" }) })}>解冻</Button>
                        ) : (
                          <Button size="sm" variant="outline" onClick={() => setFreezeTarget(school.id)}>冻结</Button>
                        )}
                        {school.status === 5 ? (
                          <ConfirmDialog title="恢复学校" description="恢复后学校和关联用户将同时恢复。" trigger={<Button size="sm" variant="outline">恢复</Button>} onConfirm={() => lifecycle.restore.mutate(school.id, { onSuccess: () => showToast({ title: "学校已恢复", variant: "success" }) })} />
                        ) : (
                          <ConfirmDialog title="注销学校" description="注销学校会软删除学校及关联用户，并使该校用户会话立即失效。" trigger={<Button size="sm" variant="destructive">注销</Button>} onConfirm={() => lifecycle.cancel.mutate(school.id, { onSuccess: () => showToast({ title: "学校已注销", variant: "success" }) })} />
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : null}
        {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={(page) => setParams((current) => ({ ...current, page }))} /> : null}
      </CardContent>
      <Dialog open={licenseTarget !== null} onOpenChange={(open) => !open && setLicenseTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>设置授权有效期</DialogTitle>
            <DialogDescription>学校服务需要设置有效期，到期后会进入受限状态，便于后续处理。</DialogDescription>
          </DialogHeader>
          <Input type="datetime-local" value={licenseEndAt} onChange={(event) => setLicenseEndAt(event.target.value)} />
          <DialogFooter>
            <Button variant="outline" onClick={() => setLicenseTarget(null)}>取消</Button>
            <Button
              disabled={!licenseEndAt}
              isLoading={licenseMutation.isPending}
              onClick={() =>
                licenseMutation.mutate(
                  { license_end_at: new Date(licenseEndAt).toISOString() },
                  { onSuccess: () => { setLicenseTarget(null); showToast({ title: "授权有效期已更新", variant: "success" }); } },
                )
              }
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={freezeTarget !== null} onOpenChange={(open) => !open && setFreezeTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>冻结学校</DialogTitle>
            <DialogDescription>冻结后该校用户将暂时无法继续使用平台。</DialogDescription>
          </DialogHeader>
          <Textarea value={freezeReason} onChange={(event) => setFreezeReason(event.target.value)} placeholder="请输入冻结原因" />
          <DialogFooter>
            <Button variant="outline" onClick={() => setFreezeTarget(null)}>取消</Button>
            <Button
              variant="destructive"
              disabled={!freezeReason.trim()}
              onClick={() => {
                if (freezeTarget !== null) {
                  lifecycle.freeze.mutate({ id: freezeTarget, reason: freezeReason }, { onSuccess: () => { setFreezeTarget(null); setFreezeReason(""); showToast({ title: "学校已冻结", variant: "success" }); } });
                }
              }}
            >
              确认冻结
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

/**
 * SchoolFormPanel 学校创建或详情编辑组件。
 */
export function SchoolFormPanel({ schoolID }: { schoolID?: ID }) {
  const isEdit = schoolID !== undefined;
  const query = useSchool(schoolID ?? "");
  const createMutation = useCreateSchoolMutation();
  const updateMutation = useUpdateSchoolMutation(schoolID ?? "");
  const { showToast } = useToast();
  const [form, setForm] = useState<CreateSchoolRequest>(EMPTY_SCHOOL);

  useEffect(() => {
    if (query.data === undefined) {
      return;
    }
    setForm({
      name: query.data.name,
      code: query.data.code,
      address: query.data.address,
      website: query.data.website,
      logo_url: query.data.logo_url,
      description: query.data.description,
      license_start_at: query.data.license_start_at ?? "",
      license_end_at: query.data.license_end_at ?? "",
      contact_name: query.data.contact_name,
      contact_phone: query.data.contact_phone,
      contact_email: query.data.contact_email,
      contact_title: query.data.contact_title,
    });
  }, [query.data]);

  if (isEdit && query.isLoading) {
    return <LoadingState />;
  }
  if (isEdit && query.isError) {
    return <ErrorState description={query.error.message} />;
  }

  const canSubmit =
    form.name.trim().length > 0 &&
    form.code.trim().length > 0 &&
    form.license_start_at.trim().length > 0 &&
    form.license_end_at.trim().length > 0 &&
    form.contact_name.trim().length > 0 &&
    form.contact_phone.trim().length > 0;
  const updatePayload: UpdateSchoolRequest = form;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{isEdit ? "学校详情/编辑" : "创建学校"}</CardTitle>
        <CardDescription>{isEdit ? "可编辑学校基础信息，并单独调整服务有效期。" : "直接创建后，系统会为学校生成首个管理员账号。"}</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-4 md:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            if (!canSubmit) {
              return;
            }
            if (isEdit && schoolID !== undefined) {
              updateMutation.mutate(updatePayload, { onSuccess: () => showToast({ title: "学校信息已更新", variant: "success" }) });
              return;
            }
            createMutation.mutate(form, { onSuccess: () => showToast({ title: "学校已创建", variant: "success" }) });
          }}
        >
          <TextInput label="学校名称" value={form.name} onChange={(name) => setForm((current) => ({ ...current, name }))} required />
          <TextInput label="学校编码" value={form.code} onChange={(code) => setForm((current) => ({ ...current, code }))} required />
          <TextInput label="学校地址" value={form.address ?? ""} onChange={(address) => setForm((current) => ({ ...current, address }))} />
          <TextInput label="学校官网" value={form.website ?? ""} onChange={(website) => setForm((current) => ({ ...current, website }))} />
          <TextInput label="LOGO URL" value={form.logo_url ?? ""} onChange={(logo_url) => setForm((current) => ({ ...current, logo_url }))} />
          <TextInput label="联系人姓名" value={form.contact_name} onChange={(contact_name) => setForm((current) => ({ ...current, contact_name }))} required />
          <TextInput label="联系人手机号" value={form.contact_phone} onChange={(contact_phone) => setForm((current) => ({ ...current, contact_phone }))} required />
          <TextInput label="联系人邮箱" value={form.contact_email ?? ""} onChange={(contact_email) => setForm((current) => ({ ...current, contact_email }))} />
          <TextInput label="职务/部门" value={form.contact_title ?? ""} onChange={(contact_title) => setForm((current) => ({ ...current, contact_title }))} />
          <FormField label="授权开始时间" required>
            <Input type="datetime-local" value={toLocalDateTime(form.license_start_at)} onChange={(event) => setForm((current) => ({ ...current, license_start_at: event.target.value ? new Date(event.target.value).toISOString() : "" }))} />
          </FormField>
          <FormField label="授权结束时间" required>
            <Input type="datetime-local" value={toLocalDateTime(form.license_end_at)} onChange={(event) => setForm((current) => ({ ...current, license_end_at: event.target.value ? new Date(event.target.value).toISOString() : "" }))} />
          </FormField>
          <FormField label="学校简介" className="md:col-span-2">
            <Textarea value={form.description ?? ""} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
          </FormField>
          {!isEdit ? (
            <div className="rounded-xl border border-border bg-muted/25 p-4 text-sm text-muted-foreground md:col-span-2">
              创建学校成功后，系统会直接生成首个学校管理员账号，并通过短信发送初始密码。
            </div>
          ) : null}
          <div className="flex gap-3 md:col-span-2">
            <Button type="submit" disabled={!canSubmit} isLoading={createMutation.isPending || updateMutation.isPending}>{isEdit ? "保存修改" : "创建学校"}</Button>
            <Link className={buttonClassName({ variant: "outline" })} href="/admin/schools">返回列表</Link>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function TextInput({ label, value, onChange, required = false }: { label: string; value: string; onChange: (value: string) => void; required?: boolean }) {
  return (
    <FormField label={label} required={required}>
      <Input value={value} onChange={(event) => onChange(event.target.value)} />
    </FormField>
  );
}

function toLocalDateTime(value: string | null | undefined) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return date.toISOString().slice(0, 16);
}
