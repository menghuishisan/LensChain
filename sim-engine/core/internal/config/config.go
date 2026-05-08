// config.go
// SimEngine Core 配置管理模块
// 使用 Viper 加载 YAML 配置文件，支持环境变量覆盖（前缀 LENSCHAIN_SIM）。
// 风格与 backend/internal/config/config.go 保持一致。

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config SimEngine Core 全局配置。
type Config struct {
	// Server 服务监听配置。
	Server ServerConfig `mapstructure:"server"`
	// Auth WebSocket JWT 鉴权配置。
	Auth AuthConfig `mapstructure:"auth"`
	// Orchestrator 场景算法容器 K8s 编排配置。
	// SceneManager 据此按需启动场景 Pod、维护 gRPC 连接池、空闲回收。
	Orchestrator OrchestratorConfig `mapstructure:"orchestrator"`
	// ObjectStorage 快照对象存储配置。
	ObjectStorage ObjectStorageConfig `mapstructure:"object_storage"`
	// Snapshot 快照运行时参数。
	Snapshot SnapshotConfig `mapstructure:"snapshot"`
	// Loop 后台循环节奏配置。
	Loop LoopConfig `mapstructure:"loop"`
}

// ServerConfig 服务监听配置。
type ServerConfig struct {
	// HTTPAddr HTTP/WebSocket 监听地址，例如 ":50052"。
	HTTPAddr string `mapstructure:"http_addr"`
	// GRPCAddr gRPC 监听地址，例如 ":50051"。
	GRPCAddr string `mapstructure:"grpc_addr"`
	// PublicBase 前端 WebSocket 连接的公开基地址。
	PublicBase string `mapstructure:"public_base"`
	// ReadHeaderTimeout HTTP 读取请求头超时时间。
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	// ShutdownTimeout 优雅关闭等待时间。
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// AuthConfig WebSocket JWT 鉴权配置。
type AuthConfig struct {
	// WSJWTSecret WebSocket JWT 验证密钥（与后端 jwt.access_secret 一致）。
	WSJWTSecret string `mapstructure:"ws_jwt_secret"`
	// WSJWTIssuer 期望的 JWT 签发方。
	WSJWTIssuer string `mapstructure:"ws_jwt_issuer"`
	// WSJWTAudience 期望的 JWT 受众。
	WSJWTAudience string `mapstructure:"ws_jwt_audience"`
}

// OrchestratorConfig 场景算法容器 K8s 编排配置。
// 实施由 internal/scene/k8s_orchestrator.go 完成，
// 详见 docs/modules/04-实验环境/06.1-场景编排实施方案.md。
type OrchestratorConfig struct {
	// InCluster 为 true 时使用集群内 ServiceAccount；false 走 KubeconfigPath。
	InCluster bool `mapstructure:"in_cluster"`
	// KubeconfigPath 集群外 kubeconfig 文件路径；空走 ~/.kube/config。
	KubeconfigPath string `mapstructure:"kubeconfig_path"`
	// Namespace 场景 Pod / Service 所在命名空间（建议与 SimEngine 同 ns）。
	Namespace string `mapstructure:"namespace"`
	// ImagePullSecretName 镜像拉取凭据 Secret 名称。
	ImagePullSecretName string `mapstructure:"image_pull_secret_name"`
	// ReadyTimeout Pod 启动 + gRPC HealthCheck 通过的总超时（对齐文档 §6.4 = 10s）。
	ReadyTimeout time.Duration `mapstructure:"ready_timeout"`
	// IdleTTL 场景 Pod 空闲多久后被自动回收。
	IdleTTL time.Duration `mapstructure:"idle_ttl"`
	// DefaultCPU / DefaultMemory 资源请求默认值。
	DefaultCPU    string `mapstructure:"default_cpu"`
	DefaultMemory string `mapstructure:"default_memory"`
	// LimitCPU / LimitMemory 资源上限。
	LimitCPU    string `mapstructure:"limit_cpu"`
	LimitMemory string `mapstructure:"limit_memory"`
}

// ObjectStorageConfig 快照对象存储配置。
type ObjectStorageConfig struct {
	// Endpoint 对象存储服务地址（例如 minio:9000）。
	Endpoint string `mapstructure:"endpoint"`
	// AccessKey 访问密钥。
	AccessKey string `mapstructure:"access_key"`
	// SecretKey 访问密钥对应的 Secret。
	SecretKey string `mapstructure:"secret_key"`
	// UseSSL 是否启用 HTTPS。
	UseSSL bool `mapstructure:"use_ssl"`
	// Bucket 快照存储桶名称。
	Bucket string `mapstructure:"bucket"`
	// Region 对象存储区域。
	Region string `mapstructure:"region"`
	// ObjectPrefix 快照对象的统一前缀。
	ObjectPrefix string `mapstructure:"object_prefix"`
	// EncryptionKey 32 字节快照加密密钥。
	EncryptionKey string `mapstructure:"encryption_key"`
	// PresignDuration 预签名 URL 有效期。
	PresignDuration time.Duration `mapstructure:"presign_duration"`
}

// SnapshotConfig 快照运行时参数。
type SnapshotConfig struct {
	// InitTimeout 初始化对象存储客户端的超时时间。
	InitTimeout time.Duration `mapstructure:"init_timeout"`
}

// LoopConfig 后台循环节奏。
type LoopConfig struct {
	// ClockInterval 时钟推进间隔。
	ClockInterval time.Duration `mapstructure:"clock_interval"`
	// TeacherSummaryInterval 教师概览刷新间隔。
	TeacherSummaryInterval time.Duration `mapstructure:"teacher_summary_interval"`
	// AutoSnapshotInterval 自动快照间隔。
	AutoSnapshotInterval time.Duration `mapstructure:"auto_snapshot_interval"`
}

// Load 加载配置文件。configPath 为空时按默认搜索路径定位 config.yaml。
// 同时启用 LENSCHAIN_SIM_* 环境变量覆盖（与 backend 的 LENSCHAIN_* 风格保持一致，加 SIM 前缀做隔离）。
func Load(configPath string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
		v.AddConfigPath("/app/configs")
	}

	v.SetEnvPrefix("LENSCHAIN_SIM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取 SimEngine 配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析 SimEngine 配置文件失败: %w", err)
	}
	return cfg, nil
}
