package ctf

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

type challengeVerificationFlowResult struct {
	Status       int16
	StepResults  []dto.VerificationStepResult
	ErrorMessage *string
	CompletedAt  time.Time
}

func executeChallengeVerificationFlow(
	ctx context.Context,
	challenge *entity.Challenge,
	environmentID string,
	pocLanguage string,
	pocContent string,
	contracts []ChallengeRuntimeContractSpec,
	assertions []ChallengeAssertionSpec,
	provisioner ChallengeEnvironmentProvisioner,
	executor ChallengeSubmissionExecutor,
) (*challengeVerificationFlowResult, error) {
	steps := buildVerificationInitialSteps()
	fail := func(stepIndex int, detail string) (*challengeVerificationFlowResult, error) {
		if stepIndex >= 0 && stepIndex < len(steps) {
			steps[stepIndex].Status = "failed"
			steps[stepIndex].Detail = detail
		}
		steps[4] = buildVerificationTerminalStep(false, detail)
		return &challengeVerificationFlowResult{
			Status:       enum.VerificationStatusFailed,
			StepResults:  steps,
			ErrorMessage: stringPtr(detail),
			CompletedAt:  time.Now().UTC(),
		}, nil
	}
	if challenge == nil {
		return fail(0, "题目不存在")
	}
	if provisioner == nil {
		return fail(0, "预验证运行时未配置")
	}
	if executor == nil {
		return fail(1, "预验证提交执行器未配置")
	}

	spec := buildChallengeEnvironmentSpec(challenge, 0, 0, environmentID, convertRuntimeContractsToEntities(contracts))
	spec.Namespace = environmentID
	result, err := provisioner.ProvisionChallengeEnvironment(ctx, spec)
	if err != nil {
		return fail(0, "部署测试环境失败："+err.Error())
	}
	steps[0].Status = "passed"
	steps[0].Detail = "测试环境部署完成"
	if result == nil || result.ChainRPCURL == nil || strings.TrimSpace(*result.ChainRPCURL) == "" {
		return fail(0, "测试环境未返回可用链地址")
	}
	if len(result.Contracts) == 0 {
		return fail(0, "测试环境未返回部署后的合约绑定")
	}

	positiveResult, err := executor.ExecuteChallengeSubmission(ctx, &ChallengeSubmissionSpec{
		Namespace:      environmentID,
		ChainRPCURL:    *result.ChainRPCURL,
		SubmissionData: pocContent,
		Contracts:      result.Contracts,
		Accounts:       loadChallengeAccounts(challenge),
		Assertions:     assertions,
	})
	if err != nil {
		return fail(1, "提交官方PoC失败："+err.Error())
	}
	if positiveResult == nil {
		return fail(1, "提交官方PoC未返回执行结果")
	}
	steps[1].Status = "passed"
	steps[1].Detail = fmt.Sprintf("官方PoC已执行，语言=%s", strings.TrimSpace(pocLanguage))
	if !positiveResult.IsCorrect {
		steps[2].Assertions = verificationAssertionDetails(positiveResult.AssertionResults)
		detail := buildVerificationAssertionFailureDetail("正向验证失败", steps[2].Assertions)
		if positiveResult.ErrorMessage != nil && strings.TrimSpace(*positiveResult.ErrorMessage) != "" {
			detail = "正向验证失败：" + strings.TrimSpace(*positiveResult.ErrorMessage)
		}
		steps[2].Status = "failed"
		steps[2].Detail = detail
		steps[4] = buildVerificationTerminalStep(false, detail)
		return &challengeVerificationFlowResult{
			Status:       enum.VerificationStatusFailed,
			StepResults:  steps,
			ErrorMessage: stringPtr(detail),
			CompletedAt:  time.Now().UTC(),
		}, nil
	}
	steps[2].Status = "passed"
	steps[2].Detail = "正向验证通过，漏洞可以被官方PoC稳定触发"
	steps[2].Assertions = verificationAssertionDetails(positiveResult.AssertionResults)

	reverseNamespace := environmentID + "-reverse"
	reverseSpec := buildChallengeEnvironmentSpec(challenge, 0, 0, reverseNamespace, convertRuntimeContractsToEntities(contracts))
	reverseSpec.Namespace = reverseNamespace
	reverseEnv, err := provisioner.ProvisionChallengeEnvironment(ctx, reverseSpec)
	if err != nil {
		return fail(3, "反向验证环境部署失败："+err.Error())
	}
	if reverseEnv == nil || reverseEnv.ChainRPCURL == nil || strings.TrimSpace(*reverseEnv.ChainRPCURL) == "" {
		return fail(3, "反向验证环境未返回可用链地址")
	}
	if len(reverseEnv.Contracts) == 0 {
		return fail(3, "反向验证环境未返回部署后的合约绑定")
	}

	reverseResult, err := executor.ExecuteChallengeSubmission(ctx, &ChallengeSubmissionSpec{
		Namespace:      reverseNamespace,
		ChainRPCURL:    *reverseEnv.ChainRPCURL,
		SubmissionData: "",
		Contracts:      reverseEnv.Contracts,
		Accounts:       loadChallengeAccounts(challenge),
		Assertions:     assertions,
	})
	if err != nil {
		return fail(3, "反向验证失败："+err.Error())
	}
	if reverseResult == nil {
		return fail(3, "反向验证未返回执行结果")
	}
	if reverseResult.IsCorrect {
		steps[3].Assertions = verificationAssertionDetails(reverseResult.AssertionResults)
		detail := "反向验证失败：未执行攻击时断言仍然通过"
		steps[3].Status = "failed"
		steps[3].Detail = detail
		steps[4] = buildVerificationTerminalStep(false, detail)
		return &challengeVerificationFlowResult{
			Status:       enum.VerificationStatusFailed,
			StepResults:  steps,
			ErrorMessage: stringPtr(detail),
			CompletedAt:  time.Now().UTC(),
		}, nil
	}
	steps[3].Status = "passed"
	steps[3].Detail = "反向验证通过，未执行攻击时断言不会误报"
	steps[3].Assertions = verificationAssertionDetails(reverseResult.AssertionResults)
	steps[4] = buildVerificationTerminalStep(true, "预验证通过")
	return &challengeVerificationFlowResult{
		Status:      enum.VerificationStatusPassed,
		StepResults: steps,
		CompletedAt: time.Now().UTC(),
	}, nil
}

func verificationAssertionDetails(results *dto.VerificationAssertionResults) []dto.VerificationAssertionResult {
	if results == nil || len(results.Results) == 0 {
		return nil
	}
	items := make([]dto.VerificationAssertionResult, len(results.Results))
	copy(items, results.Results)
	return items
}

func buildVerificationAssertionFailureDetail(prefix string, assertions []dto.VerificationAssertionResult) string {
	for _, assertion := range assertions {
		if assertion.Passed {
			continue
		}
		return fmt.Sprintf("%s：%s 实际值=%s，期望值=%s", prefix, assertion.Target, assertion.Actual, assertion.Expected)
	}
	return prefix
}

func convertRuntimeContractsToEntities(contracts []ChallengeRuntimeContractSpec) []*entity.ChallengeContract {
	items := make([]*entity.ChallengeContract, 0, len(contracts))
	for _, contract := range contracts {
		items = append(items, &entity.ChallengeContract{
			ChallengeID:     contract.ChallengeID,
			Name:            contract.ContractName,
			ABI:             datatypes.JSON([]byte(contract.ABIJSON)),
			Bytecode:        contract.Bytecode,
			ConstructorArgs: mustJSON(contract.ConstructorArgs),
			DeployOrder:     contract.DeployOrder,
		})
	}
	return items
}
