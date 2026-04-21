// instance_service_stream.go
// 模块04 — 实验环境：终端流与 SimEngine 通道辅助逻辑
// 为 WebSocket 入口提供统一的访问校验与目标信息解析

package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

// TerminalStreamInfo 远程终端只读流所需的目标信息。
type TerminalStreamInfo struct {
	Namespace     string
	PodName       string
	ContainerName string
}

// TerminalOutput 远程终端当前输出快照。
type TerminalOutput struct {
	Container string
	Data      string
}

// TerminalCommandOutput 表示一次终端命令执行结果。
type TerminalCommandOutput struct {
	Container string
	Command   string
	ExitCode  int
	Stdout    string
	Stderr    string
}

// SimEngineProxyTarget SimEngine WebSocket 代理目标信息。
type SimEngineProxyTarget struct {
	SessionID  string
	TargetURL  string
	InstanceID int64
}

// GetTerminalStreamInfo 获取教师远程查看学生终端所需的目标信息。
func (s *instanceService) GetTerminalStreamInfo(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*TerminalStreamInfo, error) {
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
	return s.resolveTerminalStreamInfo(ctx, instance, "")
}

// ExecuteTerminalCommand 在学生自己的实验实例中执行一条终端命令。
func (s *instanceService) ExecuteTerminalCommand(ctx context.Context, sc *svcctx.ServiceContext, id int64, containerName, command string) (*TerminalCommandOutput, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if instance.Status != enum.InstanceStatusRunning {
		return nil, errcode.ErrInstanceNotRunning
	}
	info, err := s.resolveTerminalStreamInfo(ctx, instance, containerName)
	if err != nil {
		return nil, err
	}

	result, err := s.k8sSvc.ExecInPod(ctx, info.Namespace, info.PodName, info.ContainerName, command)
	if err != nil {
		return nil, err
	}

	s.touchInstanceActivity(ctx, instance.ID)
	s.recordTerminalCommand(ctx, instance, info.ContainerName, command, result)

	return &TerminalCommandOutput{
		Container: info.ContainerName,
		Command:   command,
		ExitCode:  result.ExitCode,
		Stdout:    result.Stdout,
		Stderr:    result.Stderr,
	}, nil
}

// GetStudentTerminalOutput 获取学生自己实例的当前终端输出快照。
func (s *instanceService) GetStudentTerminalOutput(ctx context.Context, sc *svcctx.ServiceContext, id int64, containerName string, tailLines int) (*TerminalOutput, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if instance.Status != enum.InstanceStatusRunning {
		return nil, errcode.ErrInstanceNotRunning
	}
	info, err := s.resolveTerminalStreamInfo(ctx, instance, containerName)
	if err != nil {
		return nil, err
	}

	output, err := s.k8sSvc.GetPodLogs(ctx, info.Namespace, info.PodName, info.ContainerName, tailLines)
	if err != nil {
		return nil, err
	}
	return &TerminalOutput{
		Container: info.ContainerName,
		Data:      output,
	}, nil
}

// GetTerminalOutput 获取教师远程查看学生终端的当前输出快照。
func (s *instanceService) GetTerminalOutput(ctx context.Context, sc *svcctx.ServiceContext, id int64, tailLines int) (*TerminalOutput, error) {
	info, err := s.GetTerminalStreamInfo(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	output, err := s.k8sSvc.GetPodLogs(ctx, info.Namespace, info.PodName, info.ContainerName, tailLines)
	if err != nil {
		return nil, err
	}
	return &TerminalOutput{
		Container: info.ContainerName,
		Data:      output,
	}, nil
}

// GetGroupMemberTerminalOutput 获取组内成员只读查看的终端输出快照。
func (s *instanceService) GetGroupMemberTerminalOutput(ctx context.Context, sc *svcctx.ServiceContext, groupID, studentID int64, tailLines int) (*TerminalOutput, error) {
	if !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}

	if _, err := s.groupMemberRepo.GetByGroupAndStudent(ctx, groupID, sc.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrForbidden
		}
		return nil, err
	}

	if _, err := s.groupMemberRepo.GetByGroupAndStudent(ctx, groupID, studentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrForbidden
		}
		return nil, err
	}

	instances, err := s.instanceRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	targetInstance := buildLatestInstanceByStudent(instances)[studentID]
	if targetInstance == nil {
		return nil, errcode.ErrForbidden
	}

	info, err := s.resolveTerminalStreamInfo(ctx, targetInstance, "")
	if err != nil {
		return nil, err
	}

	output, err := s.k8sSvc.GetPodLogs(ctx, info.Namespace, info.PodName, info.ContainerName, tailLines)
	if err != nil {
		return nil, err
	}
	return &TerminalOutput{
		Container: info.ContainerName,
		Data:      output,
	}, nil
}

// GetSimEngineProxyTarget 获取 SimEngine WebSocket 代理目标。
// SimEngine 交互链路属于学生实验操作页面，必须校验当前 token 与实验实例、会话归属一致，
// 不允许课程教师、学校管理员或超管通过该交互通道代替学生操作仿真场景。
func (s *instanceService) GetSimEngineProxyTarget(ctx context.Context, sc *svcctx.ServiceContext, sessionID string) (*SimEngineProxyTarget, error) {
	instance, err := s.instanceRepo.GetBySimSessionID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrSimSessionNotFound
		}
		return nil, err
	}
	if sc == nil || !sc.IsStudent() || instance.StudentID != sc.UserID || instance.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}
	if instance.SimWebSocketURL == nil || *instance.SimWebSocketURL == "" {
		return nil, errcode.ErrSimSessionNotFound
	}
	return &SimEngineProxyTarget{
		SessionID:  sessionID,
		TargetURL:  *instance.SimWebSocketURL,
		InstanceID: instance.ID,
	}, nil
}

// TouchActivity 刷新实例最近操作时间。
func (s *instanceService) TouchActivity(ctx context.Context, id int64) {
	s.touchInstanceActivity(ctx, id)
}

// RecordSimEngineOperation 记录发往 SimEngine 的用户交互与时间控制操作。
// 模块04文档已约定 SimEngine WebSocket 使用顶层 `type + scene_code + payload` 协议，
// 这里按文档解析并落操作日志，避免继续沿用旧的 `data` 包装格式。
func (s *instanceService) RecordSimEngineOperation(ctx context.Context, sc *svcctx.ServiceContext, instanceID int64, payload []byte) {
	if len(payload) == 0 {
		return
	}

	instance, err := s.getAccessibleInstance(ctx, sc, instanceID)
	if err != nil {
		return
	}

	var envelope struct {
		Type      string          `json:"type"`
		SceneCode string          `json:"scene_code"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return
	}

	action := ""
	switch envelope.Type {
	case "action":
		action = enum.ActionSimInteraction
	case "control", "rewind_to":
		action = enum.ActionSimTimeControl
	default:
		return
	}

	var targetScene *string
	if envelope.SceneCode != "" {
		targetScene = &envelope.SceneCode
	}

	s.touchInstanceActivity(ctx, instance.ID)
	s.recordOpLog(ctx, instance.ID, sc.UserID, action, nil, targetScene, nil, nil, envelope.Payload)
}

// resolveTerminalStreamInfo 解析实例终端访问的命名空间、Pod 与容器信息。
func (s *instanceService) resolveTerminalStreamInfo(ctx context.Context, instance *entity.ExperimentInstance, containerName string) (*TerminalStreamInfo, error) {
	if instance.Status != enum.InstanceStatusRunning && instance.Status != enum.InstanceStatusCompleted {
		return nil, errcode.ErrInstanceNotRunning
	}
	if instance.Namespace == nil || *instance.Namespace == "" {
		return nil, errcode.ErrInstanceNotRunning
	}

	containers, err := s.containerRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return nil, err
	}
	sort.Slice(containers, func(i, j int) bool {
		if containers[i].Status == containers[j].Status {
			return containers[i].ContainerName < containers[j].ContainerName
		}
		return containers[i].Status > containers[j].Status
	})

	if containerName != "" {
		for _, container := range containers {
			if container.ContainerName == containerName && container.PodName != nil && *container.PodName != "" {
				return &TerminalStreamInfo{
					Namespace:     *instance.Namespace,
					PodName:       *container.PodName,
					ContainerName: container.ContainerName,
				}, nil
			}
		}
		return nil, errcode.ErrInvalidParams.WithMessage("目标容器不存在")
	}

	for _, container := range containers {
		if container.PodName == nil || *container.PodName == "" {
			continue
		}
		return &TerminalStreamInfo{
			Namespace:     *instance.Namespace,
			PodName:       *container.PodName,
			ContainerName: container.ContainerName,
		}, nil
	}
	return nil, errcode.ErrInstanceNotRunning
}

// touchInstanceActivity 刷新实例的最近操作时间。
func (s *instanceService) touchInstanceActivity(ctx context.Context, instanceID int64) {
	now := time.Now()
	_ = s.instanceRepo.UpdateLastActiveAt(ctx, instanceID, now)
}

// recordTerminalCommand 记录学生终端命令和执行结果。
func (s *instanceService) recordTerminalCommand(ctx context.Context, instance *entity.ExperimentInstance, containerName, command string, result *ExecResult) {
	if instance == nil || result == nil {
		return
	}

	commandOutput, detailPayloadMap := s.buildTerminalCommandAudit(ctx, instance.ID, result)
	detailPayloadMap["exit_code"] = result.ExitCode
	detailPayloadMap["stdout"] = truncateUTF8(result.Stdout, maxCommandOutputBytes)
	detailPayloadMap["stderr"] = truncateUTF8(result.Stderr, maxCommandOutputBytes)
	detailPayload, _ := json.Marshal(detailPayloadMap)
	targetContainer := containerName
	s.recordOpLog(ctx, instance.ID, instance.StudentID, enum.ActionTerminalCommand, &targetContainer, nil, &command, commandOutput, detailPayload)
}
