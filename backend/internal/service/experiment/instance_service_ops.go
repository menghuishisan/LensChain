// instance_service_ops.go
// 模块04 — 实验环境：实验实例操作（暂停、恢复、重启、提交、销毁、心跳）
// 从 instance_service.go 拆分，保持文件体量合理
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/datatypes"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/ws"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// 暂停实验
// ---------------------------------------------------------------------------

// Pause 暂停实验环境
// POST /api/v1/experiment-instances/:id/pause
func (s *instanceService) Pause(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.PauseInstanceResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	if instance.Status != enum.InstanceStatusRunning {
		return nil, errcode.ErrInstanceNotRunning
	}

	snapshot, err := s.createInstanceSnapshot(ctx, instance, enum.SnapshotTypePause, nil)
	if err != nil {
		return nil, err
	}

	if err := s.pauseRuntimeEnvironment(ctx, instance); err != nil {
		return nil, err
	}
	if err := s.syncPausedContainerState(ctx, instance.ID); err != nil {
		return nil, err
	}
	_ = s.quotaRepo.DecrUsedConcurrency(ctx, instance.SchoolID, 1)
	if instance.CourseID != nil {
		s.activateNextQueuedInstance(ctx, *instance.CourseID)
	}

	// 更新实例状态
	//
	// 注意：暂停时不能清掉 namespace 字段。namespace 是 Resume 重建 Pod 的目标
	// 容器（PVC / Service / NetworkPolicy 仍在其中），清掉等于让 Resume 失去定位
	// 资源的能力。SimSessionID / SimWebSocketURL 是动态会话，必须清。
	now := time.Now()
	fields := map[string]interface{}{
		"status":            enum.InstanceStatusPaused,
		"paused_at":         now,
		"updated_at":        now,
		"sim_session_id":    nil,
		"sim_websocket_url": nil,
	}
	if err := s.instanceRepo.UpdateFields(ctx, id, fields); err != nil {
		return nil, err
	}
	_ = cache.Set(ctx, fmt.Sprintf("%s:%d", cache.KeyExpInstanceStatus, id),
		fmt.Sprintf("%d", enum.InstanceStatusPaused), 24*time.Hour)
	if instance.GroupID != nil {
		s.refreshGroupStatus(ctx, *instance.GroupID)
	}

	// 记录操作日志
	pauseDetail, _ := json.Marshal(map[string]interface{}{
		"snapshot_id":  strconv.FormatInt(snapshot.ID, 10),
		"snapshot_type": enum.SnapshotTypePause,
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionPause, nil, nil, nil, nil, pauseDetail)
	s.pushCourseMonitorStatusChange(instance, enum.InstanceStatusRunning, enum.InstanceStatusPaused)

	return &dto.PauseInstanceResp{
		InstanceID: strconv.FormatInt(id, 10),
		Status:     enum.InstanceStatusPaused,
		StatusText: enum.GetInstanceStatusText(enum.InstanceStatusPaused),
		SnapshotID: strconv.FormatInt(snapshot.ID, 10),
		PausedAt:   now.Format(time.RFC3339),
	}, nil
}

// ---------------------------------------------------------------------------
// 恢复实验
// ---------------------------------------------------------------------------

// Resume 恢复实验环境
// POST /api/v1/experiment-instances/:id/resume
func (s *instanceService) Resume(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ResumeInstanceReq) (*dto.ResumeInstanceResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	if instance.Status != enum.InstanceStatusPaused {
		return nil, errcode.ErrInstanceAlreadyPaused
	}

	// 并发限制检查
	runningCount, _ := s.instanceRepo.CountRunningByStudent(ctx, sc.UserID)
	maxPerStudent := 2
	schoolQuota, _ := s.quotaRepo.GetBySchoolID(ctx, sc.SchoolID)
	if schoolQuota != nil && schoolQuota.MaxPerStudent > 0 {
		maxPerStudent = schoolQuota.MaxPerStudent
	}
	if runningCount >= int64(maxPerStudent) {
		return nil, errcode.ErrInstanceAlreadyExists
	}

	var requestedSnapshot *string
	if req != nil {
		requestedSnapshot = req.SnapshotID
	}
	snapshot, err := s.resolveResumeSnapshot(ctx, id, requestedSnapshot)
	if err != nil {
		return nil, err
	}
	_ = s.quotaRepo.IncrUsedConcurrency(ctx, sc.SchoolID, 1)

	// 更新状态为恢复中
	now := time.Now()
	fields := map[string]interface{}{
		"status":         enum.InstanceStatusInitializing,
		"last_active_at": now,
		"updated_at":     now,
	}
	if err := s.instanceRepo.UpdateFields(ctx, id, fields); err != nil {
		_ = s.quotaRepo.DecrUsedConcurrency(ctx, sc.SchoolID, 1)
		return nil, err
	}
	if instance.GroupID != nil {
		s.refreshGroupStatus(ctx, *instance.GroupID)
	}

	// 异步恢复环境，统一走公共后台任务入口。
	cronpkg.RunAsync("实验实例恢复环境", func(context.Context) {
		asyncCtx := detachContext(ctx)
		templateAggregate, _ := loadTemplateAggregate(
			asyncCtx,
			s.templateRepo,
			s.templateContainerRepo,
			s.checkpointRepo,
			s.initScriptRepo,
			s.simSceneRepo,
			nil,
			nil,
			nil,
			instance.TemplateID,
		)
		if templateAggregate == nil {
			logger.L.Error("恢复实验环境失败：加载模板聚合数据为空",
				zap.Int64("instance_id", instance.ID),
				zap.Int64("template_id", instance.TemplateID),
			)
			_ = s.instanceRepo.UpdateFields(asyncCtx, instance.ID, map[string]interface{}{
				"status":        enum.InstanceStatusError,
				"error_message": "恢复失败：实验模板数据加载异常",
				"updated_at":    time.Now(),
			})
			_ = s.quotaRepo.DecrUsedConcurrency(asyncCtx, instance.SchoolID, 1)
			if instance.CourseID != nil {
				s.activateNextQueuedInstance(asyncCtx, *instance.CourseID)
			}
			return
		}
		s.provisionEnvironment(asyncCtx, instance, templateAggregate, stringifySnapshotID(snapshot), true)
	})

	// 记录操作日志
	resumeDetail, _ := json.Marshal(map[string]interface{}{
		"snapshot_id": stringifySnapshotID(snapshot),
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionResume, nil, nil, nil, nil, resumeDetail)
	s.pushCourseMonitorStatusChange(instance, int(enum.InstanceStatusPaused), int(enum.InstanceStatusInitializing))

	return &dto.ResumeInstanceResp{
		InstanceID:            strconv.FormatInt(id, 10),
		Status:                enum.InstanceStatusInitializing,
		StatusText:            enum.GetInstanceStatusText(enum.InstanceStatusInitializing),
		EstimatedReadySeconds: 15,
	}, nil
}

// destroySimSession 关闭实例的 SimEngine 会话；失败仅记日志，由 TTL 兜底过期回收，
// 不阻塞 K8s 资源释放（K8s 才是真正占用 CPU / 内存 / 存储的部分）。Pause 与 Destroy
// 共用此清理逻辑。
func (s *instanceService) destroySimSession(ctx context.Context, instance *entity.ExperimentInstance) {
	if instance == nil || instance.SimSessionID == nil || *instance.SimSessionID == "" {
		return
	}
	if err := s.simEngineSvc.DestroySession(ctx, *instance.SimSessionID); err != nil {
		logger.L.Warn("销毁 SimEngine 会话失败，会话将由 TTL 自动过期回收",
			zap.Int64("instance_id", instance.ID),
			zap.String("session_id", *instance.SimSessionID),
			zap.Error(err),
		)
	}
}

// pauseRuntimeEnvironment 暂停时的运行时清理：销毁 SimEngine 会话 + 删除 namespace
// 内全部 Pod。Service / NetworkPolicy / PVC / namespace 全部保留，让 Resume 路径可
// 以在同一 namespace 复挂同名 PVC、复用既有 Service 路由。详见 K8s 客户端
// `DeletePodsInNamespace` 注释里的根因说明。
func (s *instanceService) pauseRuntimeEnvironment(ctx context.Context, instance *entity.ExperimentInstance) error {
	if instance == nil {
		return nil
	}
	s.destroySimSession(ctx, instance)
	if instance.Namespace != nil && *instance.Namespace != "" {
		if err := s.k8sSvc.DeletePodsInNamespace(ctx, *instance.Namespace); err != nil {
			return err
		}
	}
	return nil
}

// destroyRuntimeEnvironment 销毁时的运行时清理：销毁 SimEngine 会话 + 删除整个
// namespace。namespace 级联删除会一并回收 PVC / Service / NetworkPolicy / Pod，对应
// F-39 "重新开始：从初始状态" 的语义。失败时必须返回错误，否则 K8s 资源会泄漏占用
// 节点配额。
func (s *instanceService) destroyRuntimeEnvironment(ctx context.Context, instance *entity.ExperimentInstance) error {
	if instance == nil {
		return nil
	}
	s.destroySimSession(ctx, instance)
	if instance.Namespace != nil && *instance.Namespace != "" {
		if err := s.k8sSvc.DeleteNamespace(ctx, *instance.Namespace); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 重新开始
// ---------------------------------------------------------------------------

// Restart 重新开始实验
// POST /api/v1/experiment-instances/:id/restart
func (s *instanceService) Restart(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CreateInstanceResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	// 只有已完成/已超时/错误/已销毁状态可以重新开始
	allowRestart := map[int16]bool{
		enum.InstanceStatusCompleted: true,
		enum.InstanceStatusExpired:   true,
		enum.InstanceStatusError:     true,
		enum.InstanceStatusDestroyed: true,
	}
	if !allowRestart[instance.Status] {
		return nil, errcode.ErrInstanceNotRunning
	}

	// 销毁旧环境（如果还在运行）
	_ = s.destroyEnvironment(ctx, instance)

	// 创建新实例
	templateIDStr := strconv.FormatInt(instance.TemplateID, 10)
	req := &dto.CreateInstanceReq{
		TemplateID: templateIDStr,
	}
	if instance.CourseID != nil {
		cidStr := strconv.FormatInt(*instance.CourseID, 10)
		req.CourseID = &cidStr
	}
	if instance.LessonID != nil {
		lidStr := strconv.FormatInt(*instance.LessonID, 10)
		req.LessonID = &lidStr
	}
	if instance.AssignmentID != nil {
		aidStr := strconv.FormatInt(*instance.AssignmentID, 10)
		req.AssignmentID = &aidStr
	}
	if instance.GroupID != nil {
		gidStr := strconv.FormatInt(*instance.GroupID, 10)
		req.GroupID = &gidStr
	}

	// 记录操作日志（在创建新实例之前，只记录在原实例上）
	restartDetail, _ := json.Marshal(map[string]interface{}{
		"previous_attempt_no": instance.AttemptNo,
		"previous_status":    instance.Status,
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionRestart, nil, nil, nil, nil, restartDetail)

	return s.Create(ctx, sc, req)
}

// ---------------------------------------------------------------------------
// 提交实验
// ---------------------------------------------------------------------------

// Submit 提交实验
// POST /api/v1/experiment-instances/:id/submit
func (s *instanceService) Submit(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmitInstanceResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	if instance.Status != enum.InstanceStatusRunning {
		return nil, errcode.ErrInstanceNotRunning
	}

	// 获取模板检查点
	templateAggregate, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		nil,
		s.checkpointRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		instance.TemplateID,
	)
	if err != nil {
		return nil, err
	}
	template := templateAggregate.Template

	// 执行所有自动检查点验证
	var autoScore, autoTotal, manualTotal float64
	details := make([]dto.SubmitScoreDetail, 0, len(templateAggregate.Checkpoints))

	for _, cp := range templateAggregate.Checkpoints {
		detail := dto.SubmitScoreDetail{
			CheckpointID: strconv.FormatInt(cp.ID, 10),
			Title:        cp.Title,
			CheckType:    cp.CheckType,
			MaxScore:     cp.Score,
		}

		if cp.CheckType == enum.CheckTypeManual {
			manualTotal += cp.Score
			status := "pending_review"
			detail.Status = &status
		} else {
			autoTotal += cp.Score
			// 执行自动验证
			result := s.executeCheckpoint(ctx, instance, cp)
			passed := false
			if result.IsPassed != nil {
				passed = *result.IsPassed
			}
			detail.IsPassed = &passed
			score := float64(0)
			if result.Score != nil {
				score = *result.Score
			}
			detail.Score = &score
			autoScore += score
		}

		details = append(details, detail)
	}

	// 计算总分
	now := time.Now()
	fields := map[string]interface{}{
		"status":       enum.InstanceStatusCompleted,
		"auto_score":   autoScore,
		"submitted_at": now,
		"updated_at":   now,
	}

	// 如果没有手动评分项，直接计算总分
	if manualTotal == 0 {
		fields["total_score"] = autoScore
	} else if template.AutoWeight != nil && template.ManualWeight != nil {
		// 混合评分：暂时只记录自动分
		fields["auto_score"] = autoScore
	}

	if err := s.instanceRepo.UpdateFields(ctx, id, fields); err != nil {
		return nil, err
	}
	instance.Status = enum.InstanceStatusCompleted
	instance.AutoScore = &autoScore
	instance.SubmittedAt = &now
	instance.UpdatedAt = now
	if instance.GroupID != nil {
		s.refreshGroupStatus(ctx, *instance.GroupID)
	}

	// 记录操作日志
	submitDetail, _ := json.Marshal(map[string]interface{}{
		"auto_score":   autoScore,
		"auto_total":   autoTotal,
		"manual_total": manualTotal,
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionSubmit, nil, nil, nil, nil, submitDetail)
	s.pushCourseMonitorStatusChange(instance, int(enum.InstanceStatusRunning), int(enum.InstanceStatusCompleted))
	s.pushCourseMonitorSubmitted(instance, autoScore, manualTotal > 0)

	if manualTotal == 0 {
		totalScore := autoScore
		instance.TotalScore = &totalScore
		if err := s.syncCourseGradeIfNeeded(ctx, instance, template, nil); err != nil {
			return nil, err
		}
		if err := s.dispatchExperimentGraded(ctx, instance, template.Title, totalScore); err != nil {
			return nil, err
		}
	}

	return &dto.SubmitInstanceResp{
		InstanceID: strconv.FormatInt(id, 10),
		Status:     enum.InstanceStatusCompleted,
		StatusText: enum.GetInstanceStatusText(enum.InstanceStatusCompleted),
		Scores: dto.SubmitScoresInfo{
			AutoScore:   autoScore,
			AutoTotal:   autoTotal,
			ManualTotal: manualTotal,
			Details:     details,
		},
		CompletedAt: now.Format(time.RFC3339),
	}, nil
}

// ---------------------------------------------------------------------------
// 销毁实验
// ---------------------------------------------------------------------------

// Destroy 销毁实验环境（学生自行销毁）
func (s *instanceService) Destroy(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	instance, err := s.getManageableInstance(ctx, sc, id)
	if err != nil {
		return err
	}

	// 已销毁/已提交的不需要再销毁
	if instance.Status == enum.InstanceStatusDestroyed || instance.Status == enum.InstanceStatusCompleted {
		return nil
	}

	if err := s.destroyEnvironment(ctx, instance); err != nil {
		return err
	}

	// 记录操作日志
	destroyDetail, _ := json.Marshal(map[string]interface{}{
		"previous_status": instance.Status,
		"trigger":         "user_initiated",
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionDestroy, nil, nil, nil, nil, destroyDetail)
	s.pushCourseMonitorStatusChange(instance, int(instance.Status), int(enum.InstanceStatusDestroyed))

	return nil
}

// ForceDestroy 强制销毁实验环境（教师/管理员）
func (s *instanceService) ForceDestroy(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	instance, err := s.getManageableInstance(ctx, sc, id)
	if err != nil {
		return err
	}

	if err := s.destroyEnvironment(ctx, instance); err != nil {
		return err
	}

	// 记录操作日志
	forceDestroyDetail, _ := json.Marshal(map[string]interface{}{
		"previous_status": instance.Status,
		"trigger":         "force_by_admin",
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionForceDestroy, nil, nil, nil, nil, forceDestroyDetail)
	s.pushCourseMonitorStatusChange(instance, int(instance.Status), int(enum.InstanceStatusDestroyed))

	return nil
}

// destroyEnvironment 销毁实验环境（K8s + SimEngine）
func (s *instanceService) destroyEnvironment(ctx context.Context, instance *entity.ExperimentInstance) error {
	if err := s.destroyRuntimeEnvironment(ctx, instance); err != nil {
		return err
	}

	// 更新实例状态
	now := time.Now()
	fields := map[string]interface{}{
		"status":            enum.InstanceStatusDestroyed,
		"destroyed_at":      now,
		"updated_at":        now,
		"namespace":         nil,
		"sim_session_id":    nil,
		"sim_websocket_url": nil,
	}
	_ = s.instanceRepo.UpdateFields(ctx, instance.ID, fields)

	// 释放并发配额
	_ = s.quotaRepo.DecrUsedConcurrency(ctx, instance.SchoolID, 1)
	if instance.CourseID != nil {
		s.activateNextQueuedInstance(ctx, *instance.CourseID)
	}

	// 清除 Redis 缓存
	_ = cache.Del(ctx, fmt.Sprintf("%s:%d", cache.KeyExpInstanceStatus, instance.ID))
	_ = cache.Del(ctx, fmt.Sprintf("%s:%d", cache.KeyExpHeartbeat, instance.ID))
	if instance.GroupID != nil {
		s.refreshGroupStatus(ctx, *instance.GroupID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// 心跳
// ---------------------------------------------------------------------------

// Heartbeat 心跳上报
// POST /api/v1/experiment-instances/:id/heartbeat
func (s *instanceService) Heartbeat(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HeartbeatReq) (*dto.HeartbeatResp, error) {
	if err := s.enforceHeartbeatRateLimit(ctx, sc.UserID); err != nil {
		return nil, err
	}

	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	if instance.Status != enum.InstanceStatusRunning {
		return &dto.HeartbeatResp{
			Status:           instance.Status,
			RemainingMinutes: 0,
			IdleWarning:      false,
		}, nil
	}

	// 心跳仅表示连接仍然在线，不刷新真实操作时间。
	now := time.Now()

	// 更新 Redis 心跳缓存
	_ = cache.Set(ctx, fmt.Sprintf("%s:%d", cache.KeyExpHeartbeat, id), now.Format(time.RFC3339), 5*time.Minute)

	// 计算剩余时间
	remainingMinutes := 0
	idleWarning := false

	template, _ := s.templateRepo.GetByID(ctx, instance.TemplateID)
	if template != nil && template.MaxDuration != nil && instance.StartedAt != nil {
		elapsed := now.Sub(*instance.StartedAt)
		remaining := time.Duration(*template.MaxDuration)*time.Minute - elapsed
		if remaining > 0 {
			remainingMinutes = int(remaining.Minutes())
		}
	}

	// 空闲超时警告（距离空闲超时 ≤5 分钟）
	if template != nil && instance.LastActiveAt != nil {
		idleElapsed := now.Sub(*instance.LastActiveAt)
		idleTimeout := time.Duration(template.IdleTimeout) * time.Minute
		if idleTimeout-idleElapsed <= 5*time.Minute {
			idleWarning = true
		}
	}

	manager := ws.GetManager()
	if manager != nil {
		if idleWarning {
			if err := s.dispatchExperimentExpiring(ctx, instance, 5); err != nil {
				return nil, err
			}
			_ = manager.SendToUser(instance.StudentID, buildInstanceWSMessage("idle_warning", map[string]interface{}{
				"remaining_minutes": 5,
				"message":           "您的实验环境将在5分钟后因空闲超时被回收，请继续操作或手动暂停",
			}))
		}
		if remainingMinutes > 0 && remainingMinutes <= 10 {
			if err := s.dispatchExperimentExpiring(ctx, instance, remainingMinutes); err != nil {
				return nil, err
			}
			_ = manager.SendToUser(instance.StudentID, buildInstanceWSMessage("duration_warning", map[string]interface{}{
				"remaining_minutes": remainingMinutes,
				"message":           fmt.Sprintf("实验剩余时间%d分钟，请尽快完成并提交", remainingMinutes),
			}))
		}
	}

	return &dto.HeartbeatResp{
		Status:           instance.Status,
		RemainingMinutes: remainingMinutes,
		IdleWarning:      idleWarning,
	}, nil
}

// ---------------------------------------------------------------------------
// 内部辅助
// ---------------------------------------------------------------------------

// recordOpLog 记录实例操作日志
func (s *instanceService) recordOpLog(ctx context.Context, instanceID, studentID int64, action string, targetContainer, targetScene, command, commandOutput *string, detail json.RawMessage) {
	log := &entity.InstanceOperationLog{
		ID:              snowflake.Generate(),
		InstanceID:      instanceID,
		StudentID:       studentID,
		Action:          action,
		TargetContainer: targetContainer,
		TargetScene:     targetScene,
		Command:         command,
		CommandOutput:   commandOutput,
		Detail:          datatypes.JSON(detail),
	}
	_ = s.opLogRepo.Create(ctx, log)
}

// dispatchExperimentExpiring 在实验即将超时时向当前学生发送站内信事件。
func (s *instanceService) dispatchExperimentExpiring(ctx context.Context, instance *entity.ExperimentInstance, minutes int) error {
	if s.eventDispatcher == nil || instance == nil || instance.StudentID == 0 || minutes <= 0 {
		return nil
	}
	return s.eventDispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:   "experiment.expiring",
		ReceiverIDs: []string{strconv.FormatInt(instance.StudentID, 10)},
		Params: map[string]interface{}{
			"minutes": minutes,
		},
		SourceModule: "module_04",
		SourceType:   "experiment_instance",
		SourceID:     strconv.FormatInt(instance.ID, 10),
	})
}

// dispatchExperimentGraded 在实验评分完成后向学生发送站内信事件。
func (s *instanceService) dispatchExperimentGraded(ctx context.Context, instance *entity.ExperimentInstance, experimentName string, score float64) error {
	if s.eventDispatcher == nil || instance == nil || instance.StudentID == 0 {
		return nil
	}
	return s.eventDispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:   "experiment.graded",
		ReceiverIDs: []string{strconv.FormatInt(instance.StudentID, 10)},
		Params: map[string]interface{}{
			"experiment_name": experimentName,
			"score":           score,
		},
		SourceModule: "module_04",
		SourceType:   "experiment_instance",
		SourceID:     strconv.FormatInt(instance.ID, 10),
	})
}
