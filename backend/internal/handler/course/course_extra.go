// course_extra.go
// 模块03 — 课程与教学：选课、学习进度、课表、共享课程、学生视角、统计 HTTP 处理层
// 从 course.go 拆分而来，保持单文件 ≤ 500 行
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
)

// ========== 选课 ==========

// JoinByInviteCode 通过邀请码加入课程
// POST /api/v1/courses/join
func (h *CourseHandler) JoinByInviteCode(c *gin.Context) {
	var req dto.JoinCourseReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	courseID, err := h.contentService.JoinByInviteCode(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"course_id": courseID})
}

// AddStudent 教师添加学生
// POST /api/v1/courses/:id/students
func (h *CourseHandler) AddStudent(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AddStudentReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.AddStudent(c.Request.Context(), sc, courseID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "添加成功", nil)
}

// BatchAddStudents 批量添加学生
// POST /api/v1/courses/:id/students/batch
func (h *CourseHandler) BatchAddStudents(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.BatchAddStudentsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.BatchAddStudents(c.Request.Context(), sc, courseID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "批量添加成功", nil)
}

// RemoveStudent 移除学生
// DELETE /api/v1/courses/:id/students/:student_id
func (h *CourseHandler) RemoveStudent(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	studentID, ok := validator.ParsePathID(c, "student_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.RemoveStudent(c.Request.Context(), sc, courseID, studentID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "移除成功", nil)
}

// ListStudents 课程学生列表
// GET /api/v1/courses/:id/students
func (h *CourseHandler) ListStudents(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.StudentListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.contentService.ListStudents(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ========== 学习进度 ==========

// UpdateProgress 更新学习进度
// POST /api/v1/lessons/:id/progress
func (h *CourseHandler) UpdateProgress(c *gin.Context) {
	lessonID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateProgressReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.progressService.UpdateProgress(c.Request.Context(), sc, lessonID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// GetMyProgress 我的课程学习进度
// GET /api/v1/courses/:id/my-progress
func (h *CourseHandler) GetMyProgress(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	progress, err := h.progressService.GetMyProgress(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, progress)
}

// ListStudentsProgress 全班学习进度
// GET /api/v1/courses/:id/students-progress
func (h *CourseHandler) ListStudentsProgress(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.StudentsProgressReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.progressService.ListStudentsProgress(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ========== 课表 ==========

// SetSchedule 设置课程时间表
// PUT /api/v1/courses/:id/schedules
func (h *CourseHandler) SetSchedule(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SetScheduleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.progressService.SetSchedule(c.Request.Context(), sc, courseID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", nil)
}

// GetSchedule 获取课程时间表
// GET /api/v1/courses/:id/schedules
func (h *CourseHandler) GetSchedule(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	schedules, err := h.progressService.GetSchedule(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, schedules)
}

// GetMySchedule 我的周课程表
// GET /api/v1/my-schedule
func (h *CourseHandler) GetMySchedule(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.progressService.GetMySchedule(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// ========== 共享课程库 ==========

// ListShared 共享课程库列表
// GET /api/v1/shared-courses
func (h *CourseHandler) ListShared(c *gin.Context) {
	var req dto.SharedCourseListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.courseService.ListShared(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// GetSharedDetail 共享课程详情
// GET /api/v1/shared-courses/:id
func (h *CourseHandler) GetSharedDetail(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	course, err := h.courseService.GetSharedDetail(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, course)
}

// ========== 学生视角 ==========

// ListMyCourses 我的课程列表
// GET /api/v1/my-courses
func (h *CourseHandler) ListMyCourses(c *gin.Context) {
	var req dto.MyCourseListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.courseService.ListMyCourses(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ========== 统计 ==========

// GetCourseOverview 课程整体统计
// GET /api/v1/courses/:id/statistics/overview
func (h *CourseHandler) GetCourseOverview(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	stats, err := h.progressService.GetCourseOverview(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, stats)
}

// ========== 单课程成绩 ==========

// SetGradeConfig 配置成绩权重
// PUT /api/v1/courses/:id/grade-config
func (h *CourseHandler) SetGradeConfig(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.GradeConfigReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.gradeService.SetGradeConfig(c.Request.Context(), sc, courseID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", nil)
}

// GetGradeConfig 获取成绩权重配置
// GET /api/v1/courses/:id/grade-config
func (h *CourseHandler) GetGradeConfig(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	config, err := h.gradeService.GetGradeConfig(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, config)
}

// GetGradeSummary 获取课程成绩汇总
// GET /api/v1/courses/:id/grades
func (h *CourseHandler) GetGradeSummary(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.gradeService.GetGradeSummary(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// AdjustGrade 手动调整课程最终成绩
// PATCH /api/v1/courses/:id/grades/:student_id
func (h *CourseHandler) AdjustGrade(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	studentID, ok := validator.ParsePathID(c, "student_id")
	if !ok {
		return
	}
	var req dto.AdjustGradeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.gradeService.AdjustGrade(c.Request.Context(), sc, courseID, studentID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "成绩调整成功", nil)
}

// GetMyGrades 获取学生本人在课程内的成绩
// GET /api/v1/courses/:id/my-grades
func (h *CourseHandler) GetMyGrades(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.gradeService.GetMyGrades(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// ExportGrades 导出课程成绩单
// GET /api/v1/courses/:id/grades/export
func (h *CourseHandler) ExportGrades(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	buf, fileName, err := h.gradeService.ExportGrades(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	encodedName := url.PathEscape(fileName)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// GetAssignmentStatistics 获取课程作业统计
// GET /api/v1/courses/:id/statistics/assignments
func (h *CourseHandler) GetAssignmentStatistics(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.gradeService.GetAssignmentStatistics(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, resp)
}

// ExportCourseStatistics 导出课程统计报告
// GET /api/v1/courses/:id/statistics/export
func (h *CourseHandler) ExportCourseStatistics(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	buf, fileName, err := h.gradeService.ExportCourseStatistics(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	encodedName := url.PathEscape(fileName)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}
