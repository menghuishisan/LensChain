// course-navigation.ts
// 模块03课时导航工具，根据章节课时树解析上一课时和下一课时。

import type { ID } from "@/types/api";
import type { ChapterWithLessons } from "@/types/course";

/**
 * 相邻课时摘要。
 */
export interface AdjacentLessonSummary {
  id: ID;
  title: string;
}

/**
 * resolveAdjacentLessons 从完整章节树中计算目标课时的前后导航。
 */
export function resolveAdjacentLessons(chapters: ChapterWithLessons[], lessonID: ID) {
  const flattenedLessons = chapters.flatMap((chapter) =>
    chapter.lessons.map((lesson) => ({
      id: lesson.id,
      title: lesson.title,
    })),
  );
  const lessonIndex = flattenedLessons.findIndex((lesson) => lesson.id === lessonID);

  if (lessonIndex < 0) {
    return {
      previousLesson: null,
      nextLesson: null,
    };
  }

  return {
    previousLesson: lessonIndex > 0 ? flattenedLessons[lessonIndex - 1] : null,
    nextLesson: lessonIndex < flattenedLessons.length - 1 ? flattenedLessons[lessonIndex + 1] : null,
  };
}
