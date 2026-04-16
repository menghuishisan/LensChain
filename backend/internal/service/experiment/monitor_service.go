// monitor_service.go
// 模块04 — 实验环境：监控与统计业务逻辑
// 负责课程实验监控、课程统计、学校监控和全平台实验概览

package experiment

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

const monitorQueryPageSize = 10000

// MonitorService 监控与统计服务接口。
type MonitorService interface {
	GetCourseMonitor(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.MonitorPanelReq) (*dto.MonitorPanelResp, error)
	GetCourseStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.ExperimentStatisticsReq) (*dto.ExperimentStatisticsResp, error)
	GetSchoolMonitor(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SchoolMonitorResp, error)
	GetExperimentOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ExperimentOverviewResp, error)
	GetContainerResources(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ContainerResourceResp, error)
	GetK8sClusterStatus(ctx context.Context, sc *svcctx.ServiceContext) (*dto.K8sClusterStatusResp, error)
}

// monitorService 监控与统计服务实现。
// 统一聚合实例、课程、配额和集群状态，输出教师、校管、超管视角的监控面板数据。
type monitorService struct {
	instanceRepo       experimentrepo.InstanceRepository
	templateRepo       experimentrepo.TemplateRepository
	imageRepo          experimentrepo.ImageRepository
	scenarioRepo       experimentrepo.ScenarioRepository
	quotaRepo          experimentrepo.QuotaRepository
	checkpointRepo     experimentrepo.CheckpointRepository
	checkResultRepo    experimentrepo.CheckpointResultRepository
	courseQuerier      CourseQuerier
	courseRoster       CourseRosterQuerier
	userSummaryQuerier UserSummaryQuerier
	userNameQuerier    UserNameQuerier
	schoolNameQuerier  SchoolNameQuerier
	k8sSvc             K8sService
}

// NewMonitorService 创建监控与统计服务实例。
func NewMonitorService(
	instanceRepo experimentrepo.InstanceRepository,
	templateRepo experimentrepo.TemplateRepository,
	imageRepo experimentrepo.ImageRepository,
	scenarioRepo experimentrepo.ScenarioRepository,
	quotaRepo experimentrepo.QuotaRepository,
	checkpointRepo experimentrepo.CheckpointRepository,
	checkResultRepo experimentrepo.CheckpointResultRepository,
	courseQuerier CourseQuerier,
	courseRoster CourseRosterQuerier,
	userSummaryQuerier UserSummaryQuerier,
	userNameQuerier UserNameQuerier,
	schoolNameQuerier SchoolNameQuerier,
	k8sSvc K8sService,
) MonitorService {
	return &monitorService{
		instanceRepo:       instanceRepo,
		templateRepo:       templateRepo,
		imageRepo:          imageRepo,
		scenarioRepo:       scenarioRepo,
		quotaRepo:          quotaRepo,
		checkpointRepo:     checkpointRepo,
		checkResultRepo:    checkResultRepo,
		courseQuerier:      courseQuerier,
		courseRoster:       courseRoster,
		userSummaryQuerier: userSummaryQuerier,
		userNameQuerier:    userNameQuerier,
		schoolNameQuerier:  schoolNameQuerier,
		k8sSvc:             k8sSvc,
	}
}

// GetCourseMonitor 获取课程实验监控面板数据。
func (s *monitorService) GetCourseMonitor(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.MonitorPanelReq) (*dto.MonitorPanelResp, error) {
	if err := s.ensureCourseTeacherAccess(ctx, sc, courseID); err != nil {
		return nil, err
	}

	params := &experimentrepo.InstanceListParams{
		CourseID:  courseID,
		Page:      1,
		PageSize:  monitorQueryPageSize,
		Status:    req.Status,
		SchoolID:  sc.SchoolID,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	if sc.IsSuperAdmin() {
		params.SchoolID = 0
	}
	if req.TemplateID != "" {
		templateID, err := snowflake.ParseString(req.TemplateID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("模板ID无效")
		}
		params.TemplateID = templateID
	}

	instances, _, err := s.instanceRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	roster, err := s.courseRoster.ListCourseStudents(ctx, courseID)
	if err != nil {
		return nil, err
	}

	instanceByStudent := make(map[int64]*entity.ExperimentInstance)
	for _, instance := range instances {
		current := instanceByStudent[instance.StudentID]
		if current == nil || instance.AttemptNo > current.AttemptNo || instance.CreatedAt.After(current.CreatedAt) {
			instanceByStudent[instance.StudentID] = instance
		}
	}

	quotaResp, _ := s.buildCourseQuotaUsage(ctx, courseID, sc.SchoolID)
	students := make([]dto.MonitorStudentItem, 0, len(roster))
	summary := dto.MonitorSummary{
		TotalStudents: len(roster),
	}
	totalProgress := 0.0

	for _, student := range roster {
		item := dto.MonitorStudentItem{
			StudentID:   strconv.FormatInt(student.StudentID, 10),
			StudentName: student.Name,
			StudentNo:   student.StudentNo,
		}
		instance := instanceByStudent[student.StudentID]
		if instance == nil {
			summary.NotStarted++
			students = append(students, item)
			continue
		}

		instanceID := strconv.FormatInt(instance.ID, 10)
		statusText := enum.GetInstanceStatusText(instance.Status)
		item.InstanceID = &instanceID
		item.Status = &instance.Status
		item.StatusText = &statusText
		if instance.StartedAt != nil {
			value := instance.StartedAt.UTC().Format(time.RFC3339)
			item.StartedAt = &value
		}
		if instance.LastActiveAt != nil {
			value := instance.LastActiveAt.UTC().Format(time.RFC3339)
			item.LastActiveAt = &value
		}

		containers, checkpointPassed, checkpointTotal := s.instanceMonitorMetrics(ctx, instance)
		item.CheckpointsPassed = checkpointPassed
		item.CheckpointsTotal = checkpointTotal
		if checkpointTotal > 0 {
			item.ProgressPercent = float64(checkpointPassed) / float64(checkpointTotal) * 100
			totalProgress += item.ProgressPercent
		}
		if len(containers) > 0 {
			item.CPUUsage = containers[0].CPUUsage
			item.MemoryUsage = containers[0].MemoryUsage
		}

		switch instance.Status {
		case enum.InstanceStatusCreating, enum.InstanceStatusRunning, enum.InstanceStatusQueued, enum.InstanceStatusRestoring:
			summary.Running++
		case enum.InstanceStatusPaused:
			summary.Paused++
		case enum.InstanceStatusSubmitted:
			summary.Completed++
		default:
			summary.NotStarted++
		}
		students = append(students, item)
	}

	if len(roster) > 0 {
		summary.AvgProgress = totalProgress / float64(len(roster))
	}
	if quotaResp != nil {
		summary.ResourceUsage = dto.MonitorResourceUsage{
			CPUUsed:     quotaResp.CPUUsed,
			CPUTotal:    quotaResp.CPUTotal,
			MemoryUsed:  quotaResp.MemoryUsed,
			MemoryTotal: quotaResp.MemoryTotal,
		}
	}

	sort.Slice(students, func(i, j int) bool {
		return students[i].StudentNo < students[j].StudentNo
	})

	return &dto.MonitorPanelResp{
		Summary:  summary,
		Students: students,
	}, nil
}

// GetCourseStatistics 获取课程实验统计数据。
func (s *monitorService) GetCourseStatistics(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.ExperimentStatisticsReq) (*dto.ExperimentStatisticsResp, error) {
	if err := s.ensureCourseTeacherAccess(ctx, sc, courseID); err != nil {
		return nil, err
	}

	params := &experimentrepo.InstanceListParams{
		CourseID:  courseID,
		Page:      1,
		PageSize:  monitorQueryPageSize,
		SchoolID:  sc.SchoolID,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	if sc.IsSuperAdmin() {
		params.SchoolID = 0
	}
	if req.TemplateID != "" {
		templateID, err := snowflake.ParseString(req.TemplateID)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("模板ID无效")
		}
		params.TemplateID = templateID
	}

	instances, _, err := s.instanceRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	templateInstances := make(map[int64][]*entity.ExperimentInstance)
	for _, instance := range instances {
		templateInstances[instance.TemplateID] = append(templateInstances[instance.TemplateID], instance)
	}

	items := make([]dto.TemplateStatisticsItem, 0, len(templateInstances))
	for templateID, groupedInstances := range templateInstances {
		template, err := s.templateRepo.GetByIDWithAll(ctx, templateID)
		if err != nil {
			continue
		}
		item := dto.TemplateStatisticsItem{
			TemplateID:    strconv.FormatInt(templateID, 10),
			TemplateTitle: template.Title,
			Statistics:    s.buildTemplateStatistics(ctx, template, groupedInstances),
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].TemplateID < items[j].TemplateID
	})

	return &dto.ExperimentStatisticsResp{Templates: items}, nil
}

// GetSchoolMonitor 获取学校管理员视角的实验监控数据。
func (s *monitorService) GetSchoolMonitor(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SchoolMonitorResp, error) {
	if !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}

	schoolID := sc.SchoolID
	params := &experimentrepo.InstanceListParams{
		SchoolID:  schoolID,
		Page:      1,
		PageSize:  monitorQueryPageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	if sc.IsSuperAdmin() {
		params.SchoolID = 0
	}

	instances, _, err := s.instanceRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	resourceUsage := dto.MonitorResourceUsage{}
	if quota, err := s.quotaRepo.GetBySchoolID(ctx, schoolID); err == nil {
		resourceUsage = dto.MonitorResourceUsage{
			CPUUsed:     quota.UsedCPU,
			CPUTotal:    quota.MaxCPU,
			MemoryUsed:  quota.UsedMemory,
			MemoryTotal: quota.MaxMemory,
		}
	}

	courseMap := make(map[int64]*dto.SchoolMonitorCourseItem)
	activeStudents := make(map[int64]struct{})
	totalStudents := 0
	runningInstances := 0

	for _, instance := range instances {
		if instance.CourseID == nil {
			continue
		}
		courseID := *instance.CourseID
		item := courseMap[courseID]
		if item == nil {
			teacherName := ""
			teacherID, teacherErr := s.courseQuerier.GetCourseTeacherID(ctx, courseID)
			if teacherErr == nil {
				teacherName = s.userNameQuerier.GetUserName(ctx, teacherID)
			}
			students, _ := s.courseRoster.ListCourseStudents(ctx, courseID)
			item = &dto.SchoolMonitorCourseItem{
				CourseID:      strconv.FormatInt(courseID, 10),
				CourseTitle:   s.courseQuerier.GetCourseTitle(ctx, courseID),
				TeacherName:   teacherName,
				TotalStudents: len(students),
			}
			totalStudents += len(students)
			courseMap[courseID] = item
		}
		if isActiveInstanceStatus(instance.Status) {
			item.RunningInstances++
			runningInstances++
			activeStudents[instance.StudentID] = struct{}{}
		}
	}

	courses := make([]dto.SchoolMonitorCourseItem, 0, len(courseMap))
	for _, item := range courseMap {
		courses = append(courses, *item)
	}
	sort.Slice(courses, func(i, j int) bool {
		return courses[i].CourseID < courses[j].CourseID
	})

	return &dto.SchoolMonitorResp{
		TotalInstances:   len(instances),
		RunningInstances: runningInstances,
		TotalStudents:    totalStudents,
		ActiveStudents:   len(activeStudents),
		ResourceUsage:    resourceUsage,
		Courses:          courses,
	}, nil
}

// GetExperimentOverview 获取超管视角的全平台实验概览。
func (s *monitorService) GetExperimentOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ExperimentOverviewResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}

	instances, _, err := s.instanceRepo.ListAdmin(ctx, &experimentrepo.AdminInstanceListParams{
		Page:      1,
		PageSize:  monitorQueryPageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, err
	}
	templates, totalTemplates, err := s.templateRepo.List(ctx, &experimentrepo.TemplateListParams{
		Page:      1,
		PageSize:  1,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, err
	}
	_ = templates
	images, totalImages, err := s.imageRepo.List(ctx, &experimentrepo.ImageListParams{
		Page:      1,
		PageSize:  1,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, err
	}
	_ = images
	pendingImages, pendingScenarios := 0, 0
	if _, totalPending, pendingErr := s.imageRepo.List(ctx, &experimentrepo.ImageListParams{
		Status:   enum.ImageStatusPending,
		Page:     1,
		PageSize: 1,
	}); pendingErr == nil {
		pendingImages = int(totalPending)
	}
	if s.scenarioRepo != nil {
		if _, totalPending, pendingErr := s.scenarioRepo.List(ctx, &experimentrepo.ScenarioListParams{
			Status:   enum.ScenarioStatusPending,
			Page:     1,
			PageSize: 1,
		}); pendingErr == nil {
			pendingScenarios = int(totalPending)
		}
	}

	clusterInfo := dto.ClusterStatusInfo{}
	if s.k8sSvc != nil {
		clusterStatus, clusterErr := s.k8sSvc.GetClusterStatus(ctx)
		if clusterErr == nil && clusterStatus != nil {
			clusterInfo = dto.ClusterStatusInfo{
				Nodes:        clusterStatus.TotalNodes,
				HealthyNodes: clusterStatus.ReadyNodes,
				TotalCPU:     clusterStatus.TotalCPU,
				UsedCPU:      clusterStatus.UsedCPU,
				TotalMemory:  clusterStatus.TotalMemory,
				UsedMemory:   clusterStatus.UsedMemory,
			}
		}
	}

	schoolUsageMap := make(map[int64]*dto.SchoolUsageItem)
	runningInstances := 0
	for _, instance := range instances {
		if isActiveInstanceStatus(instance.Status) {
			runningInstances++
		}
		item := schoolUsageMap[instance.SchoolID]
		if item == nil {
			item = &dto.SchoolUsageItem{
				SchoolID:   strconv.FormatInt(instance.SchoolID, 10),
				SchoolName: s.schoolNameQuerier.GetSchoolName(ctx, instance.SchoolID),
			}
			if quota, quotaErr := s.quotaRepo.GetBySchoolID(ctx, instance.SchoolID); quotaErr == nil {
				item.QuotaCPU = quota.MaxCPU
				item.QuotaMemory = quota.MaxMemory
				item.CPUUsed = quota.UsedCPU
				item.MemoryUsed = quota.UsedMemory
			}
			schoolUsageMap[instance.SchoolID] = item
		}
		if isActiveInstanceStatus(instance.Status) {
			item.RunningInstances++
		}
	}

	schoolUsage := make([]dto.SchoolUsageItem, 0, len(schoolUsageMap))
	for _, item := range schoolUsageMap {
		schoolUsage = append(schoolUsage, *item)
	}
	sort.Slice(schoolUsage, func(i, j int) bool {
		return schoolUsage[i].SchoolID < schoolUsage[j].SchoolID
	})

	return &dto.ExperimentOverviewResp{
		TotalInstances:   len(instances),
		RunningInstances: runningInstances,
		TotalTemplates:   int(totalTemplates),
		TotalImages:      int(totalImages),
		PendingReviews:   pendingImages + pendingScenarios,
		ClusterStatus:    clusterInfo,
		SchoolUsage:      schoolUsage,
	}, nil
}

// GetContainerResources 获取全平台容器资源监控视图。
func (s *monitorService) GetContainerResources(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ContainerResourceResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	if s.k8sSvc == nil {
		return nil, errcode.ErrInternal.WithMessage("K8s 服务未初始化")
	}

	clusterStatus, err := s.k8sSvc.GetClusterStatus(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := s.k8sSvc.GetNodeStatus(ctx)
	if err != nil {
		return nil, err
	}

	resp := &dto.ContainerResourceResp{
		TotalCPU:    clusterStatus.TotalCPU,
		UsedCPU:     clusterStatus.UsedCPU,
		TotalMemory: clusterStatus.TotalMemory,
		UsedMemory:  clusterStatus.UsedMemory,
		Nodes:       make([]dto.ContainerResourceNode, 0, len(nodes)),
	}
	for _, node := range nodes {
		if node.Status == "Ready" {
			resp.RunningContainers += node.PodCount
		}
		resp.TotalContainers += node.PodCount
		resp.Nodes = append(resp.Nodes, dto.ContainerResourceNode{
			NodeName:       node.Name,
			Status:         node.Status,
			ContainerCount: node.PodCount,
			CPUCapacity:    node.CPUTotal,
			CPUUsed:        node.CPUUsed,
			MemoryCapacity: node.MemTotal,
			MemoryUsed:     node.MemUsed,
		})
	}
	return resp, nil
}

// GetK8sClusterStatus 获取 K8s 集群状态视图。
func (s *monitorService) GetK8sClusterStatus(ctx context.Context, sc *svcctx.ServiceContext) (*dto.K8sClusterStatusResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	if s.k8sSvc == nil {
		return nil, errcode.ErrInternal.WithMessage("K8s 服务未初始化")
	}

	clusterStatus, err := s.k8sSvc.GetClusterStatus(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := s.k8sSvc.GetNodeStatus(ctx)
	if err != nil {
		return nil, err
	}
	resp := &dto.K8sClusterStatusResp{
		Nodes:           make([]dto.K8sNodeStatus, 0, len(nodes)),
		TotalPods:       clusterStatus.TotalPods,
		RunningPods:     clusterStatus.RunningPods,
		PendingPods:     clusterStatus.PendingPods,
		FailedPods:      clusterStatus.FailedPods,
		TotalNamespaces: clusterStatus.Namespaces,
	}
	for _, node := range nodes {
		resp.Nodes = append(resp.Nodes, dto.K8sNodeStatus{
			Name:           node.Name,
			Status:         node.Status,
			KubeletVersion: node.KubeletVersion,
			CPUCapacity:    node.CPUTotal,
			CPUAllocatable: node.CPUAllocatable,
			MemCapacity:    node.MemTotal,
			MemAllocatable: node.MemAllocatable,
			PodCount:       node.PodCount,
			PodCapacity:    node.PodCapacity,
		})
	}
	return resp, nil
}

// ensureCourseTeacherAccess 校验课程教师/超管访问权限。
func (s *monitorService) ensureCourseTeacherAccess(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) error {
	if sc.IsSuperAdmin() {
		return nil
	}
	courseSchoolID, err := s.courseQuerier.GetCourseSchoolID(ctx, courseID)
	if err != nil {
		return err
	}
	if courseSchoolID != sc.SchoolID {
		return errcode.ErrForbidden
	}
	teacherID, err := s.courseQuerier.GetCourseTeacherID(ctx, courseID)
	if err != nil {
		return err
	}
	if sc.IsTeacher() && teacherID == sc.UserID {
		return nil
	}
	if sc.IsSchoolAdmin() {
		return nil
	}
	return errcode.ErrForbidden
}

// instanceMonitorMetrics 计算实例的监控指标。
func (s *monitorService) instanceMonitorMetrics(ctx context.Context, instance *entity.ExperimentInstance) ([]entity.InstanceContainer, int, int) {
	fullInstance, err := s.instanceRepo.GetByIDWithAll(ctx, instance.ID)
	if err != nil || fullInstance == nil {
		return nil, 0, 0
	}
	template, err := s.templateRepo.GetByIDWithAll(ctx, instance.TemplateID)
	if err != nil || template == nil {
		return fullInstance.Containers, 0, 0
	}
	passed := 0
	total := len(template.Checkpoints)
	for i := range fullInstance.CheckpointResults {
		result := fullInstance.CheckpointResults[i]
		if result.IsPassed {
			passed++
		}
	}
	return fullInstance.Containers, passed, total
}

// buildTemplateStatistics 构建单个模板的统计数据。
func (s *monitorService) buildTemplateStatistics(ctx context.Context, template *entity.ExperimentTemplate, instances []*entity.ExperimentInstance) dto.TemplateStatisticsData {
	stats := dto.TemplateStatisticsData{
		MinScore: 0,
	}
	if len(instances) == 0 {
		return stats
	}

	studentSet := make(map[int64]struct{})
	scoreCount := 0
	totalScore := 0.0
	totalAttempts := 0
	totalDurationMinutes := 0
	completedDurations := 0
	checkpointRates := make(map[int64]*dto.CheckpointPassRateItem)

	for _, cp := range template.Checkpoints {
		item := &dto.CheckpointPassRateItem{
			CheckpointID: strconv.FormatInt(cp.ID, 10),
			Title:        cp.Title,
		}
		checkpointRates[cp.ID] = item
	}

	for index, instance := range instances {
		studentSet[instance.StudentID] = struct{}{}
		totalAttempts += instance.AttemptNo
		if instance.Status == enum.InstanceStatusSubmitted {
			stats.CompletedCount++
		}
		if instance.StartedAt != nil {
			stats.StartedCount++
		}
		if instance.TotalScore != nil {
			totalScore += *instance.TotalScore
			scoreCount++
			if index == 0 || *instance.TotalScore < stats.MinScore {
				stats.MinScore = *instance.TotalScore
			}
			if *instance.TotalScore > stats.MaxScore {
				stats.MaxScore = *instance.TotalScore
			}
			switch {
			case *instance.TotalScore >= 90:
				stats.ScoreDistribution.Range90To100++
			case *instance.TotalScore >= 80:
				stats.ScoreDistribution.Range80To89++
			case *instance.TotalScore >= 70:
				stats.ScoreDistribution.Range70To79++
			case *instance.TotalScore >= 60:
				stats.ScoreDistribution.Range60To69++
			default:
				stats.ScoreDistribution.Below60++
			}
		}
		if instance.StartedAt != nil && instance.SubmittedAt != nil {
			totalDurationMinutes += int(instance.SubmittedAt.Sub(*instance.StartedAt).Minutes())
			completedDurations++
		}

		results, err := s.checkResultRepo.ListByInstanceID(ctx, instance.ID)
		if err != nil {
			continue
		}
		for _, result := range results {
			item := checkpointRates[result.CheckpointID]
			if item == nil {
				continue
			}
			if result.IsPassed {
				if item.PassRate == nil {
					item.PassRate = floatPointer(0)
				}
				*item.PassRate = *item.PassRate + 1
			}
			if result.Score != nil {
				if item.AvgScore == nil {
					item.AvgScore = floatPointer(0)
				}
				*item.AvgScore += *result.Score
				if item.MaxScore == nil || *result.Score > *item.MaxScore {
					item.MaxScore = floatPointer(*result.Score)
				}
			}
		}
	}

	stats.TotalStudents = len(studentSet)
	if stats.TotalStudents > 0 {
		stats.CompletionRate = float64(stats.CompletedCount) / float64(stats.TotalStudents) * 100
		stats.AvgAttempts = float64(totalAttempts) / float64(stats.TotalStudents)
	}
	if scoreCount > 0 {
		stats.AvgScore = totalScore / float64(scoreCount)
	}
	if completedDurations > 0 {
		stats.AvgDurationMinutes = totalDurationMinutes / completedDurations
	}

	stats.CheckpointPassRates = make([]dto.CheckpointPassRateItem, 0, len(checkpointRates))
	for _, cp := range template.Checkpoints {
		item := checkpointRates[cp.ID]
		if item == nil {
			continue
		}
		if item.PassRate != nil && stats.TotalStudents > 0 {
			*item.PassRate = *item.PassRate / float64(stats.TotalStudents) * 100
		}
		if item.AvgScore != nil && scoreCount > 0 {
			*item.AvgScore = *item.AvgScore / float64(scoreCount)
		}
		stats.CheckpointPassRates = append(stats.CheckpointPassRates, *item)
	}

	return stats
}

// buildCourseQuotaUsage 构建课程监控面板所需的配额摘要。
func (s *monitorService) buildCourseQuotaUsage(ctx context.Context, courseID, schoolID int64) (*dto.MonitorResourceUsage, error) {
	quota, err := s.quotaRepo.GetByCourseID(ctx, courseID)
	if err == nil {
		return &dto.MonitorResourceUsage{
			CPUUsed:     quota.UsedCPU,
			CPUTotal:    quota.MaxCPU,
			MemoryUsed:  quota.UsedMemory,
			MemoryTotal: quota.MaxMemory,
		}, nil
	}
	quota, err = s.quotaRepo.GetBySchoolID(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	return &dto.MonitorResourceUsage{
		CPUUsed:     quota.UsedCPU,
		CPUTotal:    quota.MaxCPU,
		MemoryUsed:  quota.UsedMemory,
		MemoryTotal: quota.MaxMemory,
	}, nil
}

// floatPointer 创建 float64 指针，便于聚合统计时原地更新。
func floatPointer(value float64) *float64 {
	return &value
}

// isActiveInstanceStatus 判断实例是否属于教师/管理员监控中的活跃运行态。
func isActiveInstanceStatus(status int) bool {
	switch status {
	case enum.InstanceStatusCreating, enum.InstanceStatusRunning, enum.InstanceStatusQueued, enum.InstanceStatusRestoring:
		return true
	default:
		return false
	}
}
