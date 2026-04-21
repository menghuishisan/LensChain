// instance_service_monitor_push.go
// 模块04 — 实验环境：教师监控实时推送辅助
// 统一在实例生命周期和检查点链路中推送教师监控面板所需的实时消息

package experiment

import (
	"context"
	"strconv"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/ws"
)

// pushCourseMonitorStatusChange 推送学生实例状态变更通知。
func (s *instanceService) pushCourseMonitorStatusChange(instance *entity.ExperimentInstance, oldStatus, newStatus int) {
	if instance == nil || instance.CourseID == nil {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
	payload := map[string]interface{}{
		"student_id":      strconv.FormatInt(instance.StudentID, 10),
		"student_name":    studentName,
		"instance_id":     strconv.FormatInt(instance.ID, 10),
		"old_status":      oldStatus,
		"new_status":      newStatus,
		"new_status_text": enum.GetInstanceStatusText(int16(newStatus)),
	}
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("student_status_change", payload))
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("student_status_change", payload))
}

// pushCourseMonitorCheckpoint 推送检查点完成通知。
func (s *instanceService) pushCourseMonitorCheckpoint(instance *entity.ExperimentInstance, checkpointTitle string, isPassed bool) {
	if instance == nil || instance.CourseID == nil {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
	payload := map[string]interface{}{
		"student_id":       strconv.FormatInt(instance.StudentID, 10),
		"student_name":     studentName,
		"checkpoint_title": checkpointTitle,
		"is_passed":        isPassed,
	}
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("checkpoint_completed", payload))
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("checkpoint_completed", payload))
}

// pushCourseMonitorSubmitted 推送实验提交通知。
func (s *instanceService) pushCourseMonitorSubmitted(instance *entity.ExperimentInstance, autoScore float64, hasManualItems bool) {
	if instance == nil || instance.CourseID == nil {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
	payload := map[string]interface{}{
		"student_id":       strconv.FormatInt(instance.StudentID, 10),
		"student_name":     studentName,
		"auto_score":       autoScore,
		"has_manual_items": hasManualItems,
	}
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("experiment_submitted", payload))
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("experiment_submitted", payload))
}

// pushCourseMonitorInstanceError 推送实例异常告警。
func (s *instanceService) pushCourseMonitorInstanceError(instance *entity.ExperimentInstance, errorMessage string) {
	if instance == nil || instance.CourseID == nil || errorMessage == "" {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
	payload := map[string]interface{}{
		"student_id":    strconv.FormatInt(instance.StudentID, 10),
		"student_name":  studentName,
		"instance_id":   strconv.FormatInt(instance.ID, 10),
		"error_message": errorMessage,
	}
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("instance_error", payload))
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("instance_error", payload))
}

// buildCourseMonitorWSMessage 构建课程实验监控通道消息。
func buildCourseMonitorWSMessage(messageType string, data interface{}) *ws.Message {
	return &ws.Message{
		Type:    messageType,
		Channel: "course-experiment-monitor",
		Data:    data,
	}
}

// svcBackgroundContext 返回实时推送使用的后台上下文。
// 推送失败不阻断主流程，因此不复用请求上下文的取消信号。
func svcBackgroundContext() context.Context {
	return context.Background()
}
