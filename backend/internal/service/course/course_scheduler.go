// course_scheduler.go
// 模块03 — 课程与教学：课程状态定时任务
// 负责按文档要求自动推进课程状态：已发布→进行中、进行中→已结束。

package course

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/logger"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// CourseScheduler 课程状态定时任务执行器
type CourseScheduler struct {
	courseRepo courserepo.CourseRepository
}

// NewCourseScheduler 创建课程状态定时任务执行器
func NewCourseScheduler(courseRepo courserepo.CourseRepository) *CourseScheduler {
	return &CourseScheduler{courseRepo: courseRepo}
}

// RunPublishedToActive 将到达开始时间的课程从“已发布”推进为“进行中”
func (s *CourseScheduler) RunPublishedToActive() {
	ctx := context.Background()
	now := time.Now().UTC()

	courses, err := s.courseRepo.ListPublishedToActivate(ctx, now)
	if err != nil {
		logger.L.Error("查询待自动开始课程失败", zap.Error(err))
		return
	}

	for _, course := range courses {
		if err := s.courseRepo.UpdateStatus(ctx, course.ID, enum.CourseStatusActive); err != nil {
			logger.L.Error("自动开始课程失败",
				zap.Int64("course_id", course.ID),
				zap.Error(err),
			)
			continue
		}
		logger.L.Info("课程状态已自动更新为进行中",
			zap.Int64("course_id", course.ID),
		)
	}
}

// RunActiveToEnded 将到达结束时间的课程从“进行中”推进为“已结束”
func (s *CourseScheduler) RunActiveToEnded() {
	ctx := context.Background()
	now := time.Now().UTC()

	courses, err := s.courseRepo.ListActiveToEnd(ctx, now)
	if err != nil {
		logger.L.Error("查询待自动结束课程失败", zap.Error(err))
		return
	}

	for _, course := range courses {
		if err := s.courseRepo.UpdateStatus(ctx, course.ID, enum.CourseStatusEnded); err != nil {
			logger.L.Error("自动结束课程失败",
				zap.Int64("course_id", course.ID),
				zap.Error(err),
			)
			continue
		}
		logger.L.Info("课程状态已自动更新为已结束",
			zap.Int64("course_id", course.ID),
		)
	}
}
