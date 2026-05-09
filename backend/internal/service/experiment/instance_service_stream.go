// instance_service_stream.go
// 模块04 — 实验环境：终端流与 SimEngine 通道辅助逻辑
// 为 WebSocket 入口提供统一的访问校验与目标信息解析

package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
)

// terminalToolWebSocketPath 是 tool_kind="terminal" 镜像必须暴露的 WebSocket 路径。
//
// 这是平台 ↔ 镜像之间的协议合约（不是 service 层业务逻辑）：所有以 tool_kind=terminal
// 注册到平台的镜像都必须在该路径上提供 PTY WebSocket。当前唯一实现是 deploy/images/
// tools/xterm-server。如果将来引入新的终端工具镜像（如 ttyd），其 Dockerfile 必须把内部
// WS 端点路由到 "/ws"——平台契约不会为单个镜像让路。
//
// 端口不在此处声明：每个镜像的端口由 images.default_ports / template_containers.ports
// JSON 字段管理（mergePorts 解析），通过 resolveContainerFirstPort 数据驱动获取。
const terminalToolWebSocketPath = "/ws"

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
//
// 不暴露任何 URL：实验 Pod 的网络可达性必须通过 K8s API 的 SPDY portforward 隧道实现，
// 不允许使用 Pod IP / Service ClusterIP / NodePort（详见 k8s_portforward.go 与
// docs/modules/09-部署与运维/02-基础设施设计.md §2.4）。
//
// 调用方拿到该目标后应调用 InstanceService.DialPodPort(ctx, Namespace, PodName, Port)
// 取得 net.Conn，再交给 gorilla/websocket Dialer.NetDialContext 完成 WS 拨号。
type TerminalProxyTarget struct {
	InstanceID    int64
	ContainerName string
	Namespace     string
	PodName       string
	Port          int
	// WebSocketPath xterm-server 内部的 WebSocket 路径（"/ws"）。
	WebSocketPath string
}

// ToolProxyTarget 工具 iframe 反代目标信息（code-server / blockscout / VNC 桌面等）。
//
// 与 TerminalProxyTarget 的区别：本类型不携带固定 WebSocketPath——工具反代是完整的 HTTP +
// WS 双协议透传（主资源、子资源、IDE 内部 WS 等），具体路径由镜像自身决定，平台层只负责
// 透传。同样不暴露 URL，拨号仅走 SPDY portforward 隧道。
type ToolProxyTarget struct {
	InstanceID    int64
	ToolKind      string
	ContainerName string
	Namespace     string
	PodName       string
	Port          int
}

// SimEngineProxyTarget SimEngine WebSocket 代理目标信息。
//
// UpstreamToken 是后端为代理本次拨号现签的 SimWS token，绑定 (UserID, SessionID, InstanceID,
// Audience=sim-engine)，与学生 access token 完全无关。SimEngine Core 通过 audience/session_id/
// instance_id 严格校验，确保 token 不能跨会话/跨用户使用。
// 不应把学生 access token 透传给 SimEngine Core：access token 是高权限通用 token，且其 claims
// 不携带 session_id/instance_id/aud，无法通过 Core 的 validateJWTClaims 校验。
type SimEngineProxyTarget struct {
	SessionID     string
	TargetURL     string
	InstanceID    int64
	UpstreamToken string
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

// ResolveTerminalProxyTarget 查找实例中的终端工具容器（images.tool_kind = 'terminal'）
// 并返回 WebSocket 代理目标。
// 当未指定容器名时，自动查找终端工具容器；当指定的容器名恰好是终端工具容器时使用 PTY 模式。
// 返回 nil, nil 表示实例未挂载终端工具容器。
// 终端 kind 取值规范见 docs/modules/04-实验环境/02-数据库设计.md §2.16。
func (s *instanceService) ResolveTerminalProxyTarget(ctx context.Context, sc *svcctx.ServiceContext, id int64, containerName string) (*TerminalProxyTarget, error) {
	namespace, containers, err := s.loadOwnedRunningInstanceContainers(ctx, sc, id)
	if err != nil {
		return nil, err
	}

	// 指定了容器名：仅当该容器是终端工具容器时走 PTY 代理
	if containerName != "" {
		for _, c := range containers {
			if c.ContainerName != containerName {
				continue
			}
			if !isReadyToolContainer(c, "terminal") {
				return nil, nil
			}
			return s.buildTerminalProxyTarget(ctx, id, namespace, c)
		}
		return nil, nil
	}

	// 未指定容器名：自动查找终端工具容器
	for _, c := range containers {
		if !isReadyToolContainer(c, "terminal") {
			continue
		}
		return s.buildTerminalProxyTarget(ctx, id, namespace, c)
	}

	return nil, nil
}

// ResolveToolProxyTarget 按 tool_kind 查找已就绪的工具容器（code-server / blockscout / VNC / monitor 等）。
//
// 与 ResolveTerminalProxyTarget 的区别：
//   - 本函数 toolKind 必填，不接受容器名覆盖（iframe 场景下前端已按 kind 选了 tab）。
//   - 返回不携 WebSocketPath 的 ToolProxyTarget：handler 负责透传 HTTP 请求路径 / WS upgrade。
//
// 返回 errcode.ErrInvalidParams 表示实例未挂载该 toolKind 容器。调用者必须是实例所有者
func (s *instanceService) ResolveToolProxyTarget(ctx context.Context, sc *svcctx.ServiceContext, id int64, toolKind string) (*ToolProxyTarget, error) {
	toolKind = strings.TrimSpace(toolKind)
	if toolKind == "" {
		return nil, errcode.ErrInvalidParams.WithMessage("tool_kind 不能为空")
	}
	namespace, containers, err := s.loadOwnedRunningInstanceContainers(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		if !isReadyToolContainer(c, toolKind) {
			continue
		}
		port, err := s.resolveContainerFirstPort(ctx, c)
		if err != nil {
			return nil, err
		}
		return &ToolProxyTarget{
			InstanceID:    id,
			ToolKind:      toolKind,
			ContainerName: c.ContainerName,
			Namespace:     namespace,
			PodName:       *c.PodName,
			Port:          port,
		}, nil
	}
	return nil, errcode.ErrInvalidParams.WithMessage("实例未挂载该 tool_kind 容器或容器未就绪")
}

// ToolProxyAccess 工具反代访问凭证签发结果。
//
// 包含已签 token、cookie path 与有效期。Cookie 写入与 HTTP 响应组装由 handler 完成
// （那是 HTTP 层职责），但"为本次请求签什么 token"是业务决策，归 service 层。
type ToolProxyAccess struct {
	Token      string
	CookiePath string
	ProxyPath  string
	ExpiresIn  time.Duration
}

// IssueToolProxyAccess 完成业务校验并签发工具反代凭证。
//
// 该方法整合两步：
//  1. ResolveToolProxyTarget：校验学生本人 + 实例 Running + 该 toolKind 容器存在；
//  2. jwtpkg.GenerateToolProxyToken：签 ToolProxyClaims，TTL 复用 cfg.AccessExpire。
//
// handler 拿到 token 后直接 SetCookie + 返回响应，不参与"签什么"的决策。
func (s *instanceService) IssueToolProxyAccess(ctx context.Context, sc *svcctx.ServiceContext, id int64, toolKind string) (*ToolProxyAccess, error) {
	target, err := s.ResolveToolProxyTarget(ctx, sc, id, toolKind)
	if err != nil {
		return nil, err
	}
	expire := s.toolProxyTokenExpire()
	token, err := jwtpkg.GenerateToolProxyToken(
		sc.UserID,
		sc.SchoolID,
		id,
		target.ToolKind,
		target.Namespace,
		target.PodName,
		target.Port,
		expire,
	)
	if err != nil {
		return nil, fmt.Errorf("sign tool proxy token: %w", err)
	}
	cookiePath := fmt.Sprintf("/instance/%d/%s", id, target.ToolKind)
	return &ToolProxyAccess{
		Token:      token,
		CookiePath: cookiePath,
		ProxyPath:  cookiePath + "/",
		ExpiresIn:  expire,
	}, nil
}

// toolProxyTokenExpire 工具反代凭证有效期。复用 access token 的过期策略，确保学生
// 主会话过期时反代 cookie 同步失效（前端会拿不到新 access token，后续刷新也失败）。
func (s *instanceService) toolProxyTokenExpire() time.Duration {
	cfg := config.Get().JWT.AccessExpire
	if cfg <= 0 {
		return 30 * time.Minute
	}
	return cfg
}

// loadOwnedRunningInstanceContainers 加载当前学生本人拥有且运行中的实例与其容器列表。
//
// 返回 (namespace, containers, error)。终端与工具反代走同一路径：都要求“本人 + Running + 有 ns”，
// 提取为辅助函数避免两份鉴权逻辑实现（任何一侧问题会同时修正）。
func (s *instanceService) loadOwnedRunningInstanceContainers(ctx context.Context, sc *svcctx.ServiceContext, id int64) (string, []*entity.InstanceContainer, error) {
	instance, err := s.getOwnedInstance(ctx, sc, id)
	if err != nil {
		return "", nil, err
	}
	if instance.Status != enum.InstanceStatusRunning {
		return "", nil, errcode.ErrInstanceNotRunning
	}
	if instance.Namespace == nil || *instance.Namespace == "" {
		return "", nil, errcode.ErrInstanceNotRunning
	}
	containers, err := s.containerRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return "", nil, err
	}
	return *instance.Namespace, containers, nil
}

// isReadyToolContainer 判断 InstanceContainer 是否为已就绪且匹配指定 tool_kind 的工具容器。
// "已就绪" 指 PodName 已写入（K8s 调度完成），不再要求 InternalIP——SPDY 隔道
// 通过 (ns, podName, port) 寧址，与 Pod IP 无关。
func isReadyToolContainer(c *entity.InstanceContainer, kind string) bool {
	if c == nil {
		return false
	}
	if c.ToolKind == nil || *c.ToolKind != kind {
		return false
	}
	if c.PodName == nil || *c.PodName == "" {
		return false
	}
	return true
}

// buildTerminalProxyTarget 把已就绪的终端工具容器封装为 TerminalProxyTarget。
// WebSocket 路径固定为 terminalToolWebSocketPath（平台与镜像的协议契约，不是业务逻辑）；
// 端口走 resolveContainerFirstPort 数据驱动。
func (s *instanceService) buildTerminalProxyTarget(ctx context.Context, instanceID int64, namespace string, c *entity.InstanceContainer) (*TerminalProxyTarget, error) {
	port, err := s.resolveContainerFirstPort(ctx, c)
	if err != nil {
		return nil, err
	}
	return &TerminalProxyTarget{
		InstanceID:    instanceID,
		ContainerName: c.ContainerName,
		Namespace:     namespace,
		PodName:       *c.PodName,
		Port:          port,
		WebSocketPath: terminalToolWebSocketPath,
	}, nil
}

// resolveContainerFirstPort 从 template_container.ports / image.default_ports 解析首个端口。
//
// 路径：instance_container.template_container_id → template_containers.ports（含 image
// 默认端口的合并结果），取首个 PortSpec.ContainerPort。任何镜像换版本 / 教师改端口都会
// 自然反映到这里，不需要更新 service 层代码。名称不带 "terminal"：本函数体零终端
// 特异逻辑，终端与工具反代都调用同一份实现。
func (s *instanceService) resolveContainerFirstPort(ctx context.Context, ic *entity.InstanceContainer) (int, error) {
	if ic == nil || ic.TemplateContainerID == 0 {
		return 0, fmt.Errorf("instance container has no template_container_id")
	}
	tc, err := s.templateContainerRepo.GetByID(ctx, ic.TemplateContainerID)
	if err != nil {
		return 0, fmt.Errorf("load template container %d: %w", ic.TemplateContainerID, err)
	}
	var image *entity.Image
	if tc.ImageVersionID != 0 && s.imageVersionRepo != nil {
		if iv, ivErr := s.imageVersionRepo.GetByID(ctx, tc.ImageVersionID); ivErr == nil && iv != nil && s.imageRepo != nil {
			if img, imgErr := s.imageRepo.GetByID(ctx, iv.ImageID); imgErr == nil {
				image = img
			}
		}
	}
	specs := mergePorts(image, json.RawMessage(tc.Ports))
	if len(specs) == 0 || specs[0].ContainerPort <= 0 {
		return 0, fmt.Errorf("template container %d has no valid port mapping", tc.ID)
	}
	return specs[0].ContainerPort, nil
}

// DialPodPort 通过 K8s API 的 SPDY portforward 隔道建立到 Pod 指定端口的 net.Conn。
//
// handler 不直接持有 K8sService，所有跨层拨号经此方法。调用者拿到的 (ns, pod, port)
// 必须来自 Resolve***ProxyTarget 返回的 Target，业务边界在 Resolve 阶段完成校验（是否本人
// 、实例是否运行、是否为该实例的容器）。本方法不重复达这些检查。
func (s *instanceService) DialPodPort(ctx context.Context, namespace, podName string, port int) (net.Conn, error) {
	if s.k8sSvc == nil {
		return nil, fmt.Errorf("k8s service is not configured")
	}
	return s.k8sSvc.DialPodPort(ctx, namespace, podName, port)
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

	// 现签一个仅 backend → SimEngine Core 一跳使用的 SimWS token。
	// access_mode = "interactive"：上方已校验调用者必须是 instance.StudentID 本人（IsStudent 且
	// instance.StudentID == sc.UserID），所以本端点签出的 token 始终是交互模式；
	// 教师/管理员观察通道（read-only）应在专用监控接口里签 access_mode = "readonly"，不复用本端点。
	upstreamToken, err := jwtpkg.GenerateSimWSToken(
		sc.UserID,
		sc.SchoolID,
		sc.Roles,
		sessionID,
		strconv.FormatInt(instance.ID, 10),
		"interactive",
		0, // 0 → 复用 cfg.AccessExpire（30m），与学生原 access token 同步
	)
	if err != nil {
		return nil, err
	}

	return &SimEngineProxyTarget{
		SessionID:     sessionID,
		TargetURL:     *instance.SimWebSocketURL,
		InstanceID:    instance.ID,
		UpstreamToken: upstreamToken,
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
