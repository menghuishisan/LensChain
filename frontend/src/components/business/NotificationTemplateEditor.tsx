"use client";

// NotificationTemplateEditor.tsx
// 模块07消息模板组件，支持模板列表、详情、更新和安全预览。

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useNotificationTemplate, useNotificationTemplateMutations, useNotificationTemplates } from "@/hooks/useNotificationTemplates";
import { stripHtmlToText } from "@/lib/notification";
import type { ID } from "@/types/api";

/** NotificationTemplateEditor 组件属性。 */
export interface NotificationTemplateEditorProps {
  templateID?: ID;
}

/** NotificationTemplateEditor 消息模板组件。 */
export function NotificationTemplateEditor({ templateID }: NotificationTemplateEditorProps) {
  const templatesQuery = useNotificationTemplates();
  const detailQuery = useNotificationTemplate(templateID ?? "");
  const mutations = useNotificationTemplateMutations(templateID);
  const [titleTemplate, setTitleTemplate] = useState("");
  const [contentTemplate, setContentTemplate] = useState("");

  useEffect(() => {
    if (!detailQuery.data) {
      return;
    }
    setTitleTemplate(detailQuery.data.title_template);
    setContentTemplate(detailQuery.data.content_template);
  }, [detailQuery.data]);

  if (!templateID) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>消息模板管理</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(templatesQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{item.event_type}</p>
                  <p className="mt-1 text-sm text-muted-foreground">{item.title_template}</p>
                </div>
                <Button size="sm" variant="outline" onClick={() => window.location.assign(`/admin/notifications/templates?id=${item.id}`)}>编辑</Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>编辑模板</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="标题模板">
            <Input value={titleTemplate} onChange={(event) => setTitleTemplate(event.target.value)} />
          </FormField>
          <FormField label="内容模板">
            <Textarea value={contentTemplate} onChange={(event) => setContentTemplate(event.target.value)} rows={10} />
          </FormField>
          <div className="rounded-xl border border-border p-4 text-sm text-muted-foreground">
            可用变量：{(detailQuery.data?.variables ?? []).map((item) => `{${item.name}}`).join("、") || "无"}
          </div>
          <div className="flex gap-2">
            <Button onClick={() => mutations.update.mutate({ title_template: titleTemplate, content_template: contentTemplate, is_enabled: detailQuery.data?.is_enabled ?? true })} isLoading={mutations.update.isPending}>保存</Button>
            <Button variant="outline" onClick={() => mutations.preview.mutate({ course_name: "区块链原理", assignment_name: "作业1", deadline: "2026-04-15 23:59" })} isLoading={mutations.preview.isPending}>预览</Button>
          </div>
        </CardContent>
      </Card>

      {mutations.preview.data ? (
        <Card>
          <CardHeader>
            <CardTitle>安全预览</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="rounded-xl border border-border p-4">
              <p className="font-semibold">{stripHtmlToText(mutations.preview.data.title)}</p>
              <p className="mt-2 text-sm text-muted-foreground">{stripHtmlToText(mutations.preview.data.content)}</p>
            </div>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
