// progress_service.go
// 模块03 — 课程与教学：学习进度、课表、统计业务逻辑
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	courserepo "github.com/lenschain/backend/internal/repository/course"
	"gorm.io/gorm"
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
	gradeConfigRepo    courserepo.GradeConfigRepository
	gradeOverrideRepo  courserepo.GradeOverrideRepository
	scheduleRepo       courserepo.ScheduleRepository
	userNameQuerier    UserNameQuerier
	userSummaryQuerier UserSummaryQuerier
}

func scheduleVisibleCourseStatuses() []int16 {
	return []int16{
		enum.CourseStatusPublished,
		enum.CourseStatusActive,
		enum.CourseStatusEnded,
	}
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
	gradeConfigRepo courserepo.GradeConfigRepository,
	gradeOverrideRepo courserepo.GradeOverrideRepository,
	scheduleRepo courserepo.ScheduleRepository,
	userNameQuerier UserNameQuerier,
	userSummaryQuerier UserSummaryQuerier,
) ProgressService {
	return &progressService{
		courseRepo: courseRepo, lessonRepo: lessonRepo,
		chapterRepo: chapterRepo, enrollmentRepo: enrollmentRepo,
		progressRepo: progressRepo, assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo, gradeConfigRepo: gradeConfigRepo,
		gradeOverrideRepo: gradeOverrideRepo, scheduleRepo: scheduleRepo,
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

	course, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, lesson.CourseID)
	if err != nil {
		return err
	}
	if err := ensureCourseLearningAllowed(course); err != nil {
		return err
	}

	now := time.Now()
	status := req.Status
	var completedAt *time.Time
	if status == enum.LearningStatusCompleted {
		status = enum.LearningStatusCompleted
		completedAt = &now
	}

	var videoProgress *int
	if req.VideoProgress != nil {
		videoProgress = req.VideoProgress
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
		return errcode.ErrCourseProgressRateLimit
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
	chapterIDs := make([]int64, 0, len(chapters))
	for _, chapter := range chapters {
		chapterIDs = append(chapterIDs, chapter.ID)
	}
	lessonMap := make(map[int64][]*entity.Lesson, len(chapters))
	if len(chapterIDs) > 0 {
		lessons, err := s.lessonRepo.ListByChapterIDs(ctx, chapterIDs)
		if err != nil {
			return nil, err
		}
		lessonMap = buildLessonsByChapterID(lessons)
	}

	// 获取学生的学习进度
	progresses, err := s.progressRepo.ListByStudentAndCourse(ctx, sc.UserID, courseID)
	if err != nil {
		return nil, err
	}
	progressMap := make(map[int64]*entity.LearningProgress)
	for _, p := range progresses {
		progressMap[p.LessonID] = p
	}

	totalLessons := 0
	completedCount := 0
	lessons := make([]dto.LessonProgressItem, 0)

	for _, ch := range chapters {
		for _, l := range lessonMap[ch.ID] {
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
				if p.VideoProgress != nil {
					item.VideoProgress = *p.VideoProgress
				}
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

	totalDuration, err := s.progressRepo.SumStudyDurationByStudent(ctx, sc.UserID, courseID)
	if err != nil {
		return nil, err
	}

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

	totalLessons, err := s.lessonRepo.CountByCourseID(ctx, courseID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.StudentProgressItem, 0, len(enrollments))
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
		duration, err := s.progressRepo.SumStudyDurationByStudent(ctx, e.StudentID, courseID)
		if err != nil {
			return nil, 0, err
		}
		progresses, err := s.progressRepo.ListByStudentAndCourse(ctx, e.StudentID, courseID)
		if err != nil {
			return nil, 0, err
		}
		var latestAccessedAt *time.Time
		for _, progressRecord := range progresses {
			if progressRecord.LastAccessedAt == nil {
				continue
			}
			if latestAccessedAt == nil || progressRecord.LastAccessedAt.After(*latestAccessedAt) {
				latest := *progressRecord.LastAccessedAt
				latestAccessedAt = &latest
			}
		}
		var lastAccessedAt *string
		if latestAccessedAt != nil {
			t := latestAccessedAt.Format(time.RFC3339)
			lastAccessedAt = &t
		}

		var progress float64
		if totalLessons > 0 {
			progress = float64(completed) / float64(totalLessons) * 100
		}

		items = append(items, &dto.StudentProgressItem{
			StudentID:       strconv.FormatInt(e.StudentID, 10),
			StudentName:     name,
			CompletedCount:  completed,
			TotalLessons:    totalLessons,
			Progress:        progress,
			TotalStudyHours: float64(duration) / 3600,
			LastAccessedAt:  lastAccessedAt,
		})
		if summary != nil {
			items[len(items)-1].StudentNo = summary.StudentNo
		}
	}
	return items, total, nil
}

// ========== 课表 ==========

// SetSchedule 设置课程时间表
func (s *progressService) SetSchedule(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.SetScheduleReq) error {
	course, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	if err != nil {
		return err
	}
	if err := ensureCourseContentEditable(course); err != nil {
		return err
	}
	if err := validateScheduleItems(req.Schedules); err != nil {
		return err
	}

	schedules := make([]*entity.CourseSchedule, 0, len(req.Schedules))
	for _, item := range req.Schedules {
		schedules = append(schedules, &entity.CourseSchedule{
			CourseID: courseID, DayOfWeek: int16(item.DayOfWeek),
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
			ID: strconv.FormatInt(sch.ID, 10), DayOfWeek: int(sch.DayOfWeek),
			StartTime: sch.StartTime, EndTime: sch.EndTime, Location: sch.Location,
		})
	}
	return items, nil
}

// GetMySchedule 获取我的课表（学生/教师）
func (s *progressService) GetMySchedule(ctx context.Context, sc *svcctx.ServiceContext) (*dto.MyScheduleResp, error) {
	courseMap := make(map[int64]*entity.Course)

	if sc.IsTeacher() {
		teacherCourses, err := s.listAllTeacherScheduleCourses(ctx, sc)
		if err != nil {
			return nil, err
		}
		for _, course := range teacherCourses {
			courseMap[course.ID] = course
		}
	}

	if sc.IsStudent() {
		studentCourses, err := s.listAllStudentScheduleCourses(ctx, sc)
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
			DayOfWeek:   int(sch.DayOfWeek), StartTime: sch.StartTime,
			EndTime: sch.EndTime, Location: sch.Location,
		})
	}
	return &dto.MyScheduleResp{Schedules: items}, nil
}

// listAllTeacherScheduleCourses 分页拉取教师课表可见的全部课程。
// 我的课表是聚合视图，不能只取第一页课程，否则课程数较多时会漏课表项。
func (s *progressService) listAllTeacherScheduleCourses(ctx context.Context, sc *svcctx.ServiceContext) ([]*entity.Course, error) {
	const pageSize = 100

	page := 1
	courses := make([]*entity.Course, 0)
	for {
		pageItems, total, err := s.courseRepo.List(ctx, &courserepo.CourseListParams{
			SchoolID:  sc.SchoolID,
			TeacherID: sc.UserID,
			Statuses:  scheduleVisibleCourseStatuses(),
			Page:      page,
			PageSize:  pageSize,
		})
		if err != nil {
			return nil, err
		}

		courses = append(courses, pageItems...)
		if len(courses) >= int(total) || len(pageItems) < pageSize {
			break
		}
		page++
	}

	return courses, nil
}

// listAllStudentScheduleCourses 分页拉取学生课表可见的全部课程。
// 学生已选课程可能超过单页上限，课表聚合必须拉全后再查课表项。
func (s *progressService) listAllStudentScheduleCourses(ctx context.Context, sc *svcctx.ServiceContext) ([]*entity.Course, error) {
	const pageSize = 100

	page := 1
	courses := make([]*entity.Course, 0)
	for {
		pageItems, total, err := s.courseRepo.ListByStudentID(ctx, sc.UserID, &courserepo.StudentCourseListParams{
			Statuses: scheduleVisibleCourseStatuses(),
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}

		courses = append(courses, pageItems...)
		if len(courses) >= int(total) || len(pageItems) < pageSize {
			break
		}
		page++
	}

	return courses, nil
}

// ========== 统计 ==========

// GetCourseOverview 获取课程概览统计
func (s *progressService) GetCourseOverview(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.CourseOverviewStatsResp, error) {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return nil, err
	}

	studentCount, err := s.courseRepo.CountStudents(ctx, courseID)
	if err != nil {
		return nil, err
	}
	lessonCount, err := s.lessonRepo.CountByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	assignmentCount, err := s.assignmentRepo.CountByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	totalDuration, err := s.progressRepo.SumStudyDurationByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// 课程统计按学生维度聚合，保证完课率、活跃度、进度分布口径一致。
	var avgProgress, avgScore, completionRate, activityRate float64
	progressDistribution := dto.CourseProgressDistributionResp{}
	if studentCount > 0 {
		enrollments, err := s.enrollmentRepo.ListAllByCourse(ctx, courseID)
		if err != nil {
			return nil, err
		}

		progresses, err := s.progressRepo.ListByCourse(ctx, courseID)
		if err != nil {
			return nil, err
		}

		progressRecordsByStudent := make(map[int64][]*entity.LearningProgress, len(enrollments))
		for _, progress := range progresses {
			progressRecordsByStudent[progress.StudentID] = append(progressRecordsByStudent[progress.StudentID], progress)
		}

		var totalProgress float64
		activeStudents := 0
		notStartedStudents := 0
		inProgressStudents := 0
		completedStudents := 0
		for _, enrollment := range enrollments {
			records := progressRecordsByStudent[enrollment.StudentID]
			completedLessons := countCompletedLessons(records)
			if lessonCount > 0 {
				totalProgress += float64(completedLessons) / float64(lessonCount) * 100
			}
			if len(records) > 0 {
				activeStudents++
			}

			switch classifyStudentProgress(records, lessonCount, completedLessons) {
			case studentProgressCompleted:
				completedStudents++
			case studentProgressInProgress:
				inProgressStudents++
			default:
				notStartedStudents++
			}
		}

		if lessonCount > 0 {
			avgProgress = totalProgress / float64(studentCount)
			completionRate = float64(completedStudents) / float64(studentCount) * 100
		}
		activityRate = float64(activeStudents) / float64(studentCount) * 100
		progressDistribution = dto.CourseProgressDistributionResp{
			NotStartedRate: round2(float64(notStartedStudents) / float64(studentCount) * 100),
			InProgressRate: round2(float64(inProgressStudents) / float64(studentCount) * 100),
			CompletedRate:  round2(float64(completedStudents) / float64(studentCount) * 100),
		}
		avgScore, err = s.calculateCourseAverageScore(ctx, courseID, enrollments)
		if err != nil {
			return nil, err
		}
	}

	return &dto.CourseOverviewStatsResp{
		StudentCount:         studentCount,
		LessonCount:          lessonCount,
		AssignmentCount:      assignmentCount,
		AvgProgress:          round2(avgProgress),
		AvgScore:             avgScore,
		CompletionRate:       round2(completionRate),
		ActivityRate:         round2(activityRate),
		TotalStudyHours:      float64(totalDuration) / 3600,
		ProgressDistribution: progressDistribution,
	}, nil
}

type studentProgressBucket int

const (
	studentProgressNotStarted studentProgressBucket = iota + 1
	studentProgressInProgress
	studentProgressCompleted
)

// countCompletedLessons 统计单个学生已完成课时数。
func countCompletedLessons(records []*entity.LearningProgress) int {
	completed := 0
	for _, record := range records {
		if record.Status == enum.LearningStatusCompleted {
			completed++
		}
	}
	return completed
}

// classifyStudentProgress 按验收标准将学生学习状态归类为未开始、进行中、已完成。
func classifyStudentProgress(records []*entity.LearningProgress, lessonCount, completedLessons int) studentProgressBucket {
	if lessonCount > 0 && completedLessons >= lessonCount {
		return studentProgressCompleted
	}
	if len(records) == 0 {
		return studentProgressNotStarted
	}
	return studentProgressInProgress
}

// calculateCourseAverageScore 计算课程整体平均成绩。
// 统计口径与模块03成绩汇总保持一致：先按成绩配置计算加权总成绩，再叠加手动调分结果。
func (s *progressService) calculateCourseAverageScore(ctx context.Context, courseID int64, enrollments []*entity.CourseEnrollment) (float64, error) {
	config, err := s.gradeConfigRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	payload := &gradeConfigPayload{}
	if err := json.Unmarshal([]byte(config.Config), payload); err != nil {
		return 0, errcode.ErrInvalidParams.WithMessage("成绩配置解析失败")
	}
	if len(payload.Items) == 0 || len(enrollments) == 0 {
		return 0, nil
	}

	assignmentIDs := make([]int64, 0, len(payload.Items))
	weightMap := make(map[int64]float64, len(payload.Items))
	for _, item := range payload.Items {
		assignmentID, err := snowflake.ParseString(item.AssignmentID)
		if err != nil {
			return 0, errcode.ErrInvalidParams.WithMessage("成绩配置中的作业ID无效")
		}
		assignmentIDs = append(assignmentIDs, assignmentID)
		weightMap[assignmentID] = item.Weight
	}

	submissions, err := s.submissionRepo.ListLatestByAssignments(ctx, assignmentIDs)
	if err != nil {
		return 0, err
	}
	scoreMap := make(map[int64]map[int64]float64, len(enrollments))
	for _, submission := range submissions {
		score := extractSubmissionScore(submission)
		if score == nil {
			continue
		}
		if _, ok := scoreMap[submission.StudentID]; !ok {
			scoreMap[submission.StudentID] = make(map[int64]float64, len(payload.Items))
		}
		scoreMap[submission.StudentID][submission.AssignmentID] = *score
	}

	overrideMap := make(map[int64]float64)
	overrides, err := s.gradeOverrideRepo.ListByCourseID(ctx, courseID)
	if err != nil {
		return 0, err
	}
	for _, override := range overrides {
		overrideMap[override.StudentID] = override.FinalScore
	}

	totalScore := 0.0
	for _, enrollment := range enrollments {
		finalScore, ok := overrideMap[enrollment.StudentID]
		if !ok {
			weightedTotal := 0.0
			for assignmentID, weight := range weightMap {
				if studentScores, exists := scoreMap[enrollment.StudentID]; exists {
					if score, exists := studentScores[assignmentID]; exists {
						weightedTotal += score * weight / 100
					}
				}
			}
			finalScore = weightedTotal
		}
		totalScore += finalScore
	}

	return round2(totalScore / float64(len(enrollments))), nil
}

// verifyCourseTeacher 校验当前用户是否为课程负责教师
func (s *progressService) verifyCourseTeacher(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	_, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	return err
}
