'use client';

// SimMonitorEnhancedView.tsx
// 教师多学生 SimEngine 监控增强视图（06.2 §九）。
// 三列布局：学生列表（左）+ 缩略图网格（中）+ 详情面板（右）。
// 含全班操作栏（广播/集中暂停/集中恢复）。

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  CircleDot,
  Eye,
  Gamepad2,
  MessageSquare,
  RotateCcw,
  SkipForward,
  LogOut,
  Megaphone,
  Pause,
  Play,
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { cn } from '@/lib/utils';
import { useTeacherSimMonitor } from '@/hooks/useExperimentRealtime';
import { teacherIntervene } from '@/services/experiment';
import type { ID } from '@/types/api';
import type {
  SimStudentStatusMark,
  SimStudentThumbnail,
  SimStudentProgress,
  TeacherInterveneRequest,
} from '@/types/experiment';

// ─── Types ──────────────────────────────────────────────────
interface StudentEntry {
  student_id: ID;
  student_name: string;
  student_no: string;
  tick: number;
  phase: string;
  status: SimStudentStatusMark;
  error_message?: string;
  selected: boolean;
}

type FilterMode = 'all' | 'error' | 'completed' | 'behind';

export interface SimMonitorEnhancedViewProps {
  experimentID: ID;
  className?: string;
}

const STATUS_COLORS: Record<SimStudentStatusMark, string> = {
  normal: 'text-green-500',
  behind: 'text-yellow-500',
  error: 'text-red-500',
  offline: 'text-muted-foreground',
};

function StatusDot({ status }: { status: SimStudentStatusMark }) {
  const colorClass = {
    normal: 'text-green-500',
    behind: 'text-yellow-500',
    error: 'text-red-500',
    offline: 'text-muted-foreground',
  }[status];
  return <CircleDot className={cn('h-3 w-3 inline-block', colorClass)} />;
}

/**
 * SimMonitorEnhancedView 教师仿真监控增强视图（06.2 §九 三列布局）。
 */
export function SimMonitorEnhancedView({ experimentID, className }: SimMonitorEnhancedViewProps) {
  const monitor = useTeacherSimMonitor(experimentID);
  const [students, setStudents] = useState<Map<ID, StudentEntry>>(new Map());
  const [selectedId, setSelectedId] = useState<ID | null>(null);
  const [filter, setFilter] = useState<FilterMode>('all');
  const [broadcastMessage, setBroadcastMessage] = useState('');

  // ─── WS 消息处理 ─────────────────────────────────
  useEffect(() => {
    const msgs = monitor.messages;
    if (msgs.length === 0) return;
    const last = msgs[msgs.length - 1];
    if (!last) return;

    setStudents((prev) => {
      const next = new Map(prev);
      const d = last.data as Record<string, unknown> | undefined;
      switch (last.type) {
        case 'student_thumbnail': {
          const t = d as unknown as SimStudentThumbnail;
          const existing = next.get(t.student_id);
          next.set(t.student_id, {
            student_id: t.student_id,
            student_name: t.student_name,
            student_no: t.student_no,
            tick: t.tick,
            phase: t.phase,
            status: t.status,
            error_message: t.error_message,
            selected: existing?.selected ?? false,
          });
          break;
        }
        case 'student_progress': {
          const p = d as unknown as SimStudentProgress;
          const e = next.get(p.student_id);
          if (e) next.set(p.student_id, { ...e, tick: p.tick, phase: p.phase, status: p.status });
          break;
        }
        case 'student_alert': {
          const a = d as unknown as { student_id: ID; error_message: string };
          const e = next.get(a.student_id);
          if (e) next.set(a.student_id, { ...e, status: 'error', error_message: a.error_message });
          break;
        }
        case 'student_join': {
          const j = d as unknown as { student_id: ID; student_name: string; student_no: string };
          if (!next.has(j.student_id)) {
            next.set(j.student_id, {
              student_id: j.student_id,
              student_name: j.student_name,
              student_no: j.student_no,
              tick: 0,
              phase: '',
              status: 'normal',
              selected: false,
            });
          }
          break;
        }
        case 'student_leave': {
          const l = d as unknown as { student_id: ID };
          const e = next.get(l.student_id);
          if (e) next.set(l.student_id, { ...e, status: 'offline' });
          break;
        }
      }
      return next;
    });
  }, [monitor.messages]);

  // ─── 派生数据 ───────────────────────────────────
  const studentList = useMemo(() => {
    const all = Array.from(students.values());
    const filtered = all.filter((s) => {
      if (filter === 'error') return s.status === 'error';
      if (filter === 'behind') return s.status === 'behind';
      if (filter === 'completed') return s.phase === 'completed';
      return true;
    });
    // 排序：error → behind → normal → offline
    const statusOrder: Record<SimStudentStatusMark, number> = { error: 0, behind: 1, normal: 2, offline: 3 };
    return filtered.sort((a, b) => statusOrder[a.status] - statusOrder[b.status]);
  }, [students, filter]);

  const selectedStudent = selectedId ? students.get(selectedId) : null;

  const summary = useMemo(() => {
    const all = Array.from(students.values());
    const online = all.filter((s) => s.status !== 'offline').length;
    const errorCount = all.filter((s) => s.status === 'error').length;
    const completedCount = all.filter((s) => s.phase === 'completed').length;
    const avgTick = all.length > 0 ? Math.round(all.reduce((sum, s) => sum + s.tick, 0) / all.length) : 0;
    return { total: all.length, online, errorCount, completedCount, avgTick };
  }, [students]);

  const selectedStudentIds = useMemo(
    () => Array.from(students.values()).filter((s) => s.selected).map((s) => s.student_id),
    [students],
  );

  const toggleSelect = useCallback((id: ID) => {
    setStudents((prev) => {
      const next = new Map(prev);
      const s = next.get(id);
      if (s) next.set(id, { ...s, selected: !s.selected });
      return next;
    });
  }, []);

  // ─── 干预操作 ───────────────────────────────────
  const intervene = useCallback(
    (action: TeacherInterveneRequest['action'], extra?: Partial<TeacherInterveneRequest>) => {
      void teacherIntervene(experimentID, {
        action,
        target_student_ids: selectedStudentIds.length > 0 ? selectedStudentIds : undefined,
        ...extra,
      });
    },
    [experimentID, selectedStudentIds],
  );

  const broadcast = useCallback(() => {
    if (!broadcastMessage.trim()) return;
    intervene('broadcast_message', { message: broadcastMessage });
    setBroadcastMessage('');
  }, [broadcastMessage, intervene]);

  return (
    <div className={cn('flex flex-col h-full', className)}>
      {/* §9.7 顶部全班操作栏 */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <Gamepad2 className="h-5 w-5 text-primary" />
          <span className="text-sm font-semibold">仿真实验监控</span>
          <Badge variant={monitor.status === 'open' ? 'success' : 'outline'} className="text-xs">
            {monitor.status === 'open' ? '实时连接' : '未连接'}
          </Badge>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>在线 {summary.online}/{summary.total}</span>
          <span>异常 {summary.errorCount}</span>
          <span>完成 {summary.completedCount}</span>
          <span>平均进度 {summary.avgTick} 步</span>
        </div>
        <div className="flex items-center gap-2">
          <Input
            className="h-7 w-48 text-xs"
            placeholder="广播消息..."
            value={broadcastMessage}
            onChange={(e) => setBroadcastMessage(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && broadcast()}
          />
          <Button size="sm" variant="outline" className="h-7 text-xs gap-1" onClick={broadcast}>
            <Megaphone className="h-3 w-3" />广播
          </Button>
          <Button size="sm" variant="outline" className="h-7 text-xs gap-1" onClick={() => intervene('force_pause_all')}>
            <Pause className="h-3 w-3" />集中暂停
          </Button>
          <Button size="sm" variant="outline" className="h-7 text-xs gap-1" onClick={() => intervene('force_resume_all')}>
            <Play className="h-3 w-3" />集中恢复
          </Button>
        </div>
      </div>

      {/* 三列布局 */}
      <div className="flex flex-1 min-h-0">
        {/* §9.4 左列：学生列表 */}
        <div className="w-60 shrink-0 border-r overflow-auto">
          <div className="p-2 space-y-1">
            <div className="flex gap-1 flex-wrap">
              {(['all', 'error', 'behind', 'completed'] as FilterMode[]).map((f) => (
                <button
                  key={f}
                  className={cn('text-[10px] px-2 py-0.5 rounded-full', filter === f ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted')}
                  onClick={() => setFilter(f)}
                >
                  {f === 'all' ? '全部' : f === 'error' ? '异常' : f === 'behind' ? '落后' : '已完成'}
                </button>
              ))}
            </div>
          </div>
          <div className="divide-y">
            {studentList.map((s) => (
              <button
                key={s.student_id}
                className={cn(
                  'w-full flex items-center gap-2 px-3 py-2 text-left text-xs hover:bg-muted/50 transition-colors',
                  selectedId === s.student_id && 'bg-primary/10',
                )}
                onClick={() => setSelectedId(s.student_id)}
              >
                <input
                  type="checkbox"
                  checked={s.selected}
                  onChange={(e) => { e.stopPropagation(); toggleSelect(s.student_id); }}
                  className="h-3 w-3"
                />
                <span className="truncate flex-1 font-medium">{s.student_name}</span>
                <StatusDot status={s.status} />
              </button>
            ))}
          </div>
        </div>

        {/* §9.5 中列：缩略图网格 */}
        <div className="flex-1 overflow-auto p-3">
          <div className="grid gap-3 grid-cols-3 2xl:grid-cols-5 xl:grid-cols-4">
            {studentList.map((s) => (
              <button
                key={s.student_id}
                className={cn(
                  'relative rounded-lg border p-2 text-left transition-all hover:shadow-md',
                  s.status === 'error' && 'border-red-500 border-2',
                  s.status === 'behind' && 'border-yellow-500',
                  selectedId === s.student_id && 'ring-2 ring-primary',
                )}
                onDoubleClick={() => setSelectedId(s.student_id)}
              >
                {/* 缩略图占位（240×120）*/}
                <div className="w-full aspect-[2/1] rounded bg-muted/30 flex items-center justify-center text-muted-foreground text-[10px]">
                  {s.student_name}
                </div>
                <div className="flex items-center justify-between mt-1">
                  <span className="text-[10px] font-mono">第 {s.tick} 步</span>
                  <Badge variant="outline" className="text-[9px] h-4">{s.phase || '-'}</Badge>
                </div>
                {s.status === 'error' && (
                  <div className="absolute top-1 right-1">
                    <AlertTriangle className="h-3 w-3 text-red-500" />
                  </div>
                )}
              </button>
            ))}
          </div>
          {studentList.length === 0 && (
            <p className="text-sm text-muted-foreground text-center py-12">暂无学生数据，等待学生连接...</p>
          )}
        </div>

        {/* §9.6 右列：详情面板 */}
        <div className="w-[360px] shrink-0 border-l overflow-auto">
          {selectedStudent ? (
            <div className="p-4 space-y-4">
              <div>
                <p className="text-sm font-semibold">{selectedStudent.student_name}</p>
                <p className="text-xs text-muted-foreground">{selectedStudent.student_no}</p>
              </div>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="rounded border p-2">
                  <p className="text-muted-foreground">当前进度</p>
                  <p className="text-lg font-bold">第 {selectedStudent.tick} 步</p>
                </div>
                <div className="rounded border p-2">
                  <p className="text-muted-foreground">阶段</p>
                  <p className="text-sm font-medium">{selectedStudent.phase || '-'}</p>
                </div>
                <div className="rounded border p-2 col-span-2">
                  <p className="text-muted-foreground">状态</p>
                  <span className={cn('text-sm font-medium', STATUS_COLORS[selectedStudent.status])}>
                    <StatusDot status={selectedStudent.status} /> {selectedStudent.status}
                  </span>
                </div>
              </div>
              {selectedStudent.error_message && (
                <div className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
                  {selectedStudent.error_message}
                </div>
              )}

              {/* 干预操作 */}
              <div className="space-y-2">
                <p className="text-xs font-medium text-muted-foreground">干预操作</p>
                <div className="grid grid-cols-2 gap-2">
                  <Button size="sm" variant="outline" className="h-8 text-xs gap-1">
                    <Eye className="h-3 w-3" />远程查看
                  </Button>
                  <Button size="sm" variant="outline" className="h-8 text-xs gap-1">
                    <MessageSquare className="h-3 w-3" />私聊
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 text-xs gap-1"
                    onClick={() => intervene('force_step', { target_student_ids: [selectedStudent.student_id] })}
                  >
                    <SkipForward className="h-3 w-3" />推进一步
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 text-xs gap-1"
                    onClick={() => intervene('force_reset', { target_student_ids: [selectedStudent.student_id] })}
                  >
                    <RotateCcw className="h-3 w-3" />强制重置
                  </Button>
                  <Button
                    size="sm"
                    variant="destructive"
                    className="h-8 text-xs gap-1 col-span-2"
                    onClick={() => intervene('kick_student', { target_student_ids: [selectedStudent.student_id] })}
                  >
                    <LogOut className="h-3 w-3" />踢出
                  </Button>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
              选择一位学生查看详情
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
