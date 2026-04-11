// course_query.go
// 模块03 — 课程与教学：课程查询、克隆、共享课程库、学生课程列表
// 从 course_service.go 拆分而来，保持单文件 ≤ 500 行
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// Clone 克隆课程（复制课程基本信息 + 章节 + 课时结构）
// Clone 克隆课程
// 复制课程基础信息、章节和课时结构，新课程以草稿状态创建
func (s *courseService) Clone(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error) {
	original, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return "", errcode.ErrCourseNotFound
	}

	inviteCode := generateInviteCode()
	cloned := &entity.Course{
		SchoolID:     sc.SchoolID,
		TeacherID:    sc.UserID,
		Title:        original.Title + "（副本）",
		Description:  original.Description,
		CoverURL:     original.CoverURL,
		CourseType:   original.CourseType,
		Difficulty:   original.Difficulty,
		Topic:        original.Topic,
		Status:       enum.CourseStatusDraft,
		InviteCode:   &inviteCode,
		MaxStudents:  original.MaxStudents,
		ClonedFromID: &original.ID,
	}

	if err := s.courseRepo.Create(ctx, cloned); err != nil {
		return "", err
	}

	// 复制章节和课时
	s.cloneContent(ctx, id, cloned.ID)

	return strconv.FormatInt(cloned.ID, 10), nil
}

// cloneContent 复制课程的章节和课时结构
func (s *courseService) cloneContent(ctx context.Context, srcCourseID, dstCourseID int64) {
	chapters, err := s.chapterRepo.ListByCourseID(ctx, srcCourseID)
	if err != nil {
		return
	}

	for _, ch := range chapters {
		newChapter := &entity.Chapter{
			CourseID:    dstCourseID,
			Title:       ch.Title,
			Description: ch.Description,
			SortOrder:   ch.SortOrder,
		}
		if err := s.chapterRepo.Create(ctx, newChapter); err != nil {
			continue
		}

		// 复制课时
		for _, l := range ch.Lessons {
			newLesson := &entity.Lesson{
				ChapterID:        newChapter.ID,
				CourseID:         dstCourseID,
				Title:            l.Title,
				ContentType:      l.ContentType,
				Content:          l.Content,
				VideoURL:         l.VideoURL,
				VideoDuration:    l.VideoDuration,
				ExperimentID:     l.ExperimentID,
				SortOrder:        l.SortOrder,
				EstimatedMinutes: l.EstimatedMinutes,
			}
			_ = s.lessonRepo.Create(ctx, newLesson)
		}
	}
}

// ListShared 获取共享课程库列表
func (s *courseService) ListShared(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SharedCourseListReq) ([]*dto.SharedCourseItem, int64, error) {
	courses, total, err := s.courseRepo.ListShared(ctx, &courserepo.SharedCourseListParams{
		Keyword: req.Keyword, CourseType: req.CourseType,
		Difficulty: req.Difficulty, Topic: req.Topic,
		Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.SharedCourseItem, 0, len(courses))
	for _, c := range courses {
		teacherName := s.userNameQuerier.GetUserName(ctx, c.TeacherID)
		schoolName := s.schoolNameQuerier.GetSchoolName(ctx, c.SchoolID)
		studentCount, _ := s.courseRepo.CountStudents(ctx, c.ID)
		avgRating, _ := s.evaluationRepo.GetAvgRating(ctx, c.ID)

		items = append(items, &dto.SharedCourseItem{
			ID:             strconv.FormatInt(c.ID, 10),
			Title:          c.Title,
			Description:    c.Description,
			CoverURL:       c.CoverURL,
			CourseType:     c.CourseType,
			CourseTypeText: enum.GetCourseTypeText(c.CourseType),
			Difficulty:     c.Difficulty,
			DifficultyText: enum.GetDifficultyText(c.Difficulty),
			Topic:          c.Topic,
			TeacherName:    teacherName,
			SchoolName:     schoolName,
			StudentCount:   studentCount,
			Rating:         avgRating,
		})
	}
	return items, total, nil
}

// GetSharedDetail 获取共享课程详情
// 仅返回已开启共享的课程，供共享课程库详情页使用
func (s *courseService) GetSharedDetail(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CourseDetailResp, error) {
	_ = sc
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrCourseNotFound
	}
	if !course.IsShared {
		return nil, errcode.ErrCourseNotFound
	}

	studentCount, err := s.courseRepo.CountStudents(ctx, id)
	if err != nil {
		return nil, err
	}
	teacherName := s.userNameQuerier.GetUserName(ctx, course.TeacherID)
	return buildCourseDetail(course, studentCount, teacherName), nil
}

// ListMyCourses 获取学生已选课程列表
func (s *courseService) ListMyCourses(ctx context.Context, sc *svcctx.ServiceContext, req *dto.MyCourseListReq) ([]*dto.MyCourseItem, int64, error) {
	courses, total, err := s.courseRepo.ListByStudentID(ctx, sc.UserID, &courserepo.StudentCourseListParams{
		Status: req.Status, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	totalLessonsMap := make(map[int64]int)
	items := make([]*dto.MyCourseItem, 0, len(courses))
	for _, c := range courses {
		teacherName := s.userNameQuerier.GetUserName(ctx, c.TeacherID)

		// 计算学习进度
		totalLessons, ok := totalLessonsMap[c.ID]
		if !ok {
			totalLessons, _ = s.lessonRepo.CountByCourseID(ctx, c.ID)
			totalLessonsMap[c.ID] = totalLessons
		}
		completed, _ := s.progressRepo.CountCompletedByStudent(ctx, sc.UserID, c.ID)
		var progress float64
		if totalLessons > 0 {
			progress = float64(completed) / float64(totalLessons) * 100
		}

		// 获取加入时间
		enrollment, _ := s.enrollmentRepo.GetByStudentAndCourse(ctx, sc.UserID, c.ID)
		joinedAt := ""
		if enrollment != nil {
			joinedAt = enrollment.JoinedAt.Format(time.RFC3339)
		}

		items = append(items, &dto.MyCourseItem{
			ID:             strconv.FormatInt(c.ID, 10),
			Title:          c.Title,
			CoverURL:       c.CoverURL,
			CourseType:     c.CourseType,
			CourseTypeText: enum.GetCourseTypeText(c.CourseType),
			TeacherName:    teacherName,
			Status:         c.Status,
			StatusText:     enum.GetCourseStatusText(c.Status),
			Progress:       progress,
			JoinedAt:       joinedAt,
		})
	}
	return items, total, nil
}

// ========== 构建辅助函数 ==========

// buildCourseDetail 构建课程详情响应
func buildCourseDetail(c *entity.Course, studentCount int, teacherName string) *dto.CourseDetailResp {
	resp := &dto.CourseDetailResp{
		ID:             strconv.FormatInt(c.ID, 10),
		Title:          c.Title,
		Description:    c.Description,
		CoverURL:       c.CoverURL,
		CourseType:     c.CourseType,
		CourseTypeText: enum.GetCourseTypeText(c.CourseType),
		Difficulty:     c.Difficulty,
		DifficultyText: enum.GetDifficultyText(c.Difficulty),
		Topic:          c.Topic,
		Status:         c.Status,
		StatusText:     enum.GetCourseStatusText(c.Status),
		IsShared:       c.IsShared,
		InviteCode:     c.InviteCode,
		MaxStudents:    c.MaxStudents,
		StudentCount:   studentCount,
		TeacherID:      strconv.FormatInt(c.TeacherID, 10),
		TeacherName:    teacherName,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      c.UpdatedAt.Format(time.RFC3339),
	}
	if c.StartAt != nil {
		s := c.StartAt.Format(time.RFC3339)
		resp.StartAt = &s
	}
	if c.EndAt != nil {
		e := c.EndAt.Format(time.RFC3339)
		resp.EndAt = &e
	}
	if c.ClonedFromID != nil {
		cid := strconv.FormatInt(*c.ClonedFromID, 10)
		resp.ClonedFromID = &cid
	}
	return resp
}

// buildCourseListItem 构建课程列表项
func buildCourseListItem(c *entity.Course, studentCount int) *dto.CourseListItem {
	item := &dto.CourseListItem{
		ID:             strconv.FormatInt(c.ID, 10),
		Title:          c.Title,
		CoverURL:       c.CoverURL,
		CourseType:     c.CourseType,
		CourseTypeText: enum.GetCourseTypeText(c.CourseType),
		Difficulty:     c.Difficulty,
		DifficultyText: enum.GetDifficultyText(c.Difficulty),
		Topic:          c.Topic,
		Status:         c.Status,
		StatusText:     enum.GetCourseStatusText(c.Status),
		IsShared:       c.IsShared,
		StudentCount:   studentCount,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
	}
	if c.StartAt != nil {
		s := c.StartAt.Format(time.RFC3339)
		item.StartAt = &s
	}
	if c.EndAt != nil {
		e := c.EndAt.Format(time.RFC3339)
		item.EndAt = &e
	}
	return item
}
