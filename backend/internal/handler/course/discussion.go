// discussion.go
// 模块03 — 课程与教学：讨论区、公告、评价 HTTP 处理层
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

// DiscussionHandler 讨论区与公告处理器
// 处理讨论帖、回复、点赞、公告、课程评价等接口
type DiscussionHandler struct {
	discussionService svc.DiscussionService
}

// NewDiscussionHandler 创建讨论区与公告处理器
func NewDiscussionHandler(discussionService svc.DiscussionService) *DiscussionHandler {
	return &DiscussionHandler{discussionService: discussionService}
}

// ========== 讨论区 ==========

// CreateDiscussion 发帖
// POST /api/v1/courses/:id/discussions
func (h *DiscussionHandler) CreateDiscussion(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateDiscussionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.discussionService.CreateDiscussion(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// GetDiscussion 帖子详情（含回复）
// GET /api/v1/discussions/:id
func (h *DiscussionHandler) GetDiscussion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.discussionService.GetDiscussion(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// ListDiscussions 帖子列表
// GET /api/v1/courses/:id/discussions
func (h *DiscussionHandler) ListDiscussions(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.DiscussionListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.discussionService.ListDiscussions(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// DeleteDiscussion 删除帖子
// DELETE /api/v1/discussions/:id
func (h *DiscussionHandler) DeleteDiscussion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.DeleteDiscussion(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// PinDiscussion 置顶/取消置顶
// PATCH /api/v1/discussions/:id/pin
func (h *DiscussionHandler) PinDiscussion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.PinDiscussionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.PinDiscussion(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "操作成功", nil)
}

// ========== 回复 ==========

// CreateReply 回复帖子
// POST /api/v1/discussions/:id/replies
func (h *DiscussionHandler) CreateReply(c *gin.Context) {
	discussionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateReplyReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.discussionService.CreateReply(c.Request.Context(), sc, discussionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// DeleteReply 删除回复
// DELETE /api/v1/discussion-replies/:id
func (h *DiscussionHandler) DeleteReply(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.DeleteReply(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ========== 点赞 ==========

// LikeDiscussion 点赞
// POST /api/v1/discussions/:id/like
func (h *DiscussionHandler) LikeDiscussion(c *gin.Context) {
	discussionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	liked, err := h.discussionService.LikeDiscussion(c.Request.Context(), sc, discussionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"liked": liked})
}

// UnlikeDiscussion 取消点赞
// DELETE /api/v1/discussions/:id/like
func (h *DiscussionHandler) UnlikeDiscussion(c *gin.Context) {
	discussionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	liked, err := h.discussionService.UnlikeDiscussion(c.Request.Context(), sc, discussionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"liked": liked})
}

// ========== 公告 ==========

// CreateAnnouncement 发布公告
// POST /api/v1/courses/:id/announcements
func (h *DiscussionHandler) CreateAnnouncement(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateAnnouncementReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.discussionService.CreateAnnouncement(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// UpdateAnnouncement 编辑公告
// PUT /api/v1/announcements/:id
func (h *DiscussionHandler) UpdateAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateAnnouncementReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.UpdateAnnouncement(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// PinAnnouncement 置顶/取消置顶公告
// PATCH /api/v1/announcements/:id/pin
func (h *DiscussionHandler) PinAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.PinAnnouncementReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.PinAnnouncement(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "操作成功", nil)
}

// DeleteAnnouncement 删除公告
// DELETE /api/v1/announcements/:id
func (h *DiscussionHandler) DeleteAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.DeleteAnnouncement(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListAnnouncements 公告列表
// GET /api/v1/courses/:id/announcements
func (h *DiscussionHandler) ListAnnouncements(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AnnouncementListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.discussionService.ListAnnouncements(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ========== 课程评价 ==========

// CreateEvaluation 提交课程评价
// POST /api/v1/courses/:id/evaluations
func (h *DiscussionHandler) CreateEvaluation(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateEvaluationReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	id, err := h.discussionService.CreateEvaluation(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// UpdateEvaluation 修改评价
// PUT /api/v1/course-evaluations/:id
func (h *DiscussionHandler) UpdateEvaluation(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateEvaluationReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.discussionService.UpdateEvaluation(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// ListEvaluations 评价列表
// GET /api/v1/courses/:id/evaluations
func (h *DiscussionHandler) ListEvaluations(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.EvaluationListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, summary, total, err := h.discussionService.ListEvaluations(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}
	response.Success(c, &dto.EvaluationListResp{
		Summary: summary,
		Items:   items,
		Pagination: dto.EvaluationPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPage,
		},
	})
}
