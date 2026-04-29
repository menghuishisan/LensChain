// content_service.go
// 模块03 — 课程与教学：章节、课时、附件、选课业务逻辑
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/contentsafety"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/storage"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// ContentService 章节课时管理服务接口
type ContentService interface {
	// 章节
	ListChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) ([]*dto.ChapterWithLessonsResp, error)
	CreateChapter(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateChapterReq) (string, error)
	UpdateChapter(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChapterReq) error
	DeleteChapter(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	SortChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.ReorderIDsReq) error
	// 课时
	CreateLesson(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.CreateLessonReq) (string, error)
	GetLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.LessonDetailResp, error)
	UpdateLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateLessonReq) error
	DeleteLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	SortLessons(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.ReorderIDsReq) error
	// 附件
	UploadCourseFile(ctx context.Context, sc *svcctx.ServiceContext, fileName string, reader io.Reader, fileSize int64, contentType string, purpose string, lessonID string) (*dto.UploadCourseFileResp, error)
	UploadAttachment(ctx context.Context, sc *svcctx.ServiceContext, lessonID int64, req *dto.UploadAttachmentReq) (string, error)
	DeleteAttachment(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	// 选课
	JoinByInviteCode(ctx context.Context, sc *svcctx.ServiceContext, req *dto.JoinCourseReq) (string, error)
	AddStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AddStudentReq) error
	BatchAddStudents(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.BatchAddStudentsReq) error
	RemoveStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64) error
	ListStudents(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.StudentListReq) ([]*dto.EnrolledStudentItem, int64, error)
}

const (
	courseFilePurposeLessonAttachment = "lesson_attachment"
	courseFilePurposeAssignmentReport = "assignment_report"
	maxCourseVideoFileSize            = 500 << 20
	maxCourseDocumentFileSize         = 50 << 20
)

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
	chapterIDs := make([]int64, 0, len(chapters))
	for _, chapter := range chapters {
		chapterIDs = append(chapterIDs, chapter.ID)
	}
	lessons, err := s.lessonRepo.ListByChapterIDs(ctx, chapterIDs)
	if err != nil {
		return nil, err
	}
	lessonMap := buildLessonsByChapterID(lessons)

	result := make([]*dto.ChapterWithLessonsResp, 0, len(chapters))
	for _, ch := range chapters {
		chapterLessons := lessonMap[ch.ID]
		items := make([]dto.LessonListItem, 0, len(chapterLessons))
		for _, l := range chapterLessons {
			items = append(items, dto.LessonListItem{
				ID: strconv.FormatInt(l.ID, 10), Title: l.Title,
				ContentType: l.ContentType, ContentTypeText: enum.GetContentTypeText(l.ContentType),
				VideoDuration: l.VideoDuration, EstimatedMinutes: l.EstimatedMinutes,
				SortOrder: l.SortOrder,
			})
			if l.ExperimentID != nil {
				eid := strconv.FormatInt(*l.ExperimentID, 10)
				items[len(items)-1].ExperimentID = &eid
			}
		}
		result = append(result, &dto.ChapterWithLessonsResp{
			ID: strconv.FormatInt(ch.ID, 10), Title: ch.Title,
			Description: ch.Description, SortOrder: ch.SortOrder,
			Lessons: items,
		})
	}
	return result, nil
}

// CreateChapter 创建章节
func (s *contentService) CreateChapter(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateChapterReq) (string, error) {
	if err := s.verifyCourseTeacherForContent(ctx, sc, courseID); err != nil {
		return "", err
	}
	chapter := &entity.Chapter{
		CourseID: courseID, Title: req.Title,
		Description: contentsafety.SanitizeOptionalMarkdown(req.Description),
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
	if err := s.verifyCourseTeacherForContent(ctx, sc, chapter.CourseID); err != nil {
		return err
	}
	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = contentsafety.SanitizeMarkdown(*req.Description)
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
	if err := s.verifyCourseTeacherForContent(ctx, sc, chapter.CourseID); err != nil {
		return err
	}

	lessons, err := s.lessonRepo.ListByChapterID(ctx, chapter.ID)
	if err != nil {
		return err
	}
	if err := s.cleanupLessonsBeforeDelete(ctx, lessons); err != nil {
		return err
	}
	return s.chapterRepo.SoftDelete(ctx, id)
}

// SortChapters 调整章节排序
func (s *contentService) SortChapters(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.ReorderIDsReq) error {
	if err := s.verifyCourseTeacherForContent(ctx, sc, courseID); err != nil {
		return err
	}
	chapters, err := s.chapterRepo.ListByCourseID(ctx, courseID)
	if err != nil {
		return err
	}
	if len(chapters) != len(req.IDs) {
		return errcode.ErrInvalidParams.WithMessage("章节排序ID必须完整覆盖当前课程全部章节")
	}

	validIDs := make(map[int64]struct{}, len(chapters))
	for _, chapter := range chapters {
		validIDs[chapter.ID] = struct{}{}
	}

	items, err := buildSequentialSortItems(req.IDs, "章节")
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, ok := validIDs[item.ID]; !ok {
			return errcode.ErrInvalidParams.WithMessage("章节排序ID必须全部属于当前课程")
		}
	}
	return s.chapterRepo.UpdateSortOrders(ctx, items)
}

// ========== 课时 ==========

func (s *contentService) CreateLesson(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.CreateLessonReq) (string, error) {
	chapter, err := s.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		return "", errcode.ErrChapterNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, chapter.CourseID); err != nil {
		return "", err
	}

	lesson := &entity.Lesson{
		ChapterID: chapterID, CourseID: chapter.CourseID,
		Title: req.Title, ContentType: req.ContentType,
		Content: contentsafety.SanitizeOptionalMarkdown(req.Content), VideoURL: req.VideoURL,
		VideoDuration: req.VideoDuration, EstimatedMinutes: req.EstimatedMinutes,
	}
	if req.ExperimentID != nil {
		experimentID, err := parseOptionalSnowflakeID(req.ExperimentID, "实验ID格式错误")
		if err != nil {
			return "", err
		}
		lesson.ExperimentID = experimentID
	}
	if err := s.lessonRepo.Create(ctx, lesson); err != nil {
		return "", err
	}
	return strconv.FormatInt(lesson.ID, 10), nil
}

// GetLesson 获取课时详情
func (s *contentService) GetLesson(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.LessonDetailResp, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrLessonNotFound
	}
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, lesson.CourseID); err != nil {
		return nil, err
	}
	attachments, err := s.attachmentRepo.ListByLessonID(ctx, lesson.ID)
	if err != nil {
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

	resp.Attachments = make([]dto.LessonAttachmentItem, 0, len(attachments))
	for _, a := range attachments {
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
	if err := s.verifyCourseTeacherForContent(ctx, sc, lesson.CourseID); err != nil {
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
		fields["content"] = contentsafety.SanitizeMarkdown(*req.Content)
	}
	if req.VideoURL != nil {
		fields["video_url"] = *req.VideoURL
	}
	if req.VideoDuration != nil {
		fields["video_duration"] = *req.VideoDuration
	}
	if req.ExperimentID != nil {
		experimentID, err := parseOptionalSnowflakeID(req.ExperimentID, "实验ID格式错误")
		if err != nil {
			return err
		}
		if experimentID == nil {
			fields["experiment_id"] = nil
		} else {
			fields["experiment_id"] = *experimentID
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
	if err := s.verifyCourseTeacherForContent(ctx, sc, lesson.CourseID); err != nil {
		return err
	}

	if err := s.cleanupLessonBeforeDelete(ctx, lesson.ID); err != nil {
		return err
	}
	return s.lessonRepo.SoftDelete(ctx, id)
}

// SortLessons 调整课时排序
func (s *contentService) SortLessons(ctx context.Context, sc *svcctx.ServiceContext, chapterID int64, req *dto.ReorderIDsReq) error {
	chapter, err := s.chapterRepo.GetByID(ctx, chapterID)
	if err != nil {
		return errcode.ErrChapterNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, chapter.CourseID); err != nil {
		return err
	}
	lessons, err := s.lessonRepo.ListByChapterID(ctx, chapterID)
	if err != nil {
		return err
	}
	if len(lessons) != len(req.IDs) {
		return errcode.ErrInvalidParams.WithMessage("课时排序ID必须完整覆盖当前章节全部课时")
	}

	validIDs := make(map[int64]struct{}, len(lessons))
	for _, lesson := range lessons {
		validIDs[lesson.ID] = struct{}{}
	}

	items, err := buildSequentialSortItems(req.IDs, "课时")
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, ok := validIDs[item.ID]; !ok {
			return errcode.ErrInvalidParams.WithMessage("课时排序ID必须全部属于当前章节")
		}
	}
	return s.lessonRepo.UpdateSortOrders(ctx, items)
}

// ========== 附件 ==========

// UploadCourseFile 上传课程文件到对象存储，返回持久化对象键和短期下载URL。
func (s *contentService) UploadCourseFile(ctx context.Context, sc *svcctx.ServiceContext, fileName string, reader io.Reader, fileSize int64, contentType string, purpose string, lessonID string) (*dto.UploadCourseFileResp, error) {
	if err := s.validateCourseUploadAccess(ctx, sc, purpose, lessonID); err != nil {
		return nil, err
	}
	if err := validateCourseFile(fileName, contentType, fileSize, purpose); err != nil {
		return nil, err
	}

	extension := strings.ToLower(filepath.Ext(fileName))
	objectName := fmt.Sprintf("course/%s/%d/%d%s", purpose, sc.UserID, snowflake.Generate(), extension)
	uploadedObject, err := storage.UploadFile(ctx, objectName, reader, fileSize, contentType)
	if err != nil {
		return nil, errcode.ErrMinIO.WithMessage("上传课程文件失败")
	}
	downloadURL, err := storage.GetFileURL(ctx, uploadedObject, time.Hour)
	if err != nil {
		return nil, errcode.ErrMinIO.WithMessage("生成课程文件下载地址失败")
	}

	return &dto.UploadCourseFileResp{
		FileName:    fileName,
		FileURL:     uploadedObject,
		DownloadURL: downloadURL,
		FileSize:    fileSize,
		FileType:    contentType,
	}, nil
}

func (s *contentService) UploadAttachment(ctx context.Context, sc *svcctx.ServiceContext, lessonID int64, req *dto.UploadAttachmentReq) (string, error) {
	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return "", errcode.ErrLessonNotFound
	}
	if err := s.verifyCourseTeacherForContent(ctx, sc, lesson.CourseID); err != nil {
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

func (s *contentService) validateCourseUploadAccess(ctx context.Context, sc *svcctx.ServiceContext, purpose string, lessonID string) error {
	switch purpose {
	case courseFilePurposeLessonAttachment:
		if !sc.IsTeacher() {
			return errcode.ErrForbidden
		}
		if strings.TrimSpace(lessonID) == "" {
			return errcode.ErrInvalidParams.WithMessage("lesson_id 不能为空")
		}
		parsedLessonID, err := snowflake.ParseString(lessonID)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("lesson_id 无效")
		}
		lesson, err := s.lessonRepo.GetByID(ctx, parsedLessonID)
		if err != nil {
			return errcode.ErrLessonNotFound
		}
		if err := s.verifyCourseTeacherForContent(ctx, sc, lesson.CourseID); err != nil {
			return err
		}
		return nil
	case courseFilePurposeAssignmentReport:
		if sc.IsStudent() {
			return nil
		}
	default:
		return errcode.ErrInvalidParams.WithMessage("不支持的课程文件用途")
	}
	return errcode.ErrForbidden
}

func validateCourseFile(fileName string, contentType string, fileSize int64, purpose string) error {
	if strings.TrimSpace(fileName) == "" || fileSize <= 0 {
		return errcode.ErrInvalidParams.WithMessage("文件不能为空")
	}
	isVideo := strings.HasPrefix(contentType, "video/")
	isDocument := isCourseDocumentContentType(contentType)

	switch purpose {
	case courseFilePurposeLessonAttachment:
		if isVideo {
			if fileSize > maxCourseVideoFileSize {
				return errcode.ErrInvalidParams.WithMessage("视频文件不能超过500MB")
			}
			return nil
		}
		if isDocument {
			if fileSize > maxCourseDocumentFileSize {
				return errcode.ErrInvalidParams.WithMessage("文档文件不能超过50MB")
			}
			return nil
		}
		return errcode.ErrInvalidParams.WithMessage("课时附件仅支持视频或PDF/Word/PPT文档")
	case courseFilePurposeAssignmentReport:
		if !isDocument {
			return errcode.ErrInvalidParams.WithMessage("实验报告仅支持PDF/Word/PPT文档")
		}
		if fileSize > maxCourseDocumentFileSize {
			return errcode.ErrInvalidParams.WithMessage("文档文件不能超过50MB")
		}
		return nil
	default:
		return errcode.ErrInvalidParams.WithMessage("不支持的课程文件用途")
	}
}

func isCourseDocumentContentType(contentType string) bool {
	switch contentType {
	case "application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return true
	default:
		return false
	}
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
	if err := s.verifyCourseTeacherForContent(ctx, sc, lesson.CourseID); err != nil {
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
	if course.Status != enum.CourseStatusPublished && course.Status != enum.CourseStatusActive {
		return "", errcode.ErrInviteCodeInvalid
	}

	enrolled, err := s.enrollmentRepo.IsEnrolled(ctx, sc.UserID, course.ID)
	if err != nil {
		return "", err
	}
	if enrolled {
		return "", errcode.ErrAlreadyEnrolled
	}

	if course.MaxStudents != nil {
		count, err := s.courseRepo.CountStudents(ctx, course.ID)
		if err != nil {
			return "", err
		}
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
	course, err := s.ensureTeacherManageableCourse(ctx, sc, courseID)
	if err != nil {
		return err
	}
	studentID, err := snowflake.ParseString(req.StudentID)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("无效的学生ID")
	}
	enrolled, err := s.enrollmentRepo.IsEnrolled(ctx, studentID, courseID)
	if err != nil {
		return err
	}
	if enrolled {
		return errcode.ErrAlreadyEnrolled
	}
	if err := s.ensureCourseStudentCandidate(ctx, sc, studentID); err != nil {
		return err
	}
	if err := s.ensureCourseEnrollmentCapacity(ctx, course, 1); err != nil {
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
	course, err := s.ensureTeacherManageableCourse(ctx, sc, courseID)
	if err != nil {
		return err
	}
	enrollments := make([]*entity.CourseEnrollment, 0)
	seenStudentIDs := make(map[int64]struct{}, len(req.StudentIDs))
	for _, sid := range req.StudentIDs {
		studentID, err := snowflake.ParseString(sid)
		if err != nil {
			continue
		}
		if _, exists := seenStudentIDs[studentID]; exists {
			continue
		}
		seenStudentIDs[studentID] = struct{}{}
		enrolled, err := s.enrollmentRepo.IsEnrolled(ctx, studentID, courseID)
		if err != nil {
			return err
		}
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
	if err := s.ensureCourseEnrollmentCapacity(ctx, course, len(enrollments)); err != nil {
		return err
	}
	return s.enrollmentRepo.BatchCreate(ctx, enrollments)
}

// RemoveStudent 移除课程学生
func (s *contentService) RemoveStudent(ctx context.Context, sc *svcctx.ServiceContext, courseID, studentID int64) error {
	if err := s.verifyCourseTeacherForEnrollment(ctx, sc, courseID); err != nil {
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

	totalLessons, err := s.lessonRepo.CountByCourseID(ctx, courseID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.EnrolledStudentItem, 0, len(enrollments))
	for _, e := range enrollments {
		summary := s.userSummaryQuerier.GetUserSummary(ctx, e.StudentID)
		name := ""
		if summary != nil {
			name = summary.Name
		}
		completed, err := s.progressRepo.CountCompletedByStudent(ctx, e.StudentID, courseID)
		if err != nil {
			return nil, 0, err
		}
		var progress float64
		if totalLessons > 0 {
			progress = float64(completed) / float64(totalLessons) * 100
		}
		items = append(items, &dto.EnrolledStudentItem{
			ID:         strconv.FormatInt(e.StudentID, 10),
			Name:       name,
			JoinMethod: e.JoinMethod, JoinMethodText: enum.GetJoinMethodText(e.JoinMethod),
			JoinedAt: e.JoinedAt.Format(time.RFC3339), Progress: progress,
		})
		if summary != nil {
			items[len(items)-1].StudentNo = summary.StudentNo
			items[len(items)-1].College = summary.College
			items[len(items)-1].Major = summary.Major
			items[len(items)-1].ClassName = summary.ClassName
		}
	}
	return items, total, nil
}

// verifyCourseTeacherForContent 校验课程教师身份，并确保课程仍允许编辑教学内容。
func (s *contentService) verifyCourseTeacherForContent(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return err
	}
	return ensureCourseContentEditable(course)
}

// verifyCourseTeacherForEnrollment 校验课程教师身份，并确保课程仍允许管理学生名单。
func (s *contentService) verifyCourseTeacherForEnrollment(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return err
	}
	return ensureCourseEnrollmentManageable(course)
}

// ensureTeacherManageableCourse 返回当前教师可管理学生名单的课程实体。
func (s *contentService) ensureTeacherManageableCourse(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*entity.Course, error) {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return nil, err
	}
	if err := ensureCourseEnrollmentManageable(course); err != nil {
		return nil, err
	}
	return course, nil
}

// ensureCourseEnrollmentCapacity 校验课程剩余名额是否足够。
// 最大学生数属于课程级约束，教师指定与邀请码加入都必须统一遵守。
func (s *contentService) ensureCourseEnrollmentCapacity(ctx context.Context, course *entity.Course, incoming int) error {
	if course == nil || course.MaxStudents == nil || incoming <= 0 {
		return nil
	}
	count, err := s.courseRepo.CountStudents(ctx, course.ID)
	if err != nil {
		return err
	}
	if count+incoming > *course.MaxStudents {
		return errcode.ErrCourseStudentFull
	}
	return nil
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

// buildLessonsByChapterID 按章节归并课时列表。
// repository 层按“单功能单实现”只返回平铺记录，这里由 service 组合目录树。
func buildLessonsByChapterID(lessons []*entity.Lesson) map[int64][]*entity.Lesson {
	lessonMap := make(map[int64][]*entity.Lesson, len(lessons))
	for _, lesson := range lessons {
		lessonMap[lesson.ChapterID] = append(lessonMap[lesson.ChapterID], lesson)
	}
	return lessonMap
}

// cleanupLessonsBeforeDelete 递归清理待删除课时的附件与学习进度引用。
// 删除章节时必须先清理子课时关联数据，再删除章节主体，避免残留无主记录。
func (s *contentService) cleanupLessonsBeforeDelete(ctx context.Context, lessons []*entity.Lesson) error {
	for _, lesson := range lessons {
		if lesson == nil {
			continue
		}
		if err := s.cleanupLessonBeforeDelete(ctx, lesson.ID); err != nil {
			return err
		}
		if err := s.lessonRepo.SoftDelete(ctx, lesson.ID); err != nil {
			return err
		}
	}
	return nil
}

// cleanupLessonBeforeDelete 清理单个课时的从属记录。
// 课时附件与学习进度都直接依附课时存在，删除课时时需同步清理。
func (s *contentService) cleanupLessonBeforeDelete(ctx context.Context, lessonID int64) error {
	attachments, err := s.attachmentRepo.ListByLessonID(ctx, lessonID)
	if err != nil {
		return err
	}
	for _, attachment := range attachments {
		if attachment == nil {
			continue
		}
		if err := s.attachmentRepo.Delete(ctx, attachment.ID); err != nil {
			return err
		}
	}
	return s.progressRepo.DeleteByLessonID(ctx, lessonID)
}

// buildSequentialSortItems 将前端提交的完整 ID 顺序转换为顺排的 sort_order 更新项。
// 文档约定排序接口提交的是目标顺序数组，而不是任意 sort_order 值，因此这里统一重写为 1..n。
func buildSequentialSortItems(ids []string, resourceName string) ([]courserepo.SortItem, error) {
	items := make([]courserepo.SortItem, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for index, rawID := range ids {
		id, err := snowflake.ParseString(rawID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("无效的" + resourceName + "ID")
		}
		if _, exists := seen[id]; exists {
			return nil, errcode.ErrInvalidParams.WithMessage(resourceName + "排序ID不能重复")
		}
		seen[id] = struct{}{}
		items = append(items, courserepo.SortItem{
			ID:        id,
			SortOrder: index + 1,
		})
	}
	return items, nil
}
