"use client";

// ContainerOrchestrationCanvas.tsx
// 容器编排画布
// 支持容器卡片网格布局、依赖关系SVG连线、容器配置弹窗、删除和排序

import { ArrowDown, Box, GripVertical, Settings, Star, Trash2 } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import type { ID } from "@/types/api";
import type { TemplateContainer } from "@/types/experiment";

/**
 * ContainerOrchestrationCanvas 属性。
 */
interface ContainerOrchestrationCanvasProps {
  containers: TemplateContainer[];
  onUpdateContainer: (containerID: ID, payload: Partial<ContainerConfigPayload>) => void;
  onDeleteContainer: (containerID: ID) => void;
  isUpdating?: boolean;
  isDeleting?: boolean;
}

/**
 * 容器配置编辑载荷。
 */
interface ContainerConfigPayload {
  container_name: string;
  is_primary: boolean;
  startup_order: number;
  env_vars: Array<{ key: string; value: string }>;
  ports: Array<{ container: number; protocol: string }>;
  volumes: Array<{ host_path: string; container_path: string }>;
  cpu_limit: string | null;
  memory_limit: string | null;
  depends_on: string[];
}

/**
 * ContainerOrchestrationCanvas 容器编排画布组件。
 * 展示容器卡片网格、SVG 依赖连线和配置弹窗。
 */
export function ContainerOrchestrationCanvas({
  containers,
  onUpdateContainer,
  onDeleteContainer,
  isUpdating,
  isDeleting,
}: ContainerOrchestrationCanvasProps) {
  const [editingID, setEditingID] = useState<ID>("");
  const canvasRef = useRef<HTMLDivElement>(null);

  // 按启动顺序排列
  const sorted = useMemo(() => [...containers].sort((a, b) => a.startup_order - b.startup_order), [containers]);

  // 计算容器卡片位置（简化为2列网格），用于 SVG 连线
  const positions = useMemo(() => {
    const cols = 2;
    const cardW = 280;
    const cardH = 140;
    const gapX = 40;
    const gapY = 30;
    const map = new Map<string, { cx: number; cy: number; top: number; bottom: number }>();
    sorted.forEach((c, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);
      const x = col * (cardW + gapX);
      const y = row * (cardH + gapY);
      map.set(c.container_name, { cx: x + cardW / 2, cy: y + cardH / 2, top: y, bottom: y + cardH });
    });
    return map;
  }, [sorted]);

  // 构造依赖连线
  const dependencyLines = useMemo(() => {
    const lines: Array<{ from: string; to: string; x1: number; y1: number; x2: number; y2: number }> = [];
    sorted.forEach((c) => {
      c.depends_on.forEach((depName) => {
        const fromPos = positions.get(c.container_name);
        const toPos = positions.get(depName);
        if (fromPos && toPos) {
          lines.push({ from: c.container_name, to: depName, x1: fromPos.cx, y1: fromPos.top, x2: toPos.cx, y2: toPos.bottom });
        }
      });
    });
    return lines;
  }, [sorted, positions]);

  const svgHeight = Math.max(200, (Math.ceil(sorted.length / 2)) * 170 + 40);

  const editingContainer = sorted.find((c) => c.id === editingID);

  return (
    <div className="space-y-4">
      {/* SVG 依赖关系连线层 */}
      {dependencyLines.length > 0 ? (
        <div className="relative overflow-hidden rounded-xl border border-border bg-muted/10 p-4">
          <p className="mb-2 text-xs font-semibold text-muted-foreground">容器依赖关系</p>
          <svg width="100%" height={svgHeight} className="pointer-events-none">
            <defs>
              <marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
                <path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor" className="text-primary/50" />
              </marker>
            </defs>
            {dependencyLines.map((line) => (
              <line
                key={`${line.from}-${line.to}`}
                x1={line.x1}
                y1={line.y1}
                x2={line.x2}
                y2={line.y2}
                stroke="currentColor"
                className="text-primary/30"
                strokeWidth={2}
                strokeDasharray="6 3"
                markerEnd="url(#arrow)"
              />
            ))}
          </svg>
        </div>
      ) : null}

      {/* 容器卡片网格 */}
      <div ref={canvasRef} className="grid gap-4 md:grid-cols-2">
        {sorted.map((container, index) => (
          <ContainerCard
            key={container.id}
            container={container}
            index={index}
            onEdit={() => setEditingID(container.id)}
            onDelete={() => onDeleteContainer(container.id)}
            isDeleting={isDeleting}
          />
        ))}
      </div>

      {sorted.length === 0 ? (
        <div className="flex items-center justify-center rounded-xl border border-dashed border-border bg-muted/10 p-8 text-sm text-muted-foreground">
          从上方选择镜像并添加容器，已添加的容器将在此处展示。
        </div>
      ) : null}

      {/* 容器配置弹窗 */}
      {editingContainer ? (
        <ContainerConfigModal
          container={editingContainer}
          allContainerNames={sorted.map((c) => c.container_name)}
          onSave={(payload) => {
            onUpdateContainer(editingContainer.id, payload);
            setEditingID("");
          }}
          onClose={() => setEditingID("")}
          isSaving={isUpdating}
        />
      ) : null}
    </div>
  );
}

/**
 * ContainerCard 单个容器卡片。
 */
function ContainerCard({
  container,
  index,
  onEdit,
  onDelete,
  isDeleting,
}: {
  container: TemplateContainer;
  index: number;
  onEdit: () => void;
  onDelete: () => void;
  isDeleting?: boolean;
}) {
  return (
    <Card className="relative overflow-hidden">
      {container.is_primary ? (
        <div className="absolute top-0 right-0 rounded-bl-lg bg-primary px-2 py-0.5">
          <Star className="h-3 w-3 text-primary-foreground" />
        </div>
      ) : null}
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <GripVertical className="h-4 w-4 text-muted-foreground/50 shrink-0" />
          <Box className="h-4 w-4 text-primary shrink-0" />
          {container.container_name}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <p className="text-muted-foreground">
          {container.image_version?.image_display_name ?? "未关联镜像"} · {container.image_version?.version ?? "-"}
        </p>
        <div className="flex flex-wrap gap-1">
          <Badge variant="outline" className="text-xs">顺序 {container.startup_order}</Badge>
          {container.cpu_limit ? <Badge variant="outline" className="text-xs">CPU {container.cpu_limit}</Badge> : null}
          {container.memory_limit ? <Badge variant="outline" className="text-xs">内存 {container.memory_limit}</Badge> : null}
          {container.ports.length > 0 ? <Badge variant="outline" className="text-xs">{container.ports.length} 端口</Badge> : null}
          {container.depends_on.length > 0 ? <Badge variant="secondary" className="text-xs">依赖 {container.depends_on.join(", ")}</Badge> : null}
        </div>
        <div className="flex gap-1 pt-1">
          <Button variant="outline" size="sm" onClick={onEdit}>
            <Settings className="h-3.5 w-3.5" />
            配置
          </Button>
          <ConfirmDialog
            title="删除容器"
            description={`确认删除容器 "${container.container_name}"？此操作不可恢复。`}
            confirmVariant="destructive"
            trigger={
              <Button variant="ghost" size="sm" className="text-destructive">
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            }
            onConfirm={onDelete}
          />
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * ContainerConfigModal 容器配置弹窗。
 */
function ContainerConfigModal({
  container,
  allContainerNames,
  onSave,
  onClose,
  isSaving,
}: {
  container: TemplateContainer;
  allContainerNames: string[];
  onSave: (payload: Partial<ContainerConfigPayload>) => void;
  onClose: () => void;
  isSaving?: boolean;
}) {
  const [name, setName] = useState(container.container_name);
  const [isPrimary, setIsPrimary] = useState(container.is_primary);
  const [startupOrder, setStartupOrder] = useState(container.startup_order);
  const [cpuLimit, setCpuLimit] = useState(container.cpu_limit ?? "");
  const [memoryLimit, setMemoryLimit] = useState(container.memory_limit ?? "");
  const [envText, setEnvText] = useState(container.env_vars.map((e) => `${e.key}=${e.value}`).join("\n"));
  const [portsText, setPortsText] = useState(container.ports.map((p) => `${p.container}/${p.protocol}`).join("\n"));
  const [volumesText, setVolumesText] = useState(container.volumes.map((v) => `${v.host_path}:${v.container_path}`).join("\n"));
  const [dependsOn, setDependsOn] = useState(container.depends_on.join(","));

  const handleSave = () => {
    const envVars = envText.split("\n").filter(Boolean).map((line) => {
      const idx = line.indexOf("=");
      return idx > 0 ? { key: line.slice(0, idx), value: line.slice(idx + 1) } : { key: line, value: "" };
    });
    const ports = portsText.split("\n").filter(Boolean).map((line) => {
      const [port, protocol] = line.split("/");
      return { container: Number(port) || 0, protocol: protocol || "tcp" };
    });
    const volumes = volumesText.split("\n").filter(Boolean).map((line) => {
      const [hostPath, containerPath] = line.split(":");
      return { host_path: hostPath || "", container_path: containerPath || hostPath || "" };
    });
    const deps = dependsOn.split(",").map((d) => d.trim()).filter(Boolean);
    onSave({
      container_name: name,
      is_primary: isPrimary,
      startup_order: startupOrder,
      cpu_limit: cpuLimit || null,
      memory_limit: memoryLimit || null,
      env_vars: envVars,
      ports,
      volumes,
      depends_on: deps,
    });
  };

  const otherContainers = allContainerNames.filter((n) => n !== container.container_name);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <Card className="w-full max-w-2xl max-h-[80vh] overflow-y-auto">
        <CardHeader>
          <CardTitle>配置容器 — {container.container_name}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="容器名称">
              <Input value={name} onChange={(e) => setName(e.target.value)} />
            </FormField>
            <FormField label="启动顺序">
              <Input type="number" value={startupOrder} onChange={(e) => setStartupOrder(Number(e.target.value))} />
            </FormField>
            <FormField label="CPU 限制">
              <Input value={cpuLimit} onChange={(e) => setCpuLimit(e.target.value)} placeholder="例如 2" />
            </FormField>
            <FormField label="内存限制">
              <Input value={memoryLimit} onChange={(e) => setMemoryLimit(e.target.value)} placeholder="例如 4Gi" />
            </FormField>
          </div>
          <div className="flex items-center gap-3">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input type="checkbox" checked={isPrimary} onChange={(e) => setIsPrimary(e.target.checked)} className="accent-primary" />
              设为主容器
            </label>
          </div>
          <FormField label="环境变量（每行一个 KEY=VALUE）">
            <Textarea value={envText} onChange={(e) => setEnvText(e.target.value)} rows={4} className="font-mono text-xs" placeholder="NODE_ENV=production" />
          </FormField>
          <FormField label="端口映射（每行一个 端口/协议）">
            <Textarea value={portsText} onChange={(e) => setPortsText(e.target.value)} rows={3} className="font-mono text-xs" placeholder="8545/tcp" />
          </FormField>
          <FormField label="挂载卷（每行一个 宿主路径:容器路径）">
            <Textarea value={volumesText} onChange={(e) => setVolumesText(e.target.value)} rows={3} className="font-mono text-xs" placeholder="/data:/data" />
          </FormField>
          <FormField label="依赖容器（逗号分隔容器名）">
            <Input value={dependsOn} onChange={(e) => setDependsOn(e.target.value)} placeholder={otherContainers.join(", ") || "无其他容器"} />
          </FormField>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" onClick={onClose}>取消</Button>
            <Button onClick={handleSave} isLoading={isSaving}>保存配置</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
