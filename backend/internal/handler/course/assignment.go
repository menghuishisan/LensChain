// assignment.go
// 模块03 — 课程与教学：作业管理、题目、提交、批改 HTTP 处理层
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package course

import (
	"github.com/gin-gonic/gin"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/course"
)

// AssignmentHandler 作业管理处理器
// 处理作业 CRUD、题目管理、学生提交、教师批改等接口
type AssignmentHandler struct {
	assignmentService svc.AssignmentService
}

// NewAssignmentHandler 创建作业管理处理器
func NewAssignmentHandler(assignmentService svc.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{assignmentService: assignmentService}
}

// ========== 作业 CRUD ==========

// CreateAssignment 创建作业
// POST /api/v1/courses/:id/assignments
func (h *AssignmentHandler) CreateAssignment(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateAssignmentReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.assignmentService.Create(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// GetAssignment 作业详情（含题目）
// GET /api/v1/assignments/:id
func (h *AssignmentHandler) GetAssignment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.assignmentService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateAssignment 编辑作业
// PUT /api/v1/assignments/:id
func (h *AssignmentHandler) UpdateAssignment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateAssignmentReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.assignmentService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteAssignment 删除作业
// DELETE /api/v1/assignments/:id
func (h *AssignmentHandler) DeleteAssignment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.assignmentService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListAssignments 作业列表
// GET /api/v1/courses/:id/assignments
func (h *AssignmentHandler) ListAssignments(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AssignmentListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.assignmentService.List(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// PublishAssignment 发布作业
// POST /api/v1/assignments/:id/publish
func (h *AssignmentHandler) PublishAssignment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.assignmentService.Publish(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发布成功", nil)
}

// ========== 题目管理 ==========

// AddQuestion 添加题目
// POST /api/v1/assignments/:id/questions
func (h *AssignmentHandler) AddQuestion(c *gin.Context) {
	assignmentID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AddQuestionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.assignmentService.AddQuestion(c.Request.Context(), sc, assignmentID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// UpdateQuestion 编辑题目
// PUT /api/v1/assignment-questions/:id
func (h *AssignmentHandler) UpdateQuestion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateQuestionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.assignmentService.UpdateQuestion(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteQuestion 删除题目
// DELETE /api/v1/assignment-questions/:id
func (h *AssignmentHandler) DeleteQuestion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.assignmentService.DeleteQuestion(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ========== 提交与批改 ==========

// SubmitAssignment 学生提交作业
// POST /api/v1/assignments/:id/submit
func (h *AssignmentHandler) SubmitAssignment(c *gin.Context) {
	assignmentID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SubmitAssignmentReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.assignmentService.Submit(c.Request.Context(), sc, assignmentID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "提交成功", resp)
}

// GetSubmission 提交详情
// GET /api/v1/submissions/:id
func (h *AssignmentHandler) GetSubmission(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.assignmentService.GetSubmission(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// ListSubmissions 教师查看所有学生提交列表
// GET /api/v1/assignments/:id/submissions
func (h *AssignmentHandler) ListSubmissions(c *gin.Context) {
	assignmentID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SubmissionListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.assignmentService.ListSubmissions(c.Request.Context(), sc, assignmentID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ListMySubmissions 学生查看自己的提交记录
// GET /api/v1/assignments/:id/my-submissions
func (h *AssignmentHandler) ListMySubmissions(c *gin.Context) {
	assignmentID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	result, err := h.assignmentService.ListMySubmissions(c.Request.Context(), sc, assignmentID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, result)
}

// GradeSubmission 教师批改提交
// POST /api/v1/submissions/:id/grade
func (h *AssignmentHandler) GradeSubmission(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.GradeSubmissionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.assignmentService.GradeSubmission(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "批改成功", nil)
}
