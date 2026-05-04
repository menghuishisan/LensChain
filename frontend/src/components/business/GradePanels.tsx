"use client";

// GradePanels.tsx
// 模块06页面级业务面板，组合学习概览、GPA、审核、申诉、预警、成绩单和分析页面。

import { Plus } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { AcademicWarningPanel } from "@/components/business/AcademicWarningPanel";
import { GpaTrendCard } from "@/components/business/GpaTrendCard";
import { GradeReviewPanel } from "@/components/business/GradeReviewPanel";
import { LearningOverviewPanel } from "@/components/business/LearningOverviewPanel";
import { SemesterGradeTable } from "@/components/business/SemesterGradeTable";
import { TranscriptPanel } from "@/components/business/TranscriptPanel";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { Textarea } from "@/components/ui/Textarea";
import { useAcademicWarningMutations, useWarningConfig } from "@/hooks/useAcademicWarnings";
import { useGradeAppeal, useGradeAppealMutations, useGradeAppeals } from "@/hooks/useGradeAppeals";
import { useGradeReviews } from "@/hooks/useGradeReviews";
import { useCourseGradeAnalytics, useLevelConfigMutations, useLevelConfigs, useMySemesterGrades, usePlatformGradeAnalytics, useSchoolGradeAnalytics, useSemesterMutations, useSemesters } from "@/hooks/useGrades";
import { GRADE_APPEAL_STATUS_OPTIONS, getGradeAppealStatusVariant } from "@/lib/grade";
import { formatGPA, formatScore } from "@/lib/format";
import type { ID } from "@/types/api";
import type { GradeAppealStatus, GradeLevelItem } from "@/types/grade";
import { useCourses } from "@/hooks/useCourses";
import { useCourseGrades } from "@/hooks/useAssignments";

/**
 * StudentGradesPanel 学生成绩总览页。
 */
export function StudentGradesPanel() {
  return (
    <div className="space-y-6">
      <LearningOverviewPanel />
      <SemesterGradeTable />
      <div className="flex flex-wrap gap-2">
        <Link className={buttonClassName()} href="/student/grades/gpa">查看GPA趋势</Link>
        <Link className={buttonClassName({ variant: "outline" })} href="/student/grades/transcripts">下载成绩单</Link>
        <Link className={buttonClassName({ variant: "ghost" })} href="/student/grades/appeals">成绩申诉</Link>
      </div>
    </div>
  );
}

/**
 * StudentGpaPanel 学生 GPA 页。
 */
export function StudentGpaPanel() {
  return <GpaTrendCard />;
}

/**
 * StudentGradeAppealsPanel 学生成绩申诉页。
 */
export function StudentGradeAppealsPanel() {
  const [status, setStatus] = useState<GradeAppealStatus | "all">("all");
  const appealsQuery = useGradeAppeals({ page: 1, page_size: 20, status: status === "all" ? undefined : status });
  const gradesQuery = useMySemesterGrades();
  const appealMutations = useGradeAppealMutations();
  const [gradeID, setGradeID] = useState("");
  const [reason, setReason] = useState("");

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>提交新申诉</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-[1fr_2fr_auto]">
          <FormField label="课程成绩">
            <Select value={gradeID} onValueChange={setGradeID}>
              <SelectTrigger><SelectValue placeholder="选择课程成绩" /></SelectTrigger>
              <SelectContent>
                {(gradesQuery.data?.grades ?? []).map((grade) => (
                  <SelectItem key={grade.grade_id} value={grade.grade_id}>{grade.course_name} · {formatScore(grade.final_score)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="申诉理由">
            <Textarea value={reason} onChange={(event) => setReason(event.target.value)} rows={4} />
          </FormField>
          <Button className="self-end" disabled={!gradeID || reason.trim().length < 20} onClick={() => appealMutations.create.mutate({ grade_id: gradeID, appeal_reason: reason })} isLoading={appealMutations.create.isPending}>
            提交申诉
          </Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>我的申诉记录</CardTitle>
          <Select value={String(status)} onValueChange={(value) => setStatus(value === "all" ? "all" : (Number(value) as GradeAppealStatus))}>
            <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              {GRADE_APPEAL_STATUS_OPTIONS.map((item) => <SelectItem key={item.value} value={String(item.value)}>{item.label}</SelectItem>)}
            </SelectContent>
          </Select>
        </CardHeader>
        <CardContent className="space-y-3">
          {(appealsQuery.data?.list ?? []).length === 0 ? <EmptyState title="暂无申诉记录" description="提交申诉后会在这里展示处理进度。" /> : null}
          {(appealsQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{item.course_name}</p>
                  <p className="mt-1 text-sm text-muted-foreground">原成绩 {formatScore(item.original_score)} · {item.semester_name}</p>
                </div>
                <Badge variant={getGradeAppealStatusVariant(item.status)}>{item.status_text}</Badge>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">申诉提交时间：{item.created_at.slice(0, 10)}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * StudentTranscriptsPanel 学生成绩单页。
 */
export function StudentTranscriptsPanel() {
  return <TranscriptPanel />;
}

/**
 * TeacherGradeReviewsPanel 教师成绩审核提交页。
 */
export function TeacherGradeReviewsPanel() {
  const semestersQuery = useSemesters({ page: 1, page_size: 20 });
  const coursesQuery = useCourses({ page: 1, page_size: 50 });
  const currentSemester = semestersQuery.data?.list.find((item) => item.is_current) ?? semestersQuery.data?.list[0];
  return (
    <div className="space-y-5">
      <GradeReviewPanel mode="teacher" />
      <Card>
        <CardHeader>
          <CardTitle>我的课程</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(coursesQuery.data?.list ?? []).map((course) => (
            <TeacherCourseReviewItem key={course.id} courseID={course.id} courseName={course.title} semesterID={currentSemester?.id} semesterName={currentSemester?.name ?? "当前学期"} />
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * TeacherGradeAppealsPanel 教师申诉处理页。
 */
export function TeacherGradeAppealsPanel() {
  const appealsQuery = useGradeAppeals({ page: 1, page_size: 20 });
  const [appealID, setAppealID] = useState("");
  const appealQuery = useGradeAppeal(appealID);
  const mutations = useGradeAppealMutations(appealID);
  const [comment, setComment] = useState("");
  const [newScore, setNewScore] = useState("90");

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>申诉处理列表</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(appealsQuery.data?.list ?? []).map((item) => (
            <button key={item.id} className="w-full rounded-xl border border-border p-4 text-left" onClick={() => setAppealID(item.id)}>
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{item.student_name} · {item.course_name}</p>
                  <p className="mt-1 text-sm text-muted-foreground">原成绩 {formatScore(item.original_score)}</p>
                </div>
                <Badge variant={getGradeAppealStatusVariant(item.status)}>{item.status_text}</Badge>
              </div>
            </button>
          ))}
        </CardContent>
      </Card>
      {appealQuery.data ? (
        <Card>
          <CardHeader>
            <CardTitle>申诉详情</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
          <div className="rounded-xl border border-border p-4 text-sm text-muted-foreground whitespace-pre-wrap">{appealQuery.data.appeal_reason}</div>
            <div className="grid gap-3 md:grid-cols-3">
              <Metric title="原成绩" value={formatScore(appealQuery.data.original_score)} />
              <Metric title="当前状态" value={appealQuery.data.status_text} />
              <Metric title="申诉时间" value={appealQuery.data.created_at ? appealQuery.data.created_at.slice(0, 10) : "-"} />
            </div>
            <div className="grid gap-3 md:grid-cols-[160px_1fr]">
              <FormField label="新成绩">
                <Input type="number" value={newScore} onChange={(event) => setNewScore(event.target.value)} />
              </FormField>
              <FormField label="处理意见">
                <Textarea value={comment} onChange={(event) => setComment(event.target.value)} rows={4} />
              </FormField>
            </div>
            <div className="flex gap-2">
              <Button onClick={() => mutations.approve.mutate({ new_score: Number(newScore), handle_comment: comment })} isLoading={mutations.approve.isPending}>同意修改</Button>
              <Button variant="destructive" onClick={() => mutations.reject.mutate({ handle_comment: comment })} isLoading={mutations.reject.isPending}>驳回申诉</Button>
            </div>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

/**
 * TeacherGradeAnalyticsPanel 教师课程成绩分析页。
 */
export function TeacherGradeAnalyticsPanel({ courseID }: { courseID: ID }) {
  const analyticsQuery = useCourseGradeAnalytics(courseID);
  const analytics = analyticsQuery.data;

  return (
    <div className="space-y-5">
      <div className="grid gap-3 md:grid-cols-5">
        <Metric title="学生数" value={String(analytics?.student_count ?? 0)} />
        <Metric title="平均分" value={formatScore(analytics?.average_score ?? 0)} />
        <Metric title="中位数" value={formatScore(analytics?.median_score ?? 0)} />
        <Metric title="最高分" value={formatScore(analytics?.max_score ?? 0)} />
        <Metric title="及格率" value={`${Math.round((analytics?.pass_rate ?? 0) * 100)}%`} />
      </div>
      <Card>
        <CardHeader>
          <CardTitle>成绩分布</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {(analytics?.score_distribution ?? []).map((item) => (
            <div key={item.range} className="flex items-center gap-3 text-sm">
              <span className="w-16">{item.range}</span>
              <div className="h-2 flex-1 rounded-full bg-muted"><div className="h-2 rounded-full bg-primary" style={{ width: `${Math.min(100, item.count * 10)}%` }} /></div>
              <span>{item.count}</span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * AdminSemestersPanel 学校管理员学期管理页。
 */
export function AdminSemestersPanel() {
  const semestersQuery = useSemesters({ page: 1, page_size: 20 });
  const mutations = useSemesterMutations();
  const [form, setForm] = useState({ name: "", code: "", start_date: "", end_date: "" });

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>创建学期</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-[1fr_1fr_1fr_1fr_auto]">
          <Input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} placeholder="学期名称" />
          <Input value={form.code} onChange={(event) => setForm((current) => ({ ...current, code: event.target.value }))} placeholder="编码" />
          <Input type="date" value={form.start_date} onChange={(event) => setForm((current) => ({ ...current, start_date: event.target.value }))} />
          <Input type="date" value={form.end_date} onChange={(event) => setForm((current) => ({ ...current, end_date: event.target.value }))} />
          <Button onClick={() => mutations.create.mutate(form)} isLoading={mutations.create.isPending}><Plus className="h-4 w-4" />创建</Button>
        </CardContent>
      </Card>
      <TableContainer>
        <Table>
          <TableHeader><TableRow><TableHead>学期</TableHead><TableHead>编码</TableHead><TableHead>起止日期</TableHead><TableHead>课程数</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
          <TableBody>
            {(semestersQuery.data?.list ?? []).map((semester) => (
              <TableRow key={semester.id}>
                <TableCell>{semester.name}</TableCell>
                <TableCell>{semester.code}</TableCell>
                <TableCell>{semester.start_date} - {semester.end_date}</TableCell>
                <TableCell>{semester.course_count ?? 0}</TableCell>
                <TableCell className="space-x-2">
                  <Button size="sm" variant="outline" onClick={() => mutations.setCurrent.mutate(semester.id)}>设为当前</Button>
                  <Button size="sm" variant="destructive" onClick={() => mutations.remove.mutate(semester.id)}>删除</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <p className="text-sm text-muted-foreground">有课程关联的学期不可删除；设为当前学期后，其他学期会自动取消当前标记。</p>
    </div>
  );
}

/**
 * AdminLevelConfigsPanel 学校管理员等级映射配置页。
 */
export function AdminLevelConfigsPanel() {
  const levelQuery = useLevelConfigs();
  const mutations = useLevelConfigMutations();
  const [levels, setLevels] = useState(levelQuery.data?.levels ?? []);

  const data = levels.length > 0 ? levels : levelQuery.data?.levels ?? [];

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle>等级映射配置</CardTitle>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => mutations.resetDefault.mutate()} isLoading={mutations.resetDefault.isPending}>重置默认</Button>
          <Button onClick={() => mutations.update.mutate(data)} isLoading={mutations.update.isPending}>保存</Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {data.map((item, index) => (
          <div key={item.id ?? `${item.level_name}-${index}`} className="grid gap-3 md:grid-cols-4">
            <Input value={item.level_name} onChange={(event) => setLevels((current) => replaceLevelRow(current, data, index, { ...item, level_name: event.target.value }))} />
            <Input type="number" value={item.min_score} onChange={(event) => setLevels((current) => replaceLevelRow(current, data, index, { ...item, min_score: Number(event.target.value) }))} />
            <Input type="number" value={item.max_score} onChange={(event) => setLevels((current) => replaceLevelRow(current, data, index, { ...item, max_score: Number(event.target.value) }))} />
            <Input type="number" value={item.gpa_point} onChange={(event) => setLevels((current) => replaceLevelRow(current, data, index, { ...item, gpa_point: Number(event.target.value) }))} />
          </div>
        ))}
        <Button variant="outline" onClick={() => setLevels((current) => [...(current.length > 0 ? current : data), { level_name: "新等级", min_score: 0, max_score: 0, gpa_point: 0 }])}>添加等级</Button>
        <div className="rounded-xl border border-border p-4">
          <p className="font-semibold">区间预览</p>
          <div className="mt-3 flex h-6 overflow-hidden rounded-full bg-muted">
            {data.map((item) => (
              <div key={`${item.level_name}-${item.min_score}`} className="grid place-items-center border-r border-background/40 bg-primary/15 text-xs font-semibold" style={{ width: `${Math.max(4, item.max_score - item.min_score + 0.01)}%` }}>
                {item.level_name}
              </div>
            ))}
          </div>
        </div>
        <p className="text-sm text-muted-foreground">保存前请确保所有区间完整覆盖 0-100 且互不重叠，绩点范围为 0-4。</p>
      </CardContent>
    </Card>
  );
}

/**
 * AdminGradeReviewsPanel 学校管理员成绩审核列表页。
 */
export function AdminGradeReviewsPanel({ reviewID }: { reviewID?: ID }) {
  return <GradeReviewPanel mode="admin" reviewID={reviewID} />;
}

/**
 * AdminWarningsPanel 学校管理员预警管理页。
 */
export function AdminWarningsPanel({ warningID }: { warningID?: ID }) {
  return <AcademicWarningPanel warningID={warningID} />;
}

/**
 * AdminWarningConfigPanel 学校管理员预警配置页。
 */
export function AdminWarningConfigPanel() {
  const configQuery = useWarningConfig();
  const mutations = useAcademicWarningMutations();
  const [gpaThreshold, setGpaThreshold] = useState(String(configQuery.data?.gpa_threshold ?? 2));
  const [failCountThreshold, setFailCountThreshold] = useState(String(configQuery.data?.fail_count_threshold ?? 2));
  const [isEnabled, setIsEnabled] = useState(configQuery.data?.is_enabled ?? true);

  return (
    <Card>
      <CardHeader>
        <CardTitle>学业预警配置</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 md:grid-cols-[1fr_1fr_auto]">
        <FormField label="GPA阈值">
          <Input type="number" step="0.01" value={gpaThreshold} onChange={(event) => setGpaThreshold(event.target.value)} />
        </FormField>
        <FormField label="挂科阈值">
          <Input type="number" value={failCountThreshold} onChange={(event) => setFailCountThreshold(event.target.value)} />
        </FormField>
        <Button className="self-end" onClick={() => mutations.updateConfig.mutate({ gpa_threshold: Number(gpaThreshold), fail_count_threshold: Number(failCountThreshold), is_enabled: isEnabled })} isLoading={mutations.updateConfig.isPending}>
          保存配置
        </Button>
      </CardContent>
    </Card>
  );
}

/**
 * AdminGradeAnalyticsPanel 学校管理员全校成绩分析页。
 */
export function AdminGradeAnalyticsPanel() {
  const analyticsQuery = useSchoolGradeAnalytics();
  const analytics = analyticsQuery.data;

  return (
    <div className="space-y-5">
      <div className="grid gap-3 md:grid-cols-4">
        <Metric title="学生总数" value={String(analytics?.total_students ?? 0)} />
        <Metric title="平均GPA" value={formatGPA(analytics?.average_gpa ?? 0)} />
        <Metric title="不及格率" value={`${Math.round((analytics?.fail_rate ?? 0) * 100)}%`} />
        <Metric title="预警数" value={String(analytics?.warning_count ?? 0)} />
      </div>
      <Card>
        <CardHeader><CardTitle>课程排行</CardTitle></CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <ListBlock title="最高平均分课程" items={analytics?.top_courses.map((item) => `${item.course_name} ${formatScore(item.average_score)}`) ?? []} />
          <ListBlock title="最低平均分课程" items={analytics?.bottom_courses.map((item) => `${item.course_name} ${formatScore(item.average_score)}`) ?? []} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader><CardTitle>GPA分布</CardTitle></CardHeader>
        <CardContent className="space-y-2">
          {(analytics?.gpa_distribution ?? []).map((item) => (
            <div key={item.range} className="flex items-center gap-3 text-sm">
              <span className="w-24">{item.range}</span>
              <div className="h-2 flex-1 rounded-full bg-muted">
                <div className="h-2 rounded-full bg-primary" style={{ width: `${Math.min(100, item.count * 5)}%` }} />
              </div>
              <span>{item.count}</span>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * SuperGradeAnalyticsPanel 超级管理员平台成绩总览页。
 */
export function SuperGradeAnalyticsPanel() {
  const analyticsQuery = usePlatformGradeAnalytics();
  const analytics = analyticsQuery.data;

  return (
    <div className="space-y-5">
      <div className="grid gap-3 md:grid-cols-3">
        <Metric title="学校数" value={String(analytics?.total_schools ?? 0)} />
        <Metric title="学生数" value={String(analytics?.total_students ?? 0)} />
        <Metric title="平台平均GPA" value={formatGPA(analytics?.platform_average_gpa ?? 0)} />
      </div>
      <TableContainer>
        <Table>
          <TableHeader><TableRow><TableHead>学校</TableHead><TableHead>学生数</TableHead><TableHead>平均GPA</TableHead></TableRow></TableHeader>
          <TableBody>
            {(analytics?.school_comparison ?? []).map((item) => (
              <TableRow key={item.school_name}>
                <TableCell>{item.school_name}</TableCell>
                <TableCell>{item.student_count}</TableCell>
                <TableCell>{formatGPA(item.average_gpa)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
}

function Metric({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-sm text-muted-foreground">{title}</p>
        <p className="mt-1 font-display text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

function ListBlock({ title, items }: { title: string; items: string[] }) {
  return (
    <div className="rounded-xl border border-border p-4">
      <p className="font-semibold">{title}</p>
      <div className="mt-3 space-y-2">
        {items.length === 0 ? <p className="text-sm text-muted-foreground">暂无数据。</p> : null}
        {items.map((item) => <div key={item} className="text-sm text-muted-foreground">{item}</div>)}
      </div>
    </div>
  );
}

function replaceLevelRow(current: GradeLevelItem[], fallback: GradeLevelItem[], index: number, nextValue: GradeLevelItem) {
  const base = Array.isArray(current) && current.length > 0 ? current : fallback;
  return base.map((item, itemIndex) => (itemIndex === index ? nextValue : item));
}

function TeacherCourseReviewItem({ courseID, courseName, semesterID, semesterName }: { courseID: ID; courseName: string; semesterID?: ID; semesterName: string }) {
  const gradeData = useCourseGrades(courseID);
  const summary = gradeData.summary.data;
  const studentCount = summary?.students.length ?? 0;
  const completedCount = summary?.students.filter((student) => Number.isFinite(student.final_score)).length ?? 0;
  const reviewStatus = completedCount === 0 ? "未提交" : completedCount < studentCount ? "未完成" : "可提交";

  return (
    <div className="rounded-xl border border-border p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="font-semibold">{courseName}</p>
          <p className="mt-1 text-sm text-muted-foreground">{semesterName} · 成绩完成 {completedCount}/{studentCount}</p>
        </div>
        <div className="flex items-center gap-2">
          <Link className={buttonClassName({ variant: "outline", size: "sm" })} href={`/teacher/grades/analytics/${courseID}`}>成绩分析</Link>
          <Badge variant={reviewStatus === "可提交" ? "success" : reviewStatus === "未提交" ? "outline" : "destructive"}>{reviewStatus}</Badge>
        </div>
      </div>
      {reviewStatus === "可提交" && semesterID ? (
        <p className="mt-2 text-xs text-muted-foreground">请在上方“成绩审核提交”区域填入课程ID `{courseID}` 和学期ID `{semesterID}` 提交审核。</p>
      ) : null}
    </div>
  );
}
