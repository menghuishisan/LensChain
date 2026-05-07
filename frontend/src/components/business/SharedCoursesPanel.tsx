"use client";

// SharedCoursesPanel.tsx
// 模块03 P-33 共享课程库，支持筛选、分页、空状态和课程卡片列表。

import React from "react";
import { useState } from "react";

import { CourseCard } from "@/components/business/CourseCard";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { useSharedCourses } from "@/hooks/useCourses";
import type { CourseDifficulty, CourseType } from "@/types/course";

// SharedCoursesPanel 展示共享课程列表，并提供课程类型、难度和主题筛选。
export function SharedCoursesPanel() {
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [courseType, setCourseType] = useState("");
  const [difficulty, setDifficulty] = useState("");
  const [topic, setTopic] = useState("");
  const query = useSharedCourses({
    page,
    page_size: 20,
    keyword: keyword.trim() || undefined,
    course_type: courseType ? (Number(courseType) as CourseType) : undefined,
    difficulty: difficulty ? (Number(difficulty) as CourseDifficulty) : undefined,
    topic: topic.trim() || undefined,
  });

  if (query.isLoading) {
    return <LoadingState variant="grid" />;
  }

  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }

  const list = query.data?.list ?? [];

  return (
    <div className="space-y-5">
      <CardHeaderBlock />

      <div className="grid gap-3 rounded-3xl border border-border/70 bg-card/90 p-4 md:grid-cols-2 xl:grid-cols-4">
        <Input
          placeholder="搜索课程标题或主题"
          value={keyword}
          onChange={(event) => {
            setKeyword(event.target.value);
            setPage(1);
          }}
        />
        <FilterSelect
          label="课程类型"
          value={courseType}
          onChange={(value) => {
            setCourseType(value);
            setPage(1);
          }}
          options={[
            { value: "", label: "全部课程类型" },
            { value: "1", label: "理论课" },
            { value: "2", label: "实验课" },
            { value: "3", label: "混合课" },
            { value: "4", label: "项目实战" },
          ]}
        />
        <FilterSelect
          label="课程难度"
          value={difficulty}
          onChange={(value) => {
            setDifficulty(value);
            setPage(1);
          }}
          options={[
            { value: "", label: "全部难度" },
            { value: "1", label: "入门" },
            { value: "2", label: "进阶" },
            { value: "3", label: "高级" },
            { value: "4", label: "研究" },
          ]}
        />
        <Input
          placeholder="主题方向"
          value={topic}
          onChange={(event) => {
            setTopic(event.target.value);
            setPage(1);
          }}
        />
      </div>

      {list.length === 0 ? (
        <EmptyState title="暂无共享课程" description="当前筛选条件下没有可克隆的课程，稍后再试或调整筛选条件。" />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {list.map((item) => (
            <CourseCard key={item.id} course={item} href={`/shared-courses/${item.id}`} />
          ))}
        </div>
      )}

      {query.data?.pagination ? (
        <Pagination
          page={query.data.pagination.page}
          totalPages={query.data.pagination.total_pages}
          total={query.data.pagination.total}
          onPageChange={setPage}
        />
      ) : null}
    </div>
  );
}

function CardHeaderBlock() {
  return (
    <div className="rounded-3xl border border-border/70 bg-[linear-gradient(135deg,hsl(var(--primary)/0.85),hsl(var(--primary)/0.65))] p-6 text-primary-foreground">
      <p className="text-sm text-primary-foreground/75">共享课程库</p>
      <h1 className="mt-2 font-display text-3xl font-semibold">浏览可复用课程并克隆为新课程</h1>
      <p className="mt-3 max-w-3xl text-sm leading-6 text-primary-foreground/80">
        共享课程库只展示已允许共享的课程。教师可按标题、课程类型、难度和主题筛选，进入详情后再执行克隆。
      </p>
    </div>
  );
}

interface FilterSelectProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
}

function FilterSelect({ label, value, onChange, options }: FilterSelectProps) {
  return (
    <label className="grid gap-2 text-sm text-muted-foreground">
      <span>{label}</span>
      <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm text-foreground" value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((item) => (
          <option key={item.value || "all"} value={item.value}>
            {item.label}
          </option>
        ))}
      </select>
    </label>
  );
}
