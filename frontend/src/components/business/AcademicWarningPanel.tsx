"use client";

// AcademicWarningPanel.tsx
// 模块06学业预警组件，展示预警列表、详情和处理动作。

import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Textarea } from "@/components/ui/Textarea";
import { useAcademicWarning, useAcademicWarningMutations, useAcademicWarnings, useWarningConfig } from "@/hooks/useAcademicWarnings";
import { ACADEMIC_WARNING_STATUS_OPTIONS, ACADEMIC_WARNING_TYPE_OPTIONS, getAcademicWarningStatusVariant } from "@/lib/grade";
import { formatDateTime, formatGPA } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * AcademicWarningPanel 组件属性。
 */
export interface AcademicWarningPanelProps {
  warningID?: ID;
}

/**
 * AcademicWarningPanel 学业预警组件。
 */
export function AcademicWarningPanel({ warningID }: AcademicWarningPanelProps) {
  const warningsQuery = useAcademicWarnings({ page: 1, page_size: 20 });
  const warningQuery = useAcademicWarning(warningID ?? "");
  const warningConfigQuery = useWarningConfig();
  const mutations = useAcademicWarningMutations(warningID);
  const [note, setNote] = useState("");

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>预警配置</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <div className="rounded-xl border border-border p-4"><p className="text-xs text-muted-foreground">GPA阈值</p><p className="mt-1 font-display text-2xl font-semibold">{formatGPA(warningConfigQuery.data?.gpa_threshold ?? 0)}</p></div>
          <div className="rounded-xl border border-border p-4"><p className="text-xs text-muted-foreground">挂科阈值</p><p className="mt-1 font-display text-2xl font-semibold">{warningConfigQuery.data?.fail_count_threshold ?? 0} 门</p></div>
          <div className="rounded-xl border border-border p-4"><p className="text-xs text-muted-foreground">开关状态</p><p className="mt-1 font-display text-2xl font-semibold">{warningConfigQuery.data?.is_enabled ? "开启" : "关闭"}</p></div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>预警列表</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(warningsQuery.data?.list ?? []).length === 0 ? <EmptyState title="暂无预警" description="成绩审核完成后自动检测学业预警。" /> : null}
          {(warningsQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{item.student_name} · {item.student_no}</p>
                  <p className="mt-1 text-sm text-muted-foreground">{item.warning_type_text} · {item.semester_name}</p>
                </div>
                <Badge variant={getAcademicWarningStatusVariant(item.status)}>{item.status_text}</Badge>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">{item.detail.current_gpa ? `当前GPA ${formatGPA(item.detail.current_gpa)}` : `挂科门数 ${item.detail.fail_count ?? 0}`}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      {warningQuery.data ? (
        <Card>
          <CardHeader>
            <CardTitle>预警详情</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Badge variant={getAcademicWarningStatusVariant(warningQuery.data.status)}>{warningQuery.data.status_text}</Badge>
              <Badge variant="outline">{warningQuery.data.warning_type_text}</Badge>
            </div>
            <div className="rounded-xl border border-border p-4 text-sm text-muted-foreground">
              {warningQuery.data.detail.semester_courses?.map((course) => `${course.course_name} ${course.score}`).join("；") ?? warningQuery.data.detail.failed_courses?.map((course) => `${course.course_name} ${course.score}`).join("；") ?? "暂无课程明细"}
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              <div className="rounded-xl border border-border p-4 text-sm">
                <p className="text-muted-foreground">预警类型</p>
                <p className="mt-1 font-semibold">{warningQuery.data.warning_type_text}</p>
              </div>
              <div className="rounded-xl border border-border p-4 text-sm">
                <p className="text-muted-foreground">当前状态</p>
                <p className="mt-1 font-semibold">{warningQuery.data.status_text}</p>
              </div>
              <div className="rounded-xl border border-border p-4 text-sm">
                <p className="text-muted-foreground">创建时间</p>
                <p className="mt-1 font-semibold">{formatDateTime(warningQuery.data.created_at)}</p>
              </div>
            </div>
            <FormField label="处理备注">
              <Textarea value={note} onChange={(event) => setNote(event.target.value)} rows={4} />
            </FormField>
            <Button onClick={() => mutations.handle.mutate({ handle_note: note })} isLoading={mutations.handle.isPending}>标记已处理</Button>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
