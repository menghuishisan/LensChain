"use client";

// SemesterGradeTable.tsx
// 模块06学期成绩表组件，展示课程、学分、分数、等级、绩点和审核状态。

import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { useMySemesterGrades } from "@/hooks/useGrades";
import { getGradeReviewStatusVariant } from "@/lib/grade";
import { formatGPA, formatScore } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * SemesterGradeTable 组件属性。
 */
export interface SemesterGradeTableProps {
  semesterID?: ID;
}

/**
 * SemesterGradeTable 学期成绩表组件。
 */
export function SemesterGradeTable({ semesterID }: SemesterGradeTableProps) {
  const gradesQuery = useMySemesterGrades(semesterID ? { semester_id: semesterID } : {});

  if (gradesQuery.isLoading) {
    return <LoadingState title="正在加载学期成绩" description="读取课程、学分、分数、等级和绩点。" />;
  }

  if (gradesQuery.isError) {
    return <ErrorState title="学期成绩加载失败" description={gradesQuery.error.message} />;
  }

  if (!gradesQuery.data || gradesQuery.data.grades.length === 0) {
    return <EmptyState title="暂无学期成绩" description="待课程成绩审核通过后在这里展示。" />;
  }

  const result = gradesQuery.data;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{result.semester?.name ?? "当前学期"}成绩</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-4">
          <SummaryStat title="学期GPA" value={formatGPA(result.summary?.semester_gpa ?? 0)} />
          <SummaryStat title="总学分" value={formatScore(result.summary?.total_credits ?? 0)} />
          <SummaryStat title="已通过" value={String(result.summary?.passed_count ?? 0)} />
          <SummaryStat title="未通过" value={String(result.summary?.failed_count ?? 0)} />
        </div>
        <TableContainer>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>课程</TableHead>
                <TableHead>教师</TableHead>
                <TableHead>学分</TableHead>
                <TableHead>成绩</TableHead>
                <TableHead>等级</TableHead>
                <TableHead>绩点</TableHead>
                <TableHead>状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.grades.map((grade) => (
                <TableRow key={grade.course_id}>
                  <TableCell>{grade.course_name}</TableCell>
                  <TableCell>{grade.teacher_name}</TableCell>
                  <TableCell>{formatScore(grade.credits, 1)}</TableCell>
                  <TableCell>{formatScore(grade.final_score)}</TableCell>
                  <TableCell>{grade.grade_level}</TableCell>
                  <TableCell>{formatGPA(grade.gpa_point)}</TableCell>
                  <TableCell>
                    <Badge variant={grade.review_status === "approved" ? getGradeReviewStatusVariant(3) : "outline"}>{grade.review_status_text}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </CardContent>
    </Card>
  );
}

function SummaryStat({ title, value }: { title: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-muted/30 p-4">
      <p className="text-sm text-muted-foreground">{title}</p>
      <p className="mt-1 font-display text-2xl font-semibold">{value}</p>
    </div>
  );
}
