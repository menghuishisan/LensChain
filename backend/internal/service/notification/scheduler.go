// scheduler.go
// 模块07 — 通知与消息：定时任务执行器。
// 负责定时公告发布、截止提醒、竞赛提醒、历史通知清理和未读缓存对账。

package notification

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/logger"
	notificationrepo "github.com/lenschain/backend/internal/repository/notification"
)

const (
	assignmentReminderWindow          = 24 * time.Hour
	competitionRegistrationWindow     = 48 * time.Hour
	competitionStartingReminderWindow = 1 * time.Hour
	expiredReadCleanupWindow          = 180 * 24 * time.Hour
)

// UnreadCounterSyncer 声明未读计数同步的最小能力。
type UnreadCounterSyncer interface {
	SyncUnreadCountForUser(ctx context.Context, userID int64) error
}

// Scheduler 模块07定时任务执行器。
type Scheduler struct {
	dispatcher       EventDispatcher
	unreadSyncer     UnreadCounterSyncer
	announcer        AnnouncementBroadcaster
	announcementRepo notificationSchedulerAnnouncementRepo
	notificationRepo notificationSchedulerNotificationRepo
	sourceRepo       notificationSchedulerSourceRepo
}

// AnnouncementBroadcaster 声明定时公告发布后的广播能力。
type AnnouncementBroadcaster interface {
	BroadcastAnnouncementPublished(announcement *entity.SystemAnnouncement)
}

// notificationSchedulerAnnouncementRepo 声明定时公告发布所需的最小仓储能力。
type notificationSchedulerAnnouncementRepo interface {
	ListPublishDue(ctx context.Context, now time.Time, limit int) ([]*entity.SystemAnnouncement, error)
	Publish(ctx context.Context, id int64, publishedAt time.Time) error
}

// notificationSchedulerNotificationRepo 声明历史通知清理所需的最小仓储能力。
type notificationSchedulerNotificationRepo interface {
	ListExpiredRead(ctx context.Context, before time.Time, limit int) ([]*entity.Notification, error)
	DeleteByIDs(ctx context.Context, ids []int64) error
}

// notificationSchedulerSourceRepo 声明提醒扫描所需的最小只读数据能力。
type notificationSchedulerSourceRepo interface {
	ListAssignmentDeadlineCandidates(ctx context.Context, now, deadline time.Time) ([]*notificationrepo.AssignmentReminderCandidate, error)
	ListAssignmentUnsubmittedStudentIDs(ctx context.Context, assignmentID int64) ([]int64, error)
	ListCompetitionStartingCandidates(ctx context.Context, now, deadline time.Time) ([]*notificationrepo.CompetitionReminderCandidate, error)
	ListCompetitionRegistrationDeadlineCandidates(ctx context.Context, now, deadline time.Time) ([]*notificationrepo.CompetitionReminderCandidate, error)
	ListCompetitionRegisteredStudentIDs(ctx context.Context, competitionID int64) ([]int64, error)
	ListCompetitionUnregisteredStudentIDs(ctx context.Context, competitionID int64, schoolID int64) ([]int64, error)
}

// NewScheduler 创建模块07定时任务执行器。
func NewScheduler(
	dispatcher EventDispatcher,
	unreadSyncer UnreadCounterSyncer,
	announcer AnnouncementBroadcaster,
	announcementRepo notificationSchedulerAnnouncementRepo,
	notificationRepo notificationSchedulerNotificationRepo,
	sourceRepo notificationSchedulerSourceRepo,
) *Scheduler {
	return &Scheduler{
		dispatcher:       dispatcher,
		unreadSyncer:     unreadSyncer,
		announcer:        announcer,
		announcementRepo: announcementRepo,
		notificationRepo: notificationRepo,
		sourceRepo:       sourceRepo,
	}
}

// RunScan 执行模块07扫描类定时任务。
func (s *Scheduler) RunScan() {
	ctx := context.Background()
	now := time.Now().UTC()
	s.publishScheduledAnnouncements(ctx, now)
	s.sendAssignmentDeadlineReminders(ctx, now)
	s.sendCompetitionRegistrationReminders(ctx, now)
	s.sendCompetitionStartingReminders(ctx, now)
}

// RunCleanup 执行模块07清理类定时任务。
func (s *Scheduler) RunCleanup() {
	ctx := context.Background()
	now := time.Now().UTC()
	s.cleanupExpiredReadNotifications(ctx, now)
	s.reconcileUnreadCache(ctx)
}

// publishScheduledAnnouncements 自动发布到达发布时间的系统公告。
func (s *Scheduler) publishScheduledAnnouncements(ctx context.Context, now time.Time) {
	if s.announcementRepo == nil {
		return
	}
	items, err := s.announcementRepo.ListPublishDue(ctx, now, 100)
	if err != nil {
		logger.L.Error("扫描定时公告失败", zap.Error(err))
		return
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if err := s.announcementRepo.Publish(ctx, item.ID, now); err != nil {
			logger.L.Error("自动发布系统公告失败", zap.Int64("announcement_id", item.ID), zap.Error(err))
			continue
		}
		item.PublishedAt = &now
		item.Status = enum.SystemAnnouncementStatusPublished
		if s.announcer != nil {
			s.announcer.BroadcastAnnouncementPublished(item)
		}
	}
}

// sendAssignmentDeadlineReminders 扫描并发送作业截止提醒。
func (s *Scheduler) sendAssignmentDeadlineReminders(ctx context.Context, now time.Time) {
	if s.dispatcher == nil || s.sourceRepo == nil {
		return
	}
	items, err := s.sourceRepo.ListAssignmentDeadlineCandidates(ctx, now, now.Add(assignmentReminderWindow))
	if err != nil {
		logger.L.Error("扫描作业截止提醒失败", zap.Error(err))
		return
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		studentIDs, studentErr := s.sourceRepo.ListAssignmentUnsubmittedStudentIDs(ctx, item.AssignmentID)
		if studentErr != nil {
			logger.L.Error("查询作业未提交学生失败", zap.Int64("assignment_id", item.AssignmentID), zap.Error(studentErr))
			continue
		}
		receiverIDs := stringifyIDs(studentIDs)
		if len(receiverIDs) == 0 {
			continue
		}
		hours := int(item.DeadlineAt.Sub(now).Hours())
		if hours <= 0 {
			hours = 1
		}
		err = s.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
			EventType:   "assignment.deadline_reminder",
			ReceiverIDs: receiverIDs,
			Params: map[string]interface{}{
				"assignment_name": item.AssignmentName,
				"course_name":     item.CourseName,
				"hours":           hours,
				"deadline":        item.DeadlineAt.UTC().Format("2006-01-02 15:04"),
			},
			SourceModule: "module_03",
			SourceType:   "assignment",
			SourceID:     strconv.FormatInt(item.AssignmentID, 10),
		})
		if err != nil {
			logger.L.Error("发送作业截止提醒失败", zap.Int64("assignment_id", item.AssignmentID), zap.Error(err))
		}
	}
}

// sendCompetitionRegistrationReminders 扫描并发送竞赛报名截止提醒。
func (s *Scheduler) sendCompetitionRegistrationReminders(ctx context.Context, now time.Time) {
	if s.dispatcher == nil || s.sourceRepo == nil {
		return
	}
	items, err := s.sourceRepo.ListCompetitionRegistrationDeadlineCandidates(ctx, now, now.Add(competitionRegistrationWindow))
	if err != nil {
		logger.L.Error("扫描竞赛报名提醒失败", zap.Error(err))
		return
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		schoolID := int64(0)
		if item.SchoolID != nil {
			schoolID = *item.SchoolID
		}
		studentIDs, studentErr := s.sourceRepo.ListCompetitionUnregisteredStudentIDs(ctx, item.CompetitionID, schoolID)
		if studentErr != nil {
			logger.L.Error("查询竞赛未报名学生失败", zap.Int64("competition_id", item.CompetitionID), zap.Error(studentErr))
			continue
		}
		receiverIDs := stringifyIDs(studentIDs)
		if len(receiverIDs) == 0 {
			continue
		}
		err = s.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
			EventType:   "competition.registration_reminder",
			ReceiverIDs: receiverIDs,
			Params: map[string]interface{}{
				"competition_name": item.Title,
				"deadline":         formatReminderTime(item.RegistrationEndAt),
			},
			SourceModule: "module_05",
			SourceType:   "competition",
			SourceID:     strconv.FormatInt(item.CompetitionID, 10),
		})
		if err != nil {
			logger.L.Error("发送竞赛报名提醒失败", zap.Int64("competition_id", item.CompetitionID), zap.Error(err))
		}
	}
}

// sendCompetitionStartingReminders 扫描并发送竞赛开始提醒。
func (s *Scheduler) sendCompetitionStartingReminders(ctx context.Context, now time.Time) {
	if s.dispatcher == nil || s.sourceRepo == nil {
		return
	}
	items, err := s.sourceRepo.ListCompetitionStartingCandidates(ctx, now, now.Add(competitionStartingReminderWindow))
	if err != nil {
		logger.L.Error("扫描竞赛开始提醒失败", zap.Error(err))
		return
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		studentIDs, studentErr := s.sourceRepo.ListCompetitionRegisteredStudentIDs(ctx, item.CompetitionID)
		if studentErr != nil {
			logger.L.Error("查询竞赛报名学生失败", zap.Int64("competition_id", item.CompetitionID), zap.Error(studentErr))
			continue
		}
		receiverIDs := stringifyIDs(studentIDs)
		if len(receiverIDs) == 0 {
			continue
		}
		err = s.dispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
			EventType:   "competition.starting_reminder",
			ReceiverIDs: receiverIDs,
			Params: map[string]interface{}{
				"competition_name": item.Title,
				"start_time":       formatReminderTime(item.StartAt),
			},
			SourceModule: "module_05",
			SourceType:   "competition",
			SourceID:     strconv.FormatInt(item.CompetitionID, 10),
		})
		if err != nil {
			logger.L.Error("发送竞赛开始提醒失败", zap.Int64("competition_id", item.CompetitionID), zap.Error(err))
		}
	}
}

// cleanupExpiredReadNotifications 清理超期已读站内信。
func (s *Scheduler) cleanupExpiredReadNotifications(ctx context.Context, now time.Time) {
	if s.notificationRepo == nil {
		return
	}
	items, err := s.notificationRepo.ListExpiredRead(ctx, now.Add(-expiredReadCleanupWindow), 1000)
	if err != nil {
		logger.L.Error("扫描超期已读站内信失败", zap.Error(err))
		return
	}
	if len(items) == 0 {
		return
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item == nil || item.ID == 0 {
			continue
		}
		ids = append(ids, item.ID)
	}
	if len(ids) == 0 {
		return
	}
	if err := s.notificationRepo.DeleteByIDs(ctx, ids); err != nil {
		logger.L.Error("清理超期已读站内信失败", zap.Error(err))
	}
}

// reconcileUnreadCache 扫描未读缓存键并重新校准用户未读数。
func (s *Scheduler) reconcileUnreadCache(ctx context.Context) {
	if s.unreadSyncer == nil || cache.Get() == nil {
		return
	}
	userIDs, err := listUnreadCacheUserIDs(ctx)
	if err != nil {
		logger.L.Error("扫描未读缓存键失败", zap.Error(err))
		return
	}
	for _, userID := range userIDs {
		if err := s.unreadSyncer.SyncUnreadCountForUser(ctx, userID); err != nil {
			logger.L.Warn("校准未读缓存失败", zap.Int64("user_id", userID), zap.Error(err))
		}
	}
}

// listUnreadCacheUserIDs 从 Redis 未读缓存键中提取用户 ID 列表。
func listUnreadCacheUserIDs(ctx context.Context) ([]int64, error) {
	if cache.Get() == nil {
		return []int64{}, nil
	}
	result := make([]int64, 0)
	seen := make(map[int64]struct{})
	var cursor uint64
	for {
		keys, nextCursor, err := cache.Get().Scan(ctx, cursor, cache.KeyNotifyUnread+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			userID, ok := parseUnreadCacheUserID(key)
			if !ok {
				continue
			}
			if _, exists := seen[userID]; exists {
				continue
			}
			seen[userID] = struct{}{}
			result = append(result, userID)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

// parseUnreadCacheUserID 解析未读缓存键中的用户 ID。
func parseUnreadCacheUserID(key string) (int64, bool) {
	raw := strings.TrimPrefix(strings.TrimSpace(key), cache.KeyNotifyUnread)
	if raw == "" {
		return 0, false
	}
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return userID, true
}

// stringifyIDs 将整型接收者列表转为 DTO 所需的字符串切片。
func stringifyIDs(ids []int64) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		result = append(result, strconv.FormatInt(id, 10))
	}
	return result
}

// formatReminderTime 将可选时间格式化为通知模板使用的固定字符串。
func formatReminderTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02 15:04")
}
