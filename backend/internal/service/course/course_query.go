// course_query.go
// 模块03 — 课程与教学：课程查询、克隆、共享课程库、学生课程列表
// 从 course_service.go 拆分而来，保持单文件 ≤ 500 行
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// Clone 克隆课程。
// 当前先保证同步克隆的事务原子性，避免中途失败时留下半成品课程。
// 大型课程的异步克隆与完成通知，后续应接入统一异步任务体系和模块07内部通知事件，不在此处直接起 goroutine。
func (s *courseService) Clone(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error) {
	original, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return "", errcode.ErrCourseNotFound
	}
	if err := ensureCourseCloneAllowed(original, sc.UserID); err != nil {
		return "", err
	}

	return s.cloneCourseAtomically(ctx, sc, original)
}

// cloneCourseAtomically 在单事务内完成课程克隆，确保课程主表、内容、作业题目要么全部成功，要么全部回滚。
func (s *courseService) cloneCourseAtomically(ctx context.Context, sc *svcctx.ServiceContext, original *entity.Course) (string, error) {
	var clonedID int64
	err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txCourseRepo := courserepo.NewCourseRepository(tx)
		txChapterRepo := courserepo.NewChapterRepository(tx)
		txLessonRepo := courserepo.NewLessonRepository(tx)
		txAttachmentRepo := courserepo.NewAttachmentRepository(tx)
		txAssignmentRepo := courserepo.NewAssignmentRepository(tx)
		txQuestionRepo := courserepo.NewQuestionRepository(tx)

		inviteCode, err := generateInviteCode()
		if err != nil {
			return err
		}
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
			StartAt:      original.StartAt,
			EndAt:        original.EndAt,
			Credits:      original.Credits,
			SemesterID:   original.SemesterID,
			MaxStudents:  original.MaxStudents,
			ClonedFromID: &original.ID,
		}

		if err := txCourseRepo.Create(ctx, cloned); err != nil {
			return err
		}
		chapterIDMap, err := s.cloneContentWithRepos(ctx, txChapterRepo, txLessonRepo, txAttachmentRepo, original.ID, cloned.ID)
		if err != nil {
			return err
		}
		if err := s.cloneAssignmentsWithRepos(ctx, txAssignmentRepo, txQuestionRepo, original.ID, cloned.ID, chapterIDMap); err != nil {
			return err
		}
		clonedID = cloned.ID
		return nil
	})
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(clonedID, 10), nil
}

// cloneContent 复制课程的章节和课时结构，并返回章节 ID 映射。
func (s *courseService) cloneContentWithRepos(
	ctx context.Context,
	chapterRepo courserepo.ChapterRepository,
	lessonRepo courserepo.LessonRepository,
	attachmentRepo courserepo.AttachmentRepository,
	srcCourseID, dstCourseID int64,
) (map[int64]int64, error) {
	chapters, err := chapterRepo.ListByCourseID(ctx, srcCourseID)
	if err != nil {
		return nil, err
	}
	chapterIDs := make([]int64, 0, len(chapters))
	for _, chapter := range chapters {
		chapterIDs = append(chapterIDs, chapter.ID)
	}
	lessons, err := lessonRepo.ListByChapterIDs(ctx, chapterIDs)
	if err != nil {
		return nil, err
	}
	lessonMap := buildLessonsByChapterID(lessons)
	lessonIDs := make([]int64, 0, len(lessons))
	for _, lesson := range lessons {
		lessonIDs = append(lessonIDs, lesson.ID)
	}
	attachments, err := attachmentRepo.ListByLessonIDs(ctx, lessonIDs)
	if err != nil {
		return nil, err
	}
	attachmentMap := buildAttachmentsByLessonID(attachments)
	chapterIDMap := make(map[int64]int64, len(chapters))

	for _, ch := range chapters {
		newChapter := &entity.Chapter{
			CourseID:    dstCourseID,
			Title:       ch.Title,
			Description: ch.Description,
			SortOrder:   ch.SortOrder,
		}
		if err := chapterRepo.Create(ctx, newChapter); err != nil {
			return nil, err
		}
		chapterIDMap[ch.ID] = newChapter.ID

		// 复制课时
		for _, l := range lessonMap[ch.ID] {
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
			if err := lessonRepo.Create(ctx, newLesson); err != nil {
				return nil, err
			}
			for _, attachment := range attachmentMap[l.ID] {
				newAttachment := &entity.LessonAttachment{
					LessonID:  newLesson.ID,
					FileName:  attachment.FileName,
					FileURL:   attachment.FileURL,
					FileSize:  attachment.FileSize,
					FileType:  attachment.FileType,
					SortOrder: attachment.SortOrder,
				}
				if err := attachmentRepo.Create(ctx, newAttachment); err != nil {
					return nil, err
				}
			}
		}
	}
	return chapterIDMap, nil
}

// cloneAssignments 复制课程下的作业和题目，新课程默认保持未发布。
func (s *courseService) cloneAssignmentsWithRepos(
	ctx context.Context,
	assignmentRepo courserepo.AssignmentRepository,
	questionRepo courserepo.QuestionRepository,
	srcCourseID, dstCourseID int64,
	chapterIDMap map[int64]int64,
) error {
	assignments, err := listAllAssignmentsForClone(ctx, assignmentRepo, srcCourseID)
	if err != nil {
		return err
	}
	assignmentIDs := make([]int64, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentIDs = append(assignmentIDs, assignment.ID)
	}
	questionMap := make(map[int64][]*entity.AssignmentQuestion, len(assignments))
	if len(assignmentIDs) > 0 {
		questions, err := questionRepo.ListByAssignmentIDs(ctx, assignmentIDs)
		if err != nil {
			return err
		}
		for _, question := range questions {
			questionMap[question.AssignmentID] = append(questionMap[question.AssignmentID], question)
		}
	}

	for _, assignment := range assignments {
		newAssignment := buildClonedAssignment(assignment, dstCourseID, chapterIDMap, questionMap[assignment.ID])
		if err := assignmentRepo.Create(ctx, newAssignment); err != nil {
			return err
		}
		for _, question := range questionMap[assignment.ID] {
			newQuestion := &entity.AssignmentQuestion{
				AssignmentID:    newAssignment.ID,
				QuestionType:    question.QuestionType,
				Title:           question.Title,
				Options:         question.Options,
				CorrectAnswer:   question.CorrectAnswer,
				ReferenceAnswer: question.ReferenceAnswer,
				Score:           question.Score,
				JudgeConfig:     question.JudgeConfig,
				SortOrder:       question.SortOrder,
			}
			if err := questionRepo.Create(ctx, newQuestion); err != nil {
				return err
			}
		}
	}
	return nil
}

// listAllAssignmentsForClone 分页拉取课程下全部作业，避免大型课程克隆时丢失后续作业。
func listAllAssignmentsForClone(
	ctx context.Context,
	assignmentRepo courserepo.AssignmentRepository,
	courseID int64,
) ([]*entity.Assignment, error) {
	const pageSize = 1000

	page := 1
	assignments := make([]*entity.Assignment, 0)
	for {
		pageItems, total, err := assignmentRepo.ListByCourseID(ctx, &courserepo.AssignmentListParams{
			CourseID: courseID,
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}

		assignments = append(assignments, pageItems...)
		if len(assignments) >= int(total) || len(pageItems) < pageSize {
			break
		}
		page++
	}

	return assignments, nil
}

// buildClonedAssignment 构建克隆后的作业实体，并将章节关联映射到新课程的章节。
func buildClonedAssignment(
	src *entity.Assignment,
	dstCourseID int64,
	chapterIDMap map[int64]int64,
	questions []*entity.AssignmentQuestion,
) *entity.Assignment {
	totalScore := 0.0
	for _, question := range questions {
		totalScore += question.Score
	}

	cloned := &entity.Assignment{
		CourseID:            dstCourseID,
		Title:               src.Title,
		Description:         src.Description,
		AssignmentType:      src.AssignmentType,
		TotalScore:          totalScore,
		DeadlineAt:          src.DeadlineAt,
		MaxSubmissions:      src.MaxSubmissions,
		LatePolicy:          src.LatePolicy,
		LateDeductionPerDay: src.LateDeductionPerDay,
		IsPublished:         false,
		SortOrder:           src.SortOrder,
	}
	if src.ChapterID != nil {
		if mappedChapterID, ok := chapterIDMap[*src.ChapterID]; ok {
			cloned.ChapterID = &mappedChapterID
		}
	}
	return cloned
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
		studentCount, err := s.courseRepo.CountStudents(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		avgRating, err := s.evaluationRepo.GetAvgRating(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}

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

// GetSharedDetail 获取共享课程详情。
// 仅返回共享课程库中的课程基础信息和目录结构，不暴露邀请码等教师私有数据。
func (s *courseService) GetSharedDetail(ctx context.Context, _ *svcctx.ServiceContext, id int64) (*dto.SharedCourseDetailResp, error) {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrCourseNotFound
	}
	if !course.IsShared {
		return nil, errcode.ErrCourseNotFound
	}
	switch course.Status {
	case enum.CourseStatusPublished, enum.CourseStatusActive, enum.CourseStatusEnded:
	default:
		return nil, errcode.ErrCourseNotFound
	}

	studentCount, err := s.courseRepo.CountStudents(ctx, id)
	if err != nil {
		return nil, err
	}
	teacherName := s.userNameQuerier.GetUserName(ctx, course.TeacherID)
	schoolName := s.schoolNameQuerier.GetSchoolName(ctx, course.SchoolID)
	rating, err := s.evaluationRepo.GetAvgRating(ctx, course.ID)
	if err != nil {
		return nil, err
	}

	chapters, err := s.chapterRepo.ListByCourseID(ctx, course.ID)
	if err != nil {
		return nil, err
	}
	chapterIDs := make([]int64, 0, len(chapters))
	for _, chapter := range chapters {
		chapterIDs = append(chapterIDs, chapter.ID)
	}
	lessons, err := s.lessonRepo.ListByChapterIDs(ctx, chapterIDs)
	if err != nil {
		return nil, err
	}
	return buildSharedCourseDetail(
		course,
		studentCount,
		teacherName,
		schoolName,
		rating,
		buildSharedCourseChapters(chapters, lessons),
	), nil
}

// ListMyCourses 获取学生已选课程列表
func (s *courseService) ListMyCourses(ctx context.Context, sc *svcctx.ServiceContext, req *dto.MyCourseListReq) ([]*dto.MyCourseItem, int64, error) {
	params := &courserepo.StudentCourseListParams{
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	switch req.Status {
	case 0:
		params.Statuses = []int16{
			enum.CourseStatusPublished,
			enum.CourseStatusActive,
			enum.CourseStatusEnded,
		}
	case enum.CourseStatusPublished, enum.CourseStatusActive, enum.CourseStatusEnded:
		params.Status = req.Status
	default:
		return []*dto.MyCourseItem{}, 0, nil
	}
	courses, total, err := s.courseRepo.ListByStudentID(ctx, sc.UserID, params)
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
			totalLessons, err = s.lessonRepo.CountByCourseID(ctx, c.ID)
			if err != nil {
				return nil, 0, err
			}
			totalLessonsMap[c.ID] = totalLessons
		}
		completed, err := s.progressRepo.CountCompletedByStudent(ctx, sc.UserID, c.ID)
		if err != nil {
			return nil, 0, err
		}
		var progress float64
		if totalLessons > 0 {
			progress = float64(completed) / float64(totalLessons) * 100
		}

		// 获取加入时间
		enrollment, err := s.enrollmentRepo.GetByStudentAndCourse(ctx, sc.UserID, c.ID)
		if err != nil {
			return nil, 0, err
		}
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

// buildCourseChapters 构建课程详情中的章节与课时目录。
func buildCourseChapters(chapters []*entity.Chapter, lessons []*entity.Lesson) []dto.ChapterWithLessonsResp {
	return buildSharedCourseChapters(chapters, lessons)
}

// buildCourseDetail 构建课程详情响应
func buildCourseDetail(c *entity.Course, studentCount int, teacherName string, chapters []dto.ChapterWithLessonsResp) *dto.CourseDetailResp {
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
		Credits:        c.Credits,
		MaxStudents:    c.MaxStudents,
		StudentCount:   studentCount,
		TeacherID:      strconv.FormatInt(c.TeacherID, 10),
		TeacherName:    teacherName,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      c.UpdatedAt.Format(time.RFC3339),
		Chapters:       chapters,
	}
	if c.StartAt != nil {
		s := c.StartAt.Format(time.RFC3339)
		resp.StartAt = &s
	}
	if c.EndAt != nil {
		e := c.EndAt.Format(time.RFC3339)
		resp.EndAt = &e
	}
	if c.SemesterID != nil {
		semesterID := strconv.FormatInt(*c.SemesterID, 10)
		resp.SemesterID = &semesterID
	}
	return resp
}

// buildSharedCourseDetail 构建共享课程详情响应。
// 共享课程详情面向其他教师浏览与克隆，不返回邀请码等仅课程教师可见字段。
func buildSharedCourseDetail(
	c *entity.Course,
	studentCount int,
	teacherName, schoolName string,
	rating float64,
	chapters []dto.ChapterWithLessonsResp,
) *dto.SharedCourseDetailResp {
	resp := &dto.SharedCourseDetailResp{
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
		Credits:        c.Credits,
		MaxStudents:    c.MaxStudents,
		StudentCount:   studentCount,
		TeacherName:    teacherName,
		SchoolName:     schoolName,
		Rating:         rating,
		Chapters:       chapters,
	}
	if c.StartAt != nil {
		startAt := c.StartAt.Format(time.RFC3339)
		resp.StartAt = &startAt
	}
	if c.EndAt != nil {
		endAt := c.EndAt.Format(time.RFC3339)
		resp.EndAt = &endAt
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

// buildSharedCourseChapters 构建共享课程详情中的目录结构。
// 共享详情只需要目录树，因此直接在课程查询服务内组合章节与课时 DTO。
func buildSharedCourseChapters(chapters []*entity.Chapter, lessons []*entity.Lesson) []dto.ChapterWithLessonsResp {
	lessonMap := buildLessonsByChapterID(lessons)
	items := make([]dto.ChapterWithLessonsResp, 0, len(chapters))
	for _, chapter := range chapters {
		chapterLessons := lessonMap[chapter.ID]
		lessonItems := make([]dto.LessonListItem, 0, len(chapterLessons))
		for _, lesson := range chapterLessons {
			item := dto.LessonListItem{
				ID:               strconv.FormatInt(lesson.ID, 10),
				Title:            lesson.Title,
				ContentType:      lesson.ContentType,
				ContentTypeText:  enum.GetContentTypeText(lesson.ContentType),
				VideoDuration:    lesson.VideoDuration,
				EstimatedMinutes: lesson.EstimatedMinutes,
				SortOrder:        lesson.SortOrder,
			}
			if lesson.ExperimentID != nil {
				experimentID := strconv.FormatInt(*lesson.ExperimentID, 10)
				item.ExperimentID = &experimentID
			}
			lessonItems = append(lessonItems, item)
		}
		items = append(items, dto.ChapterWithLessonsResp{
			ID:          strconv.FormatInt(chapter.ID, 10),
			Title:       chapter.Title,
			Description: chapter.Description,
			SortOrder:   chapter.SortOrder,
			Lessons:     lessonItems,
		})
	}
	return items
}

// buildAttachmentsByLessonID 按课时归并附件列表，供克隆和详情等上层场景复用。
func buildAttachmentsByLessonID(attachments []*entity.LessonAttachment) map[int64][]*entity.LessonAttachment {
	attachmentMap := make(map[int64][]*entity.LessonAttachment, len(attachments))
	for _, attachment := range attachments {
		attachmentMap[attachment.LessonID] = append(attachmentMap[attachment.LessonID], attachment)
	}
	return attachmentMap
}
