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
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	MinIO     MinIOConfig     `mapstructure:"minio"`
	NATS      NATSConfig      `mapstructure:"nats"`
	Snowflake SnowflakeConfig `mapstructure:"snowflake"`
	Log       LogConfig       `mapstructure:"log"`
	SMS       SMSConfig       `mapstructure:"sms"`
	CORS      CORSConfig      `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	K8s       K8sConfig       `mapstructure:"k8s"`
	SimEngine SimEngineConfig `mapstructure:"sim_engine"`
}

// K8sConfig Kubernetes 集群配置
type K8sConfig struct {
	InCluster              bool          `mapstructure:"in_cluster"`
	KubeConfigPath         string        `mapstructure:"kubeconfig_path"`
	NamespacePrefix        string        `mapstructure:"namespace_prefix"`
	DefaultCPU             string        `mapstructure:"default_cpu"`
	DefaultMemory          string        `mapstructure:"default_memory"`
	CollectorImageTemplate string        `mapstructure:"collector_image_template"`
	Timeout                time.Duration `mapstructure:"timeout"`
}

// SimEngineConfig SimEngine 仿真引擎 gRPC 配置
type SimEngineConfig struct {
	GRPCAddr   string        `mapstructure:"grpc_addr"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	TLSEnabled bool          `mapstructure:"tls_enabled"`
	CertFile   string        `mapstructure:"cert_file"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	Mode           string        `mapstructure:"mode"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
}

// DatabaseConfig PostgreSQL数据库配置
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
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
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// Addr 生成 Redis 连接地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// JWTConfig JWT认证配置
type JWTConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessExpire  time.Duration `mapstructure:"access_expire"`
	RefreshExpire time.Duration `mapstructure:"refresh_expire"`
	TempExpire    time.Duration `mapstructure:"temp_expire"`
	Issuer        string        `mapstructure:"issuer"`
}

// MinIOConfig MinIO对象存储配置
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
}

// NATSConfig NATS消息队列配置
type NATSConfig struct {
	URL           string        `mapstructure:"url"`
	MaxReconnects int           `mapstructure:"max_reconnects"`
	ReconnectWait time.Duration `mapstructure:"reconnect_wait"`
}

// SnowflakeConfig 雪花ID生成器配置
type SnowflakeConfig struct {
	NodeID int64 `mapstructure:"node_id"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// SMSConfig 短信网关配置
type SMSConfig struct {
	Provider  string `mapstructure:"provider"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	SignName  string `mapstructure:"sign_name"`
	Region    string `mapstructure:"region"`
}

// CORSConfig 跨域配置
type CORSConfig struct {
	AllowedOrigins []string      `mapstructure:"allowed_origins"`
	AllowedMethods []string      `mapstructure:"allowed_methods"`
	AllowedHeaders []string      `mapstructure:"allowed_headers"`
	MaxAge         time.Duration `mapstructure:"max_age"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerSecond int  `mapstructure:"requests_per_second"`
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
