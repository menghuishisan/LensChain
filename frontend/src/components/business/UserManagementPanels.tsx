"use client";

// UserManagementPanels.tsx
// 模块01用户管理组件：用户列表、用户详情、创建编辑表单和账号操作。

import { Archive, KeyRound, Pencil, Plus, RotateCcw, Trash2, UserCheck, UserX } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/Badge";
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
import {
  useBatchDeleteUsersMutation,
  useCreateSuperAdminMutation,
  useCreateUserMutation,
  useDeleteUserMutation,
  useResetUserPasswordMutation,
  useUnlockUserMutation,
  useUpdateUserMutation,
  useUpdateUserStatusMutation,
  useUser,
  useUsers,
} from "@/hooks/useUsers";
import { getEnumText } from "@/lib/enum";
import { formatDateTime } from "@/lib/format";
import { ROLE_TEXT } from "@/lib/permissions";
import type { ID } from "@/types/api";
import type { CreateSuperAdminRequest, CreateUserRequest, ManageableUserRole, UpdateUserRequest, UserListParams, UserStatus } from "@/types/auth";

const STATUS_TEXT: Record<UserStatus, string> = {
  1: "正常",
  2: "禁用",
  3: "归档",
};

function maskPhone(phone: string | undefined | null) {
  if (!phone || phone.length < 7) {
    return phone ?? "—";
  }
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
}

type UserCreateRole = ManageableUserRole | "super_admin";

interface UserFormState extends Omit<CreateUserRequest, "role"> {
  role: UserCreateRole;
  school_id?: string | null;
}

/**
 * UserListPanel 用户管理列表组件。
 */
export function UserListPanel({ basePath, showSchoolColumn = false, headerActions }: { basePath: string; showSchoolColumn?: boolean; headerActions?: React.ReactNode }) {
  const { showToast } = useToast();
  const userBasePath = basePath;
  const [params, setParams] = useState<UserListParams>({ page: 1, page_size: 20, sort_by: "created_at", sort_order: "desc" });
  const [selectedIDs, setSelectedIDs] = useState<ID[]>([]);
  const [resetTarget, setResetTarget] = useState<ID | null>(null);
  const [newPassword, setNewPassword] = useState("");
  const query = useUsers(params);
  const updateStatusMutation = useUpdateUserStatusMutation();
  const deleteMutation = useDeleteUserMutation();
  const batchDeleteMutation = useBatchDeleteUsersMutation();
  const resetPasswordMutation = useResetUserPasswordMutation();
  const unlockMutation = useUnlockUserMutation();

  const list = query.data?.list ?? [];
  const pagination = query.data?.pagination;
  const isAllSelected = list.length > 0 && list.every((item) => selectedIDs.includes(item.id));

  const handleStatus = (id: ID, status: UserStatus, reason: string) => {
    updateStatusMutation.mutate(
      { id, payload: { status, reason } },
      {
        onSuccess: () => showToast({ title: "账号状态已更新", variant: "success" }),
        onError: (error) => showToast({ title: "状态更新失败", description: error.message, variant: "destructive" }),
      },
    );
  };

  return (
    <div className="space-y-5">
      <Card>
      <CardHeader className="flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <CardTitle>用户管理</CardTitle>
          <CardDescription>在这里查看、筛选和维护账号信息。</CardDescription>
        </div>
          <div className="flex flex-wrap gap-2">
            <Link className={buttonClassName({ variant: "primary" })} href={`${userBasePath}/create`}>
              <Plus className="h-4 w-4" />
              添加用户
            </Link>
            {headerActions}
            <ConfirmDialog
              title="确认批量删除账号"
              description="批量删除为软删除操作，请确认已选择正确账号。"
              trigger={
                <Button variant="destructive" disabled={selectedIDs.length === 0}>
                  批量删除
                </Button>
              }
              onConfirm={() =>
                batchDeleteMutation.mutate(
                  { ids: selectedIDs },
                  {
                    onSuccess: () => {
                      setSelectedIDs([]);
                      showToast({ title: "已提交批量删除", variant: "success" });
                    },
                    onError: (error) => showToast({ title: "批量删除失败", description: error.message, variant: "destructive" }),
                  },
                )
              }
            />
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <UserFilters params={params} onChange={(next) => setParams({ ...next, page: 1 })} />
          {query.isLoading ? <LoadingState /> : null}
          {query.isError ? <ErrorState description={query.error.message} /> : null}
          {!query.isLoading && !query.isError && list.length === 0 ? <EmptyState title="暂无用户" description="可通过添加用户或导入用户创建账号。" /> : null}
          {list.length > 0 ? (
            <>
              <TableContainer>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>
                        <input
                          type="checkbox"
                          checked={isAllSelected}
                          onChange={(event) => setSelectedIDs(event.target.checked ? list.map((item) => item.id) : [])}
                          aria-label="选择全部用户"
                        />
                      </TableHead>
                      <TableHead>姓名</TableHead>
                      <TableHead>手机号</TableHead>
                      <TableHead>学号/工号</TableHead>
                      {showSchoolColumn && <TableHead>学校</TableHead>}
                      <TableHead>角色</TableHead>
                      <TableHead>学院</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>最后登录</TableHead>
                      <TableHead>操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {list.map((userItem) => (
                      <TableRow key={userItem.id}>
                        <TableCell>
                          <input
                            type="checkbox"
                            checked={selectedIDs.includes(userItem.id)}
                            onChange={(event) =>
                              setSelectedIDs((current) =>
                                event.target.checked ? [...current, userItem.id] : current.filter((id) => id !== userItem.id),
                              )
                            }
                            aria-label={`选择${userItem.name}`}
                          />
                        </TableCell>
                        <TableCell className="font-semibold">{userItem.name}</TableCell>
                        <TableCell>{maskPhone(userItem.phone)}</TableCell>
                        <TableCell>{userItem.student_no ?? "—"}</TableCell>
                        {showSchoolColumn && <TableCell>{userItem.school_name}</TableCell>}
                        <TableCell>{userItem.roles.map((role) => ROLE_TEXT[role]).join(" / ")}</TableCell>
                        <TableCell>{userItem.college ?? "—"}</TableCell>
                        <TableCell>
                          <Badge variant={userItem.status === 1 ? "success" : userItem.status === 2 ? "destructive" : "secondary"}>
                            {userItem.status_text || getEnumText(STATUS_TEXT, userItem.status)}
                          </Badge>
                        </TableCell>
                        <TableCell>{formatDateTime(userItem.last_login_at)}</TableCell>
                        <TableCell>
                          <UserActions
                            id={userItem.id}
                            status={userItem.status}
                            basePath={userBasePath}
                            onStatus={handleStatus}
                            onReset={() => setResetTarget(userItem.id)}
                            onUnlock={() =>
                              unlockMutation.mutate(userItem.id, {
                                onSuccess: () => showToast({ title: "账号已解锁", variant: "success" }),
                                onError: (error) => showToast({ title: "解锁失败", description: error.message, variant: "destructive" }),
                              })
                            }
                            onDelete={() =>
                              deleteMutation.mutate(userItem.id, {
                                onSuccess: () => showToast({ title: "账号已删除", variant: "success" }),
                                onError: (error) => showToast({ title: "删除失败", description: error.message, variant: "destructive" }),
                              })
                            }
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
              {pagination ? (
                <Pagination
                  page={pagination.page}
                  totalPages={pagination.total_pages}
                  total={pagination.total}
                  onPageChange={(page) => setParams((current) => ({ ...current, page }))}
                />
              ) : null}
            </>
          ) : null}
        </CardContent>
      </Card>
      <ResetPasswordDialog
        open={resetTarget !== null}
        value={newPassword}
        onValueChange={setNewPassword}
        onOpenChange={(open) => {
          if (!open) {
            setResetTarget(null);
            setNewPassword("");
          }
        }}
        onConfirm={() => {
          if (resetTarget === null) {
            return;
          }
          resetPasswordMutation.mutate(
            { id: resetTarget, payload: { new_password: newPassword } },
            {
              onSuccess: () => {
                setResetTarget(null);
                setNewPassword("");
                showToast({ title: "密码已重置", variant: "success" });
              },
              onError: (error) => showToast({ title: "重置失败", description: error.message, variant: "destructive" }),
            },
          );
        }}
      />
    </div>
  );
}

/**
 * UserFormPanel 用户创建/编辑表单组件。
 */
export function UserFormPanel({ userID, basePath = "/admin/users", canCreateSuperAdmin = false }: { userID?: ID; basePath?: string; canCreateSuperAdmin?: boolean }) {
  const router = useRouter();
  const { showToast } = useToast();
  const detailQuery = useUser(userID ?? "");
  const createMutation = useCreateUserMutation();
  const createSuperAdminMutation = useCreateSuperAdminMutation();
  const updateMutation = useUpdateUserMutation(userID ?? "");
  const userBasePath = basePath;
  const isEdit = userID !== undefined;
  const detail = detailQuery.data;
  const [form, setForm] = useState<UserFormState>({
    phone: "",
    name: "",
    password: "",
    role: "student",
    student_no: "",
    college: "",
    major: "",
    class_name: "",
    education_level: null,
    email: "",
    remark: "",
    school_id: "",
  });
  useEffect(() => {
    if (detail === undefined) {
      return;
    }
    setForm({
      phone: detail.phone,
      name: detail.name,
      password: "",
      role: detail.roles.includes("teacher") ? "teacher" : detail.roles.includes("super_admin") ? "super_admin" : "student",
      student_no: detail.student_no,
      college: detail.college,
      major: detail.major,
      class_name: detail.class_name,
      education_level: detail.education_level,
      email: detail.email ?? "",
      remark: detail.remark ?? "",
      school_id: detail.school_id ?? "",
    });
  }, [detail]);

  if (isEdit && detailQuery.isLoading) {
    return <LoadingState />;
  }

  if (isEdit && detailQuery.isError) {
    return <ErrorState description={detailQuery.error.message} />;
  }

  const canSubmit =
    form.phone.trim().length === 11 &&
    form.name.trim().length > 0 &&
    (isEdit || form.password.length >= 8);
  const submitPayload: UpdateUserRequest = {
    name: form.name,
    student_no: form.student_no,
    college: form.college,
    major: form.major,
    class_name: form.class_name,
    education_level: form.education_level,
    email: form.email,
    remark: form.remark,
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{isEdit ? "编辑用户" : "添加用户"}</CardTitle>
        <CardDescription>手动创建账号时必须显式设置初始密码。</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-4 lg:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            if (!canSubmit) {
              return;
            }
            if (isEdit && userID !== undefined) {
              updateMutation.mutate(submitPayload, {
                onSuccess: () => {
                  showToast({ title: "用户信息已更新", variant: "success" });
                  router.push(`${userBasePath}/${userID}`);
                },
                onError: (error) => showToast({ title: "更新失败", description: error.message, variant: "destructive" }),
              });
              return;
            }

            if (form.role === "super_admin") {
              const payload: CreateSuperAdminRequest = {
                phone: form.phone,
                name: form.name,
                password: form.password,
                school_id: "0",
                email: form.email,
                remark: form.remark,
              };
              createSuperAdminMutation.mutate(payload, {
                onSuccess: (created) => {
                  showToast({ title: "超级管理员已创建", variant: "success" });
                  router.push(`${userBasePath}/${created.id}`);
                },
                onError: (error) => showToast({ title: "创建失败", description: error.message, variant: "destructive" }),
              });
              return;
            }

            const role: ManageableUserRole = form.role;
            const createPayload: CreateUserRequest = {
              phone: form.phone,
              name: form.name,
              password: form.password,
              role,
              student_no: form.student_no,
              college: form.college,
              major: form.major,
              class_name: form.class_name,
              education_level: form.education_level,
              email: form.email,
              remark: form.remark,
            };
            createMutation.mutate(createPayload, {
              onSuccess: (created) => {
                showToast({ title: "用户已创建", variant: "success" });
                router.push(`${userBasePath}/${created.id}`);
              },
              onError: (error) => showToast({ title: "创建失败", description: error.message, variant: "destructive" }),
            });
          }}
        >
          <TextInput label="手机号" value={form.phone} onChange={(phone) => setForm((current) => ({ ...current, phone }))} required={!isEdit} disabled={isEdit} />
          <TextInput label="姓名" value={form.name} onChange={(name) => setForm((current) => ({ ...current, name }))} required />
          {!isEdit ? <TextInput label="初始密码" value={form.password} onChange={(password) => setForm((current) => ({ ...current, password }))} required type="password" /> : null}
          <FormField label="角色" required description={isEdit ? "账号创建后，角色暂不支持在此页面直接修改。" : undefined}>
            <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={form.role} disabled={isEdit} onChange={(event) => setForm((current) => ({ ...current, role: event.target.value as UserCreateRole }))}>
              <option value="student">学生</option>
              <option value="teacher">教师</option>
              {canCreateSuperAdmin ? <option value="super_admin">超级管理员</option> : null}
            </select>
          </FormField>
          <TextInput label="学号/工号" value={form.student_no ?? ""} onChange={(student_no) => setForm((current) => ({ ...current, student_no }))} />
          <TextInput label="学院" value={form.college ?? ""} onChange={(college) => setForm((current) => ({ ...current, college }))} />
          <TextInput label="专业" value={form.major ?? ""} onChange={(major) => setForm((current) => ({ ...current, major }))} />
          <TextInput label="班级" value={form.class_name ?? ""} onChange={(class_name) => setForm((current) => ({ ...current, class_name }))} />
          <TextInput label="邮箱" value={form.email ?? ""} onChange={(email) => setForm((current) => ({ ...current, email }))} />
          <FormField label="学业层次">
            <select
              className="h-10 rounded-lg border border-input bg-background px-3 text-sm"
              value={form.education_level ?? ""}
              onChange={(event) => setForm((current) => ({ ...current, education_level: event.target.value ? Number(event.target.value) : null }))}
            >
              <option value="">未填写</option>
              <option value="1">专科</option>
              <option value="2">本科</option>
              <option value="3">硕士</option>
              <option value="4">博士</option>
            </select>
          </FormField>
          <FormField label="备注" className="lg:col-span-2">
            <Textarea value={form.remark ?? ""} onChange={(event) => setForm((current) => ({ ...current, remark: event.target.value }))} />
          </FormField>
          <div className="flex gap-3 lg:col-span-2">
            <Button type="submit" disabled={!canSubmit} isLoading={createMutation.isPending || createSuperAdminMutation.isPending || updateMutation.isPending}>
              {isEdit ? "保存修改" : "创建用户"}
            </Button>
            <Link className={buttonClassName({ variant: "outline" })} href={userBasePath}>
              返回列表
            </Link>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

/**
 * UserDetailPanel 用户详情组件。
 */
export function UserDetailPanel({ userID, basePath = "/admin/users" }: { userID: ID; basePath?: string }) {
  const userBasePath = basePath;
  const query = useUser(userID);

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }
  if (query.data === undefined) {
    return <EmptyState title="用户不存在" description="该用户可能已被删除或无权访问。" />;
  }

  const user = query.data;

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader className="flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <CardTitle>{user.name}</CardTitle>
            <CardDescription>{user.school_name ?? "学校信息不可见"} · {maskPhone(user.phone)}</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Link className={buttonClassName({ variant: "primary" })} href={`${userBasePath}/${userID}/edit`}>
              <Pencil className="h-4 w-4" />
              编辑信息
            </Link>
            <InlineUserDetailActions userID={userID} status={user.status} />
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {[
            ["学号/工号", user.student_no ?? "—"],
            ["角色", user.roles.map((role) => ROLE_TEXT[role]).join(" / ")],
            ["状态", user.status_text],
            ["学院", user.college ?? "—"],
            ["专业", user.major ?? "—"],
            ["班级", user.class_name ?? "—"],
            ["学业层次", user.education_level_text ?? "—"],
            ["邮箱", user.email ?? "—"],
            ["创建时间", formatDateTime(user.created_at)],
          ].map(([label, value]) => (
            <div key={label} className="rounded-xl bg-muted/60 p-4">
              <p className="text-xs text-muted-foreground">{label}</p>
              <p className="mt-1 text-sm font-semibold">{value}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

function UserFilters({ params, onChange }: { params: UserListParams; onChange: (params: UserListParams) => void }) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
      <Input placeholder="搜索姓名、手机号、学号" value={params.keyword ?? ""} onChange={(event) => onChange({ ...params, keyword: event.target.value })} />
      <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={params.status ?? ""} onChange={(event) => onChange({ ...params, status: event.target.value ? (Number(event.target.value) as UserStatus) : undefined })}>
        <option value="">全部状态</option>
        <option value="1">正常</option>
        <option value="2">禁用</option>
        <option value="3">归档</option>
      </select>
      <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={params.role ?? ""} onChange={(event) => onChange({ ...params, role: event.target.value === "" ? undefined : (event.target.value as "teacher" | "student") })}>
        <option value="">全部角色</option>
        <option value="teacher">教师</option>
        <option value="student">学生</option>
      </select>
      <Input placeholder="学院" value={params.college ?? ""} onChange={(event) => onChange({ ...params, college: event.target.value })} />
      <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={params.education_level ?? ""} onChange={(event) => onChange({ ...params, education_level: event.target.value ? Number(event.target.value) : undefined })}>
        <option value="">全部层次</option>
        <option value="1">专科</option>
        <option value="2">本科</option>
        <option value="3">硕士</option>
        <option value="4">博士</option>
      </select>
    </div>
  );
}

function UserActions({
  id,
  status,
  basePath,
  onStatus,
  onReset,
  onUnlock,
  onDelete,
}: {
  id: ID;
  status: UserStatus;
  basePath: string;
  onStatus: (id: ID, status: UserStatus, reason: string) => void;
  onReset: () => void;
  onUnlock: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      <Link className={buttonClassName({ variant: "outline", size: "sm" })} href={`${basePath}/${id}`}>详情</Link>
      <Button size="sm" variant="outline" onClick={onReset}><KeyRound className="h-4 w-4" />重置</Button>
      {status === 1 ? (
        <>
          <ConfirmDialog title="确认禁用账号" description="禁用后该用户当前会话立即失效。" trigger={<Button size="sm" variant="outline"><UserX className="h-4 w-4" />禁用</Button>} onConfirm={() => onStatus(id, 2, "管理员禁用")} />
          <ConfirmDialog title="确认归档账号" description="归档后账号无法登录，历史数据保留。" trigger={<Button size="sm" variant="outline"><Archive className="h-4 w-4" />归档</Button>} onConfirm={() => onStatus(id, 3, "管理员归档")} />
        </>
      ) : (
        <ConfirmDialog title="确认启用账号" description="启用后用户可重新登录。" trigger={<Button size="sm" variant="outline"><UserCheck className="h-4 w-4" />启用</Button>} onConfirm={() => onStatus(id, 1, "管理员启用")} />
      )}
      <Button size="sm" variant="outline" onClick={onUnlock}><RotateCcw className="h-4 w-4" />解锁</Button>
      <ConfirmDialog title="确认删除账号" description="删除后，历史学习和操作记录仍会保留。" trigger={<Button size="sm" variant="destructive"><Trash2 className="h-4 w-4" />删除</Button>} onConfirm={onDelete} />
    </div>
  );
}

function ResetPasswordDialog({
  open,
  value,
  onValueChange,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  value: string;
  onValueChange: (value: string) => void;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>重置用户密码</DialogTitle>
          <DialogDescription>重置后用户下次登录需强制修改密码，并且历史会话立即失效。</DialogDescription>
        </DialogHeader>
        <FormField label="新密码" required>
          <Input type="password" value={value} onChange={(event) => onValueChange(event.target.value)} />
        </FormField>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button type="button" disabled={value.length < 8} onClick={onConfirm}>确认重置</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function InlineUserDetailActions({ userID, status }: { userID: ID; status: UserStatus }) {
  const { showToast } = useToast();
  const updateStatusMutation = useUpdateUserStatusMutation();
  const resetPasswordMutation = useResetUserPasswordMutation();
  const unlockMutation = useUnlockUserMutation();
  const deleteMutation = useDeleteUserMutation();
  const [open, setOpen] = useState(false);
  const [newPassword, setNewPassword] = useState("");

  return (
    <>
      <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
        <KeyRound className="h-4 w-4" />
        重置密码
      </Button>
      {status === 1 ? (
        <>
          <ConfirmDialog
            title="确认禁用账号"
            description="禁用后该用户当前会话立即失效，登录会被拒绝。"
            trigger={
              <Button size="sm" variant="outline">
                <UserX className="h-4 w-4" />
                禁用
              </Button>
            }
            onConfirm={() =>
              updateStatusMutation.mutate(
                { id: userID, payload: { status: 2, reason: "管理员禁用" } },
                {
                  onSuccess: () => showToast({ title: "账号已禁用", variant: "success" }),
                  onError: (error) => showToast({ title: "禁用失败", description: error.message, variant: "destructive" }),
                },
              )
            }
          />
          <ConfirmDialog
            title="确认归档账号"
            description="归档后账号无法登录，但历史学习数据会保留。"
            trigger={
              <Button size="sm" variant="outline">
                <Archive className="h-4 w-4" />
                归档
              </Button>
            }
            onConfirm={() =>
              updateStatusMutation.mutate(
                { id: userID, payload: { status: 3, reason: "管理员归档" } },
                {
                  onSuccess: () => showToast({ title: "账号已归档", variant: "success" }),
                  onError: (error) => showToast({ title: "归档失败", description: error.message, variant: "destructive" }),
                },
              )
            }
          />
        </>
      ) : (
        <ConfirmDialog
          title="确认启用账号"
          description="启用后用户可重新登录。"
          trigger={
            <Button size="sm" variant="outline">
              <UserCheck className="h-4 w-4" />
              启用
            </Button>
          }
          onConfirm={() =>
            updateStatusMutation.mutate(
              { id: userID, payload: { status: 1, reason: "管理员启用" } },
              {
                onSuccess: () => showToast({ title: "账号已启用", variant: "success" }),
                onError: (error) => showToast({ title: "启用失败", description: error.message, variant: "destructive" }),
              },
            )
          }
        />
      )}
      <Button
        size="sm"
        variant="outline"
        onClick={() =>
          unlockMutation.mutate(userID, {
            onSuccess: () => showToast({ title: "账号已解锁", variant: "success" }),
            onError: (error) => showToast({ title: "解锁失败", description: error.message, variant: "destructive" }),
          })
        }
      >
        <RotateCcw className="h-4 w-4" />
        解锁
      </Button>
      <ConfirmDialog
        title="确认删除账号"
        description="删除后，历史学习和操作记录仍会保留。"
        trigger={
          <Button size="sm" variant="destructive">
            <Trash2 className="h-4 w-4" />
            删除
          </Button>
        }
        onConfirm={() =>
          deleteMutation.mutate(userID, {
            onSuccess: () => showToast({ title: "账号已删除", variant: "success" }),
            onError: (error) => showToast({ title: "删除失败", description: error.message, variant: "destructive" }),
          })
        }
      />
      <ResetPasswordDialog
        open={open}
        value={newPassword}
        onValueChange={setNewPassword}
        onOpenChange={(nextOpen) => {
          setOpen(nextOpen);
          if (!nextOpen) {
            setNewPassword("");
          }
        }}
        onConfirm={() =>
          resetPasswordMutation.mutate(
            { id: userID, payload: { new_password: newPassword } },
            {
              onSuccess: () => {
                setOpen(false);
                setNewPassword("");
                showToast({ title: "密码已重置", variant: "success" });
              },
              onError: (error) => showToast({ title: "重置失败", description: error.message, variant: "destructive" }),
            },
          )
        }
      />
    </>
  );
}

function TextInput({ label, value, onChange, required = false, type = "text", disabled = false }: { label: string; value: string; onChange: (value: string) => void; required?: boolean; type?: string; disabled?: boolean }) {
  return (
    <FormField label={label} required={required}>
      <Input type={type} value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)} />
    </FormField>
  );
}
