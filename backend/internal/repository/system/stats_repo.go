// stats_repo.go
// 模块08 — 系统管理与监控：平台统计与统一审计聚合数据访问层。
// 负责 platform_statistics 的读写，以及跨模块日志/业务表的只读聚合查询。

package systemrepo

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// AuditLogItem 聚合审计日志项。
type AuditLogItem struct {
	ID           int64          `json:"id"`
	Source       string         `json:"source"`
	OperatorID   int64          `json:"operator_id"`
	OperatorName *string        `json:"operator_name,omitempty"`
	Action       string         `json:"action"`
	TargetType   *string        `json:"target_type,omitempty"`
	TargetID     *int64         `json:"target_id,omitempty"`
	Detail       datatypes.JSON `json:"detail,omitempty"`
	IP           *string        `json:"ip,omitempty"`
	UserAgent    *string        `json:"user_agent,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// AuditLogListParams 统一审计查询参数。
type AuditLogListParams struct {
	Source     string
	Keyword    string
	OperatorID int64
	Action     string
	IP         string
	DateFrom   string
	DateTo     string
	Page       int
	PageSize   int
}

// AuditSourceCounts 日志来源统计。
type AuditSourceCounts struct {
	Login      int64 `json:"login"`
	Operation  int64 `json:"operation"`
	Experiment int64 `json:"experiment"`
}

// PlatformStatisticRepository 平台统计数据访问接口。
type PlatformStatisticRepository interface {
	GetByDate(ctx context.Context, statDate time.Time) (*entity.PlatformStatistic, error)
	Upsert(ctx context.Context, statistic *entity.PlatformStatistic) error
	ListTrend(ctx context.Context, metric string, dateFrom time.Time) ([]*StatisticTrendPoint, error)
	Overview(ctx context.Context) (*PlatformStatisticsOverview, error)
	SchoolActivityRanking(ctx context.Context, limit int) ([]*SchoolActivityRankingItem, error)
}

// StatisticTrendPoint 统计趋势点位。
type StatisticTrendPoint struct {
	Date  time.Time `gorm:"column:stat_date"`
	Value int64     `gorm:"column:value"`
}

// PlatformStatisticsOverview 平台统计总览。
type PlatformStatisticsOverview struct {
	TotalUsers        int64 `gorm:"column:total_users"`
	TotalSchools      int64 `gorm:"column:total_schools"`
	TotalCourses      int64 `gorm:"column:total_courses"`
	TotalExperiments  int64 `gorm:"column:total_experiments"`
	TotalCompetitions int64 `gorm:"column:total_competitions"`
	TodayActiveUsers  int64 `gorm:"column:active_users"`
	TodayNewUsers     int64 `gorm:"column:new_users"`
	TodayExperiments  int64 `gorm:"column:total_experiments_today"`
	TodayAPIRequests  int64 `gorm:"column:api_request_count"`
}

// SchoolActivityRankingItem 学校活跃度排行项。
type SchoolActivityRankingItem struct {
	SchoolID      int64   `gorm:"column:school_id"`
	SchoolName    string  `gorm:"column:school_name"`
	ActiveUsers   int64   `gorm:"column:active_users"`
	TotalUsers    int64   `gorm:"column:total_users"`
	ActivityScore float64 `gorm:"column:activity_score"`
}

type platformStatisticRepository struct {
	db *gorm.DB
}

var statisticMetricColumnMap = map[string]string{
	"active_users":        "active_users",
	"new_users":           "new_users",
	"total_users":         "total_users",
	"total_schools":       "total_schools",
	"total_courses":       "total_courses",
	"active_courses":      "active_courses",
	"total_experiments":   "total_experiments",
	"total_competitions":  "total_competitions",
	"active_competitions": "active_competitions",
	"storage_used_gb":     "storage_used_gb",
	"api_request_count":   "api_request_count",
}

// NewPlatformStatisticRepository 创建平台统计数据访问实例。
func NewPlatformStatisticRepository(db *gorm.DB) PlatformStatisticRepository {
	return &platformStatisticRepository{db: db}
}

// GetByDate 获取指定日期的平台统计。
func (r *platformStatisticRepository) GetByDate(ctx context.Context, statDate time.Time) (*entity.PlatformStatistic, error) {
	var statistic entity.PlatformStatistic
	err := r.db.WithContext(ctx).Where("stat_date = ?", statDate).First(&statistic).Error
	if err != nil {
		return nil, err
	}
	return &statistic, nil
}

// Upsert 保存平台统计。
func (r *platformStatisticRepository) Upsert(ctx context.Context, statistic *entity.PlatformStatistic) error {
	if statistic.ID == 0 {
		statistic.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stat_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"active_users", "new_users", "total_users", "total_schools", "total_courses", "active_courses", "total_experiments", "total_competitions", "active_competitions", "storage_used_gb", "api_request_count"}),
	}).Create(statistic).Error
}

// ListTrend 查询统计趋势。
func (r *platformStatisticRepository) ListTrend(ctx context.Context, metric string, dateFrom time.Time) ([]*StatisticTrendPoint, error) {
	column, ok := statisticMetricColumnMap[metric]
	if !ok {
		column = "active_users"
	}

	var items []*StatisticTrendPoint
	err := r.db.WithContext(ctx).Model(&entity.PlatformStatistic{}).
		Select("stat_date, "+column+" AS value").
		Where("stat_date >= ?", dateFrom).
		Order("stat_date asc").
		Find(&items).Error
	return items, err
}

// Overview 查询统计总览。
func (r *platformStatisticRepository) Overview(ctx context.Context) (*PlatformStatisticsOverview, error) {
	var overview PlatformStatisticsOverview
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) AS total_users,
			(SELECT COUNT(*) FROM schools WHERE deleted_at IS NULL) AS total_schools,
			(SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL) AS total_courses,
			(SELECT COUNT(*) FROM experiment_instances) AS total_experiments,
			(SELECT COUNT(*) FROM competitions WHERE deleted_at IS NULL) AS total_competitions,
			(SELECT COALESCE(active_users, 0) FROM platform_statistics ORDER BY stat_date DESC LIMIT 1) AS active_users,
			(SELECT COALESCE(new_users, 0) FROM platform_statistics ORDER BY stat_date DESC LIMIT 1) AS new_users,
			(SELECT COALESCE(total_experiments, 0) FROM platform_statistics ORDER BY stat_date DESC LIMIT 1) AS total_experiments_today,
			(SELECT COALESCE(api_request_count, 0) FROM platform_statistics ORDER BY stat_date DESC LIMIT 1) AS api_request_count
	`).Scan(&overview).Error
	if err != nil {
		return nil, err
	}
	return &overview, nil
}

// SchoolActivityRanking 查询学校活跃度排行。
func (r *platformStatisticRepository) SchoolActivityRanking(ctx context.Context, limit int) ([]*SchoolActivityRankingItem, error) {
	var items []*SchoolActivityRankingItem
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			s.id AS school_id,
			s.name AS school_name,
			COUNT(u.id) FILTER (WHERE u.last_login_at >= NOW() - INTERVAL '30 days') AS active_users,
			COUNT(u.id) AS total_users,
			CASE WHEN COUNT(u.id) = 0 THEN 0
			     ELSE ROUND((COUNT(u.id) FILTER (WHERE u.last_login_at >= NOW() - INTERVAL '30 days'))::numeric * 100.0 / COUNT(u.id), 2)
			END AS activity_score
		FROM schools s
		LEFT JOIN users u ON u.school_id = s.id AND u.deleted_at IS NULL
		WHERE s.deleted_at IS NULL
		GROUP BY s.id, s.name
		ORDER BY activity_score DESC, active_users DESC, s.id ASC
		LIMIT ?
	`, limit).Scan(&items).Error
	return items, err
}

// AuditRepository 统一审计聚合数据访问接口。
type AuditRepository interface {
	List(ctx context.Context, params *AuditLogListParams) ([]*AuditLogItem, int64, *AuditSourceCounts, error)
}

type auditRepository struct {
	db *gorm.DB
}

type auditLoginRow struct {
	entity.LoginLog
	OperatorName *string `gorm:"column:operator_name"`
}

type auditOperationRow struct {
	entity.OperationLog
	OperatorName *string `gorm:"column:operator_name"`
}

type auditExperimentRow struct {
	entity.InstanceOperationLog
	OperatorName *string `gorm:"column:operator_name"`
}

// NewAuditRepository 创建统一审计聚合数据访问实例。
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

// List 聚合查询审计日志。
func (r *auditRepository) List(ctx context.Context, params *AuditLogListParams) ([]*AuditLogItem, int64, *AuditSourceCounts, error) {
	var items []*AuditLogItem
	counts := &AuditSourceCounts{}
	if params.Source == "" || params.Source == "login" {
		loginItems, total, err := r.listLoginLogs(ctx, params)
		if err != nil {
			return nil, 0, nil, err
		}
		items = append(items, loginItems...)
		counts.Login = total
	}
	if params.Source == "" || params.Source == "operation" {
		opItems, total, err := r.listOperationLogs(ctx, params)
		if err != nil {
			return nil, 0, nil, err
		}
		items = append(items, opItems...)
		counts.Operation = total
	}
	if params.Source == "" || params.Source == "experiment" {
		expItems, total, err := r.listExperimentLogs(ctx, params)
		if err != nil {
			return nil, 0, nil, err
		}
		items = append(items, expItems...)
		counts.Experiment = total
	}

	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := int64(len(items))
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	start := pagination.Offset(page, pageSize)
	if start >= len(items) {
		return []*AuditLogItem{}, total, counts, nil
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, counts, nil
}

func (r *auditRepository) listLoginLogs(ctx context.Context, params *AuditLogListParams) ([]*AuditLogItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.LoginLog{}).
		Select("login_logs.*, users.name AS operator_name").
		Joins("LEFT JOIN users ON users.id = login_logs.user_id")
	if params.OperatorID > 0 {
		query = query.Where("login_logs.user_id = ?", params.OperatorID)
	}
	if params.Action != "" {
		actionValue, ok := parseLoginActionCode(params.Action)
		if !ok {
			return []*AuditLogItem{}, 0, nil
		}
		query = query.Where("login_logs.action = ?", actionValue)
	}
	if params.IP != "" {
		query = query.Where("login_logs.ip = ?", params.IP)
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "users.name", "login_logs.ip"))
	}
	query = query.Scopes(database.WithDateRange("login_logs.created_at", params.DateFrom, params.DateTo))
	var logs []auditLoginRow
	if err := query.Order("login_logs.created_at desc").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*AuditLogItem, 0, len(logs))
	for _, log := range logs {
		detail, _ := json.Marshal(map[string]interface{}{
			"login_method": log.LoginMethod,
			"fail_reason":  log.FailReason,
		})
		ip := log.IP
		action := loginActionCode(log.Action)
		items = append(items, &AuditLogItem{
			ID:           log.ID,
			Source:       "login",
			OperatorID:   log.UserID,
			OperatorName: log.OperatorName,
			Action:       action,
			Detail:       detail,
			IP:           &ip,
			UserAgent:    log.UserAgent,
			CreatedAt:    log.CreatedAt,
		})
	}
	return items, int64(len(items)), nil
}

func (r *auditRepository) listOperationLogs(ctx context.Context, params *AuditLogListParams) ([]*AuditLogItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.OperationLog{}).
		Select("operation_logs.*, users.name AS operator_name").
		Joins("LEFT JOIN users ON users.id = operation_logs.operator_id")
	if params.OperatorID > 0 {
		query = query.Where("operation_logs.operator_id = ?", params.OperatorID)
	}
	if params.Action != "" {
		query = query.Where("operation_logs.action = ?", params.Action)
	}
	if params.IP != "" {
		query = query.Where("operation_logs.ip = ?", params.IP)
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "users.name", "operation_logs.action", "operation_logs.target_type", "operation_logs.ip"))
	}
	query = query.Scopes(database.WithDateRange("operation_logs.created_at", params.DateFrom, params.DateTo))
	var logs []auditOperationRow
	if err := query.Order("operation_logs.created_at desc").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*AuditLogItem, 0, len(logs))
	for _, log := range logs {
		ip := log.IP
		items = append(items, &AuditLogItem{
			ID:           log.ID,
			Source:       "operation",
			OperatorID:   log.OperatorID,
			OperatorName: log.OperatorName,
			Action:       log.Action,
			TargetType:   &log.TargetType,
			TargetID:     log.TargetID,
			Detail:       log.Detail,
			IP:           &ip,
			CreatedAt:    log.CreatedAt,
		})
	}
	return items, int64(len(items)), nil
}

func (r *auditRepository) listExperimentLogs(ctx context.Context, params *AuditLogListParams) ([]*AuditLogItem, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.InstanceOperationLog{}).
		Select("instance_operation_logs.*, users.name AS operator_name").
		Joins("LEFT JOIN users ON users.id = instance_operation_logs.student_id")
	if params.OperatorID > 0 {
		query = query.Where("instance_operation_logs.student_id = ?", params.OperatorID)
	}
	if params.Action != "" {
		query = query.Where("instance_operation_logs.action = ?", params.Action)
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "users.name", "instance_operation_logs.action", "instance_operation_logs.command"))
	}
	query = query.Scopes(database.WithDateRange("instance_operation_logs.created_at", params.DateFrom, params.DateTo))
	var logs []auditExperimentRow
	if err := query.Order("instance_operation_logs.created_at desc").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*AuditLogItem, 0, len(logs))
	for _, log := range logs {
		targetType := "experiment_instance"
		detail, _ := json.Marshal(map[string]interface{}{
			"command":          log.Command,
			"target_container": log.TargetContainer,
			"target_scene":     log.TargetScene,
		})
		items = append(items, &AuditLogItem{
			ID:           log.ID,
			Source:       "experiment",
			OperatorID:   log.StudentID,
			OperatorName: log.OperatorName,
			Action:       log.Action,
			TargetType:   &targetType,
			TargetID:     &log.InstanceID,
			Detail:       detail,
			CreatedAt:    log.CreatedAt,
		})
	}
	return items, int64(len(items)), nil
}

func loginActionCode(action int16) string {
	switch action {
	case enum.LoginActionSuccess:
		return "login_success"
	case enum.LoginActionFail:
		return "login_fail"
	case enum.LoginActionLogout:
		return "logout"
	case enum.LoginActionKicked:
		return "kicked"
	case enum.LoginActionLocked:
		return "locked"
	default:
		return "unknown"
	}
}

func parseLoginActionCode(action string) (int16, bool) {
	switch action {
	case "login_success":
		return enum.LoginActionSuccess, true
	case "login_fail":
		return enum.LoginActionFail, true
	case "logout":
		return enum.LoginActionLogout, true
	case "kicked":
		return enum.LoginActionKicked, true
	case "locked":
		return enum.LoginActionLocked, true
	default:
		return 0, false
	}
}
