"use client";

// CtfTemplateLibraryPanel.tsx
// CTF 参数化模板库浏览面板
// 模板卡片列表、变体展开预览、参数配置弹窗、生成题目

import { ChevronDown, ChevronRight, FileCode, Sparkles } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { useCtfChallengeTemplate, useCtfChallengeTemplates } from "@/hooks/useCtfChallenges";
import { useVulnerabilityConvertMutations } from "@/hooks/useCtfChallenges";
import type { ID } from "@/types/api";
import type { CtfDifficulty, CtfJsonObject } from "@/types/ctf";

const DIFFICULTY_LABELS: Record<number, string> = { 1: "Warmup", 2: "Easy", 3: "Medium", 4: "Hard", 5: "Insane" };

/**
 * CtfTemplateLibraryPanel CTF 参数化模板库浏览面板。
 * 展示模板卡片列表，支持展开变体预览和生成题目。
 */
export function CtfTemplateLibraryPanel() {
  const templatesQuery = useCtfChallengeTemplates({ page: 1, page_size: 50 });
  const [expandedID, setExpandedID] = useState<ID>("");
  const [generatingID, setGeneratingID] = useState<ID>("");

  if (templatesQuery.isLoading) {
    return <LoadingState variant="grid" title="正在加载模板库" description="读取参数化漏洞模板列表。" />;
  }

  const templates = templatesQuery.data?.list ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-3xl font-semibold">参数化模板库</h1>
        <p className="mt-2 text-sm text-muted-foreground">从内置漏洞模板快速生成 CTF 题目，支持难度调整和变体选择。</p>
      </div>

      {templates.length === 0 ? (
        <EmptyState title="暂无可用模板" description="模板库为空，请联系管理员导入漏洞模板。" />
      ) : (
        <div className="space-y-4">
          {templates.map((template) => {
            const isExpanded = expandedID === template.id;
            return (
              <Card key={template.id}>
                <CardHeader className="cursor-pointer" onClick={() => setExpandedID(isExpanded ? "" : template.id)}>
                  <CardTitle className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <FileCode className="h-5 w-5 text-primary shrink-0" />
                      {template.name}
                    </div>
                    <div className="flex items-center gap-2">
                      {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </div>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex flex-wrap gap-2">
                    <Badge>{template.vulnerability_type}</Badge>
                    <Badge variant="outline">
                      {DIFFICULTY_LABELS[template.difficulty_range.min] ?? "?"} ~ {DIFFICULTY_LABELS[template.difficulty_range.max] ?? "?"}
                    </Badge>
                    <Badge variant="secondary">{template.variant_count} 变体</Badge>
                    <Badge variant="secondary">使用 {template.usage_count} 次</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground">{template.description}</p>

                  {/* 展开区域：变体预览 + 生成按钮 */}
                  {isExpanded ? (
                    <TemplateExpandedSection
                      templateID={template.id}
                      onGenerate={() => setGeneratingID(template.id)}
                    />
                  ) : null}

                  {/* 生成题目弹窗 */}
                  {generatingID === template.id ? (
                    <GenerateChallengeForm
                      templateID={template.id}
                      onClose={() => setGeneratingID("")}
                    />
                  ) : null}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

/**
 * TemplateExpandedSection 模板展开区域，展示变体列表。
 */
function TemplateExpandedSection({ templateID, onGenerate }: { templateID: ID; onGenerate: () => void }) {
  const detailQuery = useCtfChallengeTemplate(templateID);

  if (detailQuery.isLoading) {
    return <p className="text-sm text-muted-foreground">加载变体信息...</p>;
  }

  const detail = detailQuery.data;
  if (!detail) return null;

  return (
    <div className="space-y-3 border-t border-border pt-3">
      <p className="text-sm font-semibold">变体列表</p>
      <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
        {detail.variants.map((variant, index) => (
          <div key={index} className="rounded-xl border border-border p-3">
            <p className="font-semibold text-sm">{variant.name}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              建议难度：{DIFFICULTY_LABELS[variant.suggested_difficulty] ?? "未知"}
            </p>
            {Object.keys(variant.params).length > 0 ? (
              <pre className="mt-2 overflow-auto rounded-lg bg-muted p-2 text-xs">{JSON.stringify(variant.params, null, 2)}</pre>
            ) : null}
          </div>
        ))}
      </div>

      {detail.parameters.params.length > 0 ? (
        <div className="space-y-2">
          <p className="text-sm font-semibold">可配置参数</p>
          <div className="flex flex-wrap gap-2">
            {detail.parameters.params.map((param) => (
              <Badge key={param.key} variant="outline" className="text-xs">{param.label}: {String(param.default)}</Badge>
            ))}
          </div>
        </div>
      ) : null}

      {detail.reference_events.length > 0 ? (
        <div className="space-y-2">
          <p className="text-sm font-semibold">真实漏洞参考</p>
          {detail.reference_events.map((event, index) => (
            <p key={index} className="text-xs text-muted-foreground">{event.name} ({event.date}) - 损失 {event.loss}</p>
          ))}
        </div>
      ) : null}

      <Button size="sm" onClick={onGenerate}>
        <Sparkles className="h-4 w-4" />
        从此模板生成题目
      </Button>
    </div>
  );
}

/**
 * GenerateChallengeForm 从模板生成题目的参数配置弹窗。
 */
function GenerateChallengeForm({ templateID, onClose }: { templateID: ID; onClose: () => void }) {
  const detailQuery = useCtfChallengeTemplate(templateID);
  const mutations = useVulnerabilityConvertMutations();
  const [title, setTitle] = useState("");
  const [difficulty, setDifficulty] = useState<string>("2");
  const [baseScore, setBaseScore] = useState("300");
  const [templateParams, setTemplateParams] = useState<CtfJsonObject>({});

  const detail = detailQuery.data;

  const handleGenerate = () => {
    mutations.generateFromTemplate.mutate(
      {
        template_id: templateID,
        title,
        difficulty: Number(difficulty) as CtfDifficulty,
        base_score: Number(baseScore),
        template_params: templateParams,
      },
      { onSuccess: () => onClose() },
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>从模板生成题目</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="题目名称" required>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="重入攻击入门" />
          </FormField>
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="难度">
              <Select value={difficulty} onValueChange={setDifficulty}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Warmup</SelectItem>
                  <SelectItem value="2">Easy</SelectItem>
                  <SelectItem value="3">Medium</SelectItem>
                  <SelectItem value="4">Hard</SelectItem>
                  <SelectItem value="5">Insane</SelectItem>
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="基础分">
              <Input type="number" value={baseScore} onChange={(e) => setBaseScore(e.target.value)} />
            </FormField>
          </div>

          {/* 动态参数表单 */}
          {(detail?.parameters.params ?? []).map((param) => (
            <FormField key={param.key} label={param.label}>
              {param.options ? (
                <Select
                  value={String(templateParams[param.key] ?? param.default ?? "")}
                  onValueChange={(v) => setTemplateParams((prev) => ({ ...prev, [param.key]: v }))}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {param.options.map((opt) => (
                      <SelectItem key={opt} value={opt}>{opt}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  value={String(templateParams[param.key] ?? param.default ?? "")}
                  onChange={(e) => setTemplateParams((prev) => ({ ...prev, [param.key]: e.target.value }))}
                />
              )}
            </FormField>
          ))}

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={onClose}>取消</Button>
            <Button disabled={!title} onClick={handleGenerate} isLoading={mutations.generateFromTemplate.isPending}>
              生成题目
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
