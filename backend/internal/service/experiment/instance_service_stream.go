// instance_service_stream.go
// 模块04 — 实验环境：终端流与 SimEngine 通道辅助逻辑
// 为 WebSocket 入口提供统一的访问校验与目标信息解析

package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// TerminalProxyTarget xterm-server WebSocket 代理目标信息。
type TerminalProxyTarget struct {
	InstanceID    int64
	ContainerName string
	WebSocketURL  string
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

// ResolveTerminalProxyTarget 查找实例中的 xterm-server 工具容器并返回 WebSocket 代理目标。
// 当未指定容器名时，自动查找 xterm-server 工具容器；当指定的容器名恰好是 xterm-server 时使用 PTY 模式。
// 返回 nil, nil 表示实例未挂载 xterm-server。
func (s *instanceService) ResolveTerminalProxyTarget(ctx context.Context, sc *svcctx.ServiceContext, id int64, containerName string) (*TerminalProxyTarget, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	if instance.Status != enum.InstanceStatusRunning {
		return nil, errcode.ErrInstanceNotRunning
	}
	if instance.Namespace == nil || *instance.Namespace == "" {
		return nil, errcode.ErrInstanceNotRunning
	}

	containers, err := s.containerRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return nil, err
	}

	// 指定了容器名：仅当该容器是 xterm-server 时走 PTY 代理
	if containerName != "" {
		for _, c := range containers {
			if c.ContainerName != containerName {
				continue
			}
			if c.ToolKind != nil && *c.ToolKind == "xterm-server" && c.InternalIP != nil && *c.InternalIP != "" {
				return &TerminalProxyTarget{
					InstanceID:    instance.ID,
					ContainerName: c.ContainerName,
					WebSocketURL:  buildXtermWSURL(&c),
				}, nil
			}
			return nil, nil
		}
		return nil, nil
	}

	// 未指定容器名：自动查找 xterm-server 工具容器
	for _, c := range containers {
		if c.ToolKind == nil || *c.ToolKind != "xterm-server" {
			continue
		}
		if c.InternalIP == nil || *c.InternalIP == "" {
			continue
		}
		return &TerminalProxyTarget{
			InstanceID:    instance.ID,
			ContainerName: c.ContainerName,
			WebSocketURL:  buildXtermWSURL(&c),
		}, nil
	}

	return nil, nil
}

// buildXtermWSURL 根据容器配置构建 xterm-server WebSocket 地址。
// 优先使用数据库中记录的 ProxyURL，否则回退到默认端口 3000。
func buildXtermWSURL(c *entity.InstanceContainer) string {
	if c.ProxyURL != nil && *c.ProxyURL != "" {
		return *c.ProxyURL
	}
	return fmt.Sprintf("ws://%s:3000/ws", *c.InternalIP)
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
