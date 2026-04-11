// progress_service.go
// 模块03 — 课程与教学：学习进度、课表、统计业务逻辑
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// ProgressService 学习进度与课表服务接口
type ProgressService interface {
	// 学习进度
	UpdateProgress(ctx context.Context, sc *svcctx.ServiceContext, lessonID int64, req *dto.UpdateProgressReq) error
	GetMyProgress(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.MyProgressResp, error)
	ListStudentsProgress(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.StudentsProgressReq) ([]*dto.StudentProgressItem, int64, error)
	// 课表
	SetSchedule(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.SetScheduleReq) error
	GetSchedule(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) ([]*dto.ScheduleItemResp, error)
	GetMySchedule(ctx context.Context, sc *svcctx.ServiceContext) (*dto.MyScheduleResp, error)
	// 统计
	GetCourseOverview(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseOverviewStatsResp, error)
}

type progressService struct {
	courseRepo         courserepo.CourseRepository
	lessonRepo         courserepo.LessonRepository
	chapterRepo        courserepo.ChapterRepository
	enrollmentRepo     courserepo.EnrollmentRepository
	progressRepo       courserepo.ProgressRepository
	assignmentRepo     courserepo.AssignmentRepository
	submissionRepo     courserepo.SubmissionRepository
	scheduleRepo       courserepo.ScheduleRepository
	userNameQuerier    UserNameQuerier
	userSummaryQuerier UserSummaryQuerier
}

// NewProgressService 创建学习进度与课表服务实例
func NewProgressService(
	courseRepo courserepo.CourseRepository,
	lessonRepo courserepo.LessonRepository,
	chapterRepo courserepo.ChapterRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	progressRepo courserepo.ProgressRepository,
	assignmentRepo courserepo.AssignmentRepository,
	submissionRepo courserepo.SubmissionRepository,
	scheduleRepo courserepo.ScheduleRepository,
	userNameQuerier UserNameQuerier,
	userSummaryQuerier UserSummaryQuerier,
) ProgressService {
	return &progressService{
		courseRepo: courseRepo, lessonRepo: lessonRepo,
		chapterRepo: chapterRepo, enrollmentRepo: enrollmentRepo,
		progressRepo: progressRepo, assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo, scheduleRepo: scheduleRepo,
		userNameQuerier: userNameQuerier, userSummaryQuerier: userSummaryQuerier,
	}
}

// ========== 学习进度 ==========

// UpdateProgress 更新课时学习进度
func (s *progressService) UpdateProgress(ctx context.Context, sc *svcctx.ServiceContext, lessonID int64, req *dto.UpdateProgressReq) error {
	if err := enforceProgressRateLimit(ctx, sc.UserID); err != nil {
		return err
	}

	lesson, err := s.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return errcode.ErrLessonNotFound
	}

	// 检查选课
	enrolled, _ := s.enrollmentRepo.IsEnrolled(ctx, sc.UserID, lesson.CourseID)
	if !enrolled {
		return errcode.ErrNotCourseStudent
	}

	now := time.Now()
	status := enum.LearningStatusInProgress
	var completedAt *time.Time
	if req.Completed {
		status = enum.LearningStatusCompleted
		completedAt = &now
	}

	videoProgress := 0
	if req.VideoProgress != nil {
		videoProgress = *req.VideoProgress
	}

	progress := &entity.LearningProgress{
		CourseID: lesson.CourseID, StudentID: sc.UserID,
		LessonID: lessonID, Status: status,
		VideoProgress: videoProgress, StudyDuration: req.StudyDurationIncrement,
		CompletedAt: completedAt, LastAccessedAt: &now,
	}
	return s.progressRepo.Upsert(ctx, progress)
}

// enforceProgressRateLimit 限制学习进度上报频率
// 文档要求每用户每分钟最多3次，使用 Redis 计数器做服务端兜底。
func enforceProgressRateLimit(ctx context.Context, userID int64) error {
	key := cache.KeyCourseProgressRateLimit + strconv.FormatInt(userID, 10)
	count, err := cache.IncrWithExpire(ctx, key, time.Minute)
	if err != nil {
		return err
	}
	if count > 3 {
		return errcode.ErrInvalidParams.WithMessage("视频进度上报过于频繁")
	}
	return nil
}

// GetMyProgress 获取我的课程学习进度
func (s *progressService) GetMyProgress(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.MyProgressResp, error) {
	if _, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, err
	}

	// 获取课程所有章节和课时
	chapters, err := s.chapterRepo.ListByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// 获取学生的学习进度
	progresses, _ := s.progressRepo.ListByStudentAndCourse(ctx, sc.UserID, courseID)
	progressMap := make(map[int64]*entity.LearningProgress)
	for _, p := range progresses {
		progressMap[p.LessonID] = p
	}

	totalLessons := 0
	completedCount := 0
	lessons := make([]dto.LessonProgressItem, 0)

	for _, ch := range chapters {
		for _, l := range ch.Lessons {
			totalLessons++
			item := dto.LessonProgressItem{
				LessonID:      strconv.FormatInt(l.ID, 10),
				LessonTitle:   l.Title,
				ChapterTitle:  ch.Title,
				Status:        enum.LearningStatusNotStarted,
				StatusText:    enum.GetLearningStatusText(enum.LearningStatusNotStarted),
				VideoDuration: l.VideoDuration,
			}
			if p, ok := progressMap[l.ID]; ok {
				item.Status = p.Status
				item.StatusText = enum.GetLearningStatusText(p.Status)
				item.VideoProgress = p.VideoProgress
				item.StudyDuration = p.StudyDuration
				if p.CompletedAt != nil {
					t := p.CompletedAt.Format(time.RFC3339)
					item.CompletedAt = &t
				}
				if p.LastAccessedAt != nil {
					t := p.LastAccessedAt.Format(time.RFC3339)
					item.LastAccessedAt = &t
				}
				if p.Status == enum.LearningStatusCompleted {
					completedCount++
				}
			}
			lessons = append(lessons, item)
		}
	}

	var progress float64
	if totalLessons > 0 {
		progress = float64(completedCount) / float64(totalLessons) * 100
	}

	totalDuration, _ := s.progressRepo.SumStudyDurationByStudent(ctx, sc.UserID, courseID)

	return &dto.MyProgressResp{
		CourseID:        strconv.FormatInt(courseID, 10),
		TotalLessons:    totalLessons,
		CompletedCount:  completedCount,
		Progress:        progress,
		TotalStudyHours: float64(totalDuration) / 3600,
		Lessons:         lessons,
	}, nil
}

// ListStudentsProgress 教师查看全班学习进度
func (s *progressService) ListStudentsProgress(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.StudentsProgressReq) ([]*dto.StudentProgressItem, int64, error) {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
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
	items := make([]*dto.StudentProgressItem, 0, len(enrollments))
	for _, e := range enrollments {
		summary := s.userSummaryQuerier.GetUserSummary(ctx, e.StudentID)
		name := ""
		if summary != nil {
			name = summary.Name
		}
		completed, _ := s.progressRepo.CountCompletedByStudent(ctx, e.StudentID, courseID)
		duration, _ := s.progressRepo.SumStudyDurationByStudent(ctx, e.StudentID, courseID)

		var progress float64
		if totalLessons > 0 {
			progress = float64(completed) / float64(totalLessons) * 100
		}

		items = append(items, &dto.StudentProgressItem{
			StudentID:       strconv.FormatInt(e.StudentID, 10),
			StudentName:     name,
			StudentNo:       getSummaryStudentNo(summary),
			CompletedCount:  completed,
			TotalLessons:    totalLessons,
			Progress:        progress,
			TotalStudyHours: float64(duration) / 3600,
		})
	}
	return items, total, nil
}

// ========== 课表 ==========

// SetSchedule 设置课程时间表
func (s *progressService) SetSchedule(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.SetScheduleReq) error {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return err
	}

	schedules := make([]*entity.CourseSchedule, 0, len(req.Schedules))
	for _, item := range req.Schedules {
		schedules = append(schedules, &entity.CourseSchedule{
			CourseID: courseID, DayOfWeek: item.DayOfWeek,
			StartTime: item.StartTime, EndTime: item.EndTime,
			Location: item.Location,
		})
	}
	return s.scheduleRepo.ReplaceByCourseID(ctx, courseID, schedules)
}

// GetSchedule 获取课程时间表
func (s *progressService) GetSchedule(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) ([]*dto.ScheduleItemResp, error) {
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, err
	}

	schedules, err := s.scheduleRepo.ListByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	items := make([]*dto.ScheduleItemResp, 0, len(schedules))
	for _, sch := range schedules {
		items = append(items, &dto.ScheduleItemResp{
			ID: strconv.FormatInt(sch.ID, 10), DayOfWeek: sch.DayOfWeek,
			StartTime: sch.StartTime, EndTime: sch.EndTime, Location: sch.Location,
		})
	}
	return items, nil
}

// GetMySchedule 获取我的课表（学生/教师）
func (s *progressService) GetMySchedule(ctx context.Context, sc *svcctx.ServiceContext) (*dto.MyScheduleResp, error) {
	courseMap := make(map[int64]*entity.Course)

	if sc.IsTeacher() {
		teacherCourses, _, err := s.courseRepo.List(ctx, &courserepo.CourseListParams{
			SchoolID: sc.SchoolID, TeacherID: sc.UserID,
			Status: enum.CourseStatusActive, Page: 1, PageSize: 100,
		})
		if err != nil {
			return nil, err
		}
		for _, course := range teacherCourses {
			courseMap[course.ID] = course
		}
	}

	if sc.IsStudent() || len(courseMap) == 0 {
		studentCourses, _, err := s.courseRepo.ListByStudentID(ctx, sc.UserID, &courserepo.StudentCourseListParams{
			Status: enum.CourseStatusActive, Page: 1, PageSize: 100,
		})
		if err != nil {
			return nil, err
		}
		for _, course := range studentCourses {
			courseMap[course.ID] = course
		}
	}

	if len(courseMap) == 0 {
		return &dto.MyScheduleResp{Schedules: []*dto.MyScheduleItem{}}, nil
	}

	courseIDs := make([]int64, 0, len(courseMap))
	for courseID := range courseMap {
		courseIDs = append(courseIDs, courseID)
	}

	schedules, err := s.scheduleRepo.ListByStudentCourses(ctx, courseIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.MyScheduleItem, 0, len(schedules))
	for _, sch := range schedules {
		c := courseMap[sch.CourseID]
		title := ""
		teacherName := ""
		if c != nil {
			title = c.Title
			teacherName = s.userNameQuerier.GetUserName(ctx, c.TeacherID)
		}
		items = append(items, &dto.MyScheduleItem{
			CourseID: strconv.FormatInt(sch.CourseID, 10), CourseTitle: title,
			TeacherName: teacherName,
			DayOfWeek:   sch.DayOfWeek, StartTime: sch.StartTime,
			EndTime: sch.EndTime, Location: sch.Location,
		})
	}
	return &dto.MyScheduleResp{Schedules: items}, nil
}

// ========== 统计 ==========

// GetCourseOverview 获取课程概览统计
func (s *progressService) GetCourseOverview(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseOverviewStatsResp, error) {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return nil, err
	}

	studentCount, _ := s.courseRepo.CountStudents(ctx, courseID)
	lessonCount, _ := s.lessonRepo.CountByCourseID(ctx, courseID)
	assignmentCount, _ := s.assignmentRepo.CountByCourseID(ctx, courseID)
	totalDuration, _ := s.progressRepo.SumStudyDurationByCourse(ctx, courseID)

	// 计算平均进度和完成率
	var avgProgress, completionRate float64
	if studentCount > 0 && lessonCount > 0 {
		enrollments, _, _ := s.enrollmentRepo.List(ctx, &courserepo.EnrollmentListParams{
			CourseID: courseID, Page: 1, PageSize: 1000,
		})
		var totalProgress float64
		completedStudents := 0
		for _, e := range enrollments {
			completed, _ := s.progressRepo.CountCompletedByStudent(ctx, e.StudentID, courseID)
			p := float64(completed) / float64(lessonCount) * 100
			totalProgress += p
			if completed >= lessonCount {
				completedStudents++
			}
		}
		avgProgress = totalProgress / float64(studentCount)
		completionRate = float64(completedStudents) / float64(studentCount) * 100
	}

	return &dto.CourseOverviewStatsResp{
		StudentCount:    studentCount,
		LessonCount:     lessonCount,
		AssignmentCount: assignmentCount,
		AvgProgress:     avgProgress,
		CompletionRate:  completionRate,
		TotalStudyHours: float64(totalDuration) / 3600,
	}, nil
}

// verifyCourseTeacher 校验当前用户是否为课程负责教师
func (s *progressService) verifyCourseTeacher(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	_, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	return err
}
