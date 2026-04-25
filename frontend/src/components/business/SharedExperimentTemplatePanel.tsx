"use client";

// SharedExperimentTemplatePanel.tsx
// 模块04共享实验模板库业务面板。

import { useRouter } from "next/navigation";

import { ExperimentTemplateCard } from "@/components/business/ExperimentTemplateCard";
import { LoadingState } from "@/components/ui/LoadingState";
import { useSharedExperimentTemplates } from "@/hooks/useExperimentTemplates";

/**
 * SharedExperimentTemplatePanel 共享实验模板库面板。
 */
export function SharedExperimentTemplatePanel() {
  const templatesQuery = useSharedExperimentTemplates({ page: 1, page_size: 30 });
  const router = useRouter();

  if (templatesQuery.isLoading) {
    return <LoadingState title="正在加载共享实验内容" description="正在整理平台共享的实验内容。" />;
  }

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">共享实验内容</h1>
      <div className="grid gap-4 xl:grid-cols-2">
        {(templatesQuery.data?.list ?? []).map((template) => (
          <ExperimentTemplateCard key={template.id} template={template} onOpen={(id) => router.push(`/teacher/experiment-templates/${id}`)} />
        ))}
      </div>
    </div>
  );
}
