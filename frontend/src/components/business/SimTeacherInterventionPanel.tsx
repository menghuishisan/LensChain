'use client';

/**
 * SimTeacherInterventionPanel — 教师实时干预抽屉（06.2 §7.2）。
 *
 * TopBar Settings2 按钮触发右侧抽屉（width 360px）。
 * 10 个干预按钮，与 backend internal/model/enum/experiment.go::InterveneType* 严格同名。
 * 按 redesign-proposal.html §①-B 重分 4 组：参数注入 / 事件触发 / 容器干预 / 会话管理。
 *
 * 本组件只发出“干预意图”事件；具体 HTTP 请求由父组件（SimEnginePanel）走 services 层完成。
 */

import { useState } from 'react';
import { Bug, ClipboardEdit, FastForward, LogOut, Megaphone, Pause, Play, RotateCcw, Settings2, SkipForward, Unlock } from 'lucide-react';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/Sheet';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Textarea } from '@/components/ui/Textarea';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { cn } from '@/lib/utils';

/**
 * 与 backend internal/model/enum/experiment.go::InterveneType* 严格同名的 10 个动作。
 */
export type InterventionType =
  | 'broadcast'
  | 'pause_all'
  | 'resume_all'
  | 'force_step'
  | 'force_reset'
  | 'push_step'
  | 'debug_shared_state'
  | 'unlock_link_clock'
  | 'kick_student'
  | 'annotation';

export interface InterventionPayload {
  type: InterventionType;
  /** broadcast 文字 / push_step 步骤 ID 等附加参数。 */
  data?: Record<string, unknown>;
}

export interface SimTeacherInterventionPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onIntervene: (payload: InterventionPayload) => void;
}

export function SimTeacherInterventionPanel(props: SimTeacherInterventionPanelProps) {
  const { open, onOpenChange, onIntervene } = props;
  const [broadcastText, setBroadcastText] = useState('');
  const [confirmType, setConfirmType] = useState<InterventionType | null>(null);

  const requestConfirm = (type: InterventionType) => setConfirmType(type);
  const handleConfirm = () => {
    if (!confirmType) return;
    onIntervene({ type: confirmType });
    setConfirmType(null);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="w-[360px]" className="p-0">
        <SheetHeader className="border-b border-border px-4 py-3">
          <SheetTitle className="flex items-center gap-1.5 text-sm">
            <Settings2 className="h-3.5 w-3.5" /> 教师干预
          </SheetTitle>
        </SheetHeader>
        <div className="flex flex-col gap-3 p-3">

          {/* 1． 参数注入：向全班/个人注入文字 / 内容 / 参数 */}
          <Section title="参数注入" icon={<Megaphone className="h-3.5 w-3.5" />}>
            <Textarea
              value={broadcastText}
              onChange={e => setBroadcastText(e.target.value)}
              placeholder="向全班学生广播一条文字"
              className="h-16 text-xs"
            />
            <Button
              size="sm"
              variant="primary"
              disabled={!broadcastText.trim()}
              onClick={() => {
                onIntervene({ type: 'broadcast', data: { text: broadcastText.trim() } });
                setBroadcastText('');
              }}
              className="h-7 self-end text-xs"
            >
              <Megaphone className="mr-1 h-3.5 w-3.5" /> 广播发送
            </Button>
          </Section>

          {/* 2． 事件触发：强制推进 / 重置 / 推送指定步骤 / 老师标注 */}
          <Section title="事件触发" icon={<FastForward className="h-3.5 w-3.5" />}>
            <div className="grid grid-cols-2 gap-2">
              <Button size="sm" variant="outline" onClick={() => requestConfirm('force_step')} className="h-7 text-xs">
                <FastForward className="mr-1 h-3.5 w-3.5" /> 强制步进
              </Button>
              <Button size="sm" variant="outline" onClick={() => requestConfirm('force_reset')} className="h-7 text-xs">
                <RotateCcw className="mr-1 h-3.5 w-3.5" /> 强制重置
              </Button>
              <Button size="sm" variant="outline" onClick={() => requestConfirm('push_step')} className="h-7 text-xs">
                <SkipForward className="mr-1 h-3.5 w-3.5" /> 推送步骤
              </Button>
              <Button size="sm" variant="outline" onClick={() => requestConfirm('annotation')} className="h-7 text-xs">
                <ClipboardEdit className="mr-1 h-3.5 w-3.5" /> 推送标注
              </Button>
            </div>
          </Section>

          {/* 3． 容器干预：全体启停与联动状态调试 */}
          <Section title="容器干预" icon={<Pause className="h-3.5 w-3.5" />}>
            <div className="grid grid-cols-2 gap-2">
              <Button size="sm" variant="outline" onClick={() => requestConfirm('pause_all')} className="h-7 text-xs">
                <Pause className="mr-1 h-3.5 w-3.5" /> 全部暂停
              </Button>
              <Button size="sm" variant="outline" onClick={() => requestConfirm('resume_all')} className="h-7 text-xs">
                <Play className="mr-1 h-3.5 w-3.5" /> 全部恢复
              </Button>
              <Button size="sm" variant="outline" onClick={() => requestConfirm('unlock_link_clock')} className="h-7 text-xs">
                <Unlock className="mr-1 h-3.5 w-3.5" /> 解锁同步
              </Button>
              <Button size="sm" variant="outline" onClick={() => requestConfirm('debug_shared_state')} className="h-7 text-xs">
                <Bug className="mr-1 h-3.5 w-3.5" /> 调试状态
              </Button>
            </div>
          </Section>

          {/* 4． 会话管理：踢出、后续可扩充“结束会话 / 转移” */}
          <Section title="会话管理" icon={<LogOut className="h-3.5 w-3.5" />}>
            <KickStudentRow onKick={(studentId) => onIntervene({ type: 'kick_student', data: { student_id: studentId } })} />
          </Section>
        </div>
      </SheetContent>

      <ConfirmDialog
        open={confirmType !== null}
        onOpenChange={(o) => { if (!o) setConfirmType(null); }}
        title={`确认 ${interventionLabel(confirmType)}？`}
        description="该操作会影响所选学生 / 全班，无法直接撤销。"
        confirmText="确认执行"
        confirmVariant="destructive"
        onConfirm={handleConfirm}
      />
    </Sheet>
  );
}

function Section(props: { title: string; icon?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-2 rounded border border-border/60 p-2">
      <p className="flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
        {props.icon} {props.title}
      </p>
      <div className={cn('flex flex-col gap-1.5')}>{props.children}</div>
    </div>
  );
}

function KickStudentRow(props: { onKick: (studentID: string) => void }) {
  const [studentId, setStudentId] = useState('');
  return (
    <div className="flex items-center gap-2">
      <Input
        value={studentId}
        onChange={e => setStudentId(e.target.value)}
        placeholder="学生 ID"
        className="h-7 flex-1 text-xs"
      />
      <Button
        size="sm"
        variant="destructive"
        disabled={!studentId.trim()}
        onClick={() => { props.onKick(studentId.trim()); setStudentId(''); }}
        className="h-7 text-xs"
      >
        <LogOut className="mr-1 h-3.5 w-3.5" /> 踢出
      </Button>
    </div>
  );
}

function interventionLabel(t: InterventionType | null): string {
  if (!t) return '';
  switch (t) {
    case 'broadcast': return '广播消息';
    case 'pause_all': return '全部暂停';
    case 'resume_all': return '全部恢复';
    case 'force_step': return '强制步进';
    case 'force_reset': return '强制重置';
    case 'push_step': return '推送指定步骤';
    case 'debug_shared_state': return '调试 SharedState';
    case 'unlock_link_clock': return '解锁联动时钟';
    case 'kick_student': return '踢出学生';
    case 'annotation': return '推送老师标注';
  }
}
