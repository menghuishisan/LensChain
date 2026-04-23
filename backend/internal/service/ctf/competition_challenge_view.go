// competition_challenge_view.go
// 模块05 — CTF竞赛：竞赛题目列表视图构建。
// 负责根据访问者身份构建“参赛选手脱敏视图”和“竞赛创建者完整视图”，
// 避免在竞赛主流程文件中继续堆积题目展示细节。

package ctf

import (
	"context"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// buildCompetitionChallengeSummary 构建竞赛题目概要信息。
// 参赛选手只能看到脱敏字段；竞赛创建者可额外看到完整题目配置、合约、断言和最新预验证摘要。
func (s *competitionService) buildCompetitionChallengeSummary(ctx context.Context, challenge *entity.Challenge, fullView bool) (dto.CompetitionChallengeSummary, error) {
	attachmentURLs := make([]string, 0)
	if err := decodeJSON(challenge.AttachmentURLs, &attachmentURLs); err != nil {
		attachmentURLs = []string{}
	}
	summary := dto.CompetitionChallengeSummary{
		ID:             int64String(challenge.ID),
		Title:          challenge.Title,
		Description:    challenge.Description,
		Category:       challenge.Category,
		CategoryText:   enum.GetChallengeCategoryText(challenge.Category),
		Difficulty:     challenge.Difficulty,
		DifficultyText: enum.GetCtfDifficultyText(challenge.Difficulty),
		FlagType:       challenge.FlagType,
		FlagTypeText:   enum.GetFlagTypeText(challenge.FlagType),
		AttachmentURLs: attachmentURLs,
	}
	if !fullView {
		return summary, nil
	}

	if err := s.enrichCompetitionChallengeSummary(ctx, challenge, &summary); err != nil {
		return dto.CompetitionChallengeSummary{}, err
	}
	return summary, nil
}

// enrichCompetitionChallengeSummary 为竞赛创建者补齐题目的完整配置视图。
func (s *competitionService) enrichCompetitionChallengeSummary(ctx context.Context, challenge *entity.Challenge, summary *dto.CompetitionChallengeSummary) error {
	if summary == nil {
		return nil
	}

	var chainConfig *dto.ChallengeChainConfig
	if err := decodeJSON(challenge.ChainConfig, &chainConfig); err != nil {
		return err
	}
	var environmentConfig *dto.ChallengeEnvironmentConfig
	if err := decodeJSON(challenge.EnvironmentConfig, &environmentConfig); err != nil {
		return err
	}

	summary.ChainConfig = chainConfig
	summary.SourcePath = challenge.SourcePath
	if challenge.SourcePath != nil {
		summary.SourcePathText = stringPtr(enum.GetChallengeSourcePathText(*challenge.SourcePath))
	}
	summary.SwcID = challenge.SwcID
	if challenge.TemplateID != nil {
		value := int64String(*challenge.TemplateID)
		summary.TemplateID = &value
	}
	summary.EnvironmentConfig = environmentConfig

	status := challenge.Status
	statusText := enum.GetChallengeStatusText(challenge.Status)
	isPublic := challenge.IsPublic
	usageCount := challenge.UsageCount
	summary.Status = &status
	summary.StatusText = &statusText
	summary.IsPublic = &isPublic
	summary.UsageCount = &usageCount

	contracts, err := s.contractRepo.ListByChallengeID(ctx, challenge.ID)
	if err != nil {
		return err
	}
	for _, contract := range contracts {
		summary.Contracts = append(summary.Contracts, buildChallengeContractItem(contract, true))
	}

	assertions, err := s.assertionRepo.ListByChallengeID(ctx, challenge.ID)
	if err != nil {
		return err
	}
	for _, assertion := range assertions {
		summary.Assertions = append(summary.Assertions, buildChallengeAssertionItem(assertion))
	}

	verification, err := s.verificationRepo.GetLatestByChallengeID(ctx, challenge.ID)
	if err == nil && verification != nil {
		summary.LatestVerification = &dto.VerificationSummary{
			ID:          int64String(verification.ID),
			Status:      verification.Status,
			StatusText:  enum.GetVerificationStatusText(verification.Status),
			CompletedAt: optionalTimeString(verification.CompletedAt),
		}
	}
	return nil
}
