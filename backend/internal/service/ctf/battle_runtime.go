// battle_runtime.go
// 模块05 — CTF竞赛：攻防对抗赛运行时编排辅助。
// 负责把攻防赛业务数据转换为运行时部署规格，并调用运行时适配器创建真实链资源。

package ctf

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

const adAttackLockTTL = 45 * time.Second

// createAdGroupRuntime 调用运行时适配器创建攻防赛分组链资源。
func (s *battleService) createAdGroupRuntime(
	ctx context.Context,
	competition *entity.Competition,
	groupNamespace string,
	groupID int64,
	teams []*entity.Team,
	challengeItems []*entity.CompetitionChallenge,
	challengeMap map[int64]*entity.Challenge,
	contractMap map[int64][]*entity.ChallengeContract,
) (*ADRuntimeGroupResult, error) {
	if s.runtimeProvisioner == nil {
		return nil, errcode.ErrInternal.WithMessage("攻防赛运行时编排器未配置")
	}
	cfg, err := s.loadADConfig(competition)
	if err != nil {
		return nil, err
	}
	spec, err := buildADRuntimeGroupSpec(competition, cfg, groupID, groupNamespace, teams, challengeItems, challengeMap, contractMap)
	if err != nil {
		return nil, err
	}
	return s.runtimeProvisioner.CreateADGroupRuntime(ctx, spec)
}

// buildADRuntimeGroupSpec 将攻防赛业务数据转换为运行时编排输入。
func buildADRuntimeGroupSpec(
	competition *entity.Competition,
	cfg *dto.ADCompetitionConfig,
	groupID int64,
	groupNamespace string,
	teams []*entity.Team,
	challengeItems []*entity.CompetitionChallenge,
	challengeMap map[int64]*entity.Challenge,
	contractMap map[int64][]*entity.ChallengeContract,
) (*ADRuntimeGroupSpec, error) {
	spec := &ADRuntimeGroupSpec{
		CompetitionID:    competition.ID,
		GroupID:          groupID,
		Namespace:        groupNamespace,
		JudgeChainImage:  cfg.JudgeChainImage,
		TeamChainImage:   cfg.TeamChainImage,
		RuntimeToolImage: "ctf-blockchain:latest",
		Teams:            make([]ADRuntimeTeamSpec, 0, len(teams)),
	}
	for _, team := range teams {
		teamSpec, err := buildADRuntimeTeamSpec(team, challengeItems, challengeMap, contractMap)
		if err != nil {
			return nil, err
		}
		spec.Teams = append(spec.Teams, *teamSpec)
	}
	return spec, nil
}

// buildADRuntimeTeamSpec 构建单支队伍的链部署输入。
func buildADRuntimeTeamSpec(
	team *entity.Team,
	challengeItems []*entity.CompetitionChallenge,
	challengeMap map[int64]*entity.Challenge,
	contractMap map[int64][]*entity.ChallengeContract,
) (*ADRuntimeTeamSpec, error) {
	spec := &ADRuntimeTeamSpec{
		TeamID:    team.ID,
		TeamName:  team.Name,
		Contracts: make([]ADRuntimeContractSpec, 0, len(challengeItems)),
	}
	for _, item := range challengeItems {
		challengeContracts := contractMap[item.ChallengeID]
		if len(challengeContracts) == 0 {
			return nil, errcode.ErrChallengeContractRequired.WithMessage("攻防赛题目缺少可部署合约")
		}
		for _, contract := range challengeContracts {
			constructorArgs, err := decodeChallengeConstructorArgs(contract.ConstructorArgs)
			if err != nil {
				return nil, errcode.ErrChallengeContractInvalid.WithMessage("题目合约构造参数格式不正确")
			}
			challengeTitle := contract.Name
			if challengeMap[item.ChallengeID] != nil {
				challengeTitle = challengeMap[item.ChallengeID].Title
			}
			spec.Contracts = append(spec.Contracts, ADRuntimeContractSpec{
				ChallengeID:     item.ChallengeID,
				ChallengeTitle:  challengeTitle,
				ContractName:    contract.Name,
				ABIJSON:         string(contract.ABI),
				Bytecode:        contract.Bytecode,
				ConstructorArgs: constructorArgs,
				DeployOrder:     contract.DeployOrder,
			})
		}
	}
	sort.SliceStable(spec.Contracts, func(i, j int) bool {
		left := spec.Contracts[i]
		right := spec.Contracts[j]
		if left.ChallengeID == right.ChallengeID {
			return left.DeployOrder < right.DeployOrder
		}
		return left.ChallengeID < right.ChallengeID
	})
	return spec, nil
}

// buildChallengeContractMap 将题目合约列表按题目ID分组，供运行时批量部署使用。
func buildChallengeContractMap(contracts []*entity.ChallengeContract) map[int64][]*entity.ChallengeContract {
	result := make(map[int64][]*entity.ChallengeContract)
	for _, contract := range contracts {
		if contract == nil {
			continue
		}
		result[contract.ChallengeID] = append(result[contract.ChallengeID], contract)
	}
	for challengeID := range result {
		sort.SliceStable(result[challengeID], func(i, j int) bool {
			if result[challengeID][i].DeployOrder == result[challengeID][j].DeployOrder {
				return result[challengeID][i].CreatedAt.Before(result[challengeID][j].CreatedAt)
			}
			return result[challengeID][i].DeployOrder < result[challengeID][j].DeployOrder
		})
	}
	return result
}

// decodeChallengeConstructorArgs 解析题目合约构造参数，统一为运行时可消费的动态数组。
func decodeChallengeConstructorArgs(raw []byte) ([]interface{}, error) {
	if len(raw) == 0 {
		return []interface{}{}, nil
	}
	var values []interface{}
	if err := decodeJSON(raw, &values); err == nil {
		return values, nil
	}
	var single interface{}
	if err := decodeJSON(raw, &single); err != nil {
		return nil, err
	}
	return []interface{}{single}, nil
}

// acquireAttackExecutionLock 获取目标漏洞攻击锁，命中时阻止并发攻击继续执行。
func acquireAttackExecutionLock(ctx context.Context, roundID, targetTeamID, challengeID int64) error {
	if cache.Get() == nil {
		return nil
	}
	acquired, err := cache.SetNX(ctx, buildADAttackLockKey(roundID, targetTeamID, challengeID), "1", adAttackLockTTL)
	if err != nil {
		return err
	}
	if !acquired {
		return errcode.ErrAdAttackLocked
	}
	return nil
}

// releaseAttackExecutionLock 释放目标漏洞攻击锁。
func releaseAttackExecutionLock(ctx context.Context, roundID, targetTeamID, challengeID int64) {
	if cache.Get() == nil {
		return
	}
	_ = cache.Del(ctx, buildADAttackLockKey(roundID, targetTeamID, challengeID))
}

// buildADAttackLockKey 构建同回合同目标漏洞的分布式锁键。
func buildADAttackLockKey(roundID, targetTeamID, challengeID int64) string {
	return cache.KeyCTFADAttackLock +
		strconv.FormatInt(roundID, 10) + ":" +
		strconv.FormatInt(targetTeamID, 10) + ":" +
		strconv.FormatInt(challengeID, 10)
}

// calculateAdExploitReward 根据 exploit_count 和衰减因子计算本次可窃取 Token。
func calculateAdExploitReward(baseReward int, exploitCount int, cfg *dto.ADCompetitionConfig) int {
	if exploitCount <= 0 {
		exploitCount = 1
	}
	return int(math.Round(float64(baseReward) * math.Pow(cfg.VulnerabilityDecayFactor, float64(exploitCount-1))))
}

// buildADAttackExecutionSpec 构建攻防赛攻击代理执行规格。
func (s *battleService) buildADAttackExecutionSpec(ctx context.Context, targetTeamID, challengeID int64, attackTxData string) (*ADAttackExecutionSpec, error) {
	chain, err := s.adChainRepo.GetByTeamID(ctx, targetTeamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrEnvironmentNotFound
		}
		return nil, err
	}
	if chain.ChainRPCURL == nil {
		return nil, errcode.ErrEnvironmentNotFound.WithMessage("目标队伍链未就绪")
	}
	contracts, err := decodeTeamChainContracts(chain.DeployedContracts)
	if err != nil {
		return nil, err
	}
	assertions, err := s.assertionRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	return &ADAttackExecutionSpec{
		Namespace:    buildAdGroupNamespace(chain.CompetitionID, chain.GroupID),
		ChainRPCURL:  *chain.ChainRPCURL,
		AttackTxData: attackTxData,
		Contracts:    filterTeamContractsByChallenge(contracts, challengeID),
		Assertions:   buildRuntimeAssertions(assertions),
	}, nil
}

// buildADPatchVerificationSpec 构建补丁验证运行时规格。
func (s *battleService) buildADPatchVerificationSpec(ctx context.Context, teamID, challengeID int64, source string, challenge *entity.Challenge) (*ADPatchVerificationSpec, error) {
	chain, err := s.adChainRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrEnvironmentNotFound
		}
		return nil, err
	}
	if chain.ChainRPCURL == nil {
		return nil, errcode.ErrEnvironmentNotFound.WithMessage("队伍链未就绪")
	}
	teamContracts, err := decodeTeamChainContracts(chain.DeployedContracts)
	if err != nil {
		return nil, err
	}
	originalContracts, err := listChallengeRuntimeContracts(ctx, s.contractRepo, challengeID)
	if err != nil {
		return nil, err
	}
	assertions, err := s.assertionRepo.ListByChallengeID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	officialPoc, err := s.loadOfficialPoc(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	return &ADPatchVerificationSpec{
		Namespace:          buildAdGroupNamespace(chain.CompetitionID, chain.GroupID),
		ChallengeID:        challengeID,
		ChallengeTitle:     challenge.Title,
		ChainRPCURL:        *chain.ChainRPCURL,
		PatchSourceCode:    source,
		OriginalContracts:  originalContracts,
		TargetContracts:    filterTeamContractsByChallenge(teamContracts, challengeID),
		Assertions:         buildRuntimeAssertions(assertions),
		OfficialPocContent: officialPoc,
	}, nil
}

// loadOfficialPoc 读取题目最新通过预验证中的官方 PoC。
func (s *battleService) loadOfficialPoc(ctx context.Context, challengeID int64) (string, error) {
	verification, err := s.verificationRepo.GetLatestByChallengeID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errcode.ErrChallengeVerificationAbsent
		}
		return "", err
	}
	if verification.Status != enum.VerificationStatusPassed || verification.PocContent == nil {
		return "", errcode.ErrChallengeVerificationAbsent
	}
	return decryptSensitiveTextValue(verification.PocContent), nil
}

// filterTeamContractsByChallenge 过滤队伍链中属于指定漏洞题目的部署合约。
func filterTeamContractsByChallenge(contracts []dto.TeamChainContractItem, challengeID int64) []dto.TeamChainContractItem {
	items := make([]dto.TeamChainContractItem, 0, len(contracts))
	challengeIDText := int64String(challengeID)
	for _, contract := range contracts {
		if contract.ChallengeID == challengeIDText {
			items = append(items, contract)
		}
	}
	return items
}
