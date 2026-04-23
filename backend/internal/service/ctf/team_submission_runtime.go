// team_submission_runtime.go
// 模块05 — CTF竞赛：解题赛提交验证辅助。
// 负责动态 Flag 计算、链上提交运行时规格构建和题目环境元数据读取，
// 避免把提交验证细节堆积在 team_service 主文件中。

package ctf

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

// evaluateCompetitionSubmission 按题目类型执行真实提交判定。
func (s *teamService) evaluateCompetitionSubmission(
	ctx context.Context,
	competitionID int64,
	team *entity.Team,
	challenge *entity.Challenge,
	req *dto.SubmitCompetitionChallengeReq,
) (bool, *dto.VerificationAssertionResults, *string, error) {
	if challenge == nil {
		return false, nil, stringPtr("题目不存在"), nil
	}
	switch req.SubmissionType {
	case enum.SubmissionTypeStaticFlag:
		if challenge.FlagType != enum.FlagTypeStatic {
			return false, nil, stringPtr("当前题目不接受静态 Flag 提交"), nil
		}
		return evaluateStaticFlagSubmission(challenge, req), nil, buildStaticFlagFailure(challenge, req), nil
	case enum.SubmissionTypeDynamicFlag:
		if challenge.FlagType != enum.FlagTypeDynamic {
			return false, nil, stringPtr("当前题目不接受动态 Flag 提交"), nil
		}
		return evaluateDynamicFlagSubmission(team.ID, challenge, req), nil, buildDynamicFlagFailure(team.ID, challenge, req), nil
	case enum.SubmissionTypeAttackTx:
		return s.evaluateOnChainSubmission(ctx, competitionID, team, challenge, req)
	default:
		return false, nil, stringPtr("提交类型无效"), nil
	}
}

// evaluateStaticFlagSubmission 执行静态 Flag 字符串比对。
func evaluateStaticFlagSubmission(challenge *entity.Challenge, req *dto.SubmitCompetitionChallengeReq) bool {
	if challenge == nil || challenge.StaticFlag == nil {
		return false
	}
	expected := decryptSensitiveTextValue(challenge.StaticFlag)
	return strings.TrimSpace(req.Content) == strings.TrimSpace(expected)
}

// buildStaticFlagFailure 构建静态 Flag 错误提示。
func buildStaticFlagFailure(challenge *entity.Challenge, req *dto.SubmitCompetitionChallengeReq) *string {
	if evaluateStaticFlagSubmission(challenge, req) {
		return nil
	}
	return stringPtr("Flag不正确")
}

// evaluateDynamicFlagSubmission 基于 HMAC-SHA256(secret, team_id) 计算动态 Flag。
func evaluateDynamicFlagSubmission(teamID int64, challenge *entity.Challenge, req *dto.SubmitCompetitionChallengeReq) bool {
	if challenge == nil || challenge.DynamicFlagSecret == nil {
		return false
	}
	expected := buildDynamicFlag(teamID, decryptSensitiveTextValue(challenge.DynamicFlagSecret))
	return strings.EqualFold(strings.TrimSpace(req.Content), expected)
}

// buildDynamicFlagFailure 构建动态 Flag 错误提示。
func buildDynamicFlagFailure(teamID int64, challenge *entity.Challenge, req *dto.SubmitCompetitionChallengeReq) *string {
	if evaluateDynamicFlagSubmission(teamID, challenge, req) {
		return nil
	}
	return stringPtr("Flag不正确")
}

// buildDynamicFlag 计算题目文档要求的动态 Flag 文本。
func buildDynamicFlag(teamID int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(int64String(teamID)))
	return "flag{" + hex.EncodeToString(mac.Sum(nil)) + "}"
}

// evaluateOnChainSubmission 在选手题目环境执行攻击交易并校验断言。
func (s *teamService) evaluateOnChainSubmission(
	ctx context.Context,
	competitionID int64,
	team *entity.Team,
	challenge *entity.Challenge,
	req *dto.SubmitCompetitionChallengeReq,
) (bool, *dto.VerificationAssertionResults, *string, error) {
	if challenge == nil || challenge.FlagType != enum.FlagTypeOnChain {
		return false, nil, stringPtr("当前题目不是链上状态验证题"), nil
	}
	if s.submissionExecutor == nil {
		return false, nil, stringPtr("链上验证运行时未配置"), errcode.ErrInternal.WithMessage("CTF链上验证运行时未配置")
	}
	environment, err := s.environmentRepo.GetActiveByTeamAndChallenge(ctx, competitionID, team.ID, challenge.ID)
	if err != nil || environment == nil || environment.Status != enum.ChallengeEnvStatusRunning || environment.ChainRPCURL == nil {
		return false, nil, nil, errcode.ErrSubmissionInvalid.WithMessage("题目环境未启动")
	}
	runtimeState := decodeChallengeEnvironmentRuntimeState(environment)
	if len(runtimeState.Contracts) == 0 {
		return false, nil, nil, errcode.ErrSubmissionInvalid.WithMessage("题目环境未启动")
	}
	assertions, err := s.assertionRepo.ListByChallengeID(ctx, challenge.ID)
	if err != nil {
		return false, nil, nil, err
	}
	result, err := s.submissionExecutor.ExecuteChallengeSubmission(ctx, &ChallengeSubmissionSpec{
		Namespace:      environment.Namespace,
		ChainRPCURL:    *environment.ChainRPCURL,
		SubmissionData: req.Content,
		Contracts:      runtimeState.Contracts,
		Accounts:       loadChallengeAccounts(challenge),
		Assertions:     buildRuntimeAssertions(assertions),
	})
	if err != nil {
		message := err.Error()
		return false, nil, &message, nil
	}
	if result == nil {
		return false, nil, stringPtr("链上验证未返回结果"), nil
	}
	return result.IsCorrect, result.AssertionResults, result.ErrorMessage, nil
}
