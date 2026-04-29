// instance_service_subresources.go
// 模块04 — 实验环境：实例子资源业务逻辑
// 负责检查点验证、手动评分、快照、操作日志、实验报告、教师指导消息

package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/storage"
	"github.com/lenschain/backend/internal/pkg/ws"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// VerifyCheckpoints 触发实例检查点验证。
// 支持单个自动检查点验证，也支持不指定检查点时批量验证所有自动检查点。
func (s *instanceService) VerifyCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.VerifyCheckpointReq) (*dto.VerifyCheckpointResp, error) {
	if err := s.enforceCheckpointRateLimit(ctx, sc.UserID); err != nil {
		return nil, err
	}

	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if instance.Status != enum.InstanceStatusRunning {
		return nil, errcode.ErrInstanceNotRunning
	}

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

	targets := make([]entity.TemplateCheckpoint, 0)
	if req != nil && req.CheckpointID != nil && *req.CheckpointID != "" {
		checkpointID, parseErr := snowflake.ParseString(*req.CheckpointID)
		if parseErr != nil {
			return nil, errcode.ErrCheckpointNotFound
		}
		for _, cp := range templateAggregate.Checkpoints {
			if cp.ID == checkpointID {
				targets = append(targets, *cp)
				break
			}
		}
		if len(targets) == 0 {
			return nil, errcode.ErrCheckpointNotFound
		}
	} else {
		for _, cp := range templateAggregate.Checkpoints {
			if cp.CheckType == enum.CheckTypeScript || cp.CheckType == enum.CheckTypeSimAssert {
				targets = append(targets, *cp)
			}
		}
	}

	results := make([]dto.CheckpointVerifyResultItem, 0, len(targets))
	for _, cp := range targets {
		if cp.CheckType == enum.CheckTypeManual {
			return nil, errcode.ErrInvalidParams.WithMessage("手动评分检查点不支持自动验证")
		}
		result := s.executeCheckpoint(ctx, instance, &cp)
		score := 0.0
		checkOutput := ""
		if result.Score != nil {
			score = *result.Score
		}
		if result.CheckOutput != nil {
			checkOutput = *result.CheckOutput
		}
		results = append(results, dto.CheckpointVerifyResultItem{
			CheckpointID: strconv.FormatInt(cp.ID, 10),
			Title:        cp.Title,
			IsPassed:     result.IsPassed != nil && *result.IsPassed,
			Score:        score,
			CheckOutput:  checkOutput,
			CheckedAt:    result.CheckedAt.UTC().Format(time.RFC3339),
		})
		for _, target := range s.resolveCheckpointTargetInstances(ctx, instance, &cp) {
			s.pushCourseMonitorCheckpoint(target, cp.Title, result.IsPassed != nil && *result.IsPassed)
		}
		detailPayload, _ := json.Marshal(map[string]interface{}{
			"checkpoint_id":    strconv.FormatInt(cp.ID, 10),
			"checkpoint_title": cp.Title,
			"is_passed":        result.IsPassed != nil && *result.IsPassed,
			"scope":            cp.Scope,
			"score":            score,
		})
		s.recordOpLog(ctx, id, sc.UserID, enum.ActionCheckpoint, cp.TargetContainer, nil, nil, nil, detailPayload)
	}

	s.touchInstanceActivity(ctx, instance.ID)

	return &dto.VerifyCheckpointResp{Results: results}, nil
}

// ListCheckpointResults 获取实例检查点结果列表。
func (s *instanceService) ListCheckpointResults(ctx context.Context, sc *svcctx.ServiceContext, id int64) ([]dto.InstanceCheckpointItem, error) {
	instance, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
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
	results, err := s.checkResultRepo.ListByInstanceID(ctx, id)
	if err != nil {
		return nil, err
	}

	resultByCheckpoint := make(map[int64]*entity.CheckpointResult, len(results))
	for _, result := range results {
		resultByCheckpoint[result.CheckpointID] = result
	}

	items := make([]dto.InstanceCheckpointItem, 0, len(templateAggregate.Checkpoints))
	for _, cp := range templateAggregate.Checkpoints {
		item := dto.InstanceCheckpointItem{
			CheckpointID: strconv.FormatInt(cp.ID, 10),
			Title:        cp.Title,
			CheckType:    cp.CheckType,
			Score:        cp.Score,
		}
		if result := resultByCheckpoint[cp.ID]; result != nil {
			score := 0.0
			if result.Score != nil {
				score = *result.Score
			}
			item.Result = &dto.InstanceCheckpointResult{
				ID:        strconv.FormatInt(result.ID, 10),
				IsPassed:  result.IsPassed != nil && *result.IsPassed,
				Score:     score,
				CheckedAt: result.CheckedAt.UTC().Format(time.RFC3339),
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// GradeCheckpoint 对单个手动评分检查点打分。
func (s *instanceService) GradeCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, resultID int64, req *dto.GradeCheckpointReq) error {
	result, err := s.checkResultRepo.GetByID(ctx, resultID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrCheckpointNotFound
		}
		return err
	}
	instance, err := s.getAccessibleInstance(ctx, sc, result.InstanceID)
	if err != nil {
		return err
	}
	allowed, err := s.canTeachInstance(ctx, sc, instance)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}

	checkpoint, err := s.checkpointRepo.GetByID(ctx, result.CheckpointID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrCheckpointNotFound
		}
		return err
	}
	if checkpoint.CheckType != enum.CheckTypeManual {
		return errcode.ErrInvalidParams.WithMessage("该检查点不是手动评分类型")
	}
	if req.Score > checkpoint.Score {
		return errcode.ErrInvalidParams.WithMessage("评分不能超过检查点满分")
	}

	now := time.Now()
	fields := map[string]interface{}{
		"is_passed":       req.Score >= checkpoint.Score,
		"score":           req.Score,
		"teacher_comment": req.Comment,
		"graded_by":       sc.UserID,
		"graded_at":       now,
		"updated_at":      now,
	}
	if checkpoint.Scope == enum.CheckpointScopeGroup {
		return s.persistManualCheckpointScore(ctx, instance, checkpoint, sc.UserID, req.Score, req.Comment, now)
	}
	return s.checkResultRepo.UpdateFields(ctx, resultID, fields)
}

// ManualGrade 对实验实例进行整体手动评分。
func (s *instanceService) ManualGrade(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ManualGradeReq) (*dto.ManualGradeResp, error) {
	instance, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	allowed, err := s.canTeachInstance(ctx, sc, instance)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}

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
	checkpointMap := make(map[int64]entity.TemplateCheckpoint, len(templateAggregate.Checkpoints))
	autoTotal := 0.0
	manualTotal := 0.0
	for _, cp := range templateAggregate.Checkpoints {
		checkpointMap[cp.ID] = *cp
		if cp.CheckType == enum.CheckTypeManual {
			manualTotal += cp.Score
		} else {
			autoTotal += cp.Score
		}
	}

	now := time.Now()
	manualScore := 0.0
	for _, grade := range req.CheckpointGrades {
		checkpointID, parseErr := snowflake.ParseString(grade.CheckpointID)
		if parseErr != nil {
			return nil, errcode.ErrCheckpointNotFound
		}
		cp, ok := checkpointMap[checkpointID]
		if !ok {
			return nil, errcode.ErrCheckpointNotFound
		}
		if cp.CheckType != enum.CheckTypeManual {
			return nil, errcode.ErrInvalidParams.WithMessage("该检查点不是手动评分类型")
		}
		if grade.Score > cp.Score {
			return nil, errcode.ErrInvalidParams.WithMessage("评分不能超过检查点满分")
		}

		if persistErr := s.persistManualCheckpointScore(ctx, instance, &cp, sc.UserID, grade.Score, grade.Comment, now); persistErr != nil {
			return nil, persistErr
		}
		manualScore += grade.Score
	}

	autoScore := 0.0
	if instance.AutoScore != nil {
		autoScore = *instance.AutoScore
	}

	totalScore := autoScore + manualScore
	scoreDetail := fmt.Sprintf("自动部分: %.0f/%.0f, 手动部分: %.0f/%.0f, 总分: %.0f", autoScore, autoTotal, manualScore, manualTotal, totalScore)
	if manualTotal > 0 && template.AutoWeight != nil && template.ManualWeight != nil && autoTotal > 0 {
		totalScore = (autoScore/autoTotal)*(*template.AutoWeight) + (manualScore/manualTotal)*(*template.ManualWeight)
		scoreDetail = fmt.Sprintf(
			"自动部分: %.0f/%.0f × %.0f%% = %.0f, 手动部分: %.0f/%.0f × %.0f%% = %.0f, 总分: %.0f",
			autoScore, autoTotal, *template.AutoWeight, (autoScore/autoTotal)*(*template.AutoWeight),
			manualScore, manualTotal, *template.ManualWeight, (manualScore/manualTotal)*(*template.ManualWeight),
			totalScore,
		)
	}

	if err := s.instanceRepo.UpdateFields(ctx, id, map[string]interface{}{
		"manual_score": manualScore,
		"total_score":  totalScore,
		"updated_at":   now,
	}); err != nil {
		return nil, err
	}
	instance.ManualScore = &manualScore
	instance.TotalScore = &totalScore
	instance.UpdatedAt = now

	detailPayload, _ := json.Marshal(map[string]interface{}{
		"overall_comment": req.OverallComment,
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionManualGrade, nil, nil, nil, nil, detailPayload)

	if err := s.syncCourseGradeIfNeeded(ctx, instance, template, req.OverallComment); err != nil {
		return nil, err
	}
	if err := s.dispatchExperimentGraded(ctx, instance, template.Title, totalScore); err != nil {
		return nil, err
	}

	return &dto.ManualGradeResp{
		InstanceID:  strconv.FormatInt(id, 10),
		AutoScore:   autoScore,
		ManualScore: manualScore,
		TotalScore:  totalScore,
		ScoreDetail: scoreDetail,
	}, nil
}

// ListSnapshots 获取实例快照列表。
func (s *instanceService) ListSnapshots(ctx context.Context, sc *svcctx.ServiceContext, id int64) ([]dto.SnapshotResp, error) {
	_, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	snapshots, err := s.snapshotRepo.ListByInstanceID(ctx, id)
	if err != nil {
		return nil, err
	}
	items := make([]dto.SnapshotResp, 0, len(snapshots))
	for _, snapshot := range snapshots {
		items = append(items, buildSnapshotResp(ctx, snapshot))
	}
	return items, nil
}

// CreateSnapshot 手动创建实例快照。
func (s *instanceService) CreateSnapshot(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.CreateSnapshotReq) (*dto.SnapshotResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	var description *string
	if req != nil {
		description = req.Description
	}
	snapshot, err := s.createInstanceSnapshot(ctx, instance, enum.SnapshotTypeManual, description)
	if err != nil {
		return nil, err
	}
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionSnapshotCreate, nil, nil, nil, nil, nil)
	resp := buildSnapshotResp(ctx, snapshot)
	return &resp, nil
}

// RestoreSnapshot 从指定快照恢复实验实例。
func (s *instanceService) RestoreSnapshot(ctx context.Context, sc *svcctx.ServiceContext, id, snapshotID int64) error {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return err
	}
	snapshot, err := s.snapshotRepo.GetByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrSnapshotNotFound
		}
		return err
	}
	if snapshot.InstanceID != id {
		return errcode.ErrSnapshotNotFound
	}

	releaseConcurrencyOnError := false
	switch instance.Status {
	case enum.InstanceStatusPaused, enum.InstanceStatusDestroyed, enum.InstanceStatusError, enum.InstanceStatusCompleted:
		releaseConcurrencyOnError = true
		_ = s.quotaRepo.IncrUsedConcurrency(ctx, sc.SchoolID, 1)
	}

	if instance.Status == enum.InstanceStatusRunning || instance.Status == enum.InstanceStatusInitializing {
		if err := s.teardownRuntimeEnvironment(ctx, instance); err != nil {
			if releaseConcurrencyOnError {
				_ = s.quotaRepo.DecrUsedConcurrency(ctx, sc.SchoolID, 1)
			}
			return err
		}
	}

	now := time.Now()
	if err := s.instanceRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status":            enum.InstanceStatusInitializing,
		"paused_at":         nil,
		"updated_at":        now,
		"started_at":        gorm.Expr("COALESCE(started_at, ?)", now),
		"last_active_at":    now,
		"namespace":         nil,
		"sim_session_id":    nil,
		"sim_websocket_url": nil,
	}); err != nil {
		if releaseConcurrencyOnError {
			_ = s.quotaRepo.DecrUsedConcurrency(ctx, sc.SchoolID, 1)
		}
		return err
	}

	templateAggregate, err := loadTemplateAggregate(
		ctx,
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
	if err != nil {
		return err
	}
	cronpkg.RunAsync("模块04快照恢复环境重建", func(asyncCtx context.Context) {
		s.provisionEnvironment(detachContext(ctx), instance, templateAggregate, stringifySnapshotID(snapshot), releaseConcurrencyOnError)
	})

	detailPayload, _ := json.Marshal(map[string]interface{}{
		"snapshot_id": strconv.FormatInt(snapshotID, 10),
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionSnapshotRestore, nil, nil, nil, nil, detailPayload)
	return nil
}

// DeleteSnapshot 删除指定快照。
func (s *instanceService) DeleteSnapshot(ctx context.Context, sc *svcctx.ServiceContext, id, snapshotID int64) error {
	instance, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return err
	}

	allowed := instance.StudentID == sc.UserID
	if !allowed {
		allowed, err = s.canTeachInstance(ctx, sc, instance)
		if err != nil {
			return err
		}
	}
	if !allowed {
		return errcode.ErrForbidden
	}

	snapshot, err := s.snapshotRepo.GetByID(ctx, snapshotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrSnapshotNotFound
		}
		return err
	}
	if snapshot.InstanceID != id {
		return errcode.ErrSnapshotNotFound
	}

	s.deleteSnapshotArchive(ctx, snapshot)
	if err := s.snapshotRepo.Delete(ctx, snapshotID); err != nil {
		return err
	}

	detailPayload, _ := json.Marshal(map[string]interface{}{
		"snapshot_id": strconv.FormatInt(snapshotID, 10),
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionSnapshotDelete, nil, nil, nil, nil, detailPayload)
	return nil
}

// ListOperationLogs 获取实例操作日志列表。
func (s *instanceService) ListOperationLogs(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.InstanceOpLogListReq) ([]dto.InstanceOpLogItem, int64, error) {
	_, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return nil, 0, err
	}
	logs, total, err := s.opLogRepo.List(ctx, &experimentrepo.OperationLogListParams{
		InstanceID:      id,
		Action:          req.Action,
		TargetContainer: req.TargetContainer,
		Page:            req.Page,
		PageSize:        req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.InstanceOpLogItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, dto.InstanceOpLogItem{
			ID:              strconv.FormatInt(log.ID, 10),
			Action:          log.Action,
			TargetContainer: log.TargetContainer,
			Command:         log.Command,
			Detail:          json.RawMessage(log.Detail),
			CreatedAt:       log.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, total, nil
}

// CreateReport 提交实验报告。
// 按验收标准，重复提交会覆盖上一次报告内容。
func (s *instanceService) CreateReport(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.CreateReportReq) (*dto.ReportResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if err := validateReportPayload(req.Content, req.FileURL, req.FileName, req.FileSize); err != nil {
		return nil, err
	}

	report, err := s.reportRepo.GetByInstanceAndStudent(ctx, id, sc.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	content := normalizeOptionalText(req.Content)
	fileURL := normalizeOptionalText(req.FileURL)
	fileName := normalizeOptionalText(req.FileName)
	now := time.Now()
	if report == nil {
		report = &entity.ExperimentReport{
			ID:          snowflake.Generate(),
			InstanceID:  id,
			StudentID:   instance.StudentID,
			Content:     content,
			FileURL:     fileURL,
			FileName:    fileName,
			FileSize:    req.FileSize,
			SubmittedAt: now,
		}
		if createErr := s.reportRepo.Create(ctx, report); createErr != nil {
			return nil, createErr
		}
	} else {
		if updateErr := s.reportRepo.UpdateFields(ctx, report.ID, map[string]interface{}{
			"content":      content,
			"file_url":     fileURL,
			"file_name":    fileName,
			"file_size":    req.FileSize,
			"submitted_at": now,
			"updated_at":   now,
		}); updateErr != nil {
			return nil, updateErr
		}
		report.Content = content
		report.FileURL = fileURL
		report.FileName = fileName
		report.FileSize = req.FileSize
		report.SubmittedAt = now
		report.UpdatedAt = now
	}

	s.recordOpLog(ctx, id, sc.UserID, enum.ActionReportSubmit, nil, nil, nil, nil, nil)
	return buildReportResp(report), nil
}

// GetReport 获取实验报告详情。
func (s *instanceService) GetReport(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ReportResp, error) {
	_, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	report, err := s.reportRepo.GetByInstanceID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrReportNotFound
		}
		return nil, err
	}
	return buildReportResp(report), nil
}

// UpdateReport 更新实验报告内容。
func (s *instanceService) UpdateReport(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateReportReq) (*dto.ReportResp, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if err := validateReportPayload(req.Content, req.FileURL, req.FileName, req.FileSize); err != nil {
		return nil, err
	}

	report, err := s.reportRepo.GetByInstanceAndStudent(ctx, id, instance.StudentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrReportNotFound
		}
		return nil, err
	}

	content := normalizeOptionalText(req.Content)
	fileURL := normalizeOptionalText(req.FileURL)
	fileName := normalizeOptionalText(req.FileName)
	now := time.Now()
	if err := s.reportRepo.UpdateFields(ctx, report.ID, map[string]interface{}{
		"content":    content,
		"file_url":   fileURL,
		"file_name":  fileName,
		"file_size":  req.FileSize,
		"updated_at": now,
	}); err != nil {
		return nil, err
	}
	report.Content = content
	report.FileURL = fileURL
	report.FileName = fileName
	report.FileSize = req.FileSize
	report.UpdatedAt = now

	s.recordOpLog(ctx, id, sc.UserID, enum.ActionReportUpdate, nil, nil, nil, nil, nil)
	return buildReportResp(report), nil
}

// SendGuidance 向学生发送教师指导消息。
// 消息会同时写入操作日志，并实时推送到学生实验界面。
func (s *instanceService) SendGuidance(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SendGuidanceReq) error {
	instance, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return err
	}
	allowed, err := s.canTeachInstance(ctx, sc, instance)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}

	detailPayload, _ := json.Marshal(map[string]interface{}{
		"type":    "guidance_message",
		"content": req.Content,
	})
	s.recordOpLog(ctx, id, sc.UserID, enum.ActionGuidanceMessage, nil, nil, nil, nil, detailPayload)

	manager := ws.GetManager()
	if manager != nil {
		teacherName := s.userNameQuerier.GetUserName(ctx, sc.UserID)
		_ = manager.SendToUser(instance.StudentID, buildInstanceWSMessage("guidance_message", map[string]interface{}{
			"teacher_name": teacherName,
			"content":      req.Content,
			"sent_at":      time.Now().UTC().Format(time.RFC3339),
		}))
	}
	return nil
}

// createInstanceSnapshot 创建实例快照并保存容器状态、SimEngine 状态。
func (s *instanceService) createInstanceSnapshot(ctx context.Context, instance *entity.ExperimentInstance, snapshotType int16, description *string) (*entity.InstanceSnapshot, error) {
	snapshot := &entity.InstanceSnapshot{
		ID:           snowflake.Generate(),
		InstanceID:   instance.ID,
		SnapshotType: snapshotType,
		Description:  description,
	}

	if instance.SimSessionID != nil && *instance.SimSessionID != "" {
		simSnapshot, err := s.simEngineSvc.CreateSnapshot(ctx, *instance.SimSessionID)
		if err == nil {
			stateJSON, _ := json.Marshal(simSnapshot)
			snapshot.SimEngineState = datatypes.JSON(stateJSON)
		}
	}

	runtimeStates, err := s.captureInstanceRuntimeState(ctx, instance)
	if err != nil {
		return nil, err
	}
	fullContainerStateJSON, err := encodeRuntimeContainerStates(runtimeStates)
	if err != nil {
		return nil, err
	}
	strippedContainerStateJSON, err := encodeRuntimeContainerStates(stripRuntimeArchiveData(runtimeStates))
	if err != nil {
		return nil, err
	}
	snapshot.ContainerStates = datatypes.JSON(strippedContainerStateJSON)

	objectKey, snapshotSize, err := s.uploadSnapshotArchive(ctx, snapshot, fullContainerStateJSON, json.RawMessage(snapshot.SimEngineState))
	if err != nil {
		return nil, err
	}
	snapshot.SnapshotDataURL = *objectKey
	snapshot.SnapshotSize = snapshotSize

	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

// buildSnapshotResp 构建快照响应。
func buildSnapshotResp(ctx context.Context, snapshot *entity.InstanceSnapshot) dto.SnapshotResp {
	resolvedURL := snapshot.SnapshotDataURL
	if snapshot != nil && storage.GetClient() != nil && snapshot.SnapshotDataURL != "" {
		if signedURL, err := storage.GetFileURL(ctx, snapshot.SnapshotDataURL, time.Hour); err == nil && signedURL != "" {
			resolvedURL = signedURL
		}
	}
	return dto.SnapshotResp{
		ID:               strconv.FormatInt(snapshot.ID, 10),
		InstanceID:       strconv.FormatInt(snapshot.InstanceID, 10),
		SnapshotType:     snapshot.SnapshotType,
		SnapshotTypeText: enum.GetSnapshotTypeText(snapshot.SnapshotType),
		SnapshotDataURL:  resolvedURL,
		ContainerStates:  json.RawMessage(snapshot.ContainerStates),
		SimEngineState:   json.RawMessage(snapshot.SimEngineState),
		Description:      snapshot.Description,
		CreatedAt:        snapshot.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// buildReportResp 构建实验报告响应。
func buildReportResp(report *entity.ExperimentReport) *dto.ReportResp {
	if report == nil {
		return nil
	}
	return &dto.ReportResp{
		ID:         strconv.FormatInt(report.ID, 10),
		InstanceID: strconv.FormatInt(report.InstanceID, 10),
		Content:    report.Content,
		FileURL:    report.FileURL,
		FileName:   report.FileName,
		FileSize:   report.FileSize,
		CreatedAt:  report.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  report.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
