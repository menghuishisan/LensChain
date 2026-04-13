// content_service.go
// 模块03 — 课程与教学：章节、课时、附件、选课业务逻辑
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
	"github.com/lenschain/backend/internal/pkg/snowflake"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// ContentService 章节课时管理服务接口
type ContentService interface {
	// 章节
	ListChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) ([]*dto.ChapterWithLessonsResp, error)
	CreateChapter(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateChapterReq) (string, error)
	UpdateChapter(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChapterReq) error
	DeleteChapter(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	SortChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.SortReq) error
	// 课时
	CreateLesson(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.CreateLessonReq) (string, error)
	GetLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.LessonDetailResp, error)
	UpdateLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateLessonReq) error
	DeleteLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	SortLessons(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.SortReq) error
	// 附件
	UploadAttachment(ctx context.Context, sc *svcctx.ServiceContext, lessonID int64, req *dto.UploadAttachmentReq) (string, error)
	DeleteAttachment(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	// 选课
	JoinByInviteCode(ctx context.Context, sc *svcctx.ServiceContext, req *dto.JoinCourseReq) (string, error)
	AddStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AddStudentReq) error
	BatchAddStudents(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.BatchAddStudentsReq) error
	RemoveStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64) error
	ListStudents(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.StudentListReq) ([]*dto.EnrolledStudentItem, int64, error)
}

type contentService struct {
	courseRepo         courserepo.CourseRepository
	chapterRepo        courserepo.ChapterRepository
	lessonRepo         courserepo.LessonRepository
	attachmentRepo     courserepo.AttachmentRepository
	enrollmentRepo     courserepo.EnrollmentRepository
	progressRepo       courserepo.ProgressRepository
	userSummaryQuerier UserSummaryQuerier
	userAccessChecker  UserAccessChecker
}

// NewContentService 创建章节课时管理服务实例
func NewContentService(
	courseRepo courserepo.CourseRepository,
	chapterRepo courserepo.ChapterRepository,
	lessonRepo courserepo.LessonRepository,
	attachmentRepo courserepo.AttachmentRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	progressRepo courserepo.ProgressRepository,
	userSummaryQuerier UserSummaryQuerier,
	userAccessChecker UserAccessChecker,
) ContentService {
	return &contentService{
		courseRepo: courseRepo, chapterRepo: chapterRepo,
		lessonRepo: lessonRepo, attachmentRepo: attachmentRepo,
		enrollmentRepo: enrollmentRepo, progressRepo: progressRepo,
		userSummaryQuerier: userSummaryQuerier,
		userAccessChecker:  userAccessChecker,
	}
}

// ========== 章节 ==========

func (s *contentService) ListChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) ([]*dto.ChapterWithLessonsResp, error) {
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, err
	}

	chapters, err := s.chapterRepo.ListByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.ChapterWithLessonsResp, 0, len(chapters))
	for _, ch := range chapters {
		lessons := make([]dto.LessonListItem, 0, len(ch.Lessons))
		for _, l := range ch.Lessons {
			lessons = append(lessons, dto.LessonListItem{
				ID: strconv.FormatInt(l.ID, 10), Title: l.Title,
				ContentType: l.ContentType, ContentTypeText: enum.GetContentTypeText(l.ContentType),
				VideoDuration: l.VideoDuration, EstimatedMinutes: l.EstimatedMinutes,
				SortOrder: l.SortOrder,
			})
			if l.ExperimentID != nil {
				eid := strconv.FormatInt(*l.ExperimentID, 10)
				lessons[len(lessons)-1].ExperimentID = &eid
			}
		}
		result = append(result, &dto.ChapterWithLessonsResp{
			ID: strconv.FormatInt(ch.ID, 10), Title: ch.Title,
			Description: ch.Description, SortOrder: ch.SortOrder,
			Lessons: lessons,
		})
	}
	return result, nil
}

// CreateChapter 创建章节
func (s *contentService) CreateChapter(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateChapterReq) (string, error) {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return "", err
	}
	chapter := &entity.Chapter{
		CourseID: courseID, Title: req.Title, Description: req.Description,
	}
	if err := s.chapterRepo.Create(ctx, chapter); err != nil {
		return "", err
	}
	return strconv.FormatInt(chapter.ID, 10), nil
}

// UpdateChapter 更新章节
func (s *contentService) UpdateChapter(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChapterReq) error {
	chapter, err := s.chapterRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrChapterNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, chapter.CourseID); err != nil {
		return err
	}
	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.chapterRepo.UpdateFields(ctx, id, fields)
}

// DeleteChapter 删除章节
func (s *contentService) DeleteChapter(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	chapter, err := s.chapterRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrChapterNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, chapter.CourseID); err != nil {
		return err
	}
	return s.chapterRepo.SoftDelete(ctx, id)
}

// SortChapters 调整章节排序
func (s *contentService) SortChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.SortReq) error {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return err
	}
	items := make([]courserepo.SortItem, 0, len(req.Items))
	for _, item := range req.Items {
		id, err := snowflake.ParseString(item.ID)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("无效的章节ID")
		}
		items = append(items, courserepo.SortItem{ID: id, SortOrder: item.SortOrder})
	}
	return s.chapterRepo.UpdateSortOrders(ctx, items)
}

// ========== 课时 ==========

func (s *contentService) CreateLesson(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.CreateLessonReq) (string, error) {
	chapter, err := s.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		return "", errcode.ErrChapterNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, chapter.CourseID); err != nil {
		return "", err
	}

	lesson := &entity.Lesson{
		ChapterID: chapterID, CourseID: chapter.CourseID,
		Title: req.Title, ContentType: req.ContentType,
		Content: req.Content, VideoURL: req.VideoURL,
		VideoDuration: req.VideoDuration, EstimatedMinutes: req.EstimatedMinutes,
	}
	if req.ExperimentID != nil {
		eid, err := snowflake.ParseString(*req.ExperimentID)
		if err == nil {
			lesson.ExperimentID = &eid
		}
	}
	if err := s.lessonRepo.Create(ctx, lesson); err != nil {
		return "", err
	}
	return strconv.FormatInt(lesson.ID, 10), nil
}

// GetLesson 获取课时详情
func (s *contentService) GetLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.LessonDetailResp, error) {
	lesson, err := s.lessonRepo.GetByIDWithAttachments(ctx, id)
	if err != nil {
		return nil, errcode.ErrLessonNotFound
	}
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, lesson.CourseID); err != nil {
		return nil, err
	}

	resp := &dto.LessonDetailResp{
		ID:        strconv.FormatInt(lesson.ID, 10),
		ChapterID: strconv.FormatInt(lesson.ChapterID, 10),
		CourseID:  strconv.FormatInt(lesson.CourseID, 10),
		Title:     lesson.Title, ContentType: lesson.ContentType,
		ContentTypeText: enum.GetContentTypeText(lesson.ContentType),
		Content:         lesson.Content, VideoURL: lesson.VideoURL,
		VideoDuration: lesson.VideoDuration, EstimatedMinutes: lesson.EstimatedMinutes,
		SortOrder: lesson.SortOrder,
	}
	if lesson.ExperimentID != nil {
		eid := strconv.FormatInt(*lesson.ExperimentID, 10)
		resp.ExperimentID = &eid
	}

	resp.Attachments = make([]dto.LessonAttachmentItem, 0, len(lesson.Attachments))
	for _, a := range lesson.Attachments {
		resp.Attachments = append(resp.Attachments, dto.LessonAttachmentItem{
			ID: strconv.FormatInt(a.ID, 10), FileName: a.FileName,
			FileURL: a.FileURL, FileSize: a.FileSize, FileType: a.FileType,
		})
	}
	return resp, nil
}

// UpdateLesson 更新课时
func (s *contentService) UpdateLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateLessonReq) error {
	lesson, err := s.lessonRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrLessonNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, lesson.CourseID); err != nil {
		return err
	}
	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.ContentType != nil {
		fields["content_type"] = *req.ContentType
	}
	if req.Content != nil {
		fields["content"] = *req.Content
	}
	if req.VideoURL != nil {
		fields["video_url"] = *req.VideoURL
	}
	if req.VideoDuration != nil {
		fields["video_duration"] = *req.VideoDuration
	}
	if req.ExperimentID != nil {
		eid, err := snowflake.ParseString(*req.ExperimentID)
		if err == nil {
			fields["experiment_id"] = eid
		}
	}
	if req.EstimatedMinutes != nil {
		fields["estimated_minutes"] = *req.EstimatedMinutes
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.lessonRepo.UpdateFields(ctx, id, fields)
}

// DeleteLesson 删除课时
func (s *contentService) DeleteLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	lesson, err := s.lessonRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrLessonNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, lesson.CourseID); err != nil {
		return err
	}
	return s.lessonRepo.SoftDelete(ctx, id)
}

// SortLessons 调整课时排序
func (s *contentService) SortLessons(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.SortReq) error {
	chapter, err := s.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		return errcode.ErrChapterNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, chapter.CourseID); err != nil {
		return err
	}
	items := make([]courserepo.SortItem, 0, len(req.Items))
	for _, item := range req.Items {
		id, err := snowflake.ParseString(item.ID)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("无效的课时ID")
		}
		items = append(items, courserepo.SortItem{ID: id, SortOrder: item.SortOrder})
	}
	return s.lessonRepo.UpdateSortOrders(ctx, items)
}

// ========== 附件 ==========

func (s *contentService) UploadAttachment(ctx context.Context, sc *svcctx.ServiceContext, lessonID int64, req *dto.UploadAttachmentReq) (string, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return "", errcode.ErrLessonNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, lesson.CourseID); err != nil {
		return "", err
	}
	if err := validateLessonAttachment(req.FileName, req.FileType, req.FileSize); err != nil {
		return "", err
	}
	attachment := &entity.LessonAttachment{
		LessonID: lessonID, FileName: req.FileName,
		FileURL: req.FileURL, FileSize: req.FileSize, FileType: req.FileType,
	}
	if err := s.attachmentRepo.Create(ctx, attachment); err != nil {
		return "", err
	}
	return strconv.FormatInt(attachment.ID, 10), nil
}

// DeleteAttachment 删除课时附件
func (s *contentService) DeleteAttachment(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	attachment, err := s.attachmentRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("附件不存在")
	}
	lesson, err := s.lessonRepo.GetByID(ctx, attachment.LessonID)
	if err != nil {
		return errcode.ErrLessonNotFound
	}
	if err := s.verifyCourseTeacher(ctx, sc, lesson.CourseID); err != nil {
		return err
	}
	return s.attachmentRepo.Delete(ctx, id)
}

// ========== 选课 ==========

// JoinByInviteCode 通过邀请码加入课程
func (s *contentService) JoinByInviteCode(ctx context.Context, sc *svcctx.ServiceContext, req *dto.JoinCourseReq) (string, error) {
	course, err := s.courseRepo.GetByInviteCode(ctx, req.InviteCode)
	if err != nil {
		return "", errcode.ErrInviteCodeInvalid
	}
	if !sc.IsSuperAdmin() && sc.SchoolID > 0 && course.SchoolID != sc.SchoolID {
		return "", errcode.ErrForbidden
	}
	if course.Status == enum.CourseStatusEnded || course.Status == enum.CourseStatusArchived {
		return "", errcode.ErrInvalidParams.WithMessage("课程已结束，无法加入")
	}

	enrolled, _ := s.enrollmentRepo.IsEnrolled(ctx, sc.UserID, course.ID)
	if enrolled {
		return "", errcode.ErrAlreadyEnrolled
	}

	if course.MaxStudents != nil {
		count, _ := s.courseRepo.CountStudents(ctx, course.ID)
		if count >= *course.MaxStudents {
			return "", errcode.ErrCourseStudentFull
		}
	}

	enrollment := &entity.CourseEnrollment{
		CourseID: course.ID, StudentID: sc.UserID,
		JoinMethod: enum.JoinMethodInvite, JoinedAt: time.Now(),
	}
	if err := s.enrollmentRepo.Create(ctx, enrollment); err != nil {
		return "", err
	}
	return strconv.FormatInt(course.ID, 10), nil
}

// AddStudent 教师手动添加学生
func (s *contentService) AddStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AddStudentReq) error {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return err
	}
	studentID, err := snowflake.ParseString(req.StudentID)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("无效的学生ID")
	}
	enrolled, _ := s.enrollmentRepo.IsEnrolled(ctx, studentID, courseID)
	if enrolled {
		return errcode.ErrAlreadyEnrolled
	}
	if err := s.ensureCourseStudentCandidate(ctx, sc, studentID); err != nil {
		return err
	}
	enrollment := &entity.CourseEnrollment{
		CourseID: courseID, StudentID: studentID,
		JoinMethod: enum.JoinMethodTeacher, JoinedAt: time.Now(),
	}
	return s.enrollmentRepo.Create(ctx, enrollment)
}

// BatchAddStudents 教师批量添加学生
func (s *contentService) BatchAddStudents(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.BatchAddStudentsReq) error {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return err
	}
	enrollments := make([]*entity.CourseEnrollment, 0)
	for _, sid := range req.StudentIDs {
		studentID, err := snowflake.ParseString(sid)
		if err != nil {
			continue
		}
		enrolled, _ := s.enrollmentRepo.IsEnrolled(ctx, studentID, courseID)
		if enrolled {
			continue
		}
		if err := s.ensureCourseStudentCandidate(ctx, sc, studentID); err != nil {
			return err
		}
		enrollments = append(enrollments, &entity.CourseEnrollment{
			CourseID: courseID, StudentID: studentID,
			JoinMethod: enum.JoinMethodTeacher, JoinedAt: time.Now(),
		})
	}
	if len(enrollments) == 0 {
		return nil
	}
	return s.enrollmentRepo.BatchCreate(ctx, enrollments)
}

// RemoveStudent 移除课程学生
func (s *contentService) RemoveStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64) error {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return err
	}
	return s.enrollmentRepo.Remove(ctx, courseID, studentID)
}

// ListStudents 获取课程学生列表
func (s *contentService) ListStudents(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.StudentListReq) ([]*dto.EnrolledStudentItem, int64, error) {
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID); err != nil {
		return nil, 0, err
	}

	enrollments, total, err := s.enrollmentRepo.List(ctx, &courserepo.EnrollmentListParams{
		CourseID: courseID, Keyword: req.Keyword,
		Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	totalLessons, _ := s.lessonRepo.CountByCourseID(ctx, courseID)
	items := make([]*dto.EnrolledStudentItem, 0, len(enrollments))
	for _, e := range enrollments {
		summary := s.userSummaryQuerier.GetUserSummary(ctx, e.StudentID)
		name := ""
		if summary != nil {
			name = summary.Name
		}
		completed, _ := s.progressRepo.CountCompletedByStudent(ctx, e.StudentID, courseID)
		var progress float64
		if totalLessons > 0 {
			progress = float64(completed) / float64(totalLessons) * 100
		}
		items = append(items, &dto.EnrolledStudentItem{
			ID:         strconv.FormatInt(e.StudentID, 10),
			Name:       name,
			StudentNo:  getSummaryStudentNo(summary),
			College:    getSummaryCollege(summary),
			Major:      getSummaryMajor(summary),
			ClassName:  getSummaryClassName(summary),
			JoinMethod: e.JoinMethod, JoinMethodText: enum.GetJoinMethodText(e.JoinMethod),
			JoinedAt: e.JoinedAt.Format(time.RFC3339), Progress: progress,
		})
	}
	return items, total, nil
}

// verifyCourseTeacher 校验当前用户是否为课程负责教师
func (s *contentService) verifyCourseTeacher(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	_, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	return err
}

// ensureCourseStudentCandidate 校验待添加用户是否为当前学校的学生
// 课程选课属于模块03职责，但用户身份与租户归属来自模块01，因此通过跨模块接口读取。
func (s *contentService) ensureCourseStudentCandidate(ctx context.Context, sc *svcctx.ServiceContext, studentID int64) error {
	if s.userAccessChecker == nil {
		return errcode.ErrInternal.WithMessage("用户访问校验器未初始化")
	}

	schoolID, err := s.userAccessChecker.GetUserSchoolID(ctx, studentID)
	if err != nil {
		return errcode.ErrUserNotFound
	}
	if schoolID != sc.SchoolID {
		return errcode.ErrForbidden.WithMessage("只能添加本校学生")
	}

	hasStudentRole, err := s.userAccessChecker.HasRole(ctx, studentID, enum.RoleStudent)
	if err != nil {
		return errcode.ErrUserNotFound
	}
	if !hasStudentRole {
		return errcode.ErrForbidden.WithMessage("只能添加学生角色用户")
	}

	return nil
}
