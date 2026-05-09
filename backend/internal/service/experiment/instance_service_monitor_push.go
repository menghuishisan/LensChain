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

// pushCourseMonitorStatusChange 推送学生实例状态变更通知.
// 端间数据隔离（契约见 docs/modules/04-实验环境/03-API接口设计.md §3.1 学生 / §3.3 教师）：
//   - 学生通道 status_change 仅含 {status, status_text}，不下发其它学生的姓名/学号；
//   - 教师通道 student_status_change 含 {student_id, student_name, instance_id, old_status, new_status, new_status_text}。
func (s *instanceService) pushCourseMonitorStatusChange(instance *entity.ExperimentInstance, oldStatus, newStatus int) {
	if instance == nil {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	statusText := enum.GetInstanceStatusText(int16(newStatus))
	// 学生通道：精简载荷，符合 §3.1 契约。
	studentPayload := map[string]interface{}{
		"status":      newStatus,
		"status_text": statusText,
	}
	_ = manager.BroadcastToRoom(ExperimentInstanceRoom(instance.ID), buildInstanceWSMessage("status_change", studentPayload))

	// 教师通道：仅当实例归属课程时推送，按 §3.3 契约携带学生身份信息。
	if instance.CourseID != nil {
		studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
		teacherPayload := map[string]interface{}{
			"student_id":      strconv.FormatInt(instance.StudentID, 10),
			"student_name":    studentName,
			"instance_id":     strconv.FormatInt(instance.ID, 10),
			"old_status":      oldStatus,
			"new_status":      newStatus,
			"new_status_text": statusText,
		}
		_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("student_status_change", teacherPayload))
		_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("student_status_change", teacherPayload))
	}
}

// pushCourseMonitorCheckpoint 推送检查点完成通知.
// 端间数据隔离（契约见 docs/modules/04-实验环境/03-API接口设计.md §3.1 / §3.3）：
//   - 学生通道 checkpoint_result：{checkpoint_id, title, is_passed, score}；
//   - 教师通道 checkpoint_completed：{student_id, student_name, checkpoint_title, is_passed}。
func (s *instanceService) pushCourseMonitorCheckpoint(instance *entity.ExperimentInstance, checkpointID int64, checkpointTitle string, isPassed bool, score float64) {
	if instance == nil {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	// 学生通道：仅与本实例相关的检查点结果。
	studentPayload := map[string]interface{}{
		"checkpoint_id": strconv.FormatInt(checkpointID, 10),
		"title":         checkpointTitle,
		"is_passed":     isPassed,
		"score":         score,
	}
	_ = manager.BroadcastToRoom(ExperimentInstanceRoom(instance.ID), buildInstanceWSMessage("checkpoint_result", studentPayload))

	// 教师通道：按 §3.3 契约携带学生身份。
	if instance.CourseID != nil {
		studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
		teacherPayload := map[string]interface{}{
			"student_id":       strconv.FormatInt(instance.StudentID, 10),
			"student_name":     studentName,
			"checkpoint_title": checkpointTitle,
			"is_passed":        isPassed,
		}
		_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("checkpoint_completed", teacherPayload))
		_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("checkpoint_completed", teacherPayload))
	}
}

// pushCourseMonitorSubmitted 推送实验提交通知.
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
// 端间数据隔离：
//   - 学生通道：通过紧邻的 pushCourseMonitorStatusChange 已推送 status_change（status=error），
//     无需在此处再向学生通道下发包含其他学生姓名的载荷，避免端间字段污染；
//   - 教师通道 instance_error：{student_id, student_name, instance_id, error_message}（§3.3 契约）。
func (s *instanceService) pushCourseMonitorInstanceError(instance *entity.ExperimentInstance, errorMessage string) {
	if instance == nil || errorMessage == "" {
		return
	}
	if instance.CourseID == nil {
		return
	}
	manager := ws.GetManager()
	if manager == nil {
		return
	}

	studentName := s.userNameQuerier.GetUserName(svcBackgroundContext(), instance.StudentID)
	teacherPayload := map[string]interface{}{
		"student_id":    strconv.FormatInt(instance.StudentID, 10),
		"student_name":  studentName,
		"instance_id":   strconv.FormatInt(instance.ID, 10),
		"error_message": errorMessage,
	}
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, ""), buildCourseMonitorWSMessage("instance_error", teacherPayload))
	_ = manager.BroadcastToRoom(CourseMonitorRoom(*instance.CourseID, strconv.FormatInt(instance.TemplateID, 10)), buildCourseMonitorWSMessage("instance_error", teacherPayload))
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
