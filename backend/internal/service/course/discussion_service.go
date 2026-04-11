// discussion_service.go
// 模块03 — 课程与教学：讨论区、公告、评价、成绩管理业务逻辑
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"context"
	"encoding/json"
	"math"
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

// DiscussionService 讨论区与公告服务接口
type DiscussionService interface {
	// 讨论
	CreateDiscussion(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateDiscussionReq) (string, error)
	GetDiscussion(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.DiscussionDetailResp, error)
	ListDiscussions(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.DiscussionListReq) ([]*dto.DiscussionListItem, int64, error)
	DeleteDiscussion(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	PinDiscussion(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.PinDiscussionReq) error
	// 回复
	CreateReply(ctx context.Context, sc *svcctx.ServiceContext, discussionID int64, req *dto.CreateReplyReq) (string, error)
	DeleteReply(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	// 点赞
	ToggleLike(ctx context.Context, sc *svcctx.ServiceContext, discussionID int64) (bool, error)
	// 公告
	CreateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateAnnouncementReq) (string, error)
	UpdateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateAnnouncementReq) error
	DeleteAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	ListAnnouncements(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AnnouncementListReq) ([]*dto.AnnouncementItem, int64, error)
	// 评价
	CreateEvaluation(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateEvaluationReq) (string, error)
	UpdateEvaluation(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateEvaluationReq) error
	ListEvaluations(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.EvaluationListReq) ([]*dto.EvaluationItem, *dto.EvaluationSummary, int64, error)
	// 成绩配置
	SetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.GradeConfigReq) error
	GetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.GradeConfigResp, error)
}

type discussionService struct {
	courseRepo       courserepo.CourseRepository
	discussionRepo   courserepo.DiscussionRepository
	replyRepo        courserepo.ReplyRepository
	likeRepo         courserepo.LikeRepository
	announcementRepo courserepo.AnnouncementRepository
	evaluationRepo   courserepo.EvaluationRepository
	enrollmentRepo   courserepo.EnrollmentRepository
	gradeConfigRepo  courserepo.GradeConfigRepository
	userNameQuerier  UserNameQuerier
}

// NewDiscussionService 创建讨论区与公告服务实例
func NewDiscussionService(
	courseRepo courserepo.CourseRepository,
	discussionRepo courserepo.DiscussionRepository,
	replyRepo courserepo.ReplyRepository,
	likeRepo courserepo.LikeRepository,
	announcementRepo courserepo.AnnouncementRepository,
	evaluationRepo courserepo.EvaluationRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	gradeConfigRepo courserepo.GradeConfigRepository,
	userNameQuerier UserNameQuerier,
) DiscussionService {
	return &discussionService{
		courseRepo: courseRepo, discussionRepo: discussionRepo,
		replyRepo: replyRepo, likeRepo: likeRepo,
		announcementRepo: announcementRepo, evaluationRepo: evaluationRepo,
		enrollmentRepo: enrollmentRepo, gradeConfigRepo: gradeConfigRepo,
		userNameQuerier: userNameQuerier,
	}
}

// ========== 讨论 ==========

func (s *discussionService) CreateDiscussion(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateDiscussionReq) (string, error) {
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return "", err
	}

	discussion := &entity.CourseDiscussion{
		CourseID: courseID, AuthorID: sc.UserID,
		Title: req.Title, Content: req.Content,
	}
	if err := s.discussionRepo.Create(ctx, discussion); err != nil {
		return "", err
	}
	return strconv.FormatInt(discussion.ID, 10), nil
}

func (s *discussionService) GetDiscussion(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.DiscussionDetailResp, error) {
	discussion, err := s.discussionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("讨论不存在")
	}
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, discussion.CourseID); err != nil {
		return nil, err
	}

	authorName := s.userNameQuerier.GetUserName(ctx, discussion.AuthorID)
	liked, _ := s.likeRepo.Exists(ctx, id, sc.UserID)

	resp := &dto.DiscussionDetailResp{
		ID:       strconv.FormatInt(discussion.ID, 10),
		CourseID: strconv.FormatInt(discussion.CourseID, 10),
		Title:    discussion.Title, Content: discussion.Content,
		AuthorID:   strconv.FormatInt(discussion.AuthorID, 10),
		AuthorName: authorName, IsPinned: discussion.IsPinned,
		ReplyCount: discussion.ReplyCount, LikeCount: discussion.LikeCount,
		IsLiked: liked, CreatedAt: discussion.CreatedAt.Format(time.RFC3339),
	}

	// 加载回复
	replies, _ := s.replyRepo.ListByDiscussionID(ctx, id)
	resp.Replies = make([]dto.DiscussionReplyItem, 0, len(replies))
	for _, r := range replies {
		rName := s.userNameQuerier.GetUserName(ctx, r.AuthorID)
		item := dto.DiscussionReplyItem{
			ID:         strconv.FormatInt(r.ID, 10),
			AuthorID:   strconv.FormatInt(r.AuthorID, 10),
			AuthorName: rName, Content: r.Content,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
		}
		if r.ReplyToID != nil {
			rid := strconv.FormatInt(*r.ReplyToID, 10)
			item.ReplyToID = &rid
			replyTo, replyErr := s.replyRepo.GetByID(ctx, *r.ReplyToID)
			if replyErr == nil {
				replyToName := s.userNameQuerier.GetUserName(ctx, replyTo.AuthorID)
				if replyToName != "" {
					item.ReplyToName = &replyToName
				}
			}
		}
		resp.Replies = append(resp.Replies, item)
	}
	return resp, nil
}

func (s *discussionService) ListDiscussions(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.DiscussionListReq) ([]*dto.DiscussionListItem, int64, error) {
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, 0, err
	}

	discussions, total, err := s.discussionRepo.List(ctx, &courserepo.DiscussionListParams{
		CourseID: courseID, Keyword: req.Keyword, SortBy: req.SortBy,
		Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	// 批量查询当前用户的点赞状态
	dIDs := make([]int64, 0, len(discussions))
	for _, d := range discussions {
		dIDs = append(dIDs, d.ID)
	}
	likedIDs, _ := s.likeRepo.ListByUserAndDiscussions(ctx, sc.UserID, dIDs)
	likedSet := make(map[int64]bool)
	for _, id := range likedIDs {
		likedSet[id] = true
	}

	items := make([]*dto.DiscussionListItem, 0, len(discussions))
	for _, d := range discussions {
		authorName := s.userNameQuerier.GetUserName(ctx, d.AuthorID)
		item := &dto.DiscussionListItem{
			ID: strconv.FormatInt(d.ID, 10), Title: d.Title,
			AuthorID: strconv.FormatInt(d.AuthorID, 10), AuthorName: authorName,
			IsPinned: d.IsPinned, ReplyCount: d.ReplyCount, LikeCount: d.LikeCount,
			IsLiked: likedSet[d.ID], CreatedAt: d.CreatedAt.Format(time.RFC3339),
		}
		if d.LastRepliedAt != nil {
			t := d.LastRepliedAt.Format(time.RFC3339)
			item.LastRepliedAt = &t
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (s *discussionService) DeleteDiscussion(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	discussion, err := s.discussionRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("讨论不存在")
	}
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, discussion.CourseID); err != nil {
		return err
	}
	// 作者或教师可删除
	if discussion.AuthorID != sc.UserID {
		if err := s.verifyCourseTeacher(ctx, sc, discussion.CourseID); err != nil {
			return errcode.ErrForbidden
		}
	}
	return s.discussionRepo.SoftDelete(ctx, id)
}

func (s *discussionService) PinDiscussion(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.PinDiscussionReq) error {
	discussion, err := s.discussionRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("讨论不存在")
	}
	if _, err := ensureCourseTeacher(ctx, sc, s.courseRepo, discussion.CourseID); err != nil {
		return err
	}
	return s.discussionRepo.UpdateFields(ctx, id, map[string]interface{}{
		"is_pinned": req.IsPinned, "updated_at": time.Now(),
	})
}

// ========== 回复 ==========

func (s *discussionService) CreateReply(ctx context.Context, sc *svcctx.ServiceContext, discussionID int64, req *dto.CreateReplyReq) (string, error) {
	discussion, err := s.discussionRepo.GetByID(ctx, discussionID)
	if err != nil {
		return "", errcode.ErrInvalidParams.WithMessage("讨论不存在")
	}
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, discussion.CourseID); err != nil {
		return "", err
	}

	reply := &entity.DiscussionReply{
		DiscussionID: discussionID, AuthorID: sc.UserID,
		Content: req.Content,
	}
	if req.ReplyToID != nil {
		rid, err := snowflake.ParseString(*req.ReplyToID)
		if err == nil {
			reply.ReplyToID = &rid
		}
	}
	if err := s.replyRepo.Create(ctx, reply); err != nil {
		return "", err
	}
	// 更新讨论帖回复数和最后回复时间
	_ = s.discussionRepo.IncrReplyCount(ctx, discussionID, 1)
	now := time.Now()
	_ = s.discussionRepo.UpdateFields(ctx, discussionID, map[string]interface{}{
		"last_replied_at": now,
	})
	return strconv.FormatInt(reply.ID, 10), nil
}

func (s *discussionService) DeleteReply(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	reply, err := s.replyRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("回复不存在")
	}
	discussion, _ := s.discussionRepo.GetByID(ctx, reply.DiscussionID)
	if discussion != nil {
		if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, discussion.CourseID); err != nil {
			return err
		}
	}
	if reply.AuthorID != sc.UserID {
		if discussion != nil {
			if err := s.verifyCourseTeacher(ctx, sc, discussion.CourseID); err != nil {
				return errcode.ErrForbidden
			}
		}
	}
	if err := s.replyRepo.SoftDelete(ctx, id); err != nil {
		return err
	}
	_ = s.discussionRepo.IncrReplyCount(ctx, reply.DiscussionID, -1)
	return nil
}

// ========== 点赞 ==========

func (s *discussionService) ToggleLike(ctx context.Context, sc *svcctx.ServiceContext, discussionID int64) (bool, error) {
	discussion, err := s.discussionRepo.GetByID(ctx, discussionID)
	if err != nil {
		return false, errcode.ErrInvalidParams.WithMessage("讨论不存在")
	}
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, discussion.CourseID); err != nil {
		return false, err
	}

	exists, _ := s.likeRepo.Exists(ctx, discussionID, sc.UserID)
	if exists {
		_ = s.likeRepo.Delete(ctx, discussionID, sc.UserID)
		_ = s.discussionRepo.IncrLikeCount(ctx, discussionID, -1)
		return false, nil
	}
	like := &entity.DiscussionLike{
		DiscussionID: discussionID, UserID: sc.UserID,
	}
	_ = s.likeRepo.Create(ctx, like)
	_ = s.discussionRepo.IncrLikeCount(ctx, discussionID, 1)
	return true, nil
}

// ========== 公告 ==========

func (s *discussionService) CreateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateAnnouncementReq) (string, error) {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return "", err
	}
	announcement := &entity.CourseAnnouncement{
		CourseID: courseID, TeacherID: sc.UserID,
		Title: req.Title, Content: req.Content,
	}
	if err := s.announcementRepo.Create(ctx, announcement); err != nil {
		return "", err
	}
	return strconv.FormatInt(announcement.ID, 10), nil
}

func (s *discussionService) UpdateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateAnnouncementReq) error {
	announcement, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("公告不存在")
	}
	if err := s.verifyCourseTeacher(ctx, sc, announcement.CourseID); err != nil {
		return err
	}
	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Content != nil {
		fields["content"] = *req.Content
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.announcementRepo.UpdateFields(ctx, id, fields)
}

func (s *discussionService) DeleteAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	announcement, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("公告不存在")
	}
	if err := s.verifyCourseTeacher(ctx, sc, announcement.CourseID); err != nil {
		return err
	}
	return s.announcementRepo.SoftDelete(ctx, id)
}

func (s *discussionService) ListAnnouncements(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.AnnouncementListReq) ([]*dto.AnnouncementItem, int64, error) {
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, 0, err
	}

	announcements, total, err := s.announcementRepo.List(ctx, courseID, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.AnnouncementItem, 0, len(announcements))
	for _, a := range announcements {
		teacherName := s.userNameQuerier.GetUserName(ctx, a.TeacherID)
		items = append(items, &dto.AnnouncementItem{
			ID: strconv.FormatInt(a.ID, 10), Title: a.Title,
			Content: a.Content, IsPinned: a.IsPinned,
			TeacherName: teacherName,
			CreatedAt:   a.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   a.UpdatedAt.Format(time.RFC3339),
		})
	}
	return items, total, nil
}

// ========== 评价 ==========

func (s *discussionService) CreateEvaluation(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CreateEvaluationReq) (string, error) {
	course, err := ensureCourseStudent(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID)
	if err != nil {
		return "", err
	}
	if course.Status != enum.CourseStatusEnded {
		return "", errcode.ErrInvalidParams.WithMessage("仅已结束的课程允许评价")
	}

	// 检查是否已评价
	existing, _ := s.evaluationRepo.GetByStudentAndCourse(ctx, sc.UserID, courseID)
	if existing != nil {
		return "", errcode.ErrInvalidParams.WithMessage("已提交过评价")
	}
	evaluation := &entity.CourseEvaluation{
		CourseID: courseID, StudentID: sc.UserID,
		Rating: req.Rating, Comment: req.Comment,
	}
	if err := s.evaluationRepo.Create(ctx, evaluation); err != nil {
		return "", err
	}
	return strconv.FormatInt(evaluation.ID, 10), nil
}

func (s *discussionService) UpdateEvaluation(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateEvaluationReq) error {
	evaluation, err := s.evaluationRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("评价不存在")
	}
	if evaluation.StudentID != sc.UserID {
		return errcode.ErrForbidden
	}
	fields := make(map[string]interface{})
	if req.Rating != nil {
		fields["rating"] = *req.Rating
	}
	if req.Comment != nil {
		fields["comment"] = *req.Comment
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.evaluationRepo.UpdateFields(ctx, id, fields)
}

func (s *discussionService) ListEvaluations(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.EvaluationListReq) ([]*dto.EvaluationItem, *dto.EvaluationSummary, int64, error) {
	if _, err := ensureCourseMember(ctx, sc, s.courseRepo, s.enrollmentRepo, courseID); err != nil {
		return nil, nil, 0, err
	}

	evaluations, total, err := s.evaluationRepo.List(ctx, courseID, req.Page, req.PageSize)
	if err != nil {
		return nil, nil, 0, err
	}

	items := make([]*dto.EvaluationItem, 0, len(evaluations))
	for _, e := range evaluations {
		name := s.userNameQuerier.GetUserName(ctx, e.StudentID)
		items = append(items, &dto.EvaluationItem{
			ID:          strconv.FormatInt(e.ID, 10),
			StudentID:   strconv.FormatInt(e.StudentID, 10),
			StudentName: name, Rating: e.Rating, Comment: e.Comment,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}

	avgRating, _ := s.evaluationRepo.GetAvgRating(ctx, courseID)
	dist, _ := s.evaluationRepo.GetDistribution(ctx, courseID)
	summary := &dto.EvaluationSummary{
		AvgRating: avgRating, TotalCount: int(total), Distribution: dist,
	}
	return items, summary, total, nil
}

// ========== 成绩配置 ==========

func (s *discussionService) SetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.GradeConfigReq) error {
	if err := s.verifyCourseTeacher(ctx, sc, courseID); err != nil {
		return err
	}
	var totalWeight float64
	for _, item := range req.Items {
		totalWeight += item.Weight
	}
	if math.Abs(totalWeight-100) > 0.0001 {
		return errcode.ErrInvalidParams.WithMessage("权重总和必须为100%")
	}
	configJSON, err := json.Marshal(req)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("配置格式错误")
	}
	config := &entity.CourseGradeConfig{
		CourseID: courseID, Config: string(configJSON),
	}
	return s.gradeConfigRepo.Upsert(ctx, config)
}

func (s *discussionService) GetGradeConfig(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (*dto.GradeConfigResp, error) {
	config, err := s.gradeConfigRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		// 未配置，返回空
		return &dto.GradeConfigResp{Items: []dto.GradeConfigItem{}}, nil
	}
	var resp dto.GradeConfigResp
	if err := json.Unmarshal([]byte(config.Config), &resp); err != nil {
		return &dto.GradeConfigResp{Items: []dto.GradeConfigItem{}}, nil
	}
	var totalWeight float64
	for _, item := range resp.Items {
		totalWeight += item.Weight
	}
	resp.TotalWeight = totalWeight
	return &resp, nil
}

// verifyCourseTeacher 校验当前用户是否为课程负责教师
func (s *discussionService) verifyCourseTeacher(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	_, err := ensureCourseTeacher(ctx, sc, s.courseRepo, courseID)
	return err
}
