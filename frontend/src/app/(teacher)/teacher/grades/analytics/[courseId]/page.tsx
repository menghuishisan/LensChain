// page.tsx
// 教师端课程成绩分析页。

import { GradePanelsTeacherWrapper } from "@/components/business/GradePanelsWrapper";

/** TeacherGradesAnalyticsPage 教师课程成绩分析页面。 */
export default function TeacherGradesAnalyticsPage({ params }: { params: { courseId: string } }) {
  return <GradePanelsTeacherWrapper page="analytics" courseID={params.courseId} />;
}
