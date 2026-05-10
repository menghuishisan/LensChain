'use client';

// SimTeacherInterventionPanel.tsx
// 教师实时干预面板（06.2 §七.2）。
// TopBar [干预] 按钮触发右侧抽屉 slide-in（width 360px）。
// 包含：广播消息、强制暂停/恢复、推送 step、强制重置、解锁联动时钟、调试共享状态、踢出学生。

import { useCallback, useState } from 'react';
import {
  LogOut,
  Megaphone,
  Pause,
  Play,
  RotateCcw,
  SkipForward,
  Unlock,
  Bug,
  X,
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Textarea } from '@/components/ui/Textarea';
import { cn } from '@/lib/utils';
import { teacherIntervene } from '@/services/experiment';
import type { ID } from '@/types/api';
import type { TeacherInterveneAction, TeacherInterveneRequest } from '@/types/experiment';

export interface SimTeacherInterventionPanelProps {
  experimentID: ID;
  open: boolean;
  onClose: () => void;
  className?: string;
}

/**
 * SimTeacherInterventionPanel 教师干预右侧抽屉面板（06.2 §七.2）。
 */
export function SimTeacherInterventionPanel({
  experimentID,
  open,
  onClose,
  className,
}: SimTeacherInterventionPanelProps) {
  const [broadcastMsg, setBroadcastMsg] = useState('');
  const [targetStudentId, setTargetStudentId] = useState('');
  const [targetSceneCode, setTargetSceneCode] = useState('');
  const [targetLinkGroup, setTargetLinkGroup] = useState('');
  const [debugField, setDebugField] = useState('');
  const [debugValue, setDebugValue] = useState('');
  const [loading, setLoading] = useState<TeacherInterveneAction | null>(null);

  const intervene = useCallback(
    async (action: TeacherInterveneAction, extra?: Partial<TeacherInterveneRequest>) => {
      setLoading(action);
      try {
        const req: TeacherInterveneRequest = {
          action,
          target_student_ids: targetStudentId ? [targetStudentId] : undefined,
          target_scene_code: targetSceneCode || undefined,
          ...extra,
        };
        await teacherIntervene(experimentID, req);
      } finally {
        setLoading(null);
      }
    },
    [experimentID, targetStudentId, targetSceneCode],
  );

  if (!open) return null;

  return (
    <div
      className={cn(
        'fixed right-0 top-0 z-50 h-full w-[360px] border-l bg-background shadow-2xl transition-transform',
        open ? 'translate-x-0' : 'translate-x-full',
        className,
      )}
    >
      {/* 头部 */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <p className="text-sm font-semibold">教师干预面板</p>
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="overflow-auto p-4 space-y-5" style={{ height: 'calc(100% - 49px)' }}>
        {/* 目标选择 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">目标（可选）</p>
          <Input className="h-7 text-xs" placeholder="指定学生（留空则对全体生效）" value={targetStudentId} onChange={(e) => setTargetStudentId(e.target.value)} />
          <Input className="h-7 text-xs" placeholder="指定场景（留空则对所有场景生效）" value={targetSceneCode} onChange={(e) => setTargetSceneCode(e.target.value)} />
        </div>

        {/* §7.2 广播消息 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground flex items-center gap-1"><Megaphone className="h-3 w-3" />广播消息</p>
          <div className="flex gap-2">
            <Input className="h-8 text-xs flex-1" placeholder="输入广播内容..." value={broadcastMsg} onChange={(e) => setBroadcastMsg(e.target.value)} />
            <Button
              size="sm"
              className="h-8 text-xs gap-1"
              disabled={!broadcastMsg.trim()}
              isLoading={loading === 'broadcast_message'}
              onClick={() => {
                void intervene('broadcast_message', { message: broadcastMsg });
                setBroadcastMsg('');
              }}
            >
              <Megaphone className="h-3 w-3" />发送
            </Button>
          </div>
        </div>

        {/* 控制类操作 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">控制操作</p>
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant="outline"
              size="sm"
              className="h-9 text-xs gap-1 justify-start"
              isLoading={loading === 'force_pause_all'}
              onClick={() => void intervene('force_pause_all')}
            >
              <Pause className="h-3.5 w-3.5" />强制暂停所有
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-9 text-xs gap-1 justify-start"
              isLoading={loading === 'force_resume_all'}
              onClick={() => void intervene('force_resume_all')}
            >
              <Play className="h-3.5 w-3.5" />强制恢复所有
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-9 text-xs gap-1 justify-start"
              isLoading={loading === 'force_step'}
              onClick={() => void intervene('force_step')}
            >
              <SkipForward className="h-3.5 w-3.5" />推进一步
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-9 text-xs gap-1 justify-start"
              isLoading={loading === 'force_reset'}
              onClick={() => void intervene('force_reset')}
            >
              <RotateCcw className="h-3.5 w-3.5" />强制重置
            </Button>
          </div>
        </div>

        {/* 联动操作 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">联动操作</p>
          <Input className="h-7 text-xs" placeholder="选择联动组" value={targetLinkGroup} onChange={(e) => setTargetLinkGroup(e.target.value)} />
          <Button
            variant="outline"
            size="sm"
            className="h-9 w-full text-xs gap-1 justify-start"
            isLoading={loading === 'unlock_link_clock'}
            onClick={() => void intervene('unlock_link_clock', { target_link_group_id: targetLinkGroup })}
          >
            <Unlock className="h-3.5 w-3.5" />解锁联动时钟同步
          </Button>
          <div className="grid grid-cols-[1fr_1fr_auto] gap-2">
            <Input className="h-7 text-xs" placeholder="字段名" value={debugField} onChange={(e) => setDebugField(e.target.value)} />
            <Input className="h-7 text-xs" placeholder="值" value={debugValue} onChange={(e) => setDebugValue(e.target.value)} />
            <Button
              variant="outline"
              size="sm"
              className="h-7 text-xs gap-1"
              isLoading={loading === 'debug_shared_state'}
              onClick={() => void intervene('debug_shared_state', { target_link_group_id: targetLinkGroup, field_name: debugField, field_value: debugValue })}
            >
              <Bug className="h-3 w-3" />写入
            </Button>
          </div>
        </div>

        {/* 踢出学生 */}
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">违规处理</p>
          <Button
            variant="destructive"
            size="sm"
            className="h-9 w-full text-xs gap-1"
            disabled={!targetStudentId}
            isLoading={loading === 'kick_student'}
            onClick={() => void intervene('kick_student')}
          >
            <LogOut className="h-3.5 w-3.5" />踢出学生
          </Button>
          {!targetStudentId && <p className="text-[10px] text-muted-foreground">请先在上方填写目标学生 ID。</p>}
        </div>
      </div>
    </div>
  );
}
