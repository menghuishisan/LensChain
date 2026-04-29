// common.go
// 模块05 — CTF竞赛：service 层公共类型与辅助函数。
// 该文件统一收口跨模块查询接口、时间与 JSON 解析、分页封装、排行榜排序和常用的 entity→DTO
// 转换基础能力，避免各功能域 service 文件重复实现。

package ctf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// UserSummary 表示 CTF 模块所需的用户最小摘要。
type UserSummary struct {
	UserID    int64
	Name      string
	StudentNo string
}

// UserSummaryQuerier 跨模块用户查询接口。
// CTF service 只依赖接口，不直接依赖认证模块具体实现。
type UserSummaryQuerier interface {
	GetUserSummary(ctx context.Context, userID int64) *UserSummary
	GetUserName(ctx context.Context, userID int64) string
	GetUserSummaries(ctx context.Context, userIDs []int64) map[int64]*UserSummary
}

// SchoolNameQuerier 跨模块学校名称查询接口。
type SchoolNameQuerier interface {
	GetSchoolName(ctx context.Context, schoolID int64) string
}

// NotificationEventDispatcher 跨模块接口：向模块07发送站内信事件。
// 模块05 只负责声明事件和接收者范围，通知模板与偏好过滤由模块07负责。
type NotificationEventDispatcher interface {
	DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error
}

// CompetitionAudienceResolver 跨模块接口：解析竞赛发布时可接收通知的学生集合。
// 平台级竞赛面向全平台学生，校级竞赛只面向本校学生。
type CompetitionAudienceResolver interface {
	ListCompetitionPublishStudentIDs(ctx context.Context, scope int16, schoolID *int64) ([]int64, error)
}

// CTFModule 表示模块05 service 聚合根。
// Handler 与 init 装配层通过它访问各功能域 service，避免散落的依赖注入。
type CTFModule struct {
	Competition CompetitionService
	Challenge   ChallengeService
	Team        TeamService
	Battle      BattleService
	Environment EnvironmentService
}

// paginationResp 构建分页响应 DTO。
func paginationResp(page, pageSize int, total int64) dto.PaginationResp {
	page, pageSize = pagination.NormalizeValues(page, pageSize)
	return dto.PaginationResp{
		Page:     page,
		PageSize: pageSize,
		Total:    int(total),
	}
}

// parseOptionalTime 解析可选 RFC3339 时间。
func parseOptionalTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, errcode.ErrCompetitionTimeInvalid
	}
	return &parsed, nil
}

// parseOptionalSnowflake 解析可选雪花 ID。
func parseOptionalSnowflake(value *string) (*int64, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	id, err := snowflake.ParseString(strings.TrimSpace(*value))
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	return &id, nil
}

// mustJSON 将结构体编码为 datatypes.JSON。
func mustJSON(v any) datatypes.JSON {
	if v == nil {
		return datatypes.JSON([]byte("null"))
	}
	raw, _ := json.Marshal(v)
	return datatypes.JSON(raw)
}

// decodeJSON 解码 JSONB 字段到目标结构。
func decodeJSON[T any](raw datatypes.JSON, target *T) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, target)
}

// stringPtr 返回字符串指针。
func stringPtr(v string) *string {
	return &v
}

// intPtr 返回整型指针。
func intPtr(v int) *int {
	return &v
}

// int16Ptr 返回 int16 指针。
func int16Ptr(v int16) *int16 {
	return &v
}

// int64Ptr 返回 int64 指针。
func int64Ptr(v int64) *int64 {
	return &v
}

// int64String 将雪花 ID 转为字符串。
func int64String(id int64) string {
	return strconv.FormatInt(id, 10)
}

// buildCompetitionStatusCacheKey 构建竞赛状态缓存键。
func buildCompetitionStatusCacheKey(competitionID int64) string {
	return fmt.Sprintf("%s%d:status", cache.KeyCTFCompStatus, competitionID)
}

// buildCompetitionLeaderboardCacheKey 构建解题赛排行榜缓存键。
func buildCompetitionLeaderboardCacheKey(competitionID int64) string {
	return fmt.Sprintf("%s%d", cache.KeyCTFLeaderboard, competitionID)
}

// buildCompetitionFrozenCacheKey 构建冻结榜缓存键。
func buildCompetitionFrozenCacheKey(competitionID int64) string {
	return fmt.Sprintf("%s%d:frozen", cache.KeyCTFFrozen, competitionID)
}

// buildAdLeaderboardCacheKey 构建攻防赛分组排行榜缓存键。
func buildAdLeaderboardCacheKey(competitionID, groupID int64) string {
	return fmt.Sprintf("%s%d:%d", cache.KeyCTFADLeaderboard, competitionID, groupID)
}

// buildAdRoundCacheKey 构建攻防赛分组当前回合缓存键。
func buildAdRoundCacheKey(competitionID, groupID int64) string {
	return fmt.Sprintf("%s%d:%d", cache.KeyCTFADRound, competitionID, groupID)
}

// buildChallengeScoreCacheKey 构建解题赛题目动态分值缓存键。
func buildChallengeScoreCacheKey(competitionID, challengeID int64) string {
	return fmt.Sprintf("%s%d:%d", cache.KeyCTFScore, competitionID, challengeID)
}

// buildAdTokenCacheKey 构建攻防赛队伍 Token 余额缓存键。
func buildAdTokenCacheKey(competitionID, teamID int64) string {
	return fmt.Sprintf("%s%d:%d", cache.KeyCTFADToken, competitionID, teamID)
}

// challengeScoreCachePayload 表示题目动态分值缓存结构。
type challengeScoreCachePayload struct {
	CurrentScore int `json:"current_score"`
	SolveCount   int `json:"solve_count"`
	BaseScore    int `json:"base_score"`
}

// adRoundCachePayload 表示攻防赛分组当前回合缓存结构。
type adRoundCachePayload struct {
	RoundNumber int    `json:"round_number"`
	Phase       int16  `json:"phase"`
	PhaseEndAt  string `json:"phase_end_at"`
}

// writeCompetitionStatusCache 将竞赛状态写入 Redis 热缓存，供实时接口与状态流转快速命中。
func writeCompetitionStatusCache(ctx context.Context, competitionID int64, status int16) {
	if err := cache.Set(ctx, buildCompetitionStatusCacheKey(competitionID), fmt.Sprintf("%d", status), 24*time.Hour); err != nil {
		logger.L.Debug("写入竞赛状态缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Error(err),
		)
	}
}

// writeCompetitionFrozenCache 写入或清理解题赛冻结标记缓存。
func writeCompetitionFrozenCache(ctx context.Context, competitionID int64, isFrozen bool) {
	key := buildCompetitionFrozenCacheKey(competitionID)
	if !isFrozen {
		_ = cache.Del(ctx, key)
		return
	}
	if err := cache.Set(ctx, key, "1", 24*time.Hour); err != nil {
		logger.L.Debug("写入竞赛冻结标记缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Error(err),
		)
	}
}

// writeChallengeScoreCache 写入题目动态分值缓存。
func writeChallengeScoreCache(ctx context.Context, competitionID, challengeID int64, currentScore, solveCount, baseScore int) {
	payload := challengeScoreCachePayload{
		CurrentScore: currentScore,
		SolveCount:   solveCount,
		BaseScore:    baseScore,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		logger.L.Debug("编码题目动态分值缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Int64("challenge_id", challengeID),
			zap.Error(err),
		)
		return
	}
	if err := cache.Set(ctx, buildChallengeScoreCacheKey(competitionID, challengeID), string(raw), 0); err != nil {
		logger.L.Debug("写入题目动态分值缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Int64("challenge_id", challengeID),
			zap.Error(err),
		)
	}
}

// readChallengeScoreCache 读取题目动态分值缓存。
func readChallengeScoreCache(ctx context.Context, competitionID, challengeID int64) (*challengeScoreCachePayload, bool) {
	raw, err := cache.GetString(ctx, buildChallengeScoreCacheKey(competitionID, challengeID))
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var payload challengeScoreCachePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, false
	}
	return &payload, true
}

// writeAdTokenBalanceCache 写入攻防赛队伍 Token 余额缓存。
func writeAdTokenBalanceCache(ctx context.Context, competitionID, teamID int64, balance int) {
	if err := cache.Set(ctx, buildAdTokenCacheKey(competitionID, teamID), strconv.Itoa(balance), 0); err != nil {
		logger.L.Debug("写入攻防赛 Token 缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Int64("team_id", teamID),
			zap.Error(err),
		)
	}
}

// readAdTokenBalanceCache 读取攻防赛队伍 Token 余额缓存。
func readAdTokenBalanceCache(ctx context.Context, competitionID, teamID int64) (int, bool) {
	raw, err := cache.GetString(ctx, buildAdTokenCacheKey(competitionID, teamID))
	if err != nil || strings.TrimSpace(raw) == "" {
		return 0, false
	}
	value, parseErr := strconv.Atoi(strings.TrimSpace(raw))
	if parseErr != nil {
		return 0, false
	}
	return value, true
}

// writeAdRoundCache 写入攻防赛分组当前回合状态缓存。
func writeAdRoundCache(ctx context.Context, competitionID, groupID int64, roundNumber int, phase int16, phaseEndAt time.Time) {
	payload := adRoundCachePayload{
		RoundNumber: roundNumber,
		Phase:       phase,
		PhaseEndAt:  timeString(phaseEndAt),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		logger.L.Debug("编码攻防赛回合缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Int64("group_id", groupID),
			zap.Error(err),
		)
		return
	}
	if err := cache.Set(ctx, buildAdRoundCacheKey(competitionID, groupID), string(raw), 0); err != nil {
		logger.L.Debug("写入攻防赛回合缓存失败",
			zap.Int64("competition_id", competitionID),
			zap.Int64("group_id", groupID),
			zap.Error(err),
		)
	}
}

// clearCompetitionCacheData 清理竞赛归档后不再需要保留的 Redis 业务缓存与热缓存。
func clearCompetitionCacheData(
	ctx context.Context,
	adGroupRepo ctfrepo.AdGroupRepository,
	adRoundRepo ctfrepo.AdRoundRepository,
	competitionID int64,
) {
	patterns := []string{
		buildCompetitionLeaderboardCacheKey(competitionID),
		buildCompetitionFrozenCacheKey(competitionID),
		buildCompetitionStatusCacheKey(competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFRateLimit, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFFailCount, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFCooldown, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFScore, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFSolved, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFADLeaderboard, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFADRound, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFADToken, competitionID),
		fmt.Sprintf("%s%d:*", cache.KeyCTFADExploit, competitionID),
	}
	if adGroupRepo != nil {
		groups, err := adGroupRepo.ListByCompetitionID(ctx, competitionID)
		if err == nil {
			for _, group := range groups {
				patterns = append(patterns, buildAdLeaderboardCacheKey(competitionID, group.ID))
				patterns = append(patterns, buildAdRoundCacheKey(competitionID, group.ID))
				if adRoundRepo == nil {
					continue
				}
				rounds, roundErr := adRoundRepo.ListByGroupID(ctx, group.ID)
				if roundErr != nil {
					continue
				}
				for _, round := range rounds {
					if round == nil {
						continue
					}
					patterns = append(patterns, fmt.Sprintf("%s%d:*", cache.KeyCTFADAttackLock, round.ID))
				}
			}
		}
	}
	if err := cache.DeleteByPatterns(ctx, patterns...); err != nil {
		logger.L.Debug("清理竞赛缓存数据失败",
			zap.Int64("competition_id", competitionID),
			zap.Error(err),
		)
	}
}

// derefString 解引用字符串指针，为空时返回空串。
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// derefInt 解引用整型指针，为空时返回 0。
func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

// timeString 将时间格式化为 RFC3339 字符串。
func timeString(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// optionalTimeString 将可选时间格式化为 RFC3339 字符串。
func optionalTimeString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	value := timeString(*t)
	return &value
}

// averageInt 计算整数数组平均值。
func averageInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	total := 0
	for _, value := range values {
		total += value
	}
	return total / len(values)
}

// maxInt 计算整数数组最大值。
func maxInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}

// minInt 计算整数数组最小值。
func minInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}

// derefBool 解引用布尔指针，为空时返回 false。
func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

// getCompetition 获取竞赛并将 not found 统一转成 CTF 错误码。
func getCompetition(ctx context.Context, repo ctfrepo.CompetitionRepository, id int64) (*entity.Competition, error) {
	competition, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrCompetitionNotFound
		}
		return nil, err
	}
	return competition, nil
}

// getChallenge 获取题目并将 not found 统一转成 CTF 错误码。
func getChallenge(ctx context.Context, repo ctfrepo.ChallengeRepository, id int64) (*entity.Challenge, error) {
	challenge, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrChallengeNotFound
		}
		return nil, err
	}
	return challenge, nil
}

// sortLeaderboard 对排行榜按分数、解题数、最后解题时间和团队 ID 排序。
func sortLeaderboard(items []dto.LeaderboardRankingItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TokenBalance != nil || items[j].TokenBalance != nil {
			leftBalance := derefInt(items[i].TokenBalance)
			rightBalance := derefInt(items[j].TokenBalance)
			if leftBalance != rightBalance {
				return leftBalance > rightBalance
			}
			return items[i].TeamID < items[j].TeamID
		}
		leftScore := 0
		rightScore := 0
		if items[i].Score != nil {
			leftScore = *items[i].Score
		}
		if items[j].Score != nil {
			rightScore = *items[j].Score
		}
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		leftSolve := 0
		rightSolve := 0
		if items[i].SolveCount != nil {
			leftSolve = *items[i].SolveCount
		}
		if items[j].SolveCount != nil {
			rightSolve = *items[j].SolveCount
		}
		if leftSolve != rightSolve {
			return leftSolve > rightSolve
		}
		left := ""
		right := ""
		if items[i].LastSolveAt != nil {
			left = *items[i].LastSolveAt
		}
		if items[j].LastSolveAt != nil {
			right = *items[j].LastSolveAt
		}
		if left != right {
			if left == "" {
				return false
			}
			if right == "" {
				return true
			}
			return left < right
		}
		return items[i].TeamID < items[j].TeamID
	})
}

// buildUserBrief 构建用户摘要 DTO。
func buildUserBrief(ctx context.Context, userQuerier UserSummaryQuerier, userID int64) *dto.CTFUserBrief {
	if userQuerier == nil || userID == 0 {
		return nil
	}
	summary := userQuerier.GetUserSummary(ctx, userID)
	if summary == nil {
		return &dto.CTFUserBrief{ID: int64String(userID)}
	}
	return &dto.CTFUserBrief{
		ID:   int64String(summary.UserID),
		Name: summary.Name,
	}
}

// validateCompetitionWindow 校验竞赛时间窗口。
func validateCompetitionWindow(registrationStartAt, registrationEndAt, startAt, endAt, freezeAt *time.Time) error {
	if registrationStartAt != nil && registrationEndAt != nil && !registrationStartAt.Before(*registrationEndAt) {
		return errcode.ErrCompetitionTimeInvalid
	}
	if registrationEndAt != nil && startAt != nil && !registrationEndAt.Before(*startAt) {
		return errcode.ErrCompetitionTimeInvalid
	}
	if startAt != nil && endAt != nil && !startAt.Before(*endAt) {
		return errcode.ErrCompetitionTimeInvalid
	}
	if freezeAt != nil && endAt != nil && freezeAt.After(*endAt) {
		return errcode.ErrCompetitionTimeInvalid
	}
	return nil
}

// validateCompetitionTypeConfig 校验竞赛类型与专属配置的一致性。
func validateCompetitionTypeConfig(competitionType int16, jeopardyConfig *dto.JeopardyCompetitionConfig, adConfig *dto.ADCompetitionConfig) error {
	if jeopardyConfig != nil && adConfig != nil {
		return errcode.ErrCompetitionConfigRequired.WithMessage("解题赛配置和攻防赛配置不能同时填写")
	}
	switch competitionType {
	case enum.CompetitionTypeJeopardy:
		if jeopardyConfig == nil {
			return errcode.ErrCompetitionConfigRequired.WithMessage("解题赛必须配置jeopardy_config")
		}
	case enum.CompetitionTypeAttackDefense:
		if adConfig == nil {
			return errcode.ErrCompetitionConfigRequired.WithMessage("攻防对抗赛必须配置ad_config")
		}
	default:
		return errcode.ErrCompetitionTypeInvalid
	}
	return nil
}

// validateCompetitionUpdateConfig 校验竞赛编辑时不能写入非当前类型的专属配置。
func validateCompetitionUpdateConfig(competition *entity.Competition, req *dto.UpdateCompetitionReq) error {
	if competition == nil || req == nil {
		return nil
	}
	if req.JeopardyConfig != nil && req.AdConfig != nil {
		return errcode.ErrCompetitionConfigRequired.WithMessage("解题赛配置和攻防赛配置不能同时填写")
	}
	switch competition.CompetitionType {
	case enum.CompetitionTypeJeopardy:
		if req.AdConfig != nil {
			return errcode.ErrCompetitionConfigRequired.WithMessage("解题赛不能填写ad_config")
		}
	case enum.CompetitionTypeAttackDefense:
		if req.JeopardyConfig != nil {
			return errcode.ErrCompetitionConfigRequired.WithMessage("攻防对抗赛不能填写jeopardy_config")
		}
	}
	return nil
}

// buildCompetitionListItem 构建竞赛列表项。
func buildCompetitionListItem(
	ctx context.Context,
	userQuerier UserSummaryQuerier,
	competition *entity.Competition,
	registeredTeams int64,
	challengeCount int64,
) dto.CompetitionListItem {
	item := dto.CompetitionListItem{
		ID:                  int64String(competition.ID),
		Title:               competition.Title,
		BannerURL:           competition.BannerURL,
		CompetitionType:     competition.CompetitionType,
		CompetitionTypeText: enum.GetCompetitionTypeText(competition.CompetitionType),
		Scope:               competition.Scope,
		ScopeText:           enum.GetCompetitionScopeText(competition.Scope),
		TeamMode:            competition.TeamMode,
		TeamModeText:        enum.GetTeamModeText(competition.TeamMode),
		MaxTeamSize:         competition.MaxTeamSize,
		Status:              competition.Status,
		StatusText:          enum.GetCompetitionStatusText(competition.Status),
		RegisteredTeams:     int(registeredTeams),
		MaxTeams:            competition.MaxTeams,
		ChallengeCount:      int(challengeCount),
		RegistrationStartAt: optionalTimeString(competition.RegistrationStartAt),
		RegistrationEndAt:   optionalTimeString(competition.RegistrationEndAt),
		StartAt:             optionalTimeString(competition.StartAt),
		EndAt:               optionalTimeString(competition.EndAt),
	}
	if userQuerier != nil {
		item.CreatedByName = userQuerier.GetUserName(ctx, competition.CreatedBy)
	}
	return item
}

// buildCompetitionDetail 构建竞赛详情响应。
func buildCompetitionDetail(
	ctx context.Context,
	userQuerier UserSummaryQuerier,
	competition *entity.Competition,
	registeredTeams int64,
	challengeCount int64,
) (*dto.CompetitionDetailResp, error) {
	resp := &dto.CompetitionDetailResp{
		ID:                  int64String(competition.ID),
		Title:               competition.Title,
		Description:         competition.Description,
		BannerURL:           competition.BannerURL,
		CompetitionType:     competition.CompetitionType,
		CompetitionTypeText: enum.GetCompetitionTypeText(competition.CompetitionType),
		Scope:               competition.Scope,
		ScopeText:           enum.GetCompetitionScopeText(competition.Scope),
		TeamMode:            competition.TeamMode,
		TeamModeText:        enum.GetTeamModeText(competition.TeamMode),
		MaxTeamSize:         competition.MaxTeamSize,
		MinTeamSize:         competition.MinTeamSize,
		MaxTeams:            competition.MaxTeams,
		Status:              competition.Status,
		StatusText:          enum.GetCompetitionStatusText(competition.Status),
		RegistrationStartAt: optionalTimeString(competition.RegistrationStartAt),
		RegistrationEndAt:   optionalTimeString(competition.RegistrationEndAt),
		StartAt:             optionalTimeString(competition.StartAt),
		EndAt:               optionalTimeString(competition.EndAt),
		FreezeAt:            optionalTimeString(competition.FreezeAt),
		Rules:               competition.Rules,
		RegisteredTeams:     int(registeredTeams),
		ChallengeCount:      int(challengeCount),
		CreatedBy:           buildUserBrief(ctx, userQuerier, competition.CreatedBy),
		CreatedAt:           timeString(competition.CreatedAt),
	}
	if len(competition.JeopardyConfig) > 0 {
		var cfg dto.JeopardyCompetitionConfig
		if err := decodeJSON(competition.JeopardyConfig, &cfg); err != nil {
			return nil, fmt.Errorf("decode jeopardy config: %w", err)
		}
		resp.JeopardyConfig = &cfg
	}
	if len(competition.AdConfig) > 0 {
		var cfg dto.ADCompetitionConfig
		if err := decodeJSON(competition.AdConfig, &cfg); err != nil {
			return nil, fmt.Errorf("decode ad config: %w", err)
		}
		resp.AdConfig = &cfg
	}
	return resp, nil
}
