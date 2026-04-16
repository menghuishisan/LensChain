// realtime.go
// 模块04 — 实验环境：实时推送辅助
// 统一维护实验实例、实验分组、课程监控的 WebSocket 房间命名与消息封装

package experiment

import (
	"strconv"

	"github.com/lenschain/backend/internal/pkg/ws"
)

// buildInstanceWSMessage 构建实验实例通道的标准消息。
func buildInstanceWSMessage(messageType string, data interface{}) *ws.Message {
	return &ws.Message{
		Type:    messageType,
		Channel: "experiment-instance",
		Data:    data,
	}
}

// buildGroupWSMessage 构建实验分组通道的标准消息。
func buildGroupWSMessage(messageType string, data interface{}) *ws.Message {
	return &ws.Message{
		Type:    messageType,
		Channel: "experiment-group",
		Data:    data,
	}
}

// experimentGroupRoom 返回实验分组聊天房间名。
func experimentGroupRoom(groupID int64) string {
	return "experiment-group:" + strconv.FormatInt(groupID, 10)
}

// ExperimentGroupRoom 返回实验分组聊天房间名。
// 供 handler 层建立 WebSocket 订阅关系时复用统一命名规范。
func ExperimentGroupRoom(groupID int64) string {
	return experimentGroupRoom(groupID)
}

// CourseMonitorRoom 返回课程实验监控房间名。
// templateID 为空时订阅课程下全部模板；非空时订阅课程模板维度的细粒度监控通道。
func CourseMonitorRoom(courseID int64, templateID string) string {
	room := "course-experiment-monitor:" + strconv.FormatInt(courseID, 10)
	if templateID != "" {
		room += ":" + templateID
	}
	return room
}
