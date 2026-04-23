// dashboard_stats_service.go
// 模块08 — 系统管理与监控：运维仪表盘与平台统计业务。
// 该文件聚合健康状态、资源面板、实时指标和平台统计查询，保持监控视图功能域内聚。

package system

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
)

// GetDashboardHealth 获取平台健康状态。
func (s *service) GetDashboardHealth(ctx context.Context, sc *svcctx.ServiceContext) (*dto.DashboardHealthResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	results := s.loadDashboardHealthResults(ctx)
	services := make([]dto.DashboardServiceHealth, 0, len(results))
	overallStatus := "healthy"
	for _, result := range results {
		if result.Status == "unhealthy" {
			overallStatus = "unhealthy"
		} else if result.Status == "unknown" && overallStatus == "healthy" {
			overallStatus = "unknown"
		}
		item := dto.DashboardServiceHealth{
			Name:      result.Name,
			Status:    result.Status,
			LatencyMS: int(result.LatencyMS),
		}
		if serviceKeyFromHealthResultName(result.Name) == "k8s_cluster" {
			if cluster, err := s.loadClusterStatus(ctx); err == nil && cluster != nil {
				item.Nodes = intPtr(cluster.TotalNodes)
				item.PodsRunning = intPtr(cluster.RunningPods)
			}
		}
		services = append(services, item)
	}
	return &dto.DashboardHealthResp{
		OverallStatus: overallStatus,
		Services:      services,
		LastCheckAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// loadDashboardHealthResults 优先读取健康检查缓存，缺失时回退为实时探测。
func (s *service) loadDashboardHealthResults(ctx context.Context) []serviceHealthResult {
	results, ok := s.readHealthResultsFromCache(ctx)
	if ok {
		return results
	}
	return s.collectHealthResults(ctx)
}

// GetDashboardResources 获取平台资源使用情况。
func (s *service) GetDashboardResources(ctx context.Context, sc *svcctx.ServiceContext) (*dto.DashboardResourcesResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	return s.buildDashboardResources(ctx)
}

// GetDashboardRealtime 获取实时指标。
func (s *service) GetDashboardRealtime(ctx context.Context, sc *svcctx.ServiceContext) (*dto.DashboardRealtimeResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	realtimeCounts, err := s.statRepo.GetRealtimeCounts(ctx)
	if err != nil {
		return nil, err
	}
	apiRequestsPerMinute, err := s.statRepo.CountRecentAPIRequests(ctx, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		return nil, err
	}
	alerts, err := s.alertEventRepo.ListRecentPending(ctx, defaultRecentAlertListSize)
	if err != nil {
		return nil, err
	}
	statusCounts, err := s.alertEventRepo.StatusCounts(ctx, nil)
	if err != nil {
		return nil, err
	}
	recentAlerts := make([]dto.DashboardRecentAlert, 0, len(alerts))
	for _, item := range alerts {
		if item == nil {
			continue
		}
		recentAlerts = append(recentAlerts, dto.DashboardRecentAlert{
			ID:          int64String(item.ID),
			Title:       item.Title,
			Level:       item.Level,
			TriggeredAt: item.TriggeredAt.UTC().Format(time.RFC3339),
		})
	}
	return &dto.DashboardRealtimeResp{
		OnlineUsers:          countOnlineUsers(ctx),
		ActiveExperiments:    int(realtimeCounts.ActiveExperiments),
		ActiveCompetitions:   int(realtimeCounts.ActiveCompetitions),
		APIRequestsPerMinute: int(apiRequestsPerMinute),
		PendingAlerts:        int(statusCounts.Pending),
		RecentAlerts:         recentAlerts,
	}, nil
}

// GetStatisticsOverview 获取平台统计总览。
func (s *service) GetStatisticsOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.StatisticsOverviewResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	overview, err := s.statRepo.Overview(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.StatisticsOverviewResp{
		TotalUsers:        int(overview.TotalUsers),
		TotalSchools:      int(overview.TotalSchools),
		TotalCourses:      int(overview.TotalCourses),
		TotalExperiments:  int(overview.TotalExperiments),
		TotalCompetitions: int(overview.TotalCompetitions),
		Today: dto.StatisticsTodaySummary{
			ActiveUsers:        int(overview.TodayActiveUsers),
			NewUsers:           int(overview.TodayNewUsers),
			ExperimentsStarted: int(overview.TodayExperiments),
			APIRequests:        int(overview.TodayAPIRequests),
		},
	}, nil
}

// GetStatisticsTrend 获取平台趋势数据。
func (s *service) GetStatisticsTrend(ctx context.Context, sc *svcctx.ServiceContext, req *dto.StatisticsTrendReq) (*dto.StatisticsTrendResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	dateFrom, period := resolveTrendPeriod(req.Period)
	points, err := s.statRepo.ListTrend(ctx, req.Metric, dateFrom)
	if err != nil {
		return nil, err
	}
	dataPoints := make([]dto.StatisticsTrendPoint, 0, len(points))
	for _, point := range points {
		if point == nil {
			continue
		}
		dataPoints = append(dataPoints, dto.StatisticsTrendPoint{
			Date:  point.Date.UTC().Format("2006-01-02"),
			Value: int(point.Value),
		})
	}
	return &dto.StatisticsTrendResp{
		Metric:     req.Metric,
		Period:     period,
		DataPoints: dataPoints,
	}, nil
}

// GetSchoolStatistics 获取学校活跃度排行。
func (s *service) GetSchoolStatistics(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SchoolActivityRankResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	items, err := s.statRepo.SchoolActivityRanking(ctx, defaultSchoolActivityTopSize)
	if err != nil {
		return nil, err
	}
	list := make([]dto.SchoolActivityRankItem, 0, len(items))
	for index, item := range items {
		if item == nil {
			continue
		}
		list = append(list, dto.SchoolActivityRankItem{
			Rank:          index + 1,
			SchoolID:      int64String(item.SchoolID),
			SchoolName:    item.SchoolName,
			ActiveUsers:   int(item.ActiveUsers),
			TotalUsers:    int(item.TotalUsers),
			ActivityScore: item.ActivityScore,
		})
	}
	return &dto.SchoolActivityRankResp{List: list}, nil
}

// countOnlineUsers 统计当前在线用户数。
func countOnlineUsers(ctx context.Context) int {
	client := cache.Get()
	if client == nil {
		return 0
	}
	count := 0
	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, cache.KeySession+"*", 100).Result()
		if err != nil {
			return 0
		}
		count += len(keys)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return count
}

// readHealthResultsFromCache 读取定时健康巡检写入的服务状态缓存。
func (s *service) readHealthResultsFromCache(ctx context.Context) ([]serviceHealthResult, bool) {
	client := cache.Get()
	if client == nil {
		return nil, false
	}
	serviceNames := []string{"PostgreSQL", "Redis", "NATS", "MinIO", "API Server", "K8s Cluster"}
	results := make([]serviceHealthResult, 0, len(serviceNames))
	for _, name := range serviceNames {
		raw, err := cache.GetString(ctx, cache.KeyServiceHealth+strings.ToLower(name))
		if err != nil || strings.TrimSpace(raw) == "" {
			return nil, false
		}
		var result serviceHealthResult
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return nil, false
		}
		results = append(results, result)
	}
	return results, true
}
