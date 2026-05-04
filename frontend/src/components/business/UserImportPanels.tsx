"use client";

// UserImportPanels.tsx
// 模块01用户导入组件，覆盖模板下载、文件上传预览、冲突策略和导入结果反馈。

import { FileSpreadsheet, UploadCloud } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/Dialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { useToast } from "@/components/ui/Toast";
import {
  useDownloadUserImportFailuresMutation,
  useDownloadUserImportTemplateMutation,
  useExecuteUserImportMutation,
  usePreviewUserImportMutation,
} from "@/hooks/useUserImport";
import { validateUserImportFile } from "@/lib/auth-validation";
import type {
  ExecuteUserImportResponse,
  UserImportConflictStrategy,
  UserImportPreviewResponse,
  UserImportPreviewStatus,
  UserImportType,
} from "@/types/auth";

const PREVIEW_STORAGE_KEY = "lenschain-user-import-preview";

/**
 * UserImportPanel 用户导入上传组件。
 */
export function UserImportPanel() {
  const router = useRouter();
  const { showToast } = useToast();
  const userBasePath = "/admin/users";
  const [type, setType] = useState<UserImportType>("student");
  const [file, setFile] = useState<File | null>(null);
  const templateMutation = useDownloadUserImportTemplateMutation();
  const previewMutation = usePreviewUserImportMutation();
  const validation = validateUserImportFile(file);

  return (
    <Card>
      <CardHeader>
        <CardTitle>导入用户</CardTitle>
        <CardDescription>支持学生和教师 Excel/CSV 批量导入；上传后先进入预览与冲突处理。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex flex-wrap gap-3">
          <label className="flex items-center gap-2 rounded-xl border border-border bg-card px-4 py-3 text-sm font-semibold">
            <input type="radio" checked={type === "student"} onChange={() => setType("student")} />
            学生
          </label>
          <label className="flex items-center gap-2 rounded-xl border border-border bg-card px-4 py-3 text-sm font-semibold">
            <input type="radio" checked={type === "teacher"} onChange={() => setType("teacher")} />
            教师
          </label>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <div className="rounded-xl border border-dashed border-border bg-muted/50 p-6">
            <FileSpreadsheet className="mb-4 h-9 w-9 text-primary" />
            <h3 className="font-display text-xl font-semibold">第一步：下载模板</h3>
            <p className="mt-2 text-sm text-muted-foreground">模板包含字段说明和示例数据，初始密码为必填。</p>
            <Button className="mt-5" type="button" variant="outline" isLoading={templateMutation.isPending} onClick={() => templateMutation.mutate(type)}>
              下载{type === "student" ? "学生" : "教师"}导入模板
            </Button>
          </div>

          <div className="rounded-xl border border-dashed border-border bg-muted/50 p-6">
            <UploadCloud className="mb-4 h-9 w-9 text-primary" />
            <h3 className="font-display text-xl font-semibold">第二步：上传并预览</h3>
            <p className="mt-2 text-sm text-muted-foreground">仅支持 .xlsx / .csv，单文件不超过 50MB。</p>
            <input
              className="mt-5 block w-full text-sm"
              type="file"
              accept=".xlsx,.csv"
              onChange={(event) => setFile(event.target.files?.[0] ?? null)}
            />
            {validation.errors.file ? <p className="mt-2 text-sm text-destructive">{validation.errors.file}</p> : null}
            <Button
              className="mt-5"
              type="button"
              disabled={!validation.isValid}
              isLoading={previewMutation.isPending}
              onClick={() => {
                if (file === null || !validation.isValid) {
                  return;
                }
                previewMutation.mutate(
                  { file, type },
                  {
                    onSuccess: (preview) => {
                      sessionStorage.setItem(PREVIEW_STORAGE_KEY, JSON.stringify(preview));
                      showToast({ title: "文件解析完成", variant: "success" });
                      router.push(`${userBasePath}/import/preview`);
                    },
                    onError: (error) => showToast({ title: "预览失败", description: error.message, variant: "destructive" }),
                  },
                );
              }}
            >
              上传并预览
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * UserImportPreviewPanel 导入预览与确认组件。
 */
export function UserImportPreviewPanel() {
  const { showToast } = useToast();
  const userBasePath = "/admin/users";
  const [preview, setPreview] = useState<UserImportPreviewResponse | null>(null);
  const [filter, setFilter] = useState<"all" | UserImportPreviewStatus>("all");
  const [strategy, setStrategy] = useState<UserImportConflictStrategy>("skip");
  const [overrides, setOverrides] = useState<string[]>([]);
  const [result, setResult] = useState<ExecuteUserImportResponse | null>(null);
  const executeMutation = useExecuteUserImportMutation();
  const failureMutation = useDownloadUserImportFailuresMutation();

  useEffect(() => {
    const raw = sessionStorage.getItem(PREVIEW_STORAGE_KEY);
    if (raw === null) {
      return;
    }

    const parsed: unknown = JSON.parse(raw);
    if (isImportPreview(parsed)) {
      setPreview(parsed);
    }
  }, []);

  const rows = useMemo(() => {
    if (preview === null) {
      return [];
    }
    return filter === "all" ? preview.preview_list : preview.preview_list.filter((row) => row.status === filter);
  }, [filter, preview]);

  if (preview === null) {
    return <EmptyState title="暂无导入预览" description="请先上传 Excel/CSV 文件并完成预览。" action={<Link className={buttonClassName({ variant: "primary" })} href={`${userBasePath}/import`}>返回导入页</Link>} />;
  }

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>导入预览</CardTitle>
          <CardDescription>
            解析结果：总计 {preview.total} 条 | 有效 {preview.valid} | 无效 {preview.invalid} | 冲突 {preview.conflict}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-sm font-semibold">冲突处理策略：</span>
            <label className="flex items-center gap-2 text-sm"><input type="radio" checked={strategy === "skip"} onChange={() => setStrategy("skip")} />全部跳过</label>
            <label className="flex items-center gap-2 text-sm"><input type="radio" checked={strategy === "overwrite"} onChange={() => setStrategy("overwrite")} />全部覆盖</label>
          </div>
          <Tabs value={filter} onValueChange={(value) => setFilter(value as "all" | UserImportPreviewStatus)}>
            <TabsList>
              <TabsTrigger value="all">全部</TabsTrigger>
              <TabsTrigger value="valid">有效</TabsTrigger>
              <TabsTrigger value="invalid">无效</TabsTrigger>
              <TabsTrigger value="conflict">冲突</TabsTrigger>
            </TabsList>
            <TabsContent value={filter}>
              <TableContainer>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>行号</TableHead>
                      <TableHead>姓名</TableHead>
                      <TableHead>手机号</TableHead>
                      <TableHead>学号/工号</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>说明</TableHead>
                      <TableHead>逐条覆盖</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rows.map((row) => (
                      <TableRow key={`${row.row}-${row.phone}`}>
                        <TableCell>{row.row}</TableCell>
                        <TableCell>{row.name || "—"}</TableCell>
                        <TableCell>{row.phone}</TableCell>
                        <TableCell>{row.student_no}</TableCell>
                        <TableCell><PreviewStatusBadge status={row.status} /></TableCell>
                        <TableCell>{row.message ?? "—"}</TableCell>
                        <TableCell>
                          {row.status === "conflict" ? (
                            <input
                              type="checkbox"
                              checked={overrides.includes(row.phone)}
                              onChange={(event) =>
                                setOverrides((current) =>
                                  event.target.checked ? [...current, row.phone] : current.filter((phone) => phone !== row.phone),
                                )
                              }
                              aria-label={`覆盖${row.phone}`}
                            />
                          ) : (
                            "—"
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </TabsContent>
          </Tabs>
          <div className="flex justify-between">
            <Link className={buttonClassName({ variant: "outline" })} href={`${userBasePath}/import`}>取消</Link>
            <Button
              type="button"
              disabled={preview.valid === 0 || preview.invalid > 0}
              isLoading={executeMutation.isPending}
              onClick={() =>
                executeMutation.mutate(
                  { import_id: preview.import_id, conflict_strategy: strategy, conflict_overrides: overrides },
                  {
                    onSuccess: (executeResult) => {
                      setResult(executeResult);
                      sessionStorage.removeItem(PREVIEW_STORAGE_KEY);
                    },
                    onError: (error) => showToast({ title: "导入失败", description: error.message, variant: "destructive" }),
                  },
                )
              }
            >
              确认导入
            </Button>
          </div>
          {preview.invalid > 0 ? <ErrorState title="存在无效数据" description="无效数据不可导入，请修正文件后重新上传预览。" /> : null}
        </CardContent>
      </Card>
      <Dialog open={result !== null} onOpenChange={(open) => !open && setResult(null)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>导入完成</DialogTitle>
          <DialogDescription>系统已经完成导入，并生成本次结果统计。</DialogDescription>
        </DialogHeader>
          {result ? (
            <div className="grid grid-cols-2 gap-3">
              <Stat label="成功" value={result.success_count} />
              <Stat label="失败" value={result.fail_count} />
              <Stat label="跳过" value={result.skip_count} />
              <Stat label="覆盖" value={result.overwrite_count} />
            </div>
          ) : null}
          <DialogFooter>
            {result && result.fail_count > 0 ? (
              <Button variant="outline" isLoading={failureMutation.isPending} onClick={() => failureMutation.mutate(result.import_id)}>
                下载失败明细
              </Button>
            ) : null}
            <Link className={buttonClassName({ variant: "primary" })} href={userBasePath}>完成</Link>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function PreviewStatusBadge({ status }: { status: UserImportPreviewStatus }) {
  if (status === "valid") {
    return <Badge variant="success">有效</Badge>;
  }
  if (status === "conflict") {
    return <Badge variant="secondary">冲突</Badge>;
  }
  return <Badge variant="destructive">无效</Badge>;
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-xl bg-muted/60 p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 font-display text-2xl font-semibold">{value}</p>
    </div>
  );
}

function isImportPreview(value: unknown): value is UserImportPreviewResponse {
  return typeof value === "object" && value !== null && "import_id" in value && "preview_list" in value;
}
