// ExperimentTemplateCard.tsx
// 模块04实验模板卡片，展示模板类型、拓扑、评分和发布状态。

import { Clock, Copy, FlaskConical, Play, Share2, ShieldCheck } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/Card";
import type { ExperimentTemplateListItem } from "@/types/experiment";

/**
 * ExperimentTemplateCard 组件属性。
 */
export interface ExperimentTemplateCardProps {
  template: ExperimentTemplateListItem;
  onOpen?: (templateID: string) => void;
  onPublish?: (templateID: string) => void;
  onClone?: (templateID: string) => void;
  onShare?: (templateID: string, isShared: boolean) => void;
  isOperating?: boolean;
}

function getStatusVariant(status: number) {
  if (status === 2) {
    return "success" as const;
  }
  if (status === 3) {
    return "secondary" as const;
  }
  return "outline" as const;
}

/**
 * normalizeExperimentTemplateTags 统一处理模板标签缺失场景，避免页面渲染报错。
 */
export function normalizeExperimentTemplateTags(template: Pick<ExperimentTemplateListItem, "tags">) {
  return Array.isArray(template.tags) ? template.tags : [];
}

/**
 * ExperimentTemplateCard 实验模板卡片组件。
 */
export function ExperimentTemplateCard({ template, onOpen, onPublish, onClone, onShare, isOperating = false }: ExperimentTemplateCardProps) {
  const tags = normalizeExperimentTemplateTags(template);

  return (
    <Card className="group relative overflow-hidden border-cyan-500/15 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 text-white shadow-[0_22px_70px_rgba(8,47,73,0.26)]">
      <div className="absolute right-0 top-0 h-28 w-28 rounded-bl-full bg-cyan-400/15 blur-sm" />
      <CardHeader className="relative">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <CardTitle className="line-clamp-2 text-white">{template.title}</CardTitle>
            <div className="mt-3 flex flex-wrap gap-2">
              <Badge variant={getStatusVariant(template.status)}>{template.status_text}</Badge>
              <Badge variant="outline" className="border-white/18 bg-white/8 text-white/82">
                {template.experiment_type_text}
              </Badge>
              {template.is_shared ? (
                <Badge variant="success" className="bg-emerald-400/15 text-emerald-100">
                  已共享
                </Badge>
              ) : null}
            </div>
          </div>
          <FlaskConical className="h-8 w-8 shrink-0 text-cyan-200" />
        </div>
      </CardHeader>
      <CardContent className="relative space-y-4">
        <div className="grid grid-cols-3 gap-3">
          <div className="rounded-xl border border-white/10 bg-white/7 p-3">
            <p className="text-xs text-white/50">拓扑模式</p>
            <p className="mt-1 text-sm font-semibold">{template.topology_mode_text}</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-3">
            <p className="text-xs text-white/50">检查点</p>
            <p className="mt-1 text-sm font-semibold">{template.checkpoint_count} 个</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-3">
            <p className="text-xs text-white/50">容器</p>
            <p className="mt-1 text-sm font-semibold">{template.container_count} 个</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 text-xs text-white/70">
          <span className="inline-flex items-center gap-1">
            <ShieldCheck className="h-3.5 w-3.5" />
            {template.judge_mode_text}
          </span>
          <span className="inline-flex items-center gap-1">
            <Clock className="h-3.5 w-3.5" />
            {template.max_duration} 分钟
          </span>
          <span>总分 {template.total_score}</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {tags.map((tag) => (
            <Badge key={tag.id} variant="outline" className="border-cyan-300/20 bg-cyan-300/8 text-cyan-50">
              {tag.name}
            </Badge>
          ))}
        </div>
      </CardContent>
      <CardFooter className="relative flex-wrap">
        <Button variant="secondary" size="sm" onClick={() => onOpen?.(template.id)}>
          <Play className="h-4 w-4" />
          打开
        </Button>
        <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => onPublish?.(template.id)} disabled={template.status === 2} isLoading={isOperating}>
          发布
        </Button>
        <Button variant="ghost" size="sm" className="text-white hover:bg-white/10" onClick={() => onClone?.(template.id)}>
          <Copy className="h-4 w-4" />
          克隆
        </Button>
        <Button variant="ghost" size="sm" className="text-white hover:bg-white/10" onClick={() => onShare?.(template.id, !template.is_shared)}>
          <Share2 className="h-4 w-4" />
          {template.is_shared ? "取消共享" : "共享"}
        </Button>
      </CardFooter>
    </Card>
  );
}
