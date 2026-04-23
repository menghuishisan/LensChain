// CourseCard.tsx
// 模块03课程卡片组件，用于教师课程列表、学生我的课程和共享课程库摘要展示。

import Link from "next/link";

import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { formatDateTime } from "@/lib/format";
import type { CourseListItem, MyCourseItem, SharedCourseItem } from "@/types/course";

/**
 * CourseCard 组件属性。
 */
export interface CourseCardProps {
  course: CourseListItem | MyCourseItem | SharedCourseItem;
  href: string;
}

/**
 * CourseCard 展示课程封面、标题、类型、状态和进入课程入口。
 */
export function CourseCard({ course, href }: CourseCardProps) {
  const statusText = "status_text" in course ? course.status_text : "共享课程";
  const meta = "teacher_name" in course ? course.teacher_name : "student_count" in course ? `${course.student_count}人` : "";

  return (
    <Link href={href}>
      <Card className="h-full overflow-hidden transition hover:-translate-y-1 hover:border-primary/40 hover:shadow-glow">
        <div className="h-32 bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.22),transparent_18rem),linear-gradient(135deg,hsl(174_60%_23%),hsl(34_72%_42%))]">
          {course.cover_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={course.cover_url} alt={course.title} className="h-full w-full object-cover" />
          ) : null}
        </div>
        <CardHeader>
          <div className="flex items-start justify-between gap-3">
            <CardTitle className="line-clamp-2">{course.title}</CardTitle>
            <Badge variant="secondary">{statusText}</Badge>
          </div>
          <CardDescription>{course.course_type_text} · {meta}</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            {"topic" in course ? course.topic : ""}
            {"created_at" in course ? ` · ${formatDateTime(course.created_at)}` : ""}
            {"joined_at" in course ? ` · 加入于 ${formatDateTime(course.joined_at)}` : ""}
          </p>
          {"progress" in course ? (
            <div className="mt-4 h-2 overflow-hidden rounded-full bg-muted">
              <div className="h-full bg-primary" style={{ width: `${course.progress}%` }} />
            </div>
          ) : null}
        </CardContent>
      </Card>
    </Link>
  );
}
