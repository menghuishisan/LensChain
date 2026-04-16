// sim_engine_client.go
// 模块04 — 实验环境：SimEngine 通信服务真实实现
// 基于 protobuf 生成的 gRPC 客户端与 SimEngine Core 通信

package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/errcode"
	simenginev1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_engine/v1"
	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
)

// simEngineClient SimEngine gRPC 客户端实现
type simEngineClient struct {
	conn    *grpc.ClientConn
	client  simenginev1.SimEngineServiceClient
	cfg     config.SimEngineConfig
	timeout time.Duration
}

// NewSimEngineService 创建 SimEngine gRPC 客户端实例
func NewSimEngineService(cfg config.SimEngineConfig) (SimEngineService, error) {
	var opts []grpc.DialOption

	if cfg.TLSEnabled && cfg.CertFile != "" {
		creds, err := credentials.NewClientTLSFromFile(cfg.CertFile, "")
		if err != nil {
			return nil, fmt.Errorf("加载 TLS 证书失败: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(cfg.GRPCAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("连接 SimEngine gRPC 失败: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &simEngineClient{
		conn:    conn,
		client:  simenginev1.NewSimEngineServiceClient(conn),
		cfg:     cfg,
		timeout: timeout,
	}, nil
}

// Close 关闭 gRPC 连接
func (c *simEngineClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// CreateSession 创建仿真会话
func (c *simEngineClient) CreateSession(ctx context.Context, req *CreateSimSessionRequest) (*SimSession, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	scenes := make([]*simenginev1.SceneConfig, 0, len(req.Scenes))
	for _, scene := range req.Scenes {
		scenes = append(scenes, &simenginev1.SceneConfig{
			SceneCode:            scene.SceneCode,
			ScenarioId:           scene.ScenarioID,
			LinkGroupId:          scene.LinkGroupID,
			LinkGroupCode:        scene.LinkGroupCode,
			ParamsJson:           scene.Params,
			InitialStateJson:     scene.InitialState,
			DataSourceConfigJson: scene.DataSourceConfig,
			LayoutPositionJson:   scene.LayoutPosition,
			DataSourceMode:       toProtoDataSourceMode(scene.DataSourceMode),
			SharedStateJson:      scene.SharedState,
		})
	}

	response, err := c.client.CreateSession(callCtx, &simenginev1.CreateSessionRequest{
		InstanceId:        fmt.Sprintf("%d", req.InstanceID),
		StudentId:         fmt.Sprintf("%d", req.StudentID),
		Scenes:            scenes,
		LinkageEnabled:    req.LinkageEnabled,
		SessionConfigJson: req.SessionConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errcode.ErrSimSessionCreateFailed, err)
	}

	if response.GetErrorMessage() != "" {
		return nil, fmt.Errorf("%w: %s", errcode.ErrSimSessionCreateFailed, response.GetErrorMessage())
	}

	return &SimSession{
		SessionID:        response.GetSessionId(),
		InstanceID:       response.GetInstanceId(),
		Status:           sessionStatusText(response.GetStatus()),
		WebSocketURL:     response.GetWebsocketUrl(),
		ActiveSceneCodes: append([]string(nil), response.GetActiveSceneCodes()...),
	}, nil
}

// DestroySession 销毁仿真会话
func (c *simEngineClient) DestroySession(ctx context.Context, sessionID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.DestroySession(callCtx, &simenginev1.DestroySessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return err
	}
	if !response.GetSuccess() {
		return fmt.Errorf("销毁仿真会话失败: %s", response.GetErrorMessage())
	}
	return nil
}

// GetSessionState 获取仿真会话状态
func (c *simEngineClient) GetSessionState(ctx context.Context, sessionID string) (*SimSessionState, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.GetSessionState(callCtx, &simenginev1.GetSessionStateRequest{
		SessionId: sessionID,
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, errcode.ErrSimSessionNotFound
		}
		return nil, err
	}

	return &SimSessionState{
		SessionID:        response.GetSessionId(),
		InstanceID:       response.GetInstanceId(),
		Status:           sessionStatusText(response.GetStatus()),
		SimTick:          response.GetSimTick(),
		SimTimeSeconds:   response.GetSimTimeSeconds(),
		Speed:            response.GetSpeed(),
		ActiveSceneCodes: append([]string(nil), response.GetActiveSceneCodes()...),
		LinkGroupCodes:   append([]string(nil), response.GetLinkGroupCodes()...),
		SceneState:       response.GetSceneStateJson(),
		LastAction:       response.GetLastAction(),
		UpdatedAtMS:      response.GetUpdatedAtMs(),
	}, nil
}

// SendInteraction 发送交互指令
func (c *simEngineClient) SendInteraction(ctx context.Context, sessionID string, interaction *SimInteraction) (*SimInteractionResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.SendInteraction(callCtx, &simenginev1.SendInteractionRequest{
		SessionId:  sessionID,
		SceneCode:  interaction.SceneCode,
		ActionCode: interaction.ActionCode,
		ParamsJson: interaction.Params,
		ActorId:    interaction.ActorID,
		RoleKey:    interaction.RoleKey,
	})
	if err != nil {
		return nil, err
	}

	return &SimInteractionResult{
		Success:      response.GetSuccess(),
		Data:         response.GetDataJson(),
		ErrorMessage: response.GetErrorMessage(),
	}, nil
}

// GetInteractionSchema 获取场景交互 schema
func (c *simEngineClient) GetInteractionSchema(ctx context.Context, sessionID string, sceneCode string) (*SimInteractionSchema, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.GetInteractionSchema(callCtx, &simenginev1.GetInteractionSchemaRequest{
		SessionId: sessionID,
		SceneCode: sceneCode,
	})
	if err != nil {
		return nil, err
	}

	actionsJSON, err := json.Marshal(response.GetActions())
	if err != nil {
		return nil, fmt.Errorf("序列化交互 schema 失败: %w", err)
	}

	return &SimInteractionSchema{
		SceneCode: response.GetSceneCode(),
		Actions:   actionsJSON,
	}, nil
}

// ControlTime 控制仿真时间
func (c *simEngineClient) ControlTime(ctx context.Context, sessionID string, action string, params json.RawMessage) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.ControlTime(callCtx, &simenginev1.ControlTimeRequest{
		SessionId:  sessionID,
		Command:    toProtoTimeControlCommand(action),
		ParamsJson: params,
	})
	if err != nil {
		return err
	}
	if !response.GetSuccess() {
		return fmt.Errorf("仿真时间控制失败: %s", response.GetErrorMessage())
	}
	return nil
}

// CreateSnapshot 创建仿真快照
func (c *simEngineClient) CreateSnapshot(ctx context.Context, sessionID string) (*SimSnapshot, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.CreateSnapshot(callCtx, &simenginev1.CreateSnapshotRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return nil, err
	}

	return &SimSnapshot{
		SnapshotID:   response.GetSnapshotId(),
		SessionID:    response.GetSessionId(),
		SnapshotType: response.GetSnapshotType(),
		SimTick:      response.GetSimTick(),
		ObjectURL:    response.GetObjectUrl(),
		CreatedAtMS:  response.GetCreatedAtMs(),
	}, nil
}

// RestoreSnapshot 恢复仿真快照
func (c *simEngineClient) RestoreSnapshot(ctx context.Context, sessionID string, snapshotID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.RestoreSnapshot(callCtx, &simenginev1.RestoreSnapshotRequest{
		SessionId:  sessionID,
		SnapshotId: snapshotID,
	})
	if err != nil {
		return err
	}
	if !response.GetSuccess() {
		return fmt.Errorf("恢复仿真快照失败: %s", response.GetErrorMessage())
	}
	return nil
}

// StartDataCollection 开始数据采集（混合实验用）
func (c *simEngineClient) StartDataCollection(ctx context.Context, sessionID string, dataCfg json.RawMessage) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.StartDataCollection(callCtx, &simenginev1.StartDataCollectionRequest{
		SessionId:           sessionID,
		CollectorConfigJson: dataCfg,
	})
	if err != nil {
		return err
	}
	if !response.GetSuccess() {
		return fmt.Errorf("启动数据采集失败: %s", response.GetErrorMessage())
	}
	return nil
}

// StopDataCollection 停止数据采集
func (c *simEngineClient) StopDataCollection(ctx context.Context, sessionID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.StopDataCollection(callCtx, &simenginev1.StopDataCollectionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return err
	}
	if !response.GetSuccess() {
		return fmt.Errorf("停止数据采集失败: %s", response.GetErrorMessage())
	}
	return nil
}

// toProtoDataSourceMode 将字符串数据源模式转换为协议枚举。
func toProtoDataSourceMode(mode string) simscenariov1.DataSourceMode {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "collection":
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_COLLECTION
	case "dual":
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_DUAL
	case "simulation":
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_SIMULATION
	default:
		return simscenariov1.DataSourceMode_DATA_SOURCE_MODE_UNSPECIFIED
	}
}

// toProtoTimeControlCommand 将字符串命令转换为协议枚举。
func toProtoTimeControlCommand(action string) simenginev1.TimeControlCommand {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "play":
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_PLAY
	case "pause":
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_PAUSE
	case "step":
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_STEP
	case "set_speed":
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_SET_SPEED
	case "reset":
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_RESET
	case "resume":
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_RESUME
	default:
		return simenginev1.TimeControlCommand_TIME_CONTROL_COMMAND_UNSPECIFIED
	}
}

// sessionStatusText 将协议会话状态映射为内部文本编码。
func sessionStatusText(status simenginev1.SessionStatus) string {
	switch status {
	case simenginev1.SessionStatus_SESSION_STATUS_CREATING:
		return "creating"
	case simenginev1.SessionStatus_SESSION_STATUS_RUNNING:
		return "running"
	case simenginev1.SessionStatus_SESSION_STATUS_PAUSED:
		return "paused"
	case simenginev1.SessionStatus_SESSION_STATUS_STOPPED:
		return "stopped"
	case simenginev1.SessionStatus_SESSION_STATUS_ERROR:
		return "error"
	default:
		return ""
	}
}

// isNotFoundError 判断是否为"未找到"类错误
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(strings.ToLower(msg), "not found") || strings.Contains(msg, "不存在")
}
