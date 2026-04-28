"use client";

// SharedExperimentTemplatePanel.tsx
// 模块04共享实验模板库业务面板，支持搜索、筛选和克隆到我的模板。

import { Copy, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { ExperimentTemplateCard } from "@/components/business/ExperimentTemplateCard";
import { Button } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { useExperimentTemplateMutations, useSharedExperimentTemplates } from "@/hooks/useExperimentTemplates";
import type { ID } from "@/types/api";

/**
 * SharedExperimentTemplatePanel 共享实验模板库面板。
 * 教师可搜索、筛选并克隆共享模板到自己的模板库。
 */
export function SharedExperimentTemplatePanel() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [topologyFilter, setTopologyFilter] = useState("all");
  const [page, setPage] = useState(1);
  const pageSize = 12;

  const templatesQuery = useSharedExperimentTemplates({
    page,
    page_size: pageSize,
    keyword: keyword || undefined,
    experiment_type: typeFilter !== "all" ? Number(typeFilter) : undefined,
    topology_mode: topologyFilter !== "all" ? Number(topologyFilter) : undefined,
  });
  const templates = templatesQuery.data?.list ?? [];
  const totalPages = Math.ceil((templatesQuery.data?.pagination?.total ?? 0) / pageSize);

  if (templatesQuery.isLoading && page === 1) {
    return <LoadingState title="正在加载共享实验内容" description="正在整理平台共享的实验内容。" />;
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-3xl font-semibold">共享实验库</h1>
        <p className="mt-2 text-sm text-muted-foreground">浏览平台共享的实验模板，一键克隆到我的模板后可自由修改。</p>
      </div>

      {/* 搜索与筛选 */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative w-64">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input className="pl-9" placeholder="搜索实验名称" value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1); }} />
        </div>
        <Select value={typeFilter} onValueChange={(v) => { setTypeFilter(v); setPage(1); }}>
          <SelectTrigger className="w-32"><SelectValue placeholder="实验类型" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部类型</SelectItem>
            <SelectItem value="1">纯仿真</SelectItem>
            <SelectItem value="2">真实环境</SelectItem>
            <SelectItem value="3">混合实验</SelectItem>
          </SelectContent>
        </Select>
        <Select value={topologyFilter} onValueChange={(v) => { setTopologyFilter(v); setPage(1); }}>
          <SelectTrigger className="w-36"><SelectValue placeholder="拓扑模式" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部拓扑</SelectItem>
            <SelectItem value="1">单人单节点</SelectItem>
            <SelectItem value="2">单人多节点</SelectItem>
            <SelectItem value="3">多人协作组网</SelectItem>
            <SelectItem value="4">共享基础设施</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* 模板卡片列表 */}
      {templates.length === 0 ? (
        <EmptyState title="暂无匹配的共享模板" description="调整筛选条件或等待更多教师共享实验模板。" />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          {templates.map((template) => (
            <SharedTemplateCardWithClone
              key={template.id}
              template={template}
              onOpen={(id) => router.push(`/teacher/experiment-templates/${id}`)}
            />
          ))}
        </div>
      )}

      {/* 分页 */}
      {totalPages > 1 ? (
        <div className="flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      ) : null}
    </div>
  );
}

/**
 * SharedTemplateCardWithClone 共享模板卡片，额外提供克隆按钮。
 */
function SharedTemplateCardWithClone({ template, onOpen }: { template: Parameters<typeof ExperimentTemplateCard>[0]["template"]; onOpen: (id: ID) => void }) {
  const mutations = useExperimentTemplateMutations(template.id);
  return (
    <ExperimentTemplateCard
      template={template}
      onOpen={onOpen}
      onClone={() => mutations.clone.mutate()}
      isOperating={mutations.clone.isPending}
    />
  );
}
