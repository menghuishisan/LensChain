package ctf

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// StartVerification 发起题目预验证。
func (s *challengeService) StartVerification(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.VerifyChallengeReq) (*dto.VerifyChallengeResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeManageable(sc, challenge); err != nil {
		return nil, err
	}
	if running, err := s.verificationRepo.GetRunningByChallengeID(ctx, challengeID); err == nil && running != nil {
		return nil, errcode.ErrChallengeVerificationRunning
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if challenge.FlagType != enum.FlagTypeOnChain {
		return nil, errcode.ErrInvalidParams.WithMessage("链上验证题目才需要预验证")
	}
	if strings.TrimSpace(req.PocContent) == "" {
		return nil, errcode.ErrInvalidParams.WithMessage("PoC内容不能为空")
	}

	contracts, err := s.contractRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if len(contracts) == 0 {
		return nil, errcode.ErrChallengeContractRequired
	}
	assertions, err := s.assertionRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if len(assertions) == 0 {
		return nil, errcode.ErrChallengeAssertionRequired
	}
	verificationID := snowflake.Generate()
	startedAt := time.Now().UTC()
	environmentID := buildVerificationEnvironmentID(verificationID)
	verification := &entity.ChallengeVerification{
		ID:            verificationID,
		ChallengeID:   challengeID,
		InitiatedBy:   sc.UserID,
		Status:        enum.VerificationStatusRunning,
		StepResults:   mustJSON(buildVerificationInitialSteps()),
		PocLanguage:   req.PocLanguage,
		EnvironmentID: &environmentID,
		StartedAt:     startedAt,
		CreatedAt:     startedAt,
	}
	if verification.PocContent, err = encryptSensitiveText(stringPtr(req.PocContent)); err != nil {
		return nil, err
	}
	if err := s.verificationRepo.Create(ctx, verification); err != nil {
		return nil, err
	}
	runtimeContracts := buildChallengeRuntimeContractSpecs(contracts)
	runtimeAssertions := buildRuntimeAssertions(assertions)
	flowResult, execErr := executeChallengeVerificationFlow(
		ctx,
		challenge,
		environmentID,
		derefString(req.PocLanguage),
		req.PocContent,
		runtimeContracts,
		runtimeAssertions,
		s.envProvisioner,
		s.submissionExec,
	)
	if s.provisioner != nil {
		_ = s.provisioner.DeleteNamespace(ctx, environmentID)
	}
	if execErr != nil {
		return nil, execErr
	}
	if flowResult != nil {
		updateFields := map[string]interface{}{
			"step_results":  mustJSON(flowResult.StepResults),
			"completed_at":  flowResult.CompletedAt,
			"error_message": flowResult.ErrorMessage,
		}
		if err := s.verificationRepo.UpdateFields(ctx, verification.ID, updateFields); err != nil {
			return nil, err
		}
		if err := s.verificationRepo.Complete(ctx, verification.ID, flowResult.Status, flowResult.ErrorMessage); err != nil {
			return nil, err
		}
	}
	return buildVerificationStartResp(verification), nil
}

// GetVerification 获取预验证详情。
func (s *challengeService) GetVerification(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.VerificationDetailResp, error) {
	verification, err := s.verificationRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrChallengeVerificationAbsent
		}
		return nil, err
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, verification.ChallengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeReadable(sc, challenge); err != nil {
		return nil, err
	}
	return buildVerificationDetail(verification)
}

// ListVerifications 查询题目预验证记录。
func (s *challengeService) ListVerifications(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.VerificationListResp, error) {
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureChallengeReadable(sc, challenge); err != nil {
		return nil, err
	}
	verifications, _, err := s.verificationRepo.ListByChallengeID(ctx, challengeID, 1, 100)
	if err != nil {
		return nil, err
	}
	items := make([]dto.VerificationListItem, 0, len(verifications))
	for _, verification := range verifications {
		items = append(items, dto.VerificationListItem{
			ID:           int64String(verification.ID),
			Status:       verification.Status,
			StatusText:   enum.GetVerificationStatusText(verification.Status),
			PocLanguage:  verification.PocLanguage,
			StartedAt:    timeString(verification.StartedAt),
			CompletedAt:  optionalTimeString(verification.CompletedAt),
			ErrorMessage: verification.ErrorMessage,
		})
	}
	return &dto.VerificationListResp{List: items}, nil
}

// buildVerificationEnvironmentID 构建题目预验证环境标识。
func buildVerificationEnvironmentID(verificationID int64) string {
	return "ctf-verify-" + int64String(verificationID)
}

// buildVerificationInitialSteps 构建预验证初始步骤结果。
func buildVerificationInitialSteps() []dto.VerificationStepResult {
	return []dto.VerificationStepResult{
		{Step: 1, Name: "部署测试环境", Status: "running", Detail: "正在准备链上验证环境"},
		{Step: 2, Name: "提交官方PoC", Status: "pending", Detail: "等待接收并执行教师提供的 PoC"},
		{Step: 3, Name: "正向验证", Status: "pending", Detail: "等待执行 PoC 后检查断言结果"},
		{Step: 4, Name: "反向验证", Status: "pending", Detail: "等待在不执行 PoC 的情况下检查断言"},
		{Step: 5, Name: "验证完成", Status: "pending", Detail: "等待汇总结论"},
	}
}

// buildVerificationStartResp 构建“发起预验证”接口响应。
func buildVerificationStartResp(verification *entity.ChallengeVerification) *dto.VerifyChallengeResp {
	return &dto.VerifyChallengeResp{
		VerificationID: int64String(verification.ID),
		ChallengeID:    int64String(verification.ChallengeID),
		Status:         enum.VerificationStatusRunning,
		StatusText:     enum.GetVerificationStatusText(enum.VerificationStatusRunning),
		StartedAt:      timeString(verification.StartedAt),
	}
}

// buildVerificationTerminalStep 构建预验证最终结论步骤。
func buildVerificationTerminalStep(passed bool, detail string) dto.VerificationStepResult {
	step := dto.VerificationStepResult{
		Step:   5,
		Name:   "验证通过",
		Detail: detail,
	}
	if passed {
		step.Status = "passed"
		return step
	}
	step.Name = "验证失败"
	step.Status = "failed"
	return step
}

// buildVerificationTimeoutSteps 构建超时失败场景下的步骤结果，确保详情接口仍能解释结束原因。
func buildVerificationTimeoutSteps(verification *entity.ChallengeVerification) []dto.VerificationStepResult {
	if verification != nil && len(verification.StepResults) > 0 {
		existing := []dto.VerificationStepResult{}
		if err := decodeJSON(verification.StepResults, &existing); err == nil && len(existing) > 0 {
			last := existing[len(existing)-1]
			if last.Step == 5 && last.Status == "failed" {
				return existing
			}
			existing = append(existing, buildVerificationTerminalStep(false, "预验证执行超时，系统已自动终止临时环境"))
			return existing
		}
	}
	return []dto.VerificationStepResult{
		{Step: 1, Name: "部署测试环境", Status: "failed", Detail: "预验证执行超时，系统已自动终止临时环境"},
		{Step: 5, Name: "验证失败", Status: "failed", Detail: "预验证执行超时，系统已自动终止临时环境"},
	}
}

// buildVerificationDetail 构建预验证详情响应。
func buildVerificationDetail(verification *entity.ChallengeVerification) (*dto.VerificationDetailResp, error) {
	stepResults := []dto.VerificationStepResult{}
	if err := decodeJSON(verification.StepResults, &stepResults); err != nil {
		return nil, err
	}
	pocContent := decryptSensitiveText(verification.PocContent)
	return &dto.VerificationDetailResp{
		ID:            int64String(verification.ID),
		ChallengeID:   int64String(verification.ChallengeID),
		Status:        verification.Status,
		StatusText:    enum.GetVerificationStatusText(verification.Status),
		StepResults:   stepResults,
		PocContent:    pocContent,
		PocLanguage:   verification.PocLanguage,
		EnvironmentID: verification.EnvironmentID,
		ErrorMessage:  verification.ErrorMessage,
		StartedAt:     timeString(verification.StartedAt),
		CompletedAt:   optionalTimeString(verification.CompletedAt),
		CreatedAt:     timeString(verification.CreatedAt),
	}, nil
}
