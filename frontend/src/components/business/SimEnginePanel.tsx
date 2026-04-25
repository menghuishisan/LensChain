"use client";

// SimEnginePanel.tsx
// 模块04可视化仿真面板，支持 WebSocket 状态、拖拽布局、时间控制和自定义倍速。

import { Activity, Move, Pause, Play, RefreshCcw, RotateCcw, StepForward, Zap } from "lucide-react";
import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { useSimEngineRealtime } from "@/hooks/useExperimentRealtime";
import { cn } from "@/lib/utils";
import type { ID } from "@/types/api";
import type { JsonObject, TemplateSimScene } from "@/types/experiment";

/**
 * SimEnginePanel 组件属性。
 */
export interface SimEnginePanelProps {
  sessionID: ID;
  scenes: TemplateSimScene[];
  onLayoutChange?: (layouts: Array<{ scene_id: ID; layout_position: JsonObject }>) => void;
  className?: string;
}

function getSceneCode(scene: TemplateSimScene) {
  return scene.scenario?.code ?? scene.id;
}

/**
 * SimEnginePanel 可视化仿真控制组件。
 */
export function SimEnginePanel({ sessionID, scenes, onLayoutChange, className }: SimEnginePanelProps) {
  const realtime = useSimEngineRealtime(sessionID, sessionID.length > 0);
  const [orderedSceneIDs, setOrderedSceneIDs] = useState<ID[]>([]);
  const [draggingID, setDraggingID] = useState<ID | null>(null);
  const [speed, setSpeed] = useState("1.0");

  useEffect(() => {
    setOrderedSceneIDs(scenes.map((scene) => scene.id));
  }, [scenes]);

  const orderedScenes = orderedSceneIDs
    .map((id) => scenes.find((scene) => scene.id === id))
    .filter((scene): scene is TemplateSimScene => scene !== undefined);

  const notifyLayout = (nextIDs: ID[]) => {
    onLayoutChange?.(
      nextIDs.map((sceneID, index) => ({
        scene_id: sceneID,
        layout_position: { order: index, column_span: 6 },
      })),
    );
  };

  const moveScene = (targetID: ID) => {
    if (draggingID === null || draggingID === targetID) {
      return;
    }
    const nextIDs = orderedSceneIDs.filter((id) => id !== draggingID);
    const targetIndex = nextIDs.indexOf(targetID);
    nextIDs.splice(targetIndex, 0, draggingID);
    setOrderedSceneIDs(nextIDs);
    notifyLayout(nextIDs);
  };

  const activeSceneCode = orderedScenes[0] ? getSceneCode(orderedScenes[0]) : "";
  const parsedSpeed = Number(speed);
  const canSendSpeed = Number.isFinite(parsedSpeed) && parsedSpeed > 0;

  return (
    <Card className={cn("overflow-hidden border-cyan-500/20 bg-slate-950 text-white", className)}>
      <CardHeader className="border-b border-white/10">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <CardTitle className="flex items-center gap-2 text-white">
            <Activity className="h-5 w-5 text-cyan-300" />
            SimEngine 可视化仿真
          </CardTitle>
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <span className={cn("h-2 w-2 rounded-full", realtime.status === "open" ? "bg-emerald-400" : "bg-amber-400")} />
            {realtime.status === "open" ? "实时同步中" : realtime.status === "reconnecting" ? "断线重连中" : "未连接"}
            <Button variant="ghost" size="sm" className="text-white hover:bg-white/10" onClick={realtime.reconnect}>
              <RefreshCcw className="h-4 w-4" />
              重连
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 p-5">
        {realtime.error ? <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-100">{realtime.error}</div> : null}
        {sessionID.length === 0 ? <div className="rounded-xl border border-amber-400/20 bg-amber-400/10 p-4 text-sm text-amber-100">可视化内容还在准备中，请稍后刷新查看。</div> : null}

        <div className="flex flex-wrap items-end gap-3 rounded-xl border border-white/10 bg-white/7 p-4">
          <Button size="sm" variant="secondary" onClick={() => realtime.sendControl(activeSceneCode, "play")} disabled={realtime.status !== "open"}>
            <Play className="h-4 w-4" />
            播放
          </Button>
          <Button size="sm" variant="outline" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => realtime.sendControl(activeSceneCode, "pause")} disabled={realtime.status !== "open"}>
            <Pause className="h-4 w-4" />
            暂停
          </Button>
          <Button size="sm" variant="outline" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => realtime.sendControl(activeSceneCode, "step")} disabled={realtime.status !== "open"}>
            <StepForward className="h-4 w-4" />
            单步
          </Button>
          <Button size="sm" variant="outline" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => realtime.sendControl(activeSceneCode, "reset")} disabled={realtime.status !== "open"}>
            <RotateCcw className="h-4 w-4" />
            重置
          </Button>
          <FormField label="自定义倍速" className="min-w-36">
            <Input className="border-white/10 bg-white/8 text-white" type="number" min="0.1" max="4" step="0.1" value={speed} onChange={(event) => setSpeed(event.target.value)} />
          </FormField>
          <Button size="sm" onClick={() => realtime.sendControl(activeSceneCode, "set_speed", parsedSpeed)} disabled={realtime.status !== "open" || !canSendSpeed}>
            <Zap className="h-4 w-4" />
            设置倍速
          </Button>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          {orderedScenes.map((scene) => (
            <div
              key={scene.id}
              draggable
              onDragStart={() => setDraggingID(scene.id)}
              onDragOver={(event) => event.preventDefault()}
              onDrop={() => moveScene(scene.id)}
              onDragEnd={() => setDraggingID(null)}
              className={cn("min-h-72 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900 to-cyan-950/60 p-4 transition", draggingID === scene.id ? "scale-[0.99] opacity-55" : "opacity-100")}
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-semibold">{scene.scenario?.name ?? "未命名场景"}</p>
                  <p className="mt-1 text-xs text-slate-400">{scene.scenario?.category_text ?? "未分类"} · {scene.data_source_mode_text}</p>
                </div>
                <Badge variant="outline" className="border-cyan-300/20 bg-cyan-300/8 text-cyan-50">
                  <Move className="mr-1 h-3 w-3" />
                  可拖拽
                </Badge>
              </div>
              <div className="mt-4 grid min-h-44 place-items-center rounded-xl border border-cyan-300/10 bg-black/30">
                <div className="text-center">
                  <div className="mx-auto h-16 w-16 rounded-full border border-cyan-300/30 bg-cyan-300/10 shadow-[0_0_45px_rgba(34,211,238,0.28)]" />
                  <p className="mt-3 text-sm text-slate-300">当前内容：{scene.scenario?.name ?? "暂未命名"}</p>
                  <p className="mt-1 text-xs text-slate-500">可视化画面会根据实验进度实时更新。</p>
                </div>
              </div>
            </div>
          ))}
        </div>

        <div className="rounded-xl border border-white/10 bg-black/30 p-4">
          <p className="text-sm font-semibold text-slate-100">实时动态</p>
          <div className="mt-3 max-h-48 space-y-2 overflow-y-auto text-xs text-slate-300">
            {realtime.messages.length === 0 ? <p className="text-slate-500">当前还没有新的动态。</p> : null}
            {realtime.messages.slice(-20).map((message, index) => (
              <pre key={`${message.type}-${index}`} className="rounded-lg bg-white/5 p-2">
                {JSON.stringify(message, null, 2)}
              </pre>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
