// nats.go
// NATS 消息队列客户端封装
// 用于跨模块事件通知总线
// Subject 设计：notification.{module}.{event}
// 所有模块通过 NATS 发布事件 → 模块07订阅消费 → 模板渲染 → 偏好过滤 → 通知推送

package mq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// 全局 NATS 连接
var conn *nats.Conn

// Event 事件消息结构
type Event struct {
	EventCode  string      `json:"event_code"`   // 事件编码（如 user.created, course.published）
	Module     string      `json:"module"`        // 来源模块
	ReceiverID int64       `json:"receiver_id"`   // 接收者用户ID（0表示广播）
	SchoolID   int64       `json:"school_id"`     // 学校ID
	Data       interface{} `json:"data"`          // 事件数据
	Timestamp  time.Time   `json:"timestamp"`     // 事件时间
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

// Subscribe 订阅事件消息
// handler 为消息处理函数
func Subscribe(subject string, handler func(event *Event)) (*nats.Subscription, error) {
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

	logger.L.Info("已订阅事件", zap.String("subject", subject))
	return sub, nil
}

// QueueSubscribe 队列订阅（同一队列组内只有一个消费者收到消息）
// 用于多实例部署时避免重复消费
func QueueSubscribe(subject, queue string, handler func(event *Event)) (*nats.Subscription, error) {
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

	logger.L.Info("已队列订阅事件",
		zap.String("subject", subject),
		zap.String("queue", queue),
	)
	return sub, nil
}

// ---- NATS Subject 常量 ----
// 对照 docs/modules/07-通知与消息 中的事件通知设计

const (
	SubjectNotifyUser         = "notification.user.*"
	SubjectNotifySchool       = "notification.school.*"
	SubjectNotifyCourse       = "notification.course.*"
	SubjectNotifyExperiment   = "notification.experiment.*"
	SubjectNotifyCompetition  = "notification.competition.*"
	SubjectNotifyGrade        = "notification.grade.*"
	SubjectNotifySystem       = "notification.system.*"
)
