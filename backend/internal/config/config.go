// config.go
// 配置管理模块
// 使用 Viper 加载 YAML 配置文件，支持环境变量覆盖

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	// Server HTTP 服务配置。
	Server    ServerConfig    `mapstructure:"server"`
	// Database PostgreSQL 数据库配置。
	Database  DatabaseConfig  `mapstructure:"database"`
	// Redis Redis 缓存配置。
	Redis     RedisConfig     `mapstructure:"redis"`
	// JWT JWT 认证配置。
	JWT       JWTConfig       `mapstructure:"jwt"`
	// MinIO 对象存储配置。
	MinIO     MinIOConfig     `mapstructure:"minio"`
	// NATS 消息队列配置。
	NATS      NATSConfig      `mapstructure:"nats"`
	// Snowflake 雪花 ID 生成器配置。
	Snowflake SnowflakeConfig `mapstructure:"snowflake"`
	// Log 日志配置。
	Log       LogConfig       `mapstructure:"log"`
	// SMS 短信网关配置。
	SMS       SMSConfig       `mapstructure:"sms"`
	// CORS 跨域配置。
	CORS      CORSConfig      `mapstructure:"cors"`
	// RateLimit 全局限流配置。
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	// K8s Kubernetes 相关配置。
	K8s       K8sConfig       `mapstructure:"k8s"`
	// SimEngine 仿真引擎配置。
	SimEngine SimEngineConfig `mapstructure:"sim_engine"`
}

// K8sConfig Kubernetes 集群配置
type K8sConfig struct {
	// InCluster 是否使用集群内配置。
	InCluster              bool          `mapstructure:"in_cluster"`
	// KubeConfigPath 集群外模式使用的 kubeconfig 路径。
	KubeConfigPath         string        `mapstructure:"kubeconfig_path"`
	// NamespacePrefix 实验环境命名空间前缀。
	NamespacePrefix        string        `mapstructure:"namespace_prefix"`
	// DefaultCPU 默认 CPU 资源配额。
	DefaultCPU             string        `mapstructure:"default_cpu"`
	// DefaultMemory 默认内存资源配额。
	DefaultMemory          string        `mapstructure:"default_memory"`
	// CollectorImageTemplate 混合实验采集器镜像模板。
	CollectorImageTemplate string        `mapstructure:"collector_image_template"`
	// Timeout 调用 K8s API 的超时时间。
	Timeout                time.Duration `mapstructure:"timeout"`
}

// SimEngineConfig SimEngine 仿真引擎 gRPC 配置
type SimEngineConfig struct {
	// GRPCAddr SimEngine Core 的 gRPC 服务地址。
	GRPCAddr   string        `mapstructure:"grpc_addr"`
	// Timeout 调用 SimEngine 的超时时间。
	Timeout    time.Duration `mapstructure:"timeout"`
	// MaxRetries 调用失败后的最大重试次数。
	MaxRetries int           `mapstructure:"max_retries"`
	// TLSEnabled 是否启用 gRPC TLS。
	TLSEnabled bool          `mapstructure:"tls_enabled"`
	// CertFile TLS 证书文件路径。
	CertFile   string        `mapstructure:"cert_file"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	// Host HTTP 服务监听地址。
	Host           string        `mapstructure:"host"`
	// Port HTTP 服务监听端口。
	Port           int           `mapstructure:"port"`
	// Mode Gin 运行模式。
	Mode           string        `mapstructure:"mode"`
	// ReadTimeout HTTP 请求读取超时时间。
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	// WriteTimeout HTTP 响应写入超时时间。
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	// MaxHeaderBytes HTTP 请求头最大字节数。
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
}

// DatabaseConfig PostgreSQL数据库配置
type DatabaseConfig struct {
	// Host PostgreSQL 主机地址。
	Host            string        `mapstructure:"host"`
	// Port PostgreSQL 端口。
	Port            int           `mapstructure:"port"`
	// User PostgreSQL 用户名。
	User            string        `mapstructure:"user"`
	// Password PostgreSQL 密码。
	Password        string        `mapstructure:"password"`
	// DBName PostgreSQL 数据库名。
	DBName          string        `mapstructure:"dbname"`
	// SSLMode PostgreSQL SSL 模式。
	SSLMode         string        `mapstructure:"sslmode"`
	// MaxIdleConns 连接池最大空闲连接数。
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	// MaxOpenConns 连接池最大打开连接数。
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	// ConnMaxLifetime 单个连接最大复用时长。
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	// LogLevel GORM 日志级别。
	LogLevel        string        `mapstructure:"log_level"`
}

// DSN 生成 PostgreSQL 连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// RedisConfig Redis缓存配置
type RedisConfig struct {
	// Host Redis 主机地址。
	Host         string `mapstructure:"host"`
	// Port Redis 端口。
	Port         int    `mapstructure:"port"`
	// Password Redis 密码。
	Password     string `mapstructure:"password"`
	// DB Redis 逻辑库编号。
	DB           int    `mapstructure:"db"`
	// PoolSize Redis 连接池大小。
	PoolSize     int    `mapstructure:"pool_size"`
	// MinIdleConns Redis 最小空闲连接数。
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// Addr 生成 Redis 连接地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// JWTConfig JWT认证配置
type JWTConfig struct {
	// AccessSecret Access Token 签名密钥。
	AccessSecret  string        `mapstructure:"access_secret"`
	// RefreshSecret Refresh Token 签名密钥。
	RefreshSecret string        `mapstructure:"refresh_secret"`
	// AccessExpire Access Token 有效期。
	AccessExpire  time.Duration `mapstructure:"access_expire"`
	// RefreshExpire Refresh Token 有效期。
	RefreshExpire time.Duration `mapstructure:"refresh_expire"`
	// TempExpire 临时 Token 有效期。
	TempExpire    time.Duration `mapstructure:"temp_expire"`
	// Issuer JWT 签发者标识。
	Issuer        string        `mapstructure:"issuer"`
}

// MinIOConfig MinIO对象存储配置
type MinIOConfig struct {
	// Endpoint MinIO 访问地址。
	Endpoint  string `mapstructure:"endpoint"`
	// AccessKey MinIO Access Key。
	AccessKey string `mapstructure:"access_key"`
	// SecretKey MinIO Secret Key。
	SecretKey string `mapstructure:"secret_key"`
	// UseSSL 是否通过 HTTPS 访问 MinIO。
	UseSSL    bool   `mapstructure:"use_ssl"`
	// Bucket 默认对象存储桶名称。
	Bucket    string `mapstructure:"bucket"`
	// Region MinIO / S3 区域标识。
	Region    string `mapstructure:"region"`
}

// NATSConfig NATS消息队列配置
type NATSConfig struct {
	// URL NATS 服务连接地址。
	URL           string        `mapstructure:"url"`
	// MaxReconnects 最大重连次数。
	MaxReconnects int           `mapstructure:"max_reconnects"`
	// ReconnectWait 每次重连等待时间。
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
}

// SnowflakeConfig 雪花ID生成器配置
type SnowflakeConfig struct {
	// NodeID 当前实例的雪花算法节点编号。
	NodeID int64 `mapstructure:"node_id"`
}

// LogConfig 日志配置
type LogConfig struct {
	// Level 日志级别。
	Level      string `mapstructure:"level"`
	// Format 日志格式。
	Format     string `mapstructure:"format"`
	// Output 日志输出目标。
	Output     string `mapstructure:"output"`
	// FilePath 文件输出模式下的日志路径。
	FilePath   string `mapstructure:"file_path"`
	// MaxSize 单个日志文件最大大小，单位 MB。
	MaxSize    int    `mapstructure:"max_size"`
	// MaxBackups 日志文件最大保留份数。
	MaxBackups int    `mapstructure:"max_backups"`
	// MaxAge 日志文件最大保留天数。
	MaxAge     int    `mapstructure:"max_age"`
	// Compress 是否压缩历史日志。
	Compress   bool   `mapstructure:"compress"`
}

// SMSConfig 短信网关配置
type SMSConfig struct {
	// Provider 短信服务提供商。
	Provider  string `mapstructure:"provider"`
	// AccessKey 短信服务 Access Key。
	AccessKey string `mapstructure:"access_key"`
	// SecretKey 短信服务 Secret Key。
	SecretKey string `mapstructure:"secret_key"`
	// SignName 短信签名。
	SignName  string `mapstructure:"sign_name"`
	// Region 短信服务区域标识。
	Region    string `mapstructure:"region"`
	// Endpoint 短信服务 API 访问地址；为空时使用提供商默认公网地址。
	Endpoint  string `mapstructure:"endpoint"`
	// SDKAppID 腾讯云短信应用 ID，仅 provider=tencent 时必填。
	SDKAppID  string `mapstructure:"sdk_app_id"`
}

// CORSConfig 跨域配置
type CORSConfig struct {
	// AllowedOrigins 允许跨域访问的来源列表。
	AllowedOrigins []string      `mapstructure:"allowed_origins"`
	// AllowedMethods 允许的 HTTP 方法列表。
	AllowedMethods []string      `mapstructure:"allowed_methods"`
	// AllowedHeaders 允许的请求头列表。
	AllowedHeaders []string      `mapstructure:"allowed_headers"`
	// MaxAge 预检请求缓存时长。
	MaxAge         time.Duration `mapstructure:"max_age"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// Enabled 是否启用全局限流。
	Enabled           bool `mapstructure:"enabled"`
	// RequestsPerSecond 每秒允许通过的请求数。
	RequestsPerSecond int  `mapstructure:"requests_per_second"`
	// Burst 突发流量桶容量。
	Burst             int  `mapstructure:"burst"`
}

// global 全局配置实例
var global *Config

// Load 加载配置文件
// configPath 为配置文件路径，为空则使用默认路径 ./configs/config.yaml
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
	}

	// 支持环境变量覆盖，前缀 LENSCHAIN_
	v.SetEnvPrefix("LENSCHAIN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	global = cfg
	return cfg, nil
}

// Get 获取全局配置实例
func Get() *Config {
	return global
}
