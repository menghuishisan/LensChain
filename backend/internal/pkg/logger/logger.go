// logger.go
// 该文件封装平台统一的日志初始化与输出入口，负责根据配置决定日志级别、编码格式和输出
// 位置。中间件、基础设施和业务层都通过这里获取结构化日志器，避免项目里混用多套日志实现。

package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/lenschain/backend/internal/config"
)

// 全局日志实例
var (
	L = zap.NewNop()
	S = L.Sugar()
)

// Init 初始化日志系统
func Init(cfg *config.LogConfig) error {
	level := parseLevel(cfg.Level)
	encoder := buildEncoder(cfg.Format)

	var cores []zapcore.Core

	// 控制台输出
	if cfg.Output == "stdout" || cfg.Output == "" {
		consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, consoleCore)
	}

	// 文件输出
	if cfg.Output == "file" || cfg.FilePath != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		fileCore := zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level)
		cores = append(cores, fileCore)
	}

	if len(cores) == 0 {
		consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, consoleCore)
	}

	core := zapcore.NewTee(cores...)
	L = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0), zap.AddStacktrace(zapcore.ErrorLevel))
	S = L.Sugar()

	return nil
}

// parseLevel 解析日志级别
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// buildEncoder 构建日志编码器
func buildEncoder(format string) zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if format == "console" {
		return zapcore.NewConsoleEncoder(encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

// Sync 刷新日志缓冲区
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
