// grade.go
// 模块06 — 评测与成绩：HTTP 处理层。
// 该文件负责参数绑定、权限上下文传递和统一响应，不承载任何业务规则。

package grade

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	snow "github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/grade"
)

// GradeHandler 模块06统一处理器。
type GradeHandler struct {
	service svc.Service
}

// NewGradeHandler 创建模块06处理器。
func NewGradeHandler(service svc.Service) *GradeHandler {
	return &GradeHandler{service: service}
}

// CreateSemester 创建学期。
func (h *GradeHandler) CreateSemester(c *gin.Context) {
	var req dto.SemesterReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.CreateSemester(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListSemesters 获取学期列表。
func (h *GradeHandler) ListSemesters(c *gin.Context) {
	var req dto.SemesterListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListSemesters(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateSemester 更新学期。
func (h *GradeHandler) UpdateSemester(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SemesterReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateSemester(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteSemester 删除学期。
func (h *GradeHandler) DeleteSemester(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteSemester(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// SetCurrentSemester 设置当前学期。
func (h *GradeHandler) SetCurrentSemester(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.SetCurrentSemester(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", nil)
}

// GetLevelConfigs 获取等级映射配置。
func (h *GradeHandler) GetLevelConfigs(c *gin.Context) {
	respData, err := h.service.GetLevelConfigs(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateLevelConfigs 更新等级映射配置。
func (h *GradeHandler) UpdateLevelConfigs(c *gin.Context) {
	var req dto.UpdateGradeLevelConfigsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.UpdateLevelConfigs(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ResetDefaultLevelConfigs 重置默认等级映射配置。
func (h *GradeHandler) ResetDefaultLevelConfigs(c *gin.Context) {
	respData, err := h.service.ResetDefaultLevelConfigs(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SubmitReview 提交成绩审核。
func (h *GradeHandler) SubmitReview(c *gin.Context) {
	var req dto.SubmitGradeReviewReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.SubmitReview(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListReviews 获取审核列表。
func (h *GradeHandler) ListReviews(c *gin.Context) {
	var req dto.GradeReviewListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListReviews(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetReview 获取审核详情。
func (h *GradeHandler) GetReview(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetReview(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ApproveReview 审核通过成绩。
func (h *GradeHandler) ApproveReview(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReviewHandleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.ApproveReview(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "审核通过", nil)
}

// RejectReview 驳回成绩审核。
func (h *GradeHandler) RejectReview(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReviewHandleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.RejectReview(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "审核驳回", nil)
}

// UnlockReview 解锁成绩审核记录。
func (h *GradeHandler) UnlockReview(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UnlockGradeReviewReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.UnlockReview(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "解锁成功", nil)
}

// GetStudentSemesterGrades 获取指定学生学期成绩。
func (h *GradeHandler) GetStudentSemesterGrades(c *gin.Context) {
	studentID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SemesterGradesReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.GetStudentSemesterGrades(c.Request.Context(), handlerctx.BuildServiceContext(c), studentID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetStudentGPA 获取指定学生 GPA。
func (h *GradeHandler) GetStudentGPA(c *gin.Context) {
	studentID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetStudentGPA(c.Request.Context(), handlerctx.BuildServiceContext(c), studentID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetMySemesterGrades 获取当前学生学期成绩。
func (h *GradeHandler) GetMySemesterGrades(c *gin.Context) {
	var req dto.SemesterGradesReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.service.GetStudentSemesterGrades(c.Request.Context(), sc, sc.UserID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetMyGPA 获取当前学生 GPA。
func (h *GradeHandler) GetMyGPA(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.service.GetStudentGPA(c.Request.Context(), sc, sc.UserID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetMyLearningOverview 获取当前学生学习概览。
func (h *GradeHandler) GetMyLearningOverview(c *gin.Context) {
	respData, err := h.service.GetLearningOverview(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// CreateAppeal 创建成绩申诉。
func (h *GradeHandler) CreateAppeal(c *gin.Context) {
	var req dto.CreateGradeAppealReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.CreateAppeal(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListAppeals 获取申诉列表。
func (h *GradeHandler) ListAppeals(c *gin.Context) {
	var req dto.GradeAppealListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListAppeals(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetAppeal 获取申诉详情。
func (h *GradeHandler) GetAppeal(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetAppeal(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ApproveAppeal 同意成绩申诉。
func (h *GradeHandler) ApproveAppeal(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ApproveGradeAppealReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.ApproveAppeal(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "申诉已同意", nil)
}

// RejectAppeal 驳回成绩申诉。
func (h *GradeHandler) RejectAppeal(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.RejectGradeAppealReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.RejectAppeal(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "申诉已驳回", nil)
}

// ListWarnings 获取学业预警列表。
func (h *GradeHandler) ListWarnings(c *gin.Context) {
	var req dto.AcademicWarningListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListWarnings(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetWarning 获取学业预警详情。
func (h *GradeHandler) GetWarning(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetWarning(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// HandleWarning 处理学业预警。
func (h *GradeHandler) HandleWarning(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.HandleAcademicWarningReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.HandleWarning(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "处理成功", nil)
}

// GetWarningConfig 获取预警配置。
func (h *GradeHandler) GetWarningConfig(c *gin.Context) {
	respData, err := h.service.GetWarningConfig(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateWarningConfig 更新预警配置。
func (h *GradeHandler) UpdateWarningConfig(c *gin.Context) {
	var req dto.UpdateWarningConfigReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.UpdateWarningConfig(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GenerateTranscript 生成成绩单。
func (h *GradeHandler) GenerateTranscript(c *gin.Context) {
	var req dto.GenerateTranscriptReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.GenerateTranscript(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListTranscripts 获取成绩单列表。
func (h *GradeHandler) ListTranscripts(c *gin.Context) {
	var req dto.TranscriptListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListTranscripts(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// DownloadTranscript 下载成绩单。
func (h *GradeHandler) DownloadTranscript(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	url, err := h.service.GetTranscriptDownloadURL(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

// GetCourseAnalytics 获取课程成绩分析。
func (h *GradeHandler) GetCourseAnalytics(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetCourseAnalytics(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetSchoolAnalytics 获取学校成绩分析。
func (h *GradeHandler) GetSchoolAnalytics(c *gin.Context) {
	var req struct {
		SemesterID string `form:"semester_id"`
	}
	if !validator.BindQuery(c, &req) {
		return
	}
	var semesterID int64
	if req.SemesterID != "" {
		parsedID, err := snow.ParseString(req.SemesterID)
		if err != nil {
			handlerctx.HandleError(c, errcode.ErrInvalidID)
			return
		}
		semesterID = parsedID
	}
	respData, err := h.service.GetSchoolAnalytics(c.Request.Context(), handlerctx.BuildServiceContext(c), semesterID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetPlatformAnalytics 获取平台成绩总览。
func (h *GradeHandler) GetPlatformAnalytics(c *gin.Context) {
	var req struct {
		SemesterID string `form:"semester_id"`
	}
	if !validator.BindQuery(c, &req) {
		return
	}
	var semesterID int64
	if req.SemesterID != "" {
		parsedID, err := snow.ParseString(req.SemesterID)
		if err != nil {
			handlerctx.HandleError(c, errcode.ErrInvalidID)
			return
		}
		semesterID = parsedID
	}
	respData, err := h.service.GetPlatformAnalytics(c.Request.Context(), handlerctx.BuildServiceContext(c), semesterID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}
