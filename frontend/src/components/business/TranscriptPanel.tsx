"use client";

// TranscriptPanel.tsx
// 模块06成绩单组件，支持生成、列表查看和下载，下载统一走后端接口。

import { FileText, Download } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { useTranscriptMutations, useTranscripts } from "@/hooks/useGrades";
import { formatDateTime, formatFileSize } from "@/lib/format";

/**
 * TranscriptPanel 成绩单组件。
 */
export function TranscriptPanel() {
  const transcriptsQuery = useTranscripts({ page: 1, page_size: 20 });
  const mutations = useTranscriptMutations();
  const [semesterIDs, setSemesterIDs] = useState("");

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5 text-primary" />
            成绩单生成
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-[1fr_auto]">
          <FormField label="学期ID列表" description="用逗号分隔多个学期ID，仅生成已审核通过的成绩单。">
            <Input value={semesterIDs} onChange={(event) => setSemesterIDs(event.target.value)} placeholder="例如：1880000000001,1880000000002" />
          </FormField>
          <Button className="self-end" disabled={semesterIDs.trim().length === 0} onClick={() => mutations.generate.mutate({ semester_ids: semesterIDs.split(",").map((item) => item.trim()).filter(Boolean) })} isLoading={mutations.generate.isPending}>
            生成成绩单
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>成绩单列表</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(transcriptsQuery.data?.list ?? []).length === 0 ? <EmptyState title="暂无成绩单" description="生成后在这里显示下载记录。" /> : null}
          {(transcriptsQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border p-4">
              <div>
                <p className="font-semibold">{item.student_name}</p>
                <p className="mt-1 text-sm text-muted-foreground">{item.include_semesters.join("、")} · {formatFileSize(item.file_size)} · {formatDateTime(item.generated_at)}</p>
              </div>
              <Button variant="outline" onClick={() => mutations.download.mutate(item.id)} isLoading={mutations.download.isPending}>
                <Download className="h-4 w-4" />
                下载
              </Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
