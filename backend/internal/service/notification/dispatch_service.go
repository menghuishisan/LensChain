// dispatch_service.go
// 模块07 — 通知与消息：内部事件分发与模板渲染业务。
// 该文件负责把跨模块事件转为最终站内信记录，确保模板、偏好、去重和写库流程集中收敛。

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/mq"
)

const (
	// notificationInternalEventSubject 是模块07统一消费跨模块通知事件的 NATS subject。
	notificationInternalEventSubject = "notification.internal.event"
	notificationEventQueue           = "notification-service"
)

// EventBus 定义模块07依赖的统一异步事件总线能力。
type EventBus interface {
	PublishOrFallback(subject string, event *mq.Event, fallback func(*mq.Event) error) error
}

// DispatchEvent 分发通知事件。
// 上层模块只依赖该入口；模块07内部优先通过 NATS 异步处理，队列不可用时同步降级。
func (s *service) DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error {
	if req == nil {
		return nil
	}
	if s.eventBus == nil {
		return s.ProcessEvent(ctx, req)
	}
	event := buildNotificationMQEvent(req)
	return s.eventBus.PublishOrFallback(notificationInternalEventSubject, event, func(event *mq.Event) error {
		fallbackReq, err := parseNotificationMQEvent(event)
		if err != nil {
			return err
		}
		return s.ProcessEvent(ctx, fallbackReq)
	})
}

// RegisterInternalEventConsumer 注册模块07的内部通知事件消费者。
func RegisterInternalEventConsumer(processor Service) error {
	if processor == nil {
		return nil
	}
	_, err := mq.QueueSubscribe(notificationInternalEventSubject, notificationEventQueue, func(event *mq.Event) {
		req, parseErr := parseNotificationMQEvent(event)
		if parseErr != nil {
			logger.L.Error("解析内部通知事件失败", zap.Error(parseErr))
			return
		}
		if err := processor.ProcessEvent(context.Background(), req); err != nil {
			logger.L.Error("处理内部通知事件失败",
				zap.String("event_type", req.EventType),
				zap.Error(err),
			)
		}
	})
	return err
}

// EnableAsyncDispatch 在消费者注册成功后启用异步事件发布。
// 若消费者注册失败，调用方不应启用该开关，DispatchEvent 会继续同步 ProcessEvent，避免事件发出后无人消费。
func EnableAsyncDispatch(dispatcher EventDispatcher, bus EventBus) {
	svc, ok := dispatcher.(*service)
	if !ok || svc == nil {
		return
	}
	if bus == nil {
		bus = mqEventBus{}
	}
	svc.eventBus = bus
}

// mqEventBus 适配 internal/pkg/mq，避免业务模块直接感知 NATS 细节。
type mqEventBus struct{}

// PublishOrFallback 发布异步事件，并在队列不可用时执行同步降级。
func (mqEventBus) PublishOrFallback(subject string, event *mq.Event, fallback func(*mq.Event) error) error {
	return mq.PublishOrFallback(subject, event, fallback)
}

// buildNotificationMQEvent 将模块07内部事件请求转换为统一 MQ 事件。
func buildNotificationMQEvent(req *dto.InternalSendNotificationEventReq) *mq.Event {
	if req == nil {
		return nil
	}
	receiverID := int64(0)
	receiverIDs, err := parseSnowflakeIDs(req.ReceiverIDs)
	if err == nil && len(receiverIDs) == 1 {
		receiverID = receiverIDs[0]
	}
	return &mq.Event{
		EventCode:  req.EventType,
		Module:     req.SourceModule,
		ReceiverID: receiverID,
		Data:       req,
		Timestamp:  time.Now().UTC(),
	}
}

// parseNotificationMQEvent 将统一 MQ 事件还原为模块07内部事件请求。
func parseNotificationMQEvent(event *mq.Event) (*dto.InternalSendNotificationEventReq, error) {
	if event == nil {
		return nil, fmt.Errorf("通知事件不能为空")
	}
	switch data := event.Data.(type) {
	case *dto.InternalSendNotificationEventReq:
		return data, nil
	case dto.InternalSendNotificationEventReq:
		return &data, nil
	default:
		raw, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("序列化通知事件载荷失败: %w", err)
		}
		var req dto.InternalSendNotificationEventReq
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, fmt.Errorf("解析通知事件载荷失败: %w", err)
		}
		if req.EventType == "" {
			req.EventType = event.EventCode
		}
		if req.SourceModule == "" {
			req.SourceModule = event.Module
		}
		return &req, nil
	}
}

// ProcessEvent 处理内部通知事件并生成站内信。
func (s *service) ProcessEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error {
	if req == nil {
		return nil
	}
	receiverIDs, err := parseSnowflakeIDs(req.ReceiverIDs)
	if err != nil {
		return err
	}

	template, err := s.templateRepo.GetByEventType(ctx, req.EventType)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if template != nil && !template.IsEnabled {
		return nil
	}

	category, title, content := s.resolveEventContent(req, template)

	notifications := make([]*entity.Notification, 0, len(receiverIDs))
	sourceID := buildSourceIDPointer(req.SourceID)
	now := time.Now()
	for _, receiverID := range receiverIDs {
		user, userErr := s.sourceRepo.GetUser(ctx, receiverID)
		if userErr != nil || user == nil {
			continue
		}
		if !s.canReceiveCategory(ctx, receiverID, category) {
			continue
		}
		exists, existsErr := s.notificationRepo.ExistsByEvent(ctx, receiverID, req.EventType, sourceID)
		if existsErr == nil && exists {
			continue
		}
		schoolID := user.SchoolID
		sourceType := req.SourceType
		notifications = append(notifications, &entity.Notification{
			ReceiverID:   receiverID,
			SchoolID:     &schoolID,
			Category:     category,
			EventType:    req.EventType,
			Title:        title,
			Content:      content,
			SourceModule: req.SourceModule,
			SourceType:   &sourceType,
			SourceID:     sourceID,
			CreatedAt:    now,
		})
	}
	if err := s.notificationRepo.BatchCreate(ctx, notifications); err != nil {
		return err
	}
	s.afterNotificationsCreated(ctx, notifications)
	return nil
}

// resolveEventContent 解析事件的最终分类、标题和内容。
func (s *service) resolveEventContent(req *dto.InternalSendNotificationEventReq, template *entity.NotificationTemplate) (int16, string, string) {
	category := inferCategoryFromEventType(req.EventType)
	title := req.EventType
	content := fmt.Sprintf("%v", req.Params)
	if template != nil {
		category = template.Category
		title = renderTextTemplate(template.TitleTemplate, req.Params)
		content = renderTextTemplate(template.ContentTemplate, req.Params)
	}

	// 成绩审核通过同时面向教师与学生，教师侧文案与学生侧模板文案不同。
	if req.EventType == "grade.review_approved" {
		if audience, _ := req.Params["audience"].(string); audience == "teacher" {
			courseName, _ := req.Params["course_name"].(string)
			title = "成绩审核已通过"
			content = fmt.Sprintf("课程《%s》成绩审核已通过。", courseName)
		}
	}
	return category, title, content
}
