// course_service.go
// 模块03 — 课程与教学：课程管理业务逻辑
// 负责课程CRUD、发布、结束、归档、共享、邀请码等
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"crypto/rand"
	"math/big"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/contentsafety"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/timeutil"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// UserNameQuerier 跨模块接口：查询用户名称
type UserNameQuerier interface {
	GetUserName(ctx context.Context, userID int64) string
}

// CourseUserSummary 课程模块使用的用户摘要信息
// 仅暴露模块03需要的只读字段，避免直接依赖模块01 DTO。
type CourseUserSummary struct {
	Name      string
	StudentNo *string
	College   *string
	Major     *string
	ClassName *string
}

// UserSummaryQuerier 跨模块接口：查询用户摘要信息
type UserSummaryQuerier interface {
	GetUserSummary(ctx context.Context, userID int64) *CourseUserSummary
}

// UserAccessChecker 跨模块接口：查询用户租户与角色信息
// 由模块01注入实现，避免课程模块直接依赖用户模块 service。
type UserAccessChecker interface {
	GetUserSchoolID(ctx context.Context, userID int64) (int64, error)
	HasRole(ctx context.Context, userID int64, role string) (bool, error)
}

// SchoolNameQuerier 跨模块接口：查询学校名称
type SchoolNameQuerier interface {
	GetSchoolName(ctx context.Context, schoolID int64) string
}

// CourseService 课程管理服务接口
type CourseService interface {
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateCourseReq) (*dto.CreateCourseResp, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CourseDetailResp, error)
	GetSharedDetail(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SharedCourseDetailResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateCourseReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CourseListReq) ([]*dto.CourseListItem, int64, error)
	Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	End(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	Archive(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	Clone(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error)
	ToggleShare(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ToggleShareReq) error
	RefreshInviteCode(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error)
	ListShared(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SharedCourseListReq) ([]*dto.SharedCourseItem, int64, error)
	ListMyCourses(ctx context.Context, sc *svcctx.ServiceContext, req *dto.MyCourseListReq) ([]*dto.MyCourseItem, int64, error)
}

// courseService 课程管理服务实现
type courseService struct {
	db                *gorm.DB
	courseRepo        courserepo.CourseRepository
	chapterRepo       courserepo.ChapterRepository
	lessonRepo        courserepo.LessonRepository
	attachmentRepo    courserepo.AttachmentRepository
	enrollmentRepo    courserepo.EnrollmentRepository
	assignmentRepo    courserepo.AssignmentRepository
	questionRepo      courserepo.QuestionRepository
	progressRepo      courserepo.ProgressRepository
	evaluationRepo    courserepo.EvaluationRepository
	userNameQuerier   UserNameQuerier
	schoolNameQuerier SchoolNameQuerier
}

// NewCourseService 创建课程管理服务实例
func NewCourseService(
	db *gorm.DB,
	courseRepo courserepo.CourseRepository,
	chapterRepo courserepo.ChapterRepository,
	lessonRepo courserepo.LessonRepository,
	attachmentRepo courserepo.AttachmentRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	assignmentRepo courserepo.AssignmentRepository,
	questionRepo courserepo.QuestionRepository,
	progressRepo courserepo.ProgressRepository,
	evaluationRepo courserepo.EvaluationRepository,
	userNameQuerier UserNameQuerier,
	schoolNameQuerier SchoolNameQuerier,
) CourseService {
	return &courseService{
		db: db, courseRepo: courseRepo, chapterRepo: chapterRepo,
		lessonRepo: lessonRepo, attachmentRepo: attachmentRepo, enrollmentRepo: enrollmentRepo,
		assignmentRepo: assignmentRepo, questionRepo: questionRepo,
		progressRepo: progressRepo, evaluationRepo: evaluationRepo,
		userNameQuerier: userNameQuerier, schoolNameQuerier: schoolNameQuerier,
	}
}

// Create 创建课程
func (s *courseService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateCourseReq) (*dto.CreateCourseResp, error) {
	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, err
	}
	coverURL := req.CoverURL
	if coverURL == nil {
		coverURL = getDefaultCourseCoverURL()
	}

	course := &entity.Course{
		SchoolID:    sc.SchoolID,
		TeacherID:   sc.UserID,
		Title:       req.Title,
		Description: contentsafety.SanitizeOptionalMarkdown(req.Description),
		CoverURL:    coverURL,
		CourseType:  req.CourseType,
		Difficulty:  req.Difficulty,
		Topic:       req.Topic,
		Status:      enum.CourseStatusDraft,
		InviteCode:  &inviteCode,
		Credits:     req.Credits,
		MaxStudents: req.MaxStudents,
	}

	if req.SemesterID != nil && *req.SemesterID != "" {
		semesterID, err := snowflake.ParseString(*req.SemesterID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("学期ID格式错误")
		}
		course.SemesterID = &semesterID
	}

	if req.StartAt != nil {
		t, err := timeutil.ParseRFC3339(*req.StartAt)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("开始时间格式错误")
		}
		course.StartAt = t
	}
	if req.EndAt != nil {
		t, err := timeutil.ParseRFC3339(*req.EndAt)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("结束时间格式错误")
		}
		course.EndAt = t
	}
	if err := validateCourseTimeRange(course.StartAt, course.EndAt); err != nil {
		return nil, err
	}

	if err := s.courseRepo.Create(ctx, course); err != nil {
		return nil, err
	}
	return &dto.CreateCourseResp{
		ID:         strconv.FormatInt(course.ID, 10),
		Title:      course.Title,
		Status:     course.Status,
		StatusText: enum.GetCourseStatusText(course.Status),
		InviteCode: inviteCode,
		CoverURL:   course.CoverURL,
	}, nil
}

// GetByID 获取课程详情
func (s *courseService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CourseDetailResp, error) {
	course, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, id)
	if err != nil {
		return nil, err
	}
	studentCount, err := s.courseRepo.CountStudents(ctx, id)
	if err != nil {
		return nil, err
	}
	chapters, err := s.chapterRepo.ListByCourseID(ctx, id)
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
	teacherName := s.userNameQuerier.GetUserName(ctx, course.TeacherID)
	detail := buildCourseDetail(course, studentCount, teacherName, buildCourseChapters(chapters, lessons))
	if course.TeacherID != sc.UserID {
		detail.InviteCode = nil
	}
	return detail, nil
}

// Update 编辑课程信息
func (s *courseService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateCourseReq) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrCourseNotFound
	}
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, course.ID); err != nil {
		return err
	}
	if err := ensureCourseContentEditable(course); err != nil {
		return err
	}

	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = contentsafety.SanitizeMarkdown(*req.Description)
	}
	if req.CoverURL != nil {
		fields["cover_url"] = *req.CoverURL
	}
	if req.CourseType != nil {
		fields["course_type"] = *req.CourseType
	}
	if req.Difficulty != nil {
		fields["difficulty"] = *req.Difficulty
	}
	if req.Topic != nil {
		fields["topic"] = *req.Topic
	}
	if req.Credits != nil {
		fields["credits"] = *req.Credits
	}
	if req.SemesterID != nil {
		if *req.SemesterID == "" {
			fields["semester_id"] = nil
		} else {
			semesterID, err := snowflake.ParseString(*req.SemesterID)
			if err != nil {
				return errcode.ErrInvalidParams.WithMessage("学期ID格式错误")
			}
			fields["semester_id"] = semesterID
		}
	}
	if req.StartAt != nil {
		t, err := timeutil.ParseRFC3339(*req.StartAt)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("开始时间格式错误")
		}
		fields["start_at"] = t
	}
	if req.EndAt != nil {
		t, err := timeutil.ParseRFC3339(*req.EndAt)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("结束时间格式错误")
		}
		fields["end_at"] = t
	}
	nextStartAt := course.StartAt
	if parsedStartAt, ok := fields["start_at"]; ok {
		value := parsedStartAt.(time.Time)
		nextStartAt = &value
	}
	nextEndAt := course.EndAt
	if parsedEndAt, ok := fields["end_at"]; ok {
		value := parsedEndAt.(time.Time)
		nextEndAt = &value
	}
	if err := validateCourseTimeRange(nextStartAt, nextEndAt); err != nil {
		return err
	}
	if req.MaxStudents != nil {
		fields["max_students"] = *req.MaxStudents
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.courseRepo.UpdateFields(ctx, id, fields)
}

// Delete 删除课程（仅草稿可删除）
func (s *courseService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrCourseNotFound
	}
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, course.ID); err != nil {
		return err
	}
	if course.Status != enum.CourseStatusDraft {
		return errcode.ErrCourseNotDraft
	}
	return s.courseRepo.SoftDelete(ctx, id)
}

// List 教师课程列表
func (s *courseService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CourseListReq) ([]*dto.CourseListItem, int64, error) {
	courses, total, err := s.courseRepo.List(ctx, &courserepo.CourseListParams{
		SchoolID: sc.SchoolID, TeacherID: sc.UserID,
		Keyword: req.Keyword, Status: req.Status, CourseType: req.CourseType,
		Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.CourseListItem, 0, len(courses))
	for _, c := range courses {
		count, err := s.courseRepo.CountStudents(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, buildCourseListItem(c, count))
	}
	return items, total, nil
}

// Publish 发布课程（草稿→已发布，需至少1章节+1课时）
func (s *courseService) Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrCourseNotFound
	}
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, course.ID); err != nil {
		return err
	}
	if course.Status != enum.CourseStatusDraft {
		return errcode.ErrCourseAlreadyPublished
	}
	chapterCount, err := s.chapterRepo.CountByCourseID(ctx, id)
	if err != nil {
		return err
	}
	lessonCount, err := s.lessonRepo.CountByCourseID(ctx, id)
	if err != nil {
		return err
	}
	if chapterCount == 0 || lessonCount == 0 {
		return errcode.ErrInvalidParams.WithMessage("请至少添加一个章节和一个课时")
	}
	return s.courseRepo.UpdateStatus(ctx, id, enum.CourseStatusPublished)
}

// End 手动结束课程
func (s *courseService) End(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrCourseNotFound
	}
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, course.ID); err != nil {
		return err
	}
	if course.Status != enum.CourseStatusActive {
		return errcode.ErrInvalidParams.WithMessage("仅进行中的课程可手动结束")
	}
	return s.courseRepo.UpdateStatus(ctx, id, enum.CourseStatusEnded)
}

// Archive 归档课程
func (s *courseService) Archive(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrCourseNotFound
	}
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, course.ID); err != nil {
		return err
	}
	if course.Status != enum.CourseStatusEnded {
		return errcode.ErrInvalidParams.WithMessage("仅已结束的课程可归档")
	}
	return s.courseRepo.UpdateStatus(ctx, id, enum.CourseStatusArchived)
}

// ToggleShare 切换共享状态
func (s *courseService) ToggleShare(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ToggleShareReq) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, id)
	if err != nil {
		return err
	}
	if err := ensureCourseWriteAllowed(course); err != nil {
		return err
	}
	if req.IsShared {
		switch course.Status {
		case enum.CourseStatusPublished, enum.CourseStatusActive, enum.CourseStatusEnded:
		default:
			return errcode.ErrInvalidParams.WithMessage("仅已发布/进行中/已结束的课程可共享")
		}
	}
	return s.courseRepo.UpdateFields(ctx, id, map[string]interface{}{
		"is_shared": req.IsShared, "updated_at": time.Now(),
	})
}

// RefreshInviteCode 刷新邀请码
func (s *courseService) RefreshInviteCode(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, error) {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, id)
	if err != nil {
		return "", err
	}
	if err := ensureCourseContentEditable(course); err != nil {
		return "", err
	}
	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}
	if err := s.courseRepo.UpdateFields(ctx, id, map[string]interface{}{
		"invite_code": code, "updated_at": time.Now(),
	}); err != nil {
		return "", err
	}
	return code, nil
}

// generateInviteCode 生成6位随机邀请码（排除易混淆字符 0OI1）。
// 使用加密安全随机源，失败时向上返回错误，避免静默生成无效邀请码。
func generateInviteCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", errcode.ErrInternal.WithMessage("生成邀请码失败")
		}
		code[i] = chars[n.Int64()]
	}
	return string(code), nil
}

// getDefaultCourseCoverURL 获取课程默认封面地址
// 文档已明确创建课程时需要自动补默认封面，因此统一由服务层兜底。
func getDefaultCourseCoverURL() *string {
	url := "https://oss.example.com/covers/auto_generated.png"
	return &url
}
