// ad_repo.go
// 模块05 — CTF竞赛：攻防对抗赛数据访问层。
// 负责 ad_groups、ad_rounds、ad_attacks、ad_defenses、ad_token_ledger、ad_team_chains 的读写和统计。

package ctfrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// AdGroupRepository 攻防赛分组数据访问接口。
type AdGroupRepository interface {
	Create(ctx context.Context, group *entity.AdGroup) error
	GetByID(ctx context.Context, id int64) (*entity.AdGroup, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.AdGroup, error)
	CountByCompetitionID(ctx context.Context, competitionID int64) (int64, error)
}

type adGroupRepository struct {
	db *gorm.DB
}

// NewAdGroupRepository 创建攻防赛分组数据访问实例。
func NewAdGroupRepository(db *gorm.DB) AdGroupRepository {
	return &adGroupRepository{db: db}
}

// Create 创建攻防赛分组。
func (r *adGroupRepository) Create(ctx context.Context, group *entity.AdGroup) error {
	if group.ID == 0 {
		group.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(group).Error
}

// GetByID 根据 ID 获取攻防赛分组。
func (r *adGroupRepository) GetByID(ctx context.Context, id int64) (*entity.AdGroup, error) {
	var group entity.AdGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateFields 更新攻防赛分组指定字段。
func (r *adGroupRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.AdGroup{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateStatus 更新攻防赛分组状态。
func (r *adGroupRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.AdGroup{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// ListByCompetitionID 查询竞赛攻防赛分组列表。
func (r *adGroupRepository) ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.AdGroup, error) {
	var groups []*entity.AdGroup
	err := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Order("created_at asc").
		Find(&groups).Error
	return groups, err
}

// CountByCompetitionID 统计竞赛攻防赛分组数。
func (r *adGroupRepository) CountByCompetitionID(ctx context.Context, competitionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AdGroup{}).
		Where("competition_id = ?", competitionID).
		Count(&count).Error
	return count, err
}

// AdRoundRepository 攻防赛回合数据访问接口。
type AdRoundRepository interface {
	Create(ctx context.Context, round *entity.AdRound) error
	GetByID(ctx context.Context, id int64) (*entity.AdRound, error)
	GetByGroupAndNumber(ctx context.Context, groupID int64, roundNumber int) (*entity.AdRound, error)
	GetCurrentByGroupID(ctx context.Context, groupID int64) (*entity.AdRound, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdatePhase(ctx context.Context, id int64, phase int16) error
	ListByGroupID(ctx context.Context, groupID int64) ([]*entity.AdRound, error)
	ListActivePhases(ctx context.Context, now time.Time) ([]*entity.AdRound, error)
}

type adRoundRepository struct {
	db *gorm.DB
}

// NewAdRoundRepository 创建攻防赛回合数据访问实例。
func NewAdRoundRepository(db *gorm.DB) AdRoundRepository {
	return &adRoundRepository{db: db}
}

// Create 创建攻防赛回合。
func (r *adRoundRepository) Create(ctx context.Context, round *entity.AdRound) error {
	if round.ID == 0 {
		round.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(round).Error
}

// GetByID 根据 ID 获取攻防赛回合。
func (r *adRoundRepository) GetByID(ctx context.Context, id int64) (*entity.AdRound, error) {
	var round entity.AdRound
	err := r.db.WithContext(ctx).First(&round, id).Error
	if err != nil {
		return nil, err
	}
	return &round, nil
}

// GetByGroupAndNumber 根据分组和回合编号获取回合。
func (r *adRoundRepository) GetByGroupAndNumber(ctx context.Context, groupID int64, roundNumber int) (*entity.AdRound, error) {
	var round entity.AdRound
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND round_number = ?", groupID, roundNumber).
		First(&round).Error
	if err != nil {
		return nil, err
	}
	return &round, nil
}

// GetCurrentByGroupID 获取分组当前未完成回合。
func (r *adRoundRepository) GetCurrentByGroupID(ctx context.Context, groupID int64) (*entity.AdRound, error) {
	var round entity.AdRound
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND phase <> ?", groupID, enum.RoundPhaseCompleted).
		Order("round_number asc").
		First(&round).Error
	if err != nil {
		return nil, err
	}
	return &round, nil
}

// UpdateFields 更新攻防赛回合指定字段。
func (r *adRoundRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.AdRound{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdatePhase 更新回合阶段。
func (r *adRoundRepository) UpdatePhase(ctx context.Context, id int64, phase int16) error {
	return r.db.WithContext(ctx).Model(&entity.AdRound{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"phase":      phase,
			"updated_at": time.Now(),
		}).Error
}

// ListByGroupID 查询分组回合列表。
func (r *adRoundRepository) ListByGroupID(ctx context.Context, groupID int64) ([]*entity.AdRound, error) {
	var rounds []*entity.AdRound
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("round_number asc").
		Find(&rounds).Error
	return rounds, err
}

// ListActivePhases 查询已经到达阶段结束时间的回合，供回合推进任务使用。
func (r *adRoundRepository) ListActivePhases(ctx context.Context, now time.Time) ([]*entity.AdRound, error) {
	var rounds []*entity.AdRound
	err := r.db.WithContext(ctx).
		Where(`
			(phase = ? AND attack_end_at IS NOT NULL AND attack_end_at <= ?)
			OR (phase = ? AND defense_end_at IS NOT NULL AND defense_end_at <= ?)
			OR (phase = ? AND settlement_end_at IS NOT NULL AND settlement_end_at <= ?)
		`, enum.RoundPhaseAttacking, now, enum.RoundPhaseDefending, now, enum.RoundPhaseSettling, now).
		Find(&rounds).Error
	return rounds, err
}

// AdAttackRepository 攻防赛攻击记录数据访问接口。
type AdAttackRepository interface {
	Create(ctx context.Context, attack *entity.AdAttack) error
	GetByID(ctx context.Context, id int64) (*entity.AdAttack, error)
	List(ctx context.Context, params *AdAttackListParams) ([]*entity.AdAttack, int64, error)
	CountSuccessfulByChallenge(ctx context.Context, competitionID, groupID, challengeID int64) (int64, error)
	HasSuccessfulByChallenge(ctx context.Context, competitionID, groupID, challengeID int64) (bool, error)
	CountSuccessfulByTeam(ctx context.Context, competitionID, teamID int64) (int64, error)
	CountSuccessfulByTeamsUntil(ctx context.Context, competitionID int64, teamIDs []int64, until time.Time) (map[int64]int64, error)
}

// AdAttackListParams 攻击记录列表查询参数。
type AdAttackListParams struct {
	CompetitionID  int64
	RoundID        int64
	GroupID        int64
	AttackerTeamID int64
	TargetTeamID   int64
	ChallengeID    int64
	IsSuccessful   *bool
	Page           int
	PageSize       int
}

type adAttackRepository struct {
	db *gorm.DB
}

// NewAdAttackRepository 创建攻防赛攻击记录数据访问实例。
func NewAdAttackRepository(db *gorm.DB) AdAttackRepository {
	return &adAttackRepository{db: db}
}

// Create 创建攻击记录。
func (r *adAttackRepository) Create(ctx context.Context, attack *entity.AdAttack) error {
	if attack.ID == 0 {
		attack.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(attack).Error
}

// GetByID 根据 ID 获取攻击记录。
func (r *adAttackRepository) GetByID(ctx context.Context, id int64) (*entity.AdAttack, error) {
	var attack entity.AdAttack
	err := r.db.WithContext(ctx).First(&attack, id).Error
	if err != nil {
		return nil, err
	}
	return &attack, nil
}

// List 查询攻击记录列表。
func (r *adAttackRepository) List(ctx context.Context, params *AdAttackListParams) ([]*entity.AdAttack, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AdAttack{})
	if params.CompetitionID > 0 {
		query = query.Where("competition_id = ?", params.CompetitionID)
	}
	if params.RoundID > 0 {
		query = query.Where("round_id = ?", params.RoundID)
	}
	if params.GroupID > 0 {
		query = query.Joins("JOIN ad_rounds ON ad_rounds.id = ad_attacks.round_id").
			Where("ad_rounds.group_id = ?", params.GroupID)
	}
	if params.AttackerTeamID > 0 {
		query = query.Where("attacker_team_id = ?", params.AttackerTeamID)
	}
	if params.TargetTeamID > 0 {
		query = query.Where("target_team_id = ?", params.TargetTeamID)
	}
	if params.ChallengeID > 0 {
		query = query.Where("challenge_id = ?", params.ChallengeID)
	}
	if params.IsSuccessful != nil {
		query = query.Where("is_successful = ?", *params.IsSuccessful)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("ad_attacks.created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var attacks []*entity.AdAttack
	if err := query.Find(&attacks).Error; err != nil {
		return nil, 0, err
	}
	return attacks, total, nil
}

// CountSuccessfulByChallenge 统计分组中某漏洞成功被利用次数。
func (r *adAttackRepository) CountSuccessfulByChallenge(ctx context.Context, competitionID, groupID, challengeID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AdAttack{}).
		Joins("JOIN ad_rounds ON ad_rounds.id = ad_attacks.round_id").
		Where("ad_attacks.competition_id = ? AND ad_rounds.group_id = ? AND ad_attacks.challenge_id = ? AND ad_attacks.is_successful = ?", competitionID, groupID, challengeID, true).
		Count(&count).Error
	return count, err
}

// HasSuccessfulByChallenge 判断分组中某漏洞是否已有成功攻击，用于 First Blood 判定。
func (r *adAttackRepository) HasSuccessfulByChallenge(ctx context.Context, competitionID, groupID, challengeID int64) (bool, error) {
	count, err := r.CountSuccessfulByChallenge(ctx, competitionID, groupID, challengeID)
	return count > 0, err
}

// CountSuccessfulByTeam 统计团队成功攻击次数。
func (r *adAttackRepository) CountSuccessfulByTeam(ctx context.Context, competitionID, teamID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AdAttack{}).
		Where("competition_id = ? AND attacker_team_id = ? AND is_successful = ?", competitionID, teamID, true).
		Count(&count).Error
	return count, err
}

// CountSuccessfulByTeamsUntil 统计多个团队在指定时间点之前的成功攻击次数。
func (r *adAttackRepository) CountSuccessfulByTeamsUntil(ctx context.Context, competitionID int64, teamIDs []int64, until time.Time) (map[int64]int64, error) {
	result := make(map[int64]int64, len(teamIDs))
	if len(teamIDs) == 0 {
		return result, nil
	}
	type row struct {
		TeamID int64 `gorm:"column:team_id"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&entity.AdAttack{}).
		Select("attacker_team_id AS team_id, COUNT(*) AS count").
		Where("competition_id = ? AND attacker_team_id IN ? AND is_successful = ? AND created_at <= ?", competitionID, teamIDs, true, until).
		Group("attacker_team_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.TeamID] = item.Count
	}
	return result, nil
}

// AdDefenseRepository 攻防赛防守记录数据访问接口。
type AdDefenseRepository interface {
	Create(ctx context.Context, defense *entity.AdDefense) error
	GetByID(ctx context.Context, id int64) (*entity.AdDefense, error)
	List(ctx context.Context, params *AdDefenseListParams) ([]*entity.AdDefense, int64, error)
	HasAcceptedPatch(ctx context.Context, teamID, challengeID int64) (bool, error)
	HasFirstPatch(ctx context.Context, competitionID, groupID, challengeID int64) (bool, error)
	CountAcceptedByTeam(ctx context.Context, competitionID, teamID int64) (int64, error)
	CountAcceptedByTeamsUntil(ctx context.Context, competitionID int64, teamIDs []int64, until time.Time) (map[int64]int64, error)
}

// AdDefenseListParams 防守记录列表查询参数。
type AdDefenseListParams struct {
	CompetitionID int64
	RoundID       int64
	GroupID       int64
	TeamID        int64
	ChallengeID   int64
	IsAccepted    *bool
	Page          int
	PageSize      int
}

type adDefenseRepository struct {
	db *gorm.DB
}

// NewAdDefenseRepository 创建攻防赛防守记录数据访问实例。
func NewAdDefenseRepository(db *gorm.DB) AdDefenseRepository {
	return &adDefenseRepository{db: db}
}

// Create 创建防守记录。
func (r *adDefenseRepository) Create(ctx context.Context, defense *entity.AdDefense) error {
	if defense.ID == 0 {
		defense.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(defense).Error
}

// GetByID 根据 ID 获取防守记录。
func (r *adDefenseRepository) GetByID(ctx context.Context, id int64) (*entity.AdDefense, error) {
	var defense entity.AdDefense
	err := r.db.WithContext(ctx).First(&defense, id).Error
	if err != nil {
		return nil, err
	}
	return &defense, nil
}

// List 查询防守记录列表。
func (r *adDefenseRepository) List(ctx context.Context, params *AdDefenseListParams) ([]*entity.AdDefense, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AdDefense{})
	if params.CompetitionID > 0 {
		query = query.Where("competition_id = ?", params.CompetitionID)
	}
	if params.RoundID > 0 {
		query = query.Where("round_id = ?", params.RoundID)
	}
	if params.GroupID > 0 {
		query = query.Joins("JOIN ad_rounds ON ad_rounds.id = ad_defenses.round_id").
			Where("ad_rounds.group_id = ?", params.GroupID)
	}
	if params.TeamID > 0 {
		query = query.Where("team_id = ?", params.TeamID)
	}
	if params.ChallengeID > 0 {
		query = query.Where("challenge_id = ?", params.ChallengeID)
	}
	if params.IsAccepted != nil {
		query = query.Where("is_accepted = ?", *params.IsAccepted)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("ad_defenses.created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var defenses []*entity.AdDefense
	if err := query.Find(&defenses).Error; err != nil {
		return nil, 0, err
	}
	return defenses, total, nil
}

// HasAcceptedPatch 判断团队某漏洞是否已有通过补丁。
func (r *adDefenseRepository) HasAcceptedPatch(ctx context.Context, teamID, challengeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AdDefense{}).
		Where("team_id = ? AND challenge_id = ? AND is_accepted = ?", teamID, challengeID, true).
		Count(&count).Error
	return count > 0, err
}

// HasFirstPatch 判断分组某漏洞是否已有首个通过补丁。
func (r *adDefenseRepository) HasFirstPatch(ctx context.Context, competitionID, groupID, challengeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AdDefense{}).
		Joins("JOIN ad_rounds ON ad_rounds.id = ad_defenses.round_id").
		Where("ad_defenses.competition_id = ? AND ad_rounds.group_id = ? AND ad_defenses.challenge_id = ? AND ad_defenses.is_accepted = ?", competitionID, groupID, challengeID, true).
		Count(&count).Error
	return count > 0, err
}

// CountAcceptedByTeam 统计团队通过补丁数。
func (r *adDefenseRepository) CountAcceptedByTeam(ctx context.Context, competitionID, teamID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AdDefense{}).
		Where("competition_id = ? AND team_id = ? AND is_accepted = ?", competitionID, teamID, true).
		Count(&count).Error
	return count, err
}

// CountAcceptedByTeamsUntil 统计多个团队在指定时间点之前的通过补丁数。
func (r *adDefenseRepository) CountAcceptedByTeamsUntil(ctx context.Context, competitionID int64, teamIDs []int64, until time.Time) (map[int64]int64, error) {
	result := make(map[int64]int64, len(teamIDs))
	if len(teamIDs) == 0 {
		return result, nil
	}
	type row struct {
		TeamID int64 `gorm:"column:team_id"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&entity.AdDefense{}).
		Select("team_id, COUNT(*) AS count").
		Where("competition_id = ? AND team_id IN ? AND is_accepted = ? AND created_at <= ?", competitionID, teamIDs, true, until).
		Group("team_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.TeamID] = item.Count
	}
	return result, nil
}

// AdTokenLedgerRepository Token 流水数据访问接口。
type AdTokenLedgerRepository interface {
	Create(ctx context.Context, ledger *entity.AdTokenLedger) error
	BatchCreate(ctx context.Context, ledgers []*entity.AdTokenLedger) error
	List(ctx context.Context, params *AdTokenLedgerListParams) ([]*entity.AdTokenLedger, int64, error)
	GetLatestByTeamID(ctx context.Context, teamID int64) (*entity.AdTokenLedger, error)
	CountByTeamsAndChangeTypeUntil(ctx context.Context, competitionID int64, teamIDs []int64, changeType int16, until time.Time) (map[int64]int64, error)
}

// AdTokenLedgerListParams Token 流水列表查询参数。
type AdTokenLedgerListParams struct {
	CompetitionID int64
	GroupID       int64
	RoundID       int64
	TeamID        int64
	ChangeType    int16
	Page          int
	PageSize      int
}

type adTokenLedgerRepository struct {
	db *gorm.DB
}

// NewAdTokenLedgerRepository 创建 Token 流水数据访问实例。
func NewAdTokenLedgerRepository(db *gorm.DB) AdTokenLedgerRepository {
	return &adTokenLedgerRepository{db: db}
}

// Create 创建 Token 流水。
func (r *adTokenLedgerRepository) Create(ctx context.Context, ledger *entity.AdTokenLedger) error {
	if ledger.ID == 0 {
		ledger.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(ledger).Error
}

// BatchCreate 批量创建 Token 流水。
func (r *adTokenLedgerRepository) BatchCreate(ctx context.Context, ledgers []*entity.AdTokenLedger) error {
	if len(ledgers) == 0 {
		return nil
	}
	for i := range ledgers {
		if ledgers[i].ID == 0 {
			ledgers[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(ledgers, 100).Error
}

// List 查询 Token 流水列表。
func (r *adTokenLedgerRepository) List(ctx context.Context, params *AdTokenLedgerListParams) ([]*entity.AdTokenLedger, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AdTokenLedger{})
	if params.CompetitionID > 0 {
		query = query.Where("competition_id = ?", params.CompetitionID)
	}
	if params.GroupID > 0 {
		query = query.Where("group_id = ?", params.GroupID)
	}
	if params.RoundID > 0 {
		query = query.Where("round_id = ?", params.RoundID)
	}
	if params.TeamID > 0 {
		query = query.Where("team_id = ?", params.TeamID)
	}
	if params.ChangeType > 0 {
		query = query.Where("change_type = ?", params.ChangeType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var ledgers []*entity.AdTokenLedger
	if err := query.Find(&ledgers).Error; err != nil {
		return nil, 0, err
	}
	return ledgers, total, nil
}

// GetLatestByTeamID 获取团队最新 Token 流水。
func (r *adTokenLedgerRepository) GetLatestByTeamID(ctx context.Context, teamID int64) (*entity.AdTokenLedger, error) {
	var ledger entity.AdTokenLedger
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at desc").
		First(&ledger).Error
	if err != nil {
		return nil, err
	}
	return &ledger, nil
}

// CountByTeamsAndChangeTypeUntil 统计多个团队在指定时间点前某类 Token 流水出现次数。
func (r *adTokenLedgerRepository) CountByTeamsAndChangeTypeUntil(ctx context.Context, competitionID int64, teamIDs []int64, changeType int16, until time.Time) (map[int64]int64, error) {
	result := make(map[int64]int64, len(teamIDs))
	if len(teamIDs) == 0 {
		return result, nil
	}
	type row struct {
		TeamID int64 `gorm:"column:team_id"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&entity.AdTokenLedger{}).
		Select("team_id, COUNT(*) AS count").
		Where("competition_id = ? AND team_id IN ? AND change_type = ? AND created_at <= ?", competitionID, teamIDs, changeType, until).
		Group("team_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result[item.TeamID] = item.Count
	}
	return result, nil
}

// AdTeamChainRepository 攻防赛队伍链数据访问接口。
type AdTeamChainRepository interface {
	Create(ctx context.Context, chain *entity.AdTeamChain) error
	GetByID(ctx context.Context, id int64) (*entity.AdTeamChain, error)
	GetByTeamID(ctx context.Context, teamID int64) (*entity.AdTeamChain, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	ListByGroupID(ctx context.Context, groupID int64) ([]*entity.AdTeamChain, error)
	ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.AdTeamChain, error)
	StopByCompetitionID(ctx context.Context, competitionID int64) error
}

type adTeamChainRepository struct {
	db *gorm.DB
}

// NewAdTeamChainRepository 创建攻防赛队伍链数据访问实例。
func NewAdTeamChainRepository(db *gorm.DB) AdTeamChainRepository {
	return &adTeamChainRepository{db: db}
}

// Create 创建队伍链记录。
func (r *adTeamChainRepository) Create(ctx context.Context, chain *entity.AdTeamChain) error {
	if chain.ID == 0 {
		chain.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(chain).Error
}

// GetByID 根据 ID 获取队伍链。
func (r *adTeamChainRepository) GetByID(ctx context.Context, id int64) (*entity.AdTeamChain, error) {
	var chain entity.AdTeamChain
	err := r.db.WithContext(ctx).First(&chain, id).Error
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// GetByTeamID 根据团队 ID 获取队伍链。
func (r *adTeamChainRepository) GetByTeamID(ctx context.Context, teamID int64) (*entity.AdTeamChain, error) {
	var chain entity.AdTeamChain
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		First(&chain).Error
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// UpdateFields 更新队伍链指定字段。
func (r *adTeamChainRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.AdTeamChain{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateStatus 更新队伍链状态。
func (r *adTeamChainRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.AdTeamChain{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// ListByGroupID 查询分组所有队伍链。
func (r *adTeamChainRepository) ListByGroupID(ctx context.Context, groupID int64) ([]*entity.AdTeamChain, error) {
	var chains []*entity.AdTeamChain
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("team_id asc").
		Find(&chains).Error
	return chains, err
}

// ListByCompetitionID 查询竞赛所有队伍链。
func (r *adTeamChainRepository) ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.AdTeamChain, error) {
	var chains []*entity.AdTeamChain
	err := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Order("group_id asc, team_id asc").
		Find(&chains).Error
	return chains, err
}

// StopByCompetitionID 将竞赛队伍链标记为已停止。
func (r *adTeamChainRepository) StopByCompetitionID(ctx context.Context, competitionID int64) error {
	return r.db.WithContext(ctx).Model(&entity.AdTeamChain{}).
		Where("competition_id = ? AND status <> ?", competitionID, enum.AdTeamChainStatusStopped).
		Updates(map[string]interface{}{
			"status":     enum.AdTeamChainStatusStopped,
			"updated_at": time.Now(),
		}).Error
}
