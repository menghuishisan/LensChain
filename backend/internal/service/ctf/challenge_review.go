package ctf

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// SubmitReview 提交审核。
func (s *challengeService) SubmitReview(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmitChallengeReviewResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return nil, err
	}
	if challenge.Status != enum.ChallengeStatusDraft && challenge.Status != enum.ChallengeStatusRejected {
		return nil, errcode.ErrChallengeStatusInvalid
	}
	if challenge.FlagType == enum.FlagTypeOnChain {
		hasPassed, err := s.challengeRepo.HasPassedVerification(ctx, challenge.ID)
		if err != nil {
			return nil, err
		}
		if !hasPassed {
			return nil, errcode.ErrChallengeVerificationAbsent
		}
	}
	if err := s.challengeRepo.UpdateStatus(ctx, challenge.ID, enum.ChallengeStatusPending); err != nil {
		return nil, err
	}
	return &dto.SubmitChallengeReviewResp{
		ID:         int64String(challenge.ID),
		Status:     enum.ChallengeStatusPending,
		StatusText: enum.GetChallengeStatusText(enum.ChallengeStatusPending),
	}, nil
}

// ListPendingReviews 查询待审核题目列表。
func (s *challengeService) ListPendingReviews(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ChallengeListReq) (*dto.ChallengeReviewListResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	challenges, _, err := s.challengeRepo.ListPendingReview(ctx, &ctfrepo.ChallengeListParams{
		Category:   req.Category,
		Difficulty: req.Difficulty,
		Keyword:    req.Keyword,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]dto.ChallengeReviewListItem, 0, len(challenges))
	for _, challenge := range challenges {
		items = append(items, dto.ChallengeReviewListItem{
			ID:             int64String(challenge.ID),
			Title:          challenge.Title,
			Category:       challenge.Category,
			CategoryText:   enum.GetChallengeCategoryText(challenge.Category),
			Difficulty:     challenge.Difficulty,
			DifficultyText: enum.GetCtfDifficultyText(challenge.Difficulty),
			AuthorName:     s.userQuerier.GetUserName(ctx, challenge.AuthorID),
			SchoolName:     s.schoolQuerier.GetSchoolName(ctx, challenge.SchoolID),
			SubmittedAt:    timeString(challenge.UpdatedAt),
		})
	}
	return &dto.ChallengeReviewListResp{List: items}, nil
}

// Review 审核题目。
func (s *challengeService) Review(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.ReviewChallengeReq) (*dto.ReviewChallengeActionResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	if !enum.IsValidReviewAction(req.Action) {
		return nil, errcode.ErrInvalidParams.WithMessage("审核动作无效")
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if challenge.Status != enum.ChallengeStatusPending {
		return nil, errcode.ErrChallengeStatusInvalid
	}
	review := &entity.ChallengeReview{
		ID:          snowflake.Generate(),
		ChallengeID: challengeID,
		ReviewerID:  sc.UserID,
		Action:      req.Action,
		Comment:     req.Comment,
		CreatedAt:   time.Now(),
	}
	nextStatus, isPublic := resolveReviewOutcome(req.Action)
	if req.Action == enum.ReviewActionApprove {
		if challenge.FlagType == enum.FlagTypeOnChain {
			hasPassed, verifyErr := s.challengeRepo.HasPassedVerification(ctx, challenge.ID)
			if verifyErr != nil {
				return nil, verifyErr
			}
			if !hasPassed {
				return nil, errcode.ErrChallengeVerificationAbsent
			}
		}
		nextStatus = int16(enum.ChallengeStatusApproved)
	}
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txReviewRepo := ctfrepo.NewChallengeReviewRepository(tx)
		txChallengeRepo := ctfrepo.NewChallengeRepository(tx)
		if err := txReviewRepo.Create(ctx, review); err != nil {
			return err
		}
		return txChallengeRepo.UpdateFields(ctx, challengeID, map[string]interface{}{
			"status":    nextStatus,
			"is_public": isPublic,
		})
	})
	if err != nil {
		return nil, err
	}
	return buildReviewActionResp(review, nextStatus), nil
}

// resolveReviewOutcome 根据审核动作计算题目下一状态和是否进入公共题库。
func resolveReviewOutcome(action int16) (int16, bool) {
	if action == enum.ReviewActionApprove {
		return enum.ChallengeStatusApproved, true
	}
	return enum.ChallengeStatusRejected, false
}

// ListReviews 查询题目审核记录。
func (s *challengeService) ListReviews(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.ChallengeReviewHistoryResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeReadable(sc, challenge); err != nil {
		return nil, err
	}
	reviews, _, err := s.reviewRepo.ListByChallengeID(ctx, challengeID, 1, 100)
	if err != nil {
		return nil, err
	}
	items := make([]dto.ChallengeReviewResp, 0, len(reviews))
	for _, review := range reviews {
		reviewerName := s.userQuerier.GetUserName(ctx, review.ReviewerID)
		items = append(items, dto.ChallengeReviewResp{
			ID:           int64String(review.ID),
			ChallengeID:  int64String(review.ChallengeID),
			ReviewerID:   int64String(review.ReviewerID),
			ReviewerName: &reviewerName,
			Action:       review.Action,
			ActionText:   enum.GetReviewActionText(review.Action),
			Comment:      review.Comment,
			CreatedAt:    timeString(review.CreatedAt),
		})
	}
	return &dto.ChallengeReviewHistoryResp{List: items}, nil
}

// buildReviewActionResp 构建题目审核动作响应。
func buildReviewActionResp(review *entity.ChallengeReview, status int16) *dto.ReviewChallengeActionResp {
	return &dto.ReviewChallengeActionResp{
		ChallengeID: int64String(review.ChallengeID),
		Status:      status,
		StatusText:  enum.GetChallengeStatusText(status),
		ReviewID:    int64String(review.ID),
	}
}
