// battle_operations.go
// 模块05 — CTF竞赛：攻防赛回合、攻防提交与操作面访问控制。

package ctf

import (
	"context"
	"errors"
	"math"
	"strings"
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

func (s *battleService) ListRounds(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.AdRoundListResp, error) {
	group, err := s.requireReadableGroup(ctx, sc, groupID)
	if err != nil {
		return nil, err
	}
	_ = group
	rounds, err := s.adRoundRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdRoundListItem, 0, len(rounds))
	for _, round := range rounds {
		items = append(items, dto.AdRoundListItem{
			ID:          int64String(round.ID),
			RoundNumber: round.RoundNumber,
			Phase:       round.Phase,
			PhaseText:   enum.GetRoundPhaseText(round.Phase),
		})
	}
	return &dto.AdRoundListResp{List: items}, nil
}

// GetRound 获取回合详情。
func (s *battleService) GetRound(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AdRoundDetailResp, error) {
	round, err := s.adRoundRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdRoundNotFound
		}
		return nil, err
	}
	if _, err := s.requireReadableGroup(ctx, sc, round.GroupID); err != nil {
		return nil, err
	}
	return buildAdRoundDetailResp(round), nil
}

// GetCurrentRound 获取分组当前回合状态。
func (s *battleService) GetCurrentRound(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.CurrentRoundResp, error) {
	// 文档将“当前回合状态”限定为分组内选手视角，不向竞赛创建者或其他管理角色开放，
	// 因为返回体包含当前用户所属队伍的余额和排名，属于分组内实时对抗信息。
	group, err := s.requireStudentReadableGroup(ctx, sc, groupID)
	if err != nil {
		return nil, err
	}
	round, err := s.adRoundRepo.GetCurrentByGroupID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdRoundNotFound
		}
		return nil, err
	}
	competition, err := getCompetition(ctx, s.competitionRepo, group.CompetitionID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.loadADConfig(competition)
	if err != nil {
		return nil, err
	}
	phaseStart, phaseEnd := roundPhaseWindow(round)
	now := time.Now()
	remainingSeconds := 0
	if phaseEnd != nil && phaseEnd.After(now) {
		remainingSeconds = int(phaseEnd.Sub(now).Seconds())
	}
	resp := &dto.CurrentRoundResp{
		GroupID:          int64String(groupID),
		RoundID:          int64String(round.ID),
		RoundNumber:      round.RoundNumber,
		TotalRounds:      cfg.TotalRounds,
		Phase:            round.Phase,
		PhaseText:        enum.GetRoundPhaseText(round.Phase),
		PhaseStartAt:     timeString(*phaseStart),
		PhaseEndAt:       timeString(*phaseEnd),
		RemainingSeconds: remainingSeconds,
	}
	if sc.IsStudent() {
		if _, myTeam, teamErr := s.getCurrentMemberTeam(ctx, competition.ID, sc.UserID); teamErr == nil && myTeam.AdGroupID != nil && *myTeam.AdGroupID == groupID {
			rank := s.calculateAdTeamRank(ctx, groupID, myTeam.ID)
			resp.MyTeam = &dto.CurrentRoundTeamInfo{
				ID:           int64String(myTeam.ID),
				Name:         myTeam.Name,
				TokenBalance: derefInt(myTeam.TokenBalance),
				Rank:         rank,
			}
		}
	}
	return resp, nil
}

// SubmitAttack 提交攻击交易。
func (s *battleService) SubmitAttack(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.SubmitAdAttackReq) (*dto.AdAttackResp, error) {
	if !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	round, group, competition, cfg, attackerTeam, err := s.loadRoundActionContext(ctx, sc, roundID)
	if err != nil {
		return nil, err
	}
	if round.Phase != enum.RoundPhaseAttacking {
		return nil, errcode.ErrNotInAttackPhase
	}
	targetTeamID, err := snowflake.ParseString(req.TargetTeamID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	challengeID, err := snowflake.ParseString(req.ChallengeID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	if targetTeamID == attackerTeam.ID {
		return nil, errcode.ErrAdSelfAttackForbidden
	}
	targetTeam, err := s.teamRepo.GetByID(ctx, targetTeamID)
	if err != nil {
		return nil, errcode.ErrTeamNotFound
	}
	if targetTeam.AdGroupID == nil || *targetTeam.AdGroupID != group.ID {
		return nil, errcode.ErrAdCrossGroupForbidden
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if _, err := s.compChallengeRepo.GetByCompetitionAndChallenge(ctx, competition.ID, challengeID); err != nil {
		return nil, errcode.ErrSubmissionChallengeGone
	}
	patched, err := s.adDefenseRepo.HasAcceptedPatch(ctx, targetTeam.ID, challengeID)
	if err != nil {
		return nil, err
	}
	if patched {
		return nil, errcode.ErrAdChallengePatched
	}
	if err := acquireAttackExecutionLock(ctx, round.ID, targetTeam.ID, challengeID); err != nil {
		return nil, err
	}
	defer releaseAttackExecutionLock(ctx, round.ID, targetTeam.ID, challengeID)
	if s.attackExecutor == nil {
		return nil, errcode.ErrInternal.WithMessage("CTF攻防攻击执行器未配置")
	}
	attackExecSpec, err := s.buildADAttackExecutionSpec(ctx, targetTeam.ID, challengeID, req.AttackTxData)
	if err != nil {
		return nil, err
	}
	attackExecResult, err := s.attackExecutor.ExecuteADAttack(ctx, attackExecSpec)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("攻击执行失败：" + err.Error())
	}
	success := attackExecResult != nil && attackExecResult.IsSuccessful
	var assertionResults *dto.VerificationAssertionResults
	var errorMessage *string
	if attackExecResult != nil {
		assertionResults = attackExecResult.AssertionResults
		errorMessage = attackExecResult.ErrorMessage
	}
	attack := &entity.AdAttack{
		ID:               snowflake.Generate(),
		CompetitionID:    competition.ID,
		RoundID:          round.ID,
		AttackerTeamID:   attackerTeam.ID,
		TargetTeamID:     targetTeam.ID,
		ChallengeID:      challengeID,
		AttackTxData:     req.AttackTxData,
		IsSuccessful:     success,
		AssertionResults: mustJSON(assertionResults),
		ErrorMessage:     errorMessage,
		CreatedAt:        time.Now(),
	}
	resp := &dto.AdAttackResp{
		AttackID:         int64String(attack.ID),
		IsSuccessful:     success,
		AssertionResults: assertionResults,
		ErrorMessage:     errorMessage,
	}
	if !success {
		if err := s.adAttackRepo.Create(ctx, attack); err != nil {
			return nil, err
		}
		s.publishAttackResult(ctx, competition.ID, group.ID, round.RoundNumber, attackerTeam, targetTeam, challenge, resp)
		return resp, nil
	}

	exploitCountBase, err := s.adAttackRepo.CountSuccessfulByChallenge(ctx, competition.ID, group.ID, challengeID)
	if err != nil {
		return nil, err
	}
	exploitCount := int(exploitCountBase) + 1
	isFirstBlood := exploitCount == 1
	stolenAmount := calculateAdExploitReward(challenge.BaseScore, exploitCount, cfg)
	bonusAmount := int(math.Round(float64(stolenAmount) * cfg.AttackBonusRatio))
	firstBloodAmount := 0
	if isFirstBlood {
		firstBloodAmount = int(math.Round(float64(stolenAmount) * cfg.FirstBloodBonusRatio))
	}
	totalGain := stolenAmount + bonusAmount + firstBloodAmount
	attackerBalanceAfter := derefInt(attackerTeam.TokenBalance) + totalGain
	targetBalanceAfter := derefInt(targetTeam.TokenBalance) - stolenAmount
	if targetBalanceAfter < 0 {
		targetBalanceAfter = 0
	}
	attack.TokenReward = &stolenAmount
	attack.ExploitCount = &exploitCount
	attack.IsFirstBlood = isFirstBlood

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txAttackRepo := ctfrepo.NewAdAttackRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txLedgerRepo := ctfrepo.NewAdTokenLedgerRepository(tx)
		if err := txAttackRepo.Create(ctx, attack); err != nil {
			return err
		}
		if err := txTeamRepo.UpdateTokenBalance(ctx, attackerTeam.ID, attackerBalanceAfter); err != nil {
			return err
		}
		if err := txTeamRepo.UpdateTokenBalance(ctx, targetTeam.ID, targetBalanceAfter); err != nil {
			return err
		}
		ledgers := buildAttackLedgers(round, group.ID, attack, attackerBalanceAfter, targetBalanceAfter, stolenAmount, bonusAmount, firstBloodAmount, targetTeam.Name, challenge.Title)
		return txLedgerRepo.BatchCreate(ctx, ledgers)
	})
	if err != nil {
		return nil, err
	}

	resp.TokenReward = &stolenAmount
	resp.IsFirstBlood = &isFirstBlood
	resp.ExploitCount = &exploitCount
	resp.AttackerBalanceAfter = &attackerBalanceAfter
	resp.TargetBalanceAfter = &targetBalanceAfter
	writeAdTokenBalanceCache(ctx, competition.ID, attackerTeam.ID, attackerBalanceAfter)
	writeAdTokenBalanceCache(ctx, competition.ID, targetTeam.ID, targetBalanceAfter)
	s.publishAttackResult(ctx, competition.ID, group.ID, round.RoundNumber, attackerTeam, targetTeam, challenge, resp)
	s.publishBattleLeaderboard(ctx, competition.ID, group.ID, &LeaderboardRealtimeTrigger{
		TeamName:       attackerTeam.Name,
		ChallengeTitle: challenge.Title,
		IsFirstBlood:   isFirstBlood,
	})
	return resp, nil
}

// ListRoundAttacks 查询回合攻击记录。
func (s *battleService) ListRoundAttacks(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.AdAttackListReq) (*dto.AdAttackListResp, error) {
	round, err := s.adRoundRepo.GetByID(ctx, roundID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdRoundNotFound
		}
		return nil, err
	}
	// 回合级攻击记录属于参赛选手操作视图，不向竞赛创建者开放。
	if _, err := s.requireStudentReadableGroup(ctx, sc, round.GroupID); err != nil {
		return nil, err
	}
	params, err := s.buildAdAttackListParams(round.CompetitionID, roundID, round.GroupID, req)
	if err != nil {
		return nil, err
	}
	return s.listAttacks(ctx, params, req.Page, req.PageSize)
}

// ListGroupAttacks 查询分组攻击记录。
func (s *battleService) ListGroupAttacks(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.AdAttackListReq) (*dto.AdAttackListResp, error) {
	group, err := s.requireReadableGroup(ctx, sc, groupID)
	if err != nil {
		return nil, err
	}
	params, err := s.buildAdAttackListParams(group.CompetitionID, 0, groupID, req)
	if err != nil {
		return nil, err
	}
	return s.listAttacks(ctx, params, req.Page, req.PageSize)
}

// SubmitDefense 提交补丁合约。
func (s *battleService) SubmitDefense(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.SubmitAdDefenseReq) (*dto.AdDefenseResp, error) {
	if !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	round, group, competition, cfg, team, err := s.loadRoundActionContext(ctx, sc, roundID)
	if err != nil {
		return nil, err
	}
	if round.Phase != enum.RoundPhaseDefending {
		return nil, errcode.ErrNotInDefensePhase
	}
	if strings.TrimSpace(req.PatchSourceCode) == "" {
		return nil, errcode.ErrAdPatchSourceRequired
	}
	challengeID, err := snowflake.ParseString(req.ChallengeID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	if _, err := s.compChallengeRepo.GetByCompetitionAndChallenge(ctx, competition.ID, challengeID); err != nil {
		return nil, errcode.ErrSubmissionChallengeGone
	}
	acceptedBefore, err := s.adDefenseRepo.HasAcceptedPatch(ctx, team.ID, challengeID)
	if err != nil {
		return nil, err
	}
	if acceptedBefore {
		return nil, errcode.ErrAdPatchAlreadyAccepted
	}
	if s.patchVerifier == nil {
		return nil, errcode.ErrInternal.WithMessage("CTF补丁验证执行器未配置")
	}
	patchSpec, err := s.buildADPatchVerificationSpec(ctx, team.ID, challengeID, req.PatchSourceCode, challenge)
	if err != nil {
		return nil, err
	}
	patchResult, err := s.patchVerifier.VerifyADPatch(ctx, patchSpec)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("补丁验证失败：" + err.Error())
	}
	functionalityPassed := patchResult != nil && patchResult.FunctionalityPassed
	vulnerabilityFixed := patchResult != nil && patchResult.VulnerabilityFixed
	isAccepted := functionalityPassed && vulnerabilityFixed
	rejectionReason := resolvePatchRejectionReason(patchResult, functionalityPassed, vulnerabilityFixed)
	defense := &entity.AdDefense{
		ID:                  snowflake.Generate(),
		CompetitionID:       competition.ID,
		RoundID:             round.ID,
		TeamID:              team.ID,
		ChallengeID:         challengeID,
		PatchSourceCode:     req.PatchSourceCode,
		IsAccepted:          isAccepted,
		FunctionalityPassed: &functionalityPassed,
		VulnerabilityFixed:  &vulnerabilityFixed,
		RejectionReason:     rejectionReason,
		CreatedAt:           time.Now(),
	}
	resp := &dto.AdDefenseResp{
		DefenseID:           int64String(defense.ID),
		IsAccepted:          isAccepted,
		FunctionalityPassed: &functionalityPassed,
		VulnerabilityFixed:  &vulnerabilityFixed,
		RejectionReason:     rejectionReason,
	}
	if !isAccepted {
		if err := s.adDefenseRepo.Create(ctx, defense); err != nil {
			return nil, err
		}
		return resp, nil
	}

	isFirstPatch, err := s.isFirstPatch(ctx, competition.ID, group.ID, challengeID)
	if err != nil {
		return nil, err
	}
	tokenReward := 0
	if isFirstPatch {
		tokenReward = cfg.FirstPatchBonus
	}
	balanceAfter := derefInt(team.TokenBalance) + tokenReward
	defense.IsFirstPatch = isFirstPatch
	defense.TokenReward = &tokenReward

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txDefenseRepo := ctfrepo.NewAdDefenseRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txLedgerRepo := ctfrepo.NewAdTokenLedgerRepository(tx)
		txChainRepo := ctfrepo.NewAdTeamChainRepository(tx)
		if err := txDefenseRepo.Create(ctx, defense); err != nil {
			return err
		}
		if err := txTeamRepo.UpdateTokenBalance(ctx, team.ID, balanceAfter); err != nil {
			return err
		}
		if tokenReward > 0 {
			ledger := buildDefenseLedger(round, group.ID, defense, balanceAfter, challenge.Title)
			if err := txLedgerRepo.Create(ctx, ledger); err != nil {
				return err
			}
		}
		chain, chainErr := txChainRepo.GetByTeamID(ctx, team.ID)
		if chainErr == nil && chain != nil {
			contracts, decodeErr := decodeTeamChainContracts(chain.DeployedContracts)
			if decodeErr == nil {
				if patchResult != nil && len(patchResult.PatchedContracts) > 0 {
					contracts = patchResult.PatchedContracts
				} else {
					contracts = markPatchedContract(contracts, challengeID)
				}
				fields := map[string]interface{}{
					"deployed_contracts":    mustJSON(contracts),
					"current_patch_version": chain.CurrentPatchVersion + 1,
					"updated_at":            time.Now(),
				}
				if err := txChainRepo.UpdateFields(ctx, chain.ID, fields); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	resp.IsFirstPatch = &isFirstPatch
	resp.TokenReward = &tokenReward
	resp.TeamBalanceAfter = &balanceAfter
	writeAdTokenBalanceCache(ctx, competition.ID, team.ID, balanceAfter)
	s.publishBattleLeaderboard(ctx, competition.ID, group.ID, nil)
	return resp, nil
}

// ListRoundDefenses 查询回合防守记录。
func (s *battleService) ListRoundDefenses(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.AdDefenseListReq) (*dto.AdDefenseListResp, error) {
	round, err := s.adRoundRepo.GetByID(ctx, roundID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdRoundNotFound
		}
		return nil, err
	}
	// 回合级防守记录同样只开放给分组内参赛选手。
	if _, err := s.requireStudentReadableGroup(ctx, sc, round.GroupID); err != nil {
		return nil, err
	}
	params, err := s.buildAdDefenseListParams(round.CompetitionID, roundID, round.GroupID, req)
	if err != nil {
		return nil, err
	}
	return s.listDefenses(ctx, params, req.Page, req.PageSize)
}

// ListTokenLedgerByCompetition 查询竞赛 Token 流水。
func (s *battleService) ListTokenLedgerByCompetition(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.TokenLedgerListReq) (*dto.TokenLedgerListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	params, err := s.buildLedgerListParams(competitionID, 0, req)
	if err != nil {
		return nil, err
	}
	return s.listLedgers(ctx, params, req.Page, req.PageSize)
}

// ListTokenLedgerByTeam 查询团队 Token 流水。
func (s *battleService) ListTokenLedgerByTeam(ctx context.Context, sc *svcctx.ServiceContext, teamID int64, req *dto.TokenLedgerListReq) (*dto.TokenLedgerListResp, error) {
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		return nil, errcode.ErrTeamNotFound
	}
	if err := s.requireMemberReadableTeam(ctx, sc, teamID); err != nil {
		return nil, err
	}
	params, err := s.buildLedgerListParams(team.CompetitionID, teamID, req)
	if err != nil {
		return nil, err
	}
	return s.listLedgers(ctx, params, req.Page, req.PageSize)
}

// GetTeamChain 获取队伍链详情。
func (s *battleService) GetTeamChain(ctx context.Context, sc *svcctx.ServiceContext, teamID int64) (*dto.TeamChainResp, error) {
	if _, err := s.teamRepo.GetByID(ctx, teamID); err != nil {
		return nil, errcode.ErrTeamNotFound
	}
	if err := s.requireMemberReadableTeam(ctx, sc, teamID); err != nil {
		return nil, err
	}
	chain, err := s.adChainRepo.GetByTeamID(ctx, teamID)
	if err != nil {
		return nil, errcode.ErrEnvironmentNotFound
	}
	return buildTeamChainResp(chain)
}

// ListGroupChains 查询分组全部队伍链。
func (s *battleService) ListGroupChains(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.TeamChainListResp, error) {
	if _, err := s.requireStudentReadableGroup(ctx, sc, groupID); err != nil {
		return nil, err
	}
	chains, err := s.adChainRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.TeamChainResp, 0, len(chains))
	for _, chain := range chains {
		resp, respErr := buildTeamChainResp(chain)
		if respErr != nil {
			return nil, respErr
		}
		items = append(items, *resp)
	}
	return &dto.TeamChainListResp{List: items}, nil
}

// ensureCompetitionManageable 校验当前上下文是否可管理攻防赛。
func (s *battleService) ensureCompetitionManageable(sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if sc == nil || competition == nil {
		return errcode.ErrForbidden
	}
	if competition.CompetitionType != enum.CompetitionTypeAttackDefense {
		return errcode.ErrAdCompetitionOnly
	}
	if competition.CreatedBy == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureAdCompetitionReadable 校验当前上下文是否可读取攻防赛分组视图。
func (s *battleService) ensureAdCompetitionReadable(ctx context.Context, sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if competition.CompetitionType != enum.CompetitionTypeAttackDefense {
		return errcode.ErrAdCompetitionOnly
	}
	// 模块文档中的攻防赛分组视图只开放给竞赛创建者和已参赛学生，
	// 管理员如需排查运行态应使用监控接口，而不是选手分组视图。
	if competition.CreatedBy == sc.UserID {
		return nil
	}
	if !sc.IsStudent() {
		return errcode.ErrForbidden
	}
	_, _, err := s.getCurrentMemberTeam(ctx, competition.ID, sc.UserID)
	return err
}

// canReadGroup 判断当前上下文是否可读取指定分组。
func (s *battleService) canReadGroup(ctx context.Context, sc *svcctx.ServiceContext, group *entity.AdGroup) bool {
	competition, err := getCompetition(ctx, s.competitionRepo, group.CompetitionID)
	if err == nil && competition.CreatedBy == sc.UserID {
		return true
	}
	if !sc.IsStudent() {
		return false
	}
	_, team, err := s.getCurrentMemberTeam(ctx, group.CompetitionID, sc.UserID)
	if err != nil || team.AdGroupID == nil {
		return false
	}
	return *team.AdGroupID == group.ID
}

// canStudentReadGroup 判断当前学生是否属于指定分组。
func (s *battleService) canStudentReadGroup(ctx context.Context, sc *svcctx.ServiceContext, group *entity.AdGroup) bool {
	if sc == nil || group == nil || !sc.IsStudent() {
		return false
	}
	_, team, err := s.getCurrentMemberTeam(ctx, group.CompetitionID, sc.UserID)
	if err != nil || team.AdGroupID == nil {
		return false
	}
	return *team.AdGroupID == group.ID
}

// requireMemberReadableTeam 校验当前上下文是否为团队成员。
// 团队 Token 流水和队伍链地址都属于团队内部敏感数据面，不通过管理员身份放宽。
func (s *battleService) requireMemberReadableTeam(ctx context.Context, sc *svcctx.ServiceContext, teamID int64) error {
	if sc == nil {
		return errcode.ErrForbidden
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, teamID, sc.UserID)
	if err != nil {
		return err
	}
	if !isMember {
		return errcode.ErrForbidden
	}
	return nil
}

// getCurrentMemberTeam 获取当前学生在竞赛中的团队成员关系和团队实体。
func (s *battleService) getCurrentMemberTeam(ctx context.Context, competitionID, studentID int64) (*entity.TeamMember, *entity.Team, error) {
	return getCompetitionMemberTeam(ctx, s.teamMemberRepo, s.teamRepo, competitionID, studentID)
}

// requireReadableGroup 获取并校验分组读取权限。
func (s *battleService) requireReadableGroup(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*entity.AdGroup, error) {
	group, err := s.adGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdGroupNotFound
		}
		return nil, err
	}
	if !s.canReadGroup(ctx, sc, group) {
		return nil, errcode.ErrForbidden
	}
	return group, nil
}

// requireStudentReadableGroup 获取并校验“当前回合状态”所需的分组读取权限。
// 当前回合接口只允许分组内选手访问，不允许创建者/超管代查他组实时状态。
func (s *battleService) requireStudentReadableGroup(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*entity.AdGroup, error) {
	group, err := s.adGroupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdGroupNotFound
		}
		return nil, err
	}
	if !s.canStudentReadGroup(ctx, sc, group) {
		return nil, errcode.ErrForbidden
	}
	return group, nil
}

// loadADConfig 读取攻防赛配置并补齐文档默认值。
func (s *battleService) loadADConfig(competition *entity.Competition) (*dto.ADCompetitionConfig, error) {
	if competition == nil || competition.CompetitionType != enum.CompetitionTypeAttackDefense {
		return nil, errcode.ErrAdCompetitionOnly
	}
	cfg := &dto.ADCompetitionConfig{
		TotalRounds:              10,
		AttackDurationMinutes:    10,
		DefenseDurationMinutes:   5,
		InitialToken:             10000,
		AttackBonusRatio:         0.05,
		DefenseRewardPerRound:    50,
		FirstPatchBonus:          200,
		FirstBloodBonusRatio:     0.10,
		VulnerabilityDecayFactor: 0.8,
		MaxTeamsPerGroup:         4,
		JudgeChainImage:          "geth-dev:latest",
		TeamChainImage:           "ganache:latest",
	}
	if len(competition.AdConfig) == 0 {
		return cfg, nil
	}
	if err := decodeJSON(competition.AdConfig, cfg); err != nil {
		return nil, errcode.ErrCompetitionConfigRequired
	}
	if cfg.TotalRounds <= 0 {
		cfg.TotalRounds = 10
	}
	if cfg.AttackDurationMinutes <= 0 {
		cfg.AttackDurationMinutes = 10
	}
	if cfg.DefenseDurationMinutes <= 0 {
		cfg.DefenseDurationMinutes = 5
	}
	if cfg.InitialToken <= 0 {
		cfg.InitialToken = 10000
	}
	if cfg.AttackBonusRatio <= 0 {
		cfg.AttackBonusRatio = 0.05
	}
	if cfg.DefenseRewardPerRound <= 0 {
		cfg.DefenseRewardPerRound = 50
	}
	if cfg.FirstPatchBonus <= 0 {
		cfg.FirstPatchBonus = 200
	}
	if cfg.FirstBloodBonusRatio <= 0 {
		cfg.FirstBloodBonusRatio = 0.10
	}
	if cfg.VulnerabilityDecayFactor <= 0 {
		cfg.VulnerabilityDecayFactor = 0.8
	}
	if cfg.MaxTeamsPerGroup <= 0 {
		cfg.MaxTeamsPerGroup = 4
	}
	if strings.TrimSpace(cfg.JudgeChainImage) == "" {
		cfg.JudgeChainImage = "geth-dev:latest"
	}
	if strings.TrimSpace(cfg.TeamChainImage) == "" {
		cfg.TeamChainImage = "ganache:latest"
	}
	return cfg, nil
}

// loadAssignableTeams 加载待分组团队并校验归属。
func (s *battleService) loadAssignableTeams(ctx context.Context, competitionID int64, teamIDs []int64) ([]*entity.Team, error) {
	teams, err := s.teamRepo.ListByIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	if len(teams) != len(teamIDs) {
		return nil, errcode.ErrTeamNotFound
	}
	for _, team := range teams {
		if team.CompetitionID != competitionID {
			return nil, errcode.ErrTeamNotFound
		}
		if team.AdGroupID != nil && *team.AdGroupID > 0 {
			return nil, errcode.ErrCompetitionConfigRequired.WithMessage("存在已分组队伍，不能重复分配")
		}
	}
	return teams, nil
}

// publishAttackResult 在攻击提交后广播攻击结果事件。
func (s *battleService) publishAttackResult(ctx context.Context, competitionID, groupID int64, roundNumber int, attackerTeam, targetTeam *entity.Team, challenge *entity.Challenge, resp *dto.AdAttackResp) {
	if s.realtimePublisher == nil || attackerTeam == nil || targetTeam == nil || challenge == nil || resp == nil {
		return
	}
	_ = s.realtimePublisher.PublishAttackResult(ctx, competitionID, groupID, &AttackRealtimePayload{
		Event:            "attack_result",
		RoundNumber:      roundNumber,
		AttackerTeam:     AttackRealtimeTeam{ID: int64String(attackerTeam.ID), Name: attackerTeam.Name},
		TargetTeam:       AttackRealtimeTeam{ID: int64String(targetTeam.ID), Name: targetTeam.Name},
		ChallengeTitle:   challenge.Title,
		IsSuccessful:     resp.IsSuccessful,
		IsFirstBlood:     derefBool(resp.IsFirstBlood),
		TokenReward:      resp.TokenReward,
		AttackerBalance:  resp.AttackerBalanceAfter,
		TargetBalance:    resp.TargetBalanceAfter,
		ErrorMessage:     resp.ErrorMessage,
		AssertionResults: resp.AssertionResults,
	})
}

// publishBattleLeaderboard 在攻防赛 Token 变化后广播最新分组排行榜。
func (s *battleService) publishBattleLeaderboard(ctx context.Context, competitionID, groupID int64, trigger *LeaderboardRealtimeTrigger) {
	if s.realtimePublisher == nil {
		return
	}
	_ = s.realtimePublisher.PublishLeaderboardUpdate(ctx, competitionID, &groupID, trigger)
}
