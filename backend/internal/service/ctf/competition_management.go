// competition_management.go
// 模块05 — CTF竞赛：公告、资源、监控与结果管理业务逻辑。

package ctf

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

func (s *competitionService) CreateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CreateCtfAnnouncementReq) (*dto.CtfAnnouncementResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return nil, err
	}
	announcement := &entity.CtfAnnouncement{
		ID:               snowflake.Generate(),
		CompetitionID:    competitionID,
		Title:            req.Title,
		Content:          req.Content,
		AnnouncementType: req.AnnouncementType,
		PublishedBy:      sc.UserID,
	}
	if req.ChallengeID != nil {
		challengeID, parseErr := parseOptionalSnowflake(req.ChallengeID)
		if parseErr != nil {
			return nil, parseErr
		}
		announcement.ChallengeID = challengeID
	}
	if err := s.announcementRepo.Create(ctx, announcement); err != nil {
		return nil, err
	}
	resp := &dto.CtfAnnouncementResp{
		ID:                   int64String(announcement.ID),
		Title:                announcement.Title,
		AnnouncementType:     announcement.AnnouncementType,
		AnnouncementTypeText: enum.GetCtfAnnouncementTypeText(announcement.AnnouncementType),
		ChallengeID:          req.ChallengeID,
		PublishedByName:      s.userQuerier.GetUserName(ctx, sc.UserID),
		CreatedAt:            timeString(announcement.CreatedAt),
	}
	if s.realtimePublisher != nil {
		var challengeTitle *string
		if announcement.ChallengeID != nil {
			if challenge, challengeErr := s.challengeRepo.GetByID(ctx, *announcement.ChallengeID); challengeErr == nil && challenge != nil {
				title := challenge.Title
				challengeTitle = &title
			}
		}
		_ = s.realtimePublisher.PublishAnnouncement(ctx, competitionID, &AnnouncementRealtimePayload{
			Event: "new_announcement",
			Announcement: dto.CtfAnnouncementItem{
				ID:                   resp.ID,
				Title:                announcement.Title,
				Content:              announcement.Content,
				AnnouncementType:     announcement.AnnouncementType,
				AnnouncementTypeText: enum.GetCtfAnnouncementTypeText(announcement.AnnouncementType),
				ChallengeID:          req.ChallengeID,
				ChallengeTitle:       challengeTitle,
				PublishedByName:      resp.PublishedByName,
				CreatedAt:            resp.CreatedAt,
			},
		})
	}
	return resp, nil
}

// ListAnnouncements 查询公告列表。
func (s *competitionService) ListAnnouncements(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CtfAnnouncementListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	announcements, err := s.announcementRepo.ListByCompetitionID(ctx, competitionID, 0)
	if err != nil {
		return nil, err
	}
	challengeTitles, err := s.loadAnnouncementChallengeTitles(ctx, announcements)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CtfAnnouncementItem, 0, len(announcements))
	for _, announcement := range announcements {
		var challengeID *string
		var challengeTitle *string
		if announcement.ChallengeID != nil {
			value := int64String(*announcement.ChallengeID)
			challengeID = &value
			if title, ok := challengeTitles[*announcement.ChallengeID]; ok {
				challengeTitle = &title
			}
		}
		items = append(items, dto.CtfAnnouncementItem{
			ID:                   int64String(announcement.ID),
			Title:                announcement.Title,
			Content:              announcement.Content,
			AnnouncementType:     announcement.AnnouncementType,
			AnnouncementTypeText: enum.GetCtfAnnouncementTypeText(announcement.AnnouncementType),
			ChallengeID:          challengeID,
			ChallengeTitle:       challengeTitle,
			PublishedByName:      s.userQuerier.GetUserName(ctx, announcement.PublishedBy),
			CreatedAt:            timeString(announcement.CreatedAt),
		})
	}
	return &dto.CtfAnnouncementListResp{List: items}, nil
}

// GetAnnouncement 获取公告详情。
func (s *competitionService) GetAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CtfAnnouncementResp, error) {
	announcement, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAnnouncementNotFoundCTF
		}
		return nil, err
	}
	competition, err := getCompetition(ctx, s.competitionRepo, announcement.CompetitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	var challengeID *string
	var challengeTitle *string
	if announcement.ChallengeID != nil {
		value := int64String(*announcement.ChallengeID)
		challengeID = &value
		if challenge, challengeErr := s.challengeRepo.GetByID(ctx, *announcement.ChallengeID); challengeErr == nil && challenge != nil {
			title := challenge.Title
			challengeTitle = &title
		}
	}
	return &dto.CtfAnnouncementResp{
		ID:                   int64String(announcement.ID),
		Title:                announcement.Title,
		Content:              announcement.Content,
		AnnouncementType:     announcement.AnnouncementType,
		AnnouncementTypeText: enum.GetCtfAnnouncementTypeText(announcement.AnnouncementType),
		ChallengeID:          challengeID,
		ChallengeTitle:       challengeTitle,
		PublishedByName:      s.userQuerier.GetUserName(ctx, announcement.PublishedBy),
		CreatedAt:            timeString(announcement.CreatedAt),
	}, nil
}

// loadAnnouncementChallengeTitles 批量加载公告关联题目名称，避免公告列表逐条查库。
func (s *competitionService) loadAnnouncementChallengeTitles(ctx context.Context, announcements []*entity.CtfAnnouncement) (map[int64]string, error) {
	challengeIDs := make([]int64, 0, len(announcements))
	seen := make(map[int64]struct{}, len(announcements))
	for _, announcement := range announcements {
		if announcement == nil || announcement.ChallengeID == nil {
			continue
		}
		if _, ok := seen[*announcement.ChallengeID]; ok {
			continue
		}
		seen[*announcement.ChallengeID] = struct{}{}
		challengeIDs = append(challengeIDs, *announcement.ChallengeID)
	}
	challenges, err := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]string, len(challenges))
	for _, challenge := range challenges {
		if challenge == nil {
			continue
		}
		result[challenge.ID] = challenge.Title
	}
	return result, nil
}

// DeleteAnnouncement 删除公告。
func (s *competitionService) DeleteAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	announcement, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAnnouncementNotFoundCTF
		}
		return err
	}
	competition, err := getCompetition(ctx, s.competitionRepo, announcement.CompetitionID)
	if err != nil {
		return err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return err
	}
	return s.announcementRepo.Delete(ctx, id)
}

// GetResourceQuota 获取竞赛资源配额。
func (s *competitionService) GetResourceQuota(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.ResourceQuotaResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	quota, err := s.quotaRepo.GetByCompetitionID(ctx, competitionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.ResourceQuotaResp{CompetitionID: int64String(competitionID)}, nil
		}
		return nil, err
	}
	return buildQuotaResp(quota), nil
}

// UpdateResourceQuota 设置竞赛资源配额。
func (s *competitionService) UpdateResourceQuota(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.UpdateResourceQuotaReq) (*dto.ResourceQuotaResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	quota := &entity.CtfResourceQuota{
		ID:            snowflake.Generate(),
		CompetitionID: competitionID,
		MaxCPU:        &req.MaxCPU,
		MaxMemory:     &req.MaxMemory,
		MaxStorage:    &req.MaxStorage,
		MaxNamespaces: &req.MaxNamespaces,
	}
	if err := s.quotaRepo.Upsert(ctx, quota); err != nil {
		return nil, err
	}
	savedQuota, err := s.quotaRepo.GetByCompetitionID(ctx, competition.ID)
	if err != nil {
		return nil, err
	}
	return buildQuotaResp(savedQuota), nil
}

// GetMonitor 获取竞赛运行监控。
func (s *competitionService) GetMonitor(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionMonitorResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	submissionStats, _ := s.submissionRepo.CountByCompetition(ctx, competitionID)
	environments, _ := s.environmentRepo.ListByCompetitionID(ctx, competitionID)
	challengeStats, _ := s.submissionRepo.CountByChallenge(ctx, competitionID)
	recentSubmissions, _ := s.submissionRepo.ListRecent(ctx, competitionID, 10)
	registrations, _ := s.registrationRepo.CountActiveByCompetitionID(ctx, competitionID)
	runningEnvironments := 0
	for _, env := range environments {
		if env.Status == enum.ChallengeEnvStatusRunning {
			runningEnvironments++
		}
	}
	activeGroupNamespaces := 0
	if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
		groups, groupErr := s.adGroupRepo.ListByCompetitionID(ctx, competitionID)
		if groupErr == nil {
			for _, group := range groups {
				if group != nil && group.Namespace != nil && strings.TrimSpace(*group.Namespace) != "" {
					activeGroupNamespaces++
				}
			}
		}
	}
	quota, _ := s.quotaRepo.GetByCompetitionID(ctx, competitionID)
	totalEnvironments := len(environments)
	if activeGroupNamespaces > 0 {
		totalEnvironments += activeGroupNamespaces
		runningEnvironments += activeGroupNamespaces
	}
	resp := &dto.CompetitionMonitorResp{
		CompetitionID:   int64String(competitionID),
		CompetitionType: competition.CompetitionType,
		Status:          competition.Status,
		StatusText:      enum.GetCompetitionStatusText(competition.Status),
		Overview: dto.CompetitionMonitorOverview{
			RegisteredTeams:     int(registrations),
			ActiveTeams:         int(registrations),
			TotalSubmissions:    int(valueOrZeroSubmissionStat(submissionStats, true)),
			CorrectSubmissions:  int(valueOrZeroSubmissionStat(submissionStats, false)),
			TotalEnvironments:   totalEnvironments,
			RunningEnvironments: runningEnvironments,
		},
	}
	if quota != nil {
		resp.ResourceUsage = dto.CompetitionMonitorResourceUsage{
			CPUUsed:        quota.UsedCPU,
			CPUMax:         derefString(quota.MaxCPU),
			MemoryUsed:     quota.UsedMemory,
			MemoryMax:      derefString(quota.MaxMemory),
			NamespacesUsed: quota.CurrentNamespaces,
			NamespacesMax:  derefInt(quota.MaxNamespaces),
		}
	} else if activeGroupNamespaces > 0 || runningEnvironments > 0 {
		resp.ResourceUsage = dto.CompetitionMonitorResourceUsage{
			NamespacesUsed: runningEnvironments,
		}
	}
	resp.ChallengeStats = s.buildMonitorChallengeStats(ctx, competitionID, challengeStats, environments)
	for _, submission := range recentSubmissions {
		challengeTitle := ""
		if challenge, challengeErr := s.challengeRepo.GetByID(ctx, submission.ChallengeID); challengeErr == nil {
			challengeTitle = challenge.Title
		}
		teamName := ""
		if team, teamErr := s.teamRepo.GetByID(ctx, submission.TeamID); teamErr == nil {
			teamName = team.Name
		}
		resp.RecentSubmissions = append(resp.RecentSubmissions, dto.CompetitionMonitorRecentSubmission{
			TeamName:       teamName,
			ChallengeTitle: challengeTitle,
			IsCorrect:      submission.IsCorrect,
			SubmittedAt:    timeString(submission.CreatedAt),
		})
	}
	return resp, nil
}

// GetStatistics 获取竞赛统计数据。
func (s *competitionService) GetStatistics(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionStatisticsResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	return s.buildCompetitionStatisticsResp(ctx, competitionID, competition.StartAt, competition.EndAt)
}

// GetResults 获取竞赛最终结果。
func (s *competitionService) GetResults(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionResultsResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	finalLeaderboard, err := s.GetFinalLeaderboard(ctx, sc, competitionID)
	if err != nil {
		return nil, err
	}
	statistics, err := s.buildCompetitionStatisticsResp(ctx, competitionID, competition.StartAt, competition.EndAt)
	if err != nil {
		return nil, err
	}
	return &dto.CompetitionResultsResp{
		CompetitionID: finalLeaderboard.CompetitionID,
		Summary: dto.CompetitionResultsSummary{
			TotalTeams:        statistics.Summary.TotalTeams,
			TotalParticipants: statistics.Summary.TotalParticipants,
			TotalSubmissions:  statistics.Summary.TotalSubmissions,
			TotalCorrect:      statistics.Summary.TotalCorrect,
			OverallSolveRate:  statistics.Summary.OverallSolveRate,
			AverageScore:      statistics.Summary.AverageScore,
			HighestScore:      statistics.Summary.HighestScore,
			LowestScore:       statistics.Summary.LowestScore,
		},
		Rankings: finalLeaderboard.Rankings,
	}, nil
}

// GetAdminOverview 获取全平台竞赛概览。
func (s *competitionService) GetAdminOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.CtfAdminOverviewResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	overview, err := s.competitionRepo.Overview(ctx)
	if err != nil {
		return nil, err
	}
	competitions, _, err := s.competitionRepo.List(ctx, &ctfrepo.CompetitionListParams{Page: 1, PageSize: 10000})
	if err != nil {
		return nil, err
	}
	quotas, err := s.quotaRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &dto.CtfAdminOverviewResp{
		TotalCompetitions:    int(overview.TotalCompetitions),
		RunningCompetitions:  int(overview.RunningCompetitions),
		UpcomingCompetitions: int(overview.UpcomingCompetitions),
		Alerts:               []dto.AdminCompetitionAlert{},
	}
	namespacesActive := 0
	totalParticipants := 0
	for _, competition := range competitions {
		participants, _ := s.competitionRepo.CountParticipants(ctx, competition.ID)
		totalParticipants += int(participants)
		environments, _ := s.environmentRepo.ListByCompetitionID(ctx, competition.ID)
		runningEnvironments := 0
		for _, env := range environments {
			if env.Status == enum.ChallengeEnvStatusRunning {
				runningEnvironments++
			}
		}
		activeGroupNamespaces := 0
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			groups, groupErr := s.adGroupRepo.ListByCompetitionID(ctx, competition.ID)
			if groupErr == nil {
				for _, group := range groups {
					if group != nil && group.Namespace != nil && strings.TrimSpace(*group.Namespace) != "" {
						activeGroupNamespaces++
					}
				}
			}
			runningEnvironments += activeGroupNamespaces
		}
		quota := findCompetitionQuota(quotas, competition.ID)
		if quota != nil {
			namespacesActive += quota.CurrentNamespaces
			runningEnvironments = quota.CurrentNamespaces
		} else {
			namespacesActive += runningEnvironments
		}
		if competition.Status == enum.CompetitionStatusRunning {
			registeredTeams, _ := s.registrationRepo.CountActiveByCompetitionID(ctx, competition.ID)
			resp.RunningCompetitionsList = append(resp.RunningCompetitionsList, dto.RunningCompetitionOverviewItem{
				ID:                  int64String(competition.ID),
				Title:               competition.Title,
				CompetitionType:     competition.CompetitionType,
				CompetitionTypeText: enum.GetCompetitionTypeText(competition.CompetitionType),
				Status:              competition.Status,
				StatusText:          enum.GetCompetitionStatusText(competition.Status),
				Teams:               int(registeredTeams),
				EnvironmentsRunning: runningEnvironments,
				StartAt:             optionalTimeString(competition.StartAt),
				EndAt:               optionalTimeString(competition.EndAt),
			})
		}
		s.appendCompetitionResourceAlert(resp, competition, quotas)
		if competition.EndAt != nil && competition.EndAt.Before(time.Now()) && competition.Status == enum.CompetitionStatusRunning {
			resp.Alerts = append(resp.Alerts, dto.AdminCompetitionAlert{
				CompetitionID: int64String(competition.ID),
				Type:          "status_warning",
				Message:       fmt.Sprintf("竞赛 %s 已超过结束时间但状态仍为进行中", competition.Title),
				CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
	resp.TotalParticipants = totalParticipants
	totalCPUUsed, totalMemoryUsed := aggregateQuotaUsage(quotas)
	resp.TotalResourceUsage = dto.AdminTotalResourceUsage{
		CPUUsed:          totalCPUUsed,
		MemoryUsed:       totalMemoryUsed,
		NamespacesActive: namespacesActive,
	}
	return resp, nil
}

// aggregateQuotaUsage 汇总全部竞赛资源配额中的已用 CPU 和内存。
func aggregateQuotaUsage(quotas []*entity.CtfResourceQuota) (string, string) {
	totalCPU := resource.MustParse("0")
	totalMemory := resource.MustParse("0")
	for _, quota := range quotas {
		if quota == nil {
			continue
		}
		addQuotaQuantity(&totalCPU, quota.UsedCPU)
		addQuotaQuantity(&totalMemory, quota.UsedMemory)
	}
	return formatCPUUsage(totalCPU), totalMemory.String()
}

// appendCompetitionResourceAlert 在竞赛资源使用超过阈值时追加平台告警。
func (s *competitionService) appendCompetitionResourceAlert(resp *dto.CtfAdminOverviewResp, competition *entity.Competition, quotas []*entity.CtfResourceQuota) {
	if resp == nil || competition == nil || competition.Status != enum.CompetitionStatusRunning {
		return
	}
	quota := findCompetitionQuota(quotas, competition.ID)
	if quota == nil || !isQuotaUsageWarning(quota) {
		return
	}
	resp.Alerts = append(resp.Alerts, dto.AdminCompetitionAlert{
		CompetitionID: int64String(competition.ID),
		Type:          "resource_warning",
		Message:       fmt.Sprintf("竞赛 '%s' 资源使用率超过80%%", competition.Title),
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	})
}

// findCompetitionQuota 按竞赛 ID 从配额列表中定位对应配额记录。
func findCompetitionQuota(quotas []*entity.CtfResourceQuota, competitionID int64) *entity.CtfResourceQuota {
	for _, quota := range quotas {
		if quota != nil && quota.CompetitionID == competitionID {
			return quota
		}
	}
	return nil
}

// isQuotaUsageWarning 判断竞赛 CPU、内存或命名空间是否达到资源告警阈值。
func isQuotaUsageWarning(quota *entity.CtfResourceQuota) bool {
	if quota == nil {
		return false
	}
	return isQuantityUsageThresholdReached(quota.UsedCPU, quota.MaxCPU) ||
		isQuantityUsageThresholdReached(quota.UsedMemory, quota.MaxMemory) ||
		isNamespaceUsageThresholdReached(quota.CurrentNamespaces, quota.MaxNamespaces)
}

// isQuantityUsageThresholdReached 判断定量资源的已用比例是否达到 80% 告警阈值。
func isQuantityUsageThresholdReached(used string, max *string) bool {
	if max == nil {
		return false
	}
	usedQuantity, usedOK := parseQuotaQuantity(used)
	maxQuantity, maxOK := parseQuotaQuantity(*max)
	if !usedOK || !maxOK || maxQuantity.Sign() <= 0 {
		return false
	}
	usedMilli := usedQuantity.MilliValue()
	maxMilli := maxQuantity.MilliValue()
	if maxMilli <= 0 {
		return false
	}
	return usedMilli*100 >= maxMilli*80
}

// isNamespaceUsageThresholdReached 判断命名空间使用量是否达到 80% 告警阈值。
func isNamespaceUsageThresholdReached(used int, max *int) bool {
	if max == nil || *max <= 0 {
		return false
	}
	return used*100 >= *max*80
}

// addQuotaQuantity 将字符串资源值累加到总量中，无法解析的值直接忽略。
func addQuotaQuantity(total *resource.Quantity, raw string) {
	if total == nil || strings.TrimSpace(raw) == "" {
		return
	}
	quantity, ok := parseQuotaQuantity(raw)
	if !ok {
		return
	}
	total.Add(quantity)
}

// parseQuotaQuantity 解析资源配额字符串，供统计与告警计算复用。
func parseQuotaQuantity(raw string) (resource.Quantity, bool) {
	quantity, err := resource.ParseQuantity(strings.TrimSpace(raw))
	if err != nil {
		return resource.Quantity{}, false
	}
	return quantity, true
}

// formatCPUUsage 将 CPU 总量格式化为接口展示字符串。
func formatCPUUsage(total resource.Quantity) string {
	value := total.AsApproximateFloat64()
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// ensureCompetitionReadable 校验当前上下文是否可读取竞赛。
