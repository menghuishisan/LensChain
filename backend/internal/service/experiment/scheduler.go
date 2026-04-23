// scheduler.go
// 模块04 — 实验环境：后台回收调度
// 负责空闲回收、时长超限回收和课程结束后的统一回收，确保模块04满足后台验收链路

package experiment

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/ws"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
	"go.uber.org/zap"
)

const (
	defaultSubmittedRetentionMinutes = 30
	courseEndWarningLead             = 10 * time.Minute
	destroyedSnapshotRetention       = 30 * 24 * time.Hour
)

// ExperimentScheduler 实验环境后台调度器。
type ExperimentScheduler struct {
	instanceSvc        *instanceService
	instanceRepo       InstanceRepositoryLike
	templateRepo       TemplateRepositoryLike
	endedCourseQuerier EndedCourseQuerier
	imageRepo          SchedulerImageRepositoryLike
	versionRepo        SchedulerImageVersionRepositoryLike
	k8sSvc             K8sService
}

// NewExperimentScheduler 创建实验环境后台调度器。
func NewExperimentScheduler(
	instanceSvc InstanceService,
	instanceRepo InstanceRepositoryLike,
	templateRepo TemplateRepositoryLike,
	endedCourseQuerier EndedCourseQuerier,
	imageRepo SchedulerImageRepositoryLike,
	versionRepo SchedulerImageVersionRepositoryLike,
	k8sSvc K8sService,
) *ExperimentScheduler {
	concrete, ok := instanceSvc.(*instanceService)
	if !ok || concrete == nil {
		return nil
	}
	return &ExperimentScheduler{
		instanceSvc:        concrete,
		instanceRepo:       instanceRepo,
		templateRepo:       templateRepo,
		endedCourseQuerier: endedCourseQuerier,
		imageRepo:          imageRepo,
		versionRepo:        versionRepo,
		k8sSvc:             k8sSvc,
	}
}

// InstanceRepositoryLike 调度器所需的最小实例仓储接口。
type InstanceRepositoryLike interface {
	List(ctx context.Context, params *experimentrepo.InstanceListParams) ([]*entity.ExperimentInstance, int64, error)
}

// TemplateRepositoryLike 调度器所需的最小模板仓储接口。
type TemplateRepositoryLike interface {
	GetByID(ctx context.Context, id int64) (*entity.ExperimentTemplate, error)
}

// SchedulerImageRepositoryLike 调度器所需的最小镜像仓储接口。
type SchedulerImageRepositoryLike interface {
	List(ctx context.Context, params *experimentrepo.ImageListParams) ([]*entity.Image, int64, error)
}

// SchedulerImageVersionRepositoryLike 调度器所需的最小镜像版本仓储接口。
type SchedulerImageVersionRepositoryLike interface {
	ListByImageID(ctx context.Context, imageID int64) ([]*entity.ImageVersion, error)
}

// RunIdleReclaim 执行空闲实验回收。
func (s *ExperimentScheduler) RunIdleReclaim() {
	s.scanAndReclaim(context.Background(), false)
}

// RunAutoSnapshot 执行运行中实例的定时自动快照。
func (s *ExperimentScheduler) RunAutoSnapshot() {
	s.createScheduledSnapshots(context.Background())
}

// RunExpiredCleanup 执行时长超限和课程结束后的实验回收。
func (s *ExperimentScheduler) RunExpiredCleanup() {
	s.scanAndReclaim(context.Background(), true)
	s.cleanupExpiredSnapshots(context.Background())
}

// RunRuntimeHealthCheck 扫描运行中实例的容器状态，并在异常时触发告警与自动恢复。
func (s *ExperimentScheduler) RunRuntimeHealthCheck() {
	s.scanRuntimeHealth(context.Background())
}

// RunImagePrePullReconcile 对已正常发布的镜像执行预拉取对账。
// 用于补偿审核通过时节点离线等场景，确保节点恢复后自动补拉。
func (s *ExperimentScheduler) RunImagePrePullReconcile() {
	if s.imageRepo == nil || s.versionRepo == nil || s.k8sSvc == nil {
		return
	}

	ctx := context.Background()
	readyNodes, err := s.listReadyNodeNames(ctx)
	if err != nil {
		logger.L.Error("镜像预拉取对账失败：查询就绪节点失败", zap.Error(err))
		return
	}
	if len(readyNodes) == 0 {
		return
	}

	images, _, err := s.imageRepo.List(ctx, &experimentrepo.ImageListParams{
		Status:    enum.ImageStatusNormal,
		Page:      1,
		PageSize:  10000,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		logger.L.Error("镜像预拉取对账失败：查询镜像列表失败", zap.Error(err))
		return
	}

	for _, image := range images {
		versions, versionErr := s.versionRepo.ListByImageID(ctx, image.ID)
		if versionErr != nil {
			logger.L.Warn("镜像预拉取对账跳过：查询镜像版本失败",
				zap.Int64("image_id", image.ID),
				zap.String("image_name", image.Name),
				zap.Error(versionErr),
			)
			continue
		}
		for _, version := range versions {
			if version.RegistryURL == "" {
				continue
			}
			statuses, statusErr := s.k8sSvc.GetImagePullStatus(ctx, version.RegistryURL)
			if statusErr != nil {
				logger.L.Warn("镜像预拉取对账跳过：查询节点拉取状态失败",
					zap.Int64("image_id", image.ID),
					zap.String("image_name", image.Name),
					zap.String("registry_url", version.RegistryURL),
					zap.Error(statusErr),
				)
				continue
			}
			pendingNodes := collectNodesNeedingPull(statuses, readyNodes)
			if len(pendingNodes) == 0 {
				continue
			}
			if pullErr := s.k8sSvc.PrePullImage(ctx, version.RegistryURL, pendingNodes); pullErr != nil {
				logger.L.Warn("镜像预拉取对账补拉失败",
					zap.Int64("image_id", image.ID),
					zap.String("image_name", image.Name),
					zap.String("registry_url", version.RegistryURL),
					zap.Strings("nodes", pendingNodes),
					zap.Error(pullErr),
				)
			}
		}
	}
}

// scanRuntimeHealth 扫描运行中实例的 Pod 健康状态。
func (s *ExperimentScheduler) scanRuntimeHealth(ctx context.Context) {
	if s.instanceSvc == nil || s.k8sSvc == nil {
		return
	}
	instances, _, err := s.instanceRepo.List(ctx, &experimentrepo.InstanceListParams{
		Statuses: []int16{enum.InstanceStatusRunning, enum.InstanceStatusInitializing},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return
	}
	for _, instance := range instances {
		s.inspectInstanceRuntimeHealth(ctx, instance)
	}
}

// inspectInstanceRuntimeHealth 检查单个实例的运行容器状态，并在需要时触发恢复。
func (s *ExperimentScheduler) inspectInstanceRuntimeHealth(ctx context.Context, instance *entity.ExperimentInstance) {
	if instance == nil || instance.Namespace == nil || strings.TrimSpace(*instance.Namespace) == "" {
		return
	}
	instanceAggregate, err := loadInstanceAggregate(
		ctx,
		s.instanceSvc.instanceRepo,
		s.instanceSvc.containerRepo,
		s.instanceSvc.checkResultRepo,
		instance.ID,
	)
	if err != nil || instanceAggregate == nil {
		return
	}
	fullInstance := instanceAggregate.Instance
	for _, container := range instanceAggregate.Containers {
		if container.PodName == nil || strings.TrimSpace(*container.PodName) == "" {
			continue
		}
		podStatus, statusErr := s.k8sSvc.GetPodStatus(ctx, *fullInstance.Namespace, *container.PodName)
		unhealthy, message := evaluatePodHealth(podStatus, statusErr)
		if !unhealthy {
			continue
		}
		s.handleInstanceRuntimeFailure(ctx, fullInstance, message)
		return
	}
}

// handleInstanceRuntimeFailure 将运行时故障实例置为异常，并在有快照时尝试自动恢复。
func (s *ExperimentScheduler) handleInstanceRuntimeFailure(ctx context.Context, instance *entity.ExperimentInstance, errorMessage string) {
	if instance == nil {
		return
	}
	now := time.Now()
	oldStatus := instance.Status
	_ = s.instanceSvc.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
		"status":        enum.InstanceStatusError,
		"error_message": errorMessage,
		"updated_at":    now,
	})
	instance.Status = enum.InstanceStatusError
	instance.ErrorMessage = &errorMessage
	s.instanceSvc.pushCourseMonitorStatusChange(instance, int(oldStatus), int(enum.InstanceStatusError))
	s.instanceSvc.pushCourseMonitorInstanceError(instance, errorMessage)

	snapshots, err := s.instanceSvc.snapshotRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return
	}
	snapshotID := resolveSnapshotIDForResume(nil, snapshots)
	if snapshotID == 0 {
		return
	}
	templateAggregate, err := loadTemplateAggregate(
		ctx,
		s.instanceSvc.templateRepo,
		s.instanceSvc.templateContainerRepo,
		s.instanceSvc.checkpointRepo,
		s.instanceSvc.initScriptRepo,
		s.instanceSvc.simSceneRepo,
		nil,
		nil,
		nil,
		instance.TemplateID,
	)
	if err != nil || templateAggregate == nil {
		return
	}
	if destroyErr := s.instanceSvc.destroyEnvironment(ctx, instance); destroyErr != nil {
		return
	}
	_ = s.instanceSvc.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
		"status":        enum.InstanceStatusInitializing,
		"error_message": errorMessage,
		"updated_at":    time.Now(),
	})
	instance.Status = enum.InstanceStatusInitializing
	cronpkg.RunAsync("模块04异常实例自动恢复", func(asyncCtx context.Context) {
		s.instanceSvc.provisionEnvironment(asyncCtx, instance, templateAggregate, strconv.FormatInt(snapshotID, 10), false)
	})
}

// scanAndReclaim 扫描运行中的实例，并按条件执行回收。
func (s *ExperimentScheduler) scanAndReclaim(ctx context.Context, includeCourseEnd bool) {
	if s.instanceSvc == nil {
		return
	}

	instances, _, err := s.instanceRepo.List(ctx, &experimentrepo.InstanceListParams{
		Statuses: []int16{
			enum.InstanceStatusRunning,
			enum.InstanceStatusPaused,
			enum.InstanceStatusInitializing,
			enum.InstanceStatusCompleted,
		},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return
	}

	endedCourses := make(map[int64]struct{})
	endingSoonCourses := make(map[int64]time.Time)
	if includeCourseEnd && s.endedCourseQuerier != nil {
		if courseIDs, courseErr := s.endedCourseQuerier.ListEndedCourseIDs(ctx); courseErr == nil {
			for _, courseID := range courseIDs {
				endedCourses[courseID] = struct{}{}
			}
		}
		if courseWindows, courseErr := s.endedCourseQuerier.ListCourseIDsEndingWithin(ctx, courseEndWarningLead); courseErr == nil {
			endingSoonCourses = courseWindows
		}
	}

	now := time.Now()
	for _, instance := range instances {
		if includeCourseEnd {
			s.pushCourseEndingWarning(ctx, instance, endingSoonCourses)
		}

		template, templateErr := s.templateRepo.GetByID(ctx, instance.TemplateID)
		if templateErr != nil || template == nil {
			continue
		}

		reason := ""
		if includeCourseEnd && instance.CourseID != nil {
			if _, ok := endedCourses[*instance.CourseID]; ok {
				reason = "course_ended"
			}
		}
		if reason == "" && instance.Status == enum.InstanceStatusCompleted &&
			shouldReclaimSubmitted(instance, template, now) {
			reason = "submitted_retention"
		}
		if reason == "" && instance.Status == enum.InstanceStatusRunning &&
			template.IdleTimeout > 0 && instance.LastActiveAt != nil &&
			now.Sub(*instance.LastActiveAt) >= time.Duration(template.IdleTimeout)*time.Minute {
			reason = "idle_timeout"
		}
		if reason == "" && instance.Status == enum.InstanceStatusRunning &&
			template.MaxDuration != nil && instance.StartedAt != nil &&
			now.Sub(*instance.StartedAt) >= time.Duration(*template.MaxDuration)*time.Minute {
			reason = "max_duration"
		}
		if reason == "" {
			continue
		}

		s.reclaimInstance(ctx, instance, template, reason)
	}
}

// createScheduledSnapshots 为运行中的实验实例补齐每5分钟自动快照。
func (s *ExperimentScheduler) createScheduledSnapshots(ctx context.Context) {
	if s.instanceSvc == nil {
		return
	}

	instances, _, err := s.instanceRepo.List(ctx, &experimentrepo.InstanceListParams{
		Statuses: []int16{enum.InstanceStatusRunning},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return
	}

	now := time.Now()
	for _, instance := range instances {
		snapshots, snapshotErr := s.instanceSvc.snapshotRepo.ListByInstanceID(ctx, instance.ID)
		if snapshotErr != nil {
			continue
		}
		if len(snapshots) > 0 && now.Sub(snapshots[0].CreatedAt) < 5*time.Minute {
			continue
		}

		description := "系统自动快照"
		if _, snapshotErr := s.instanceSvc.createInstanceSnapshot(ctx, instance, enum.SnapshotTypeScheduled, &description); snapshotErr != nil {
			logger.L.Warn("自动快照保存失败",
				zap.Int64("instance_id", instance.ID),
				zap.Error(snapshotErr),
			)
		}
	}
}

// reclaimInstance 保存快照并回收实验环境，将实例状态收口为文档定义的最终状态。
func (s *ExperimentScheduler) reclaimInstance(ctx context.Context, instance *entity.ExperimentInstance, template *entity.ExperimentTemplate, reason string) {
	if s.instanceSvc == nil || instance == nil {
		return
	}

	if reason == "max_duration" {
		s.finalizeTimeoutScore(ctx, instance)
	}

	description := fmt.Sprintf("系统自动回收：%s", reason)
	if _, snapshotErr := s.instanceSvc.createInstanceSnapshot(ctx, instance, enum.SnapshotTypeScheduled, &description); snapshotErr != nil {
		logger.L.Warn("自动回收前保存快照失败，已跳过本次回收",
			zap.Int64("instance_id", instance.ID),
			zap.String("reason", reason),
			zap.Error(snapshotErr),
		)
		return
	}
	oldStatus := instance.Status
	if err := s.instanceSvc.destroyEnvironment(ctx, instance); err != nil {
		return
	}

	now := time.Now()
	finalStatus := int16(enum.InstanceStatusExpired)
	if reason == "submitted_retention" {
		finalStatus = enum.InstanceStatusCompleted
	}
	_ = s.instanceSvc.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
		"status":       finalStatus,
		"destroyed_at": now,
		"updated_at":   now,
	})
	instance.Status = finalStatus
	instance.DestroyedAt = &now
	instance.UpdatedAt = now
	if int16(finalStatus) != oldStatus {
		s.instanceSvc.pushCourseMonitorStatusChange(instance, int(oldStatus), int(finalStatus))
	}
	// 最长时长自动结算会直接产出一次最终自动得分。
	// 该场景终态为“已过期”而非“已完成”，因此成绩回传不能只认 completed。
	if reason == "max_duration" && template != nil {
		_ = s.instanceSvc.syncCourseGradeIfNeeded(ctx, instance, template, nil)
	}
}

// finalizeTimeoutScore 在最长运行时间到达时自动执行一次自动检查点结算。
func (s *ExperimentScheduler) finalizeTimeoutScore(ctx context.Context, instance *entity.ExperimentInstance) {
	templateAggregate, err := loadTemplateAggregate(
		ctx,
		s.instanceSvc.templateRepo,
		nil,
		s.instanceSvc.checkpointRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		instance.TemplateID,
	)
	if err != nil || templateAggregate == nil {
		return
	}

	autoScore := 0.0
	manualTotal := 0.0
	for _, checkpoint := range templateAggregate.Checkpoints {
		if checkpoint.CheckType == enum.CheckTypeManual {
			manualTotal += checkpoint.Score
			continue
		}
		result := s.instanceSvc.executeCheckpoint(ctx, instance, checkpoint)
		if result.Score != nil {
			autoScore += *result.Score
		}
	}

	fields := map[string]interface{}{
		"auto_score":   autoScore,
		"updated_at":   time.Now(),
		"submitted_at": time.Now(),
	}
	if manualTotal == 0 {
		fields["total_score"] = autoScore
	}
	_ = s.instanceSvc.instanceRepo.UpdateFields(ctx, instance.ID, fields)
	instance.AutoScore = &autoScore
	if manualTotal == 0 {
		instance.TotalScore = &autoScore
	}
}

// shouldReclaimSubmitted 判断已提交实例是否已达到保留期，可自动回收运行时环境。
func shouldReclaimSubmitted(instance *entity.ExperimentInstance, template *entity.ExperimentTemplate, now time.Time) bool {
	if instance == nil || instance.SubmittedAt == nil {
		return false
	}
	return now.Sub(*instance.SubmittedAt) >= submittedRetentionDuration(template)
}

// submittedRetentionDuration 返回已提交实例供教师查看的保留时长。
// 文档未单独定义该时长时，优先复用模板空闲超时，缺省回落为30分钟。
func submittedRetentionDuration(template *entity.ExperimentTemplate) time.Duration {
	if template != nil && template.IdleTimeout > 0 {
		return time.Duration(template.IdleTimeout) * time.Minute
	}
	return defaultSubmittedRetentionMinutes * time.Minute
}

// cleanupExpiredSnapshots 清理已销毁实例超过 30 天的快照数据。
func (s *ExperimentScheduler) cleanupExpiredSnapshots(ctx context.Context) {
	if s.instanceSvc == nil {
		return
	}
	instances, _, err := s.instanceRepo.List(ctx, &experimentrepo.InstanceListParams{
		Statuses: []int16{
			enum.InstanceStatusCompleted,
			enum.InstanceStatusExpired,
			enum.InstanceStatusDestroyed,
		},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return
	}
	now := time.Now()
	for _, instance := range instances {
		if !shouldCleanupSnapshotArchives(instance, now) {
			continue
		}
		snapshots, listErr := s.instanceSvc.snapshotRepo.ListByInstanceID(ctx, instance.ID)
		if listErr == nil {
			for _, snapshot := range snapshots {
				s.instanceSvc.deleteSnapshotArchive(ctx, snapshot)
			}
		}
		_ = s.instanceSvc.snapshotRepo.DeleteByInstanceID(ctx, instance.ID)
	}
}

// shouldCleanupSnapshotArchives 判断实例的运行时环境是否已销毁满 30 天，可清理快照归档。
// 自动回收后的实例终态可能仍为“已完成/已过期”，但 destroyed_at 记录的仍是运行时被回收的时间。
func shouldCleanupSnapshotArchives(instance *entity.ExperimentInstance, now time.Time) bool {
	if instance == nil || instance.DestroyedAt == nil {
		return false
	}
	switch instance.Status {
	case enum.InstanceStatusCompleted, enum.InstanceStatusExpired, enum.InstanceStatusDestroyed:
		return now.Sub(*instance.DestroyedAt) >= destroyedSnapshotRetention
	default:
		return false
	}
}

// pushCourseEndingWarning 在课程结束前 10 分钟向学生推送一次性预警。
func (s *ExperimentScheduler) pushCourseEndingWarning(ctx context.Context, instance *entity.ExperimentInstance, endingSoonCourses map[int64]time.Time) {
	if instance == nil || instance.CourseID == nil {
		return
	}
	endAt, ok := endingSoonCourses[*instance.CourseID]
	if !ok {
		return
	}
	if instance.Status != enum.InstanceStatusRunning && instance.Status != enum.InstanceStatusPaused && instance.Status != enum.InstanceStatusInitializing {
		return
	}

	key := fmt.Sprintf("%s%d:%d", cache.KeyExpCourseEndWarn, instance.ID, endAt.Unix())
	set, err := cache.SetNX(ctx, key, "1", courseEndWarningLead+time.Minute)
	if err != nil || !set {
		return
	}

	manager := ws.GetManager()
	if manager == nil {
		return
	}
	remainingMinutes := int(time.Until(endAt).Minutes())
	if remainingMinutes < 1 {
		remainingMinutes = 1
	}
	// 课程结束前预警当前先走模块04 WebSocket 通道。
	// 若后续确认该场景需要进入模块07站内通知，应在此业务节点补发 experiment.expiring 事件，
	// 仍通过 service 层接口解耦接入，不在调度器外层重复实现。
	_ = manager.SendToUser(instance.StudentID, buildInstanceWSMessage("course_end_warning", map[string]interface{}{
		"instance_id":       fmt.Sprintf("%d", instance.ID),
		"course_id":         fmt.Sprintf("%d", *instance.CourseID),
		"course_end_at":     endAt.UTC().Format(time.RFC3339),
		"remaining_minutes": remainingMinutes,
		"warning_message":   "课程即将结束，实验环境将在10分钟后回收",
	}))
}

// listReadyNodeNames 返回当前集群中 Ready 节点集合。
func (s *ExperimentScheduler) listReadyNodeNames(ctx context.Context) (map[string]struct{}, error) {
	nodes, err := s.k8sSvc.GetNodeStatus(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if node == nil || node.Status != "Ready" {
			continue
		}
		result[node.Name] = struct{}{}
	}
	return result, nil
}

// collectNodesNeedingPull 过滤仍需执行预拉取的就绪节点。
func collectNodesNeedingPull(statuses []*ImagePullNodeStatus, readyNodes map[string]struct{}) []string {
	result := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if status == nil {
			continue
		}
		if _, ok := readyNodes[status.NodeName]; !ok {
			continue
		}
		if status.Status == "pulled" || status.Status == "pulling" {
			continue
		}
		result = append(result, status.NodeName)
	}
	return result
}
