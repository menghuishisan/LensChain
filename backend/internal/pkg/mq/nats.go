// nats.go
// 该文件封装平台事件总线使用的 NATS 连接与基础发布订阅能力，主要服务于跨模块异步事件
// 派发，例如模块业务触发通知事件后由模块07消费处理。它的职责是统一消息结构和连接管理，
// 不直接承载具体通知模板渲染或业务补偿逻辑。

package mq

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// 全局 NATS 连接
var conn *nats.Conn

var errNATSNotInitialized = errors.New("NATS客户端未初始化")

// Event 事件消息结构
type Event struct {
	EventCode  string      `json:"event_code"`  // 事件编码（如 user.created, course.published）
	Module     string      `json:"module"`      // 来源模块
	ReceiverID int64       `json:"receiver_id"` // 接收者用户ID（0表示广播）
	SchoolID   int64       `json:"school_id"`   // 学校ID
	Data       interface{} `json:"data"`        // 事件数据
	Timestamp  time.Time   `json:"timestamp"`   // 事件时间
}

// Init 初始化 NATS 连接
func Init(cfg *config.NATSConfig) error {
	var err error
	conn, err = nats.Connect(cfg.URL,
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.L.Warn("NATS连接断开", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.L.Info("NATS重新连接成功", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.L.Error("NATS错误", zap.Error(err))
		}),
	)
	if err != nil {
		return fmt.Errorf("连接NATS失败: %w", err)
	}

	logger.L.Info("NATS连接成功", zap.String("url", cfg.URL))
	return nil
}

// GetConn 获取 NATS 连接
func GetConn() *nats.Conn {
	return conn
}

// IsAvailable 判断当前 NATS 连接是否处于可发布状态。
// 上层在决定走异步还是同步降级路径时，可以通过它快速判断消息总线是否可用。
func IsAvailable() bool {
	return conn != nil && conn.Status() == nats.CONNECTED && !conn.IsClosed()
}

// Close 关闭 NATS 连接
func Close() {
	if conn != nil {
		conn.Drain()
		conn.Close()
	}
}

// Publish 发布事件消息
// subject 格式：notification.{module}.{event}
func Publish(subject string, event *Event) error {
	if !IsAvailable() {
		return errNATSNotInitialized
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件消息失败: %w", err)
	}

	if err := conn.Publish(subject, data); err != nil {
		return fmt.Errorf("发布事件消息失败: %w", err)
	}

	logger.L.Debug("事件已发布",
		zap.String("subject", subject),
		zap.String("event_code", event.EventCode),
	)

	return nil
}

// PublishOrFallback 优先发布异步事件，失败时同步执行降级处理。
// 这用于对齐模块07文档里的“队列不可用时同步处理并记录日志”要求，让上层只保留一套调用入口。
func PublishOrFallback(subject string, event *Event, fallback func(*Event) error) error {
	if event == nil {
		return fmt.Errorf("事件不能为空")
	}

	// 优先走 NATS，总线健康时保持异步解耦。
	if err := Publish(subject, event); err == nil {
		return nil
	} else {
		logger.L.Warn("NATS发布失败，降级为同步处理",
			zap.String("subject", subject),
			zap.String("event_code", event.EventCode),
			zap.Error(err),
		)
	}

	if fallback == nil {
		return fmt.Errorf("消息队列不可用且未提供同步降级处理器")
	}

	// 当消息队列不可用时，同步执行业务降级逻辑，保证核心通知链路不因基础设施故障直接中断。
	if err := fallback(event); err != nil {
		return fmt.Errorf("同步降级处理失败: %w", err)
	}
	return nil
}

// Subscribe 订阅事件消息
// handler 为消息处理函数
func Subscribe(subject string, handler func(event *Event)) (*nats.Subscription, error) {
	if conn == nil {
		return nil, errNATSNotInitialized
	}
	sub, err := conn.Subscribe(subject, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.L.Error("反序列化事件消息失败",
				zap.String("subject", subject),
				zap.Error(err),
			)
			return
		}
		handler(&event)
	})
	if err != nil {
		return nil, fmt.Errorf("订阅事件失败: %w", err)
	}
	if err := conn.FlushTimeout(5 * time.Second); err != nil {
		_ = sub.Unsubscribe()
		return nil, fmt.Errorf("确认订阅生效失败: %w", err)
	}

	logger.L.Info("已订阅事件", zap.String("subject", subject))
	return sub, nil
}

// QueueSubscribe 队列订阅（同一队列组内只有一个消费者收到消息）
// 用于多实例部署时避免重复消费
func QueueSubscribe(subject, queue string, handler func(event *Event)) (*nats.Subscription, error) {
	if conn == nil {
		return nil, errNATSNotInitialized
	}
	sub, err := conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.L.Error("反序列化事件消息失败",
				zap.String("subject", subject),
				zap.Error(err),
			)
			return
		}
		handler(&event)
	})
	if err != nil {
		return nil, fmt.Errorf("队列订阅失败: %w", err)
	}
	if err := conn.FlushTimeout(5 * time.Second); err != nil {
		_ = sub.Unsubscribe()
		return nil, fmt.Errorf("确认队列订阅生效失败: %w", err)
	}

	logger.L.Info("已队列订阅事件",
		zap.String("subject", subject),
		zap.String("queue", queue),
	)
	return sub, nil
}

// ---- NATS Subject 常量 ----
// 对照 docs/modules/07-通知与消息 中的事件通知设计

const (
	SubjectNotifyUser        = "notification.user.*"
	SubjectNotifySchool      = "notification.school.*"
	SubjectNotifyCourse      = "notification.course.*"
	SubjectNotifyExperiment  = "notification.experiment.*"
	SubjectNotifyCompetition = "notification.competition.*"
	SubjectNotifyGrade       = "notification.grade.*"
	SubjectNotifySystem      = "notification.system.*"
)
