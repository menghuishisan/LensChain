// course.go
// 模块03 — 课程与教学：课程管理、章节课时 HTTP 处理层
// 选课、学习进度、课表、共享课程、统计等拆分至 course_extra.go
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

// CourseHandler 课程管理处理器
// 处理课程 CRUD、发布、结束、归档、克隆、共享、邀请码等接口
type CourseHandler struct {
	courseService   svc.CourseService
	gradeService    svc.GradeService
	contentService  svc.ContentService
	progressService svc.ProgressService
}

// NewCourseHandler 创建课程管理处理器
func NewCourseHandler(
	courseService svc.CourseService,
	gradeService svc.GradeService,
	contentService svc.ContentService,
	progressService svc.ProgressService,
) *CourseHandler {
	return &CourseHandler{
		courseService:   courseService,
		gradeService:    gradeService,
		contentService:  contentService,
		progressService: progressService,
	}
}

// ========== 课程 CRUD ==========

// Create 创建课程
// POST /api/v1/courses
func (h *CourseHandler) Create(c *gin.Context) {
	var req dto.CreateCourseReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.courseService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", resp)
}

// GetByID 获取课程详情
// GET /api/v1/courses/:id
func (h *CourseHandler) GetByID(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.courseService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// Update 编辑课程信息
// PUT /api/v1/courses/:id
func (h *CourseHandler) Update(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCourseReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.courseService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// Delete 删除课程（仅草稿）
// DELETE /api/v1/courses/:id
func (h *CourseHandler) Delete(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.courseService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// List 课程列表（教师视角）
// GET /api/v1/courses
func (h *CourseHandler) List(c *gin.Context) {
	var req dto.CourseListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.courseService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ========== 课程状态操作 ==========

// Publish 发布课程
// POST /api/v1/courses/:id/publish
func (h *CourseHandler) Publish(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.courseService.Publish(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发布成功", nil)
}

// End 结束课程
// POST /api/v1/courses/:id/end
func (h *CourseHandler) End(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.courseService.End(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "课程已结束", nil)
}

// Archive 归档课程
// POST /api/v1/courses/:id/archive
func (h *CourseHandler) Archive(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.courseService.Archive(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "归档成功", nil)
}

// Clone 克隆课程
// POST /api/v1/courses/:id/clone
func (h *CourseHandler) Clone(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	newID, err := h.courseService.Clone(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": newID})
}

// ToggleShare 切换共享状态
// PATCH /api/v1/courses/:id/share
func (h *CourseHandler) ToggleShare(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ToggleShareReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.courseService.ToggleShare(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "操作成功", nil)
}

// RefreshInviteCode 刷新邀请码
// POST /api/v1/courses/:id/invite-code/refresh
func (h *CourseHandler) RefreshInviteCode(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	code, err := h.courseService.RefreshInviteCode(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"invite_code": code})
}

// ========== 章节 ==========

// ListChapters 获取章节列表（含课时）
// GET /api/v1/courses/:id/chapters
func (h *CourseHandler) ListChapters(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	chapters, err := h.contentService.ListChapters(c.Request.Context(), sc, courseID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, chapters)
}

// CreateChapter 创建章节
// POST /api/v1/courses/:id/chapters
func (h *CourseHandler) CreateChapter(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateChapterReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.contentService.CreateChapter(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// SortChapters 章节排序
// PUT /api/v1/courses/:id/chapters/sort
func (h *CourseHandler) SortChapters(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReorderIDsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.SortChapters(c.Request.Context(), sc, courseID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "排序成功", nil)
}

// UpdateChapter 编辑章节
// PUT /api/v1/chapters/:id
func (h *CourseHandler) UpdateChapter(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateChapterReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.UpdateChapter(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteChapter 删除章节
// DELETE /api/v1/chapters/:id
func (h *CourseHandler) DeleteChapter(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.DeleteChapter(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ========== 课时 ==========

// CreateLesson 创建课时
// POST /api/v1/chapters/:id/lessons
func (h *CourseHandler) CreateLesson(c *gin.Context) {
	chapterID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateLessonReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.contentService.CreateLesson(c.Request.Context(), sc, chapterID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// SortLessons 课时排序
// PUT /api/v1/chapters/:id/lessons/sort
func (h *CourseHandler) SortLessons(c *gin.Context) {
	chapterID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReorderIDsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.SortLessons(c.Request.Context(), sc, chapterID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "排序成功", nil)
}

// GetLesson 课时详情
// GET /api/v1/lessons/:id
func (h *CourseHandler) GetLesson(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.contentService.GetLesson(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateLesson 编辑课时
// PUT /api/v1/lessons/:id
func (h *CourseHandler) UpdateLesson(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateLessonReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.UpdateLesson(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteLesson 删除课时
// DELETE /api/v1/lessons/:id
func (h *CourseHandler) DeleteLesson(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.DeleteLesson(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// UploadAttachment 上传课时附件
// POST /api/v1/lessons/:id/attachments
func (h *CourseHandler) UploadAttachment(c *gin.Context) {
	lessonID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UploadAttachmentReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.contentService.UploadAttachment(c.Request.Context(), sc, lessonID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// DeleteAttachment 删除课时附件
// DELETE /api/v1/lesson-attachments/:id
func (h *CourseHandler) DeleteAttachment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.contentService.DeleteAttachment(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}
