"use client";

// CourseSettingsPanel.tsx
// 模块03 P-12 课程设置页，聚合课程基础编辑、生命周期、共享策略和课程表配置。

import React from "react";
import { useState } from "react";

import { CourseEditorForm } from "@/components/business/CourseEditorForm";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { useCourse, useCourseLifecycleMutations, useCourseSchedule, useSetCourseScheduleMutation } from "@/hooks/useCourses";
import type { ID } from "@/types/api";

const DAY_OPTIONS = [
  { value: 1, label: "周一" },
  { value: 2, label: "周二" },
  { value: 3, label: "周三" },
  { value: 4, label: "周四" },
  { value: 5, label: "周五" },
  { value: 6, label: "周六" },
  { value: 7, label: "周日" },
] as const;

function getDayLabel(dayOfWeek: number) {
  return DAY_OPTIONS.find((item) => item.value === dayOfWeek)?.label ?? `周${dayOfWeek}`;
}

// CourseSettingsPanel 展示课程编辑、共享状态、生命周期限制和课程表配置。
export function CourseSettingsPanel({ courseID }: { courseID: ID }) {
  const course = useCourse(courseID);
  const lifecycle = useCourseLifecycleMutations(courseID);
  const schedule = useCourseSchedule(courseID);
  const setSchedule = useSetCourseScheduleMutation(courseID);
  const [day, setDay] = useState(3);
  const [start, setStart] = useState("08:00");
  const [end, setEnd] = useState("09:40");
  const [location, setLocation] = useState("");

  const shareHint =
    course.data?.status === 1
      ? "草稿课程不可共享，删除仅允许草稿课程。"
      : "仅草稿课程允许删除；已发布、进行中、已结束课程可共享到课程库。";

  return (
    <div className="space-y-6">
      <CourseEditorForm courseID={courseID} />

      <Card>
        <CardHeader>
          <CardTitle>共享与生命周期</CardTitle>
          <CardDescription>{shareHint}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-3 md:grid-cols-3">
            <div className="rounded-2xl border border-border/70 bg-muted/25 p-4">
              <p className="text-sm text-muted-foreground">当前状态</p>
              <p className="mt-2 text-lg font-semibold">{course.data?.status_text ?? "-"}</p>
            </div>
            <div className="rounded-2xl border border-border/70 bg-muted/25 p-4">
              <p className="text-sm text-muted-foreground">共享状态</p>
              <p className="mt-2 text-lg font-semibold">{course.data?.is_shared ? "已共享" : "未共享"}</p>
            </div>
            <div className="rounded-2xl border border-border/70 bg-muted/25 p-4">
              <p className="text-sm text-muted-foreground">邀请码</p>
              <p className="mt-2 text-lg font-semibold">{course.data?.invite_code ?? "未生成"}</p>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <Button variant="outline" onClick={() => lifecycle.share.mutate(!(course.data?.is_shared ?? false))}>
              {course.data?.is_shared ? "取消共享" : "共享到课程库"}
            </Button>
            <Button variant="outline" onClick={() => lifecycle.clone.mutate()}>
              克隆课程
            </Button>
            <Button variant="destructive" onClick={() => lifecycle.remove.mutate()}>
              删除课程
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>课程表</CardTitle>
          <CardDescription>教师设置每周上课时间、地点，学生端按周视图展示。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-[10rem_1fr_1fr_1.4fr_auto]">
            <select
              aria-label="上课星期"
              className="h-10 rounded-lg border border-input bg-background px-3 text-sm"
              value={day}
              onChange={(event) => setDay(Number(event.target.value))}
            >
              {DAY_OPTIONS.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
            <Input aria-label="开始时间" value={start} onChange={(event) => setStart(event.target.value)} />
            <Input aria-label="结束时间" value={end} onChange={(event) => setEnd(event.target.value)} />
            <Input placeholder="地点" value={location} onChange={(event) => setLocation(event.target.value)} />
            <Button
              onClick={() =>
                setSchedule.mutate({
                  schedules: [
                    {
                      day_of_week: day,
                      start_time: start,
                      end_time: end,
                      location,
                    },
                  ],
                })
              }
            >
              保存课程表
            </Button>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            {(schedule.data ?? []).map((item) => (
              <div key={item.id} className="rounded-2xl border border-border/70 bg-muted/25 p-4 text-sm text-muted-foreground">
                <p className="font-medium text-foreground">
                  {getDayLabel(item.day_of_week)} {item.start_time}-{item.end_time}
                </p>
                <p className="mt-1">{item.location || "地点待定"}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
