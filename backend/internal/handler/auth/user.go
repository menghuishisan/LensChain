// user.go
// 模块01 — 用户与认证：用户管理 HTTP 处理器
// 负责用户 CRUD、状态变更、密码重置、解锁、批量删除、导入、个人中心
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/auth"
)

// UserHandler 用户管理处理器
// 处理 /api/v1/users 路径下的用户管理请求
type UserHandler struct {
	userService    svc.UserService
	profileService svc.ProfileService
	importService  svc.ImportService
}

// NewUserHandler 创建用户管理处理器实例
func NewUserHandler(
	userService svc.UserService,
	profileService svc.ProfileService,
	importService svc.ImportService,
) *UserHandler {
	return &UserHandler{
		userService:    userService,
		profileService: profileService,
		importService:  importService,
	}
}

// List 用户列表
// GET /api/v1/users
func (h *UserHandler) List(c *gin.Context) {
	var req dto.UserListReq
	if !validator.BindQuery(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.userService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// Get 用户详情
// GET /api/v1/users/:id
func (h *UserHandler) Get(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.userService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, detail)
}

// Create 创建用户
// POST /api/v1/users
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.userService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Created(c, resp)
}

// Update 更新用户信息
// PUT /api/v1/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateUserReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.userService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "更新成功", nil)
}

// Delete 删除用户（软删除）
// DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.userService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "删除成功", nil)
}

// UpdateStatus 变更账号状态
// PATCH /api/v1/users/:id/status
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateStatusReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.userService.UpdateStatus(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "状态更新成功", nil)
}

// ResetPassword 重置用户密码
// POST /api/v1/users/:id/reset-password
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	var req dto.ResetPasswordReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.userService.ResetPassword(c.Request.Context(), sc, id, req.NewPassword); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "密码重置成功", nil)
}

// Unlock 解锁账号
// POST /api/v1/users/:id/unlock
func (h *UserHandler) Unlock(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.userService.UnlockUser(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "账号解锁成功", nil)
}

// BatchDelete 批量删除
// POST /api/v1/users/batch-delete
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchDeleteReq
	if !validator.BindJSON(c, &req) {
		return
	}

	ids, ok := validator.ParseIDList(c, req.IDs)
	if !ok {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.userService.BatchDelete(c.Request.Context(), sc, ids); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "批量删除成功", nil)
}

// ========== 用户导入接口 ==========

// DownloadTemplate 下载导入模板
// GET /api/v1/user-imports/template
// P1-5 修复：使用 RFC 5987 编码中文文件名
func (h *UserHandler) DownloadTemplate(c *gin.Context) {
	importType := c.Query("type")
	if importType != "student" && importType != "teacher" {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("type 必须为 student 或 teacher"))
		return
	}

	buf, fileName, err := h.importService.BuildTemplate(importType)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	encodedName := url.PathEscape(fileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportPreview 上传文件预览
// POST /api/v1/user-imports/preview
// P3-9 修复：限制上传文件大小（最大 10MB）
func (h *UserHandler) ImportPreview(c *gin.Context) {
	importType := c.PostForm("type")
	if importType != "student" && importType != "teacher" {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("type 必须为 student 或 teacher"))
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("请上传文件"))
		return
	}

	// P3-9 修复：限制文件大小（10MB）
	const maxFileSize = 10 << 20 // 10MB
	if file.Size > maxFileSize {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("文件大小不能超过10MB"))
		return
	}

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		response.Error(c, errcode.ErrInternal.WithMessage("打开文件失败"))
		return
	}
	defer src.Close()

	dataRows, err := h.importService.ParseFile(file.Filename, src)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams.WithMessage(err.Error()))
		return
	}

	if len(dataRows) == 0 {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("文件内容为空（仅有表头）"))
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.importService.Preview(c.Request.Context(), sc, importType, dataRows)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, resp)
}

// ImportExecute 确认执行导入
// POST /api/v1/user-imports/execute
func (h *UserHandler) ImportExecute(c *gin.Context) {
	var req dto.ImportExecuteReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.importService.Execute(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "导入完成", resp)
}

// ImportFailures 下载失败明细
// GET /api/v1/user-imports/:id/failures
// 从缓存中获取失败记录，生成 Excel 文件流式下载
func (h *UserHandler) ImportFailures(c *gin.Context) {
	importID := c.Param("id")
	if importID == "" {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("缺少导入批次ID"))
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	rows, err := h.importService.GetImportFailures(c.Request.Context(), sc, importID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	if len(rows) == 0 {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("没有失败记录"))
		return
	}

	buf, fileName, err := h.importService.BuildFailureFile(rows)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	encodedName := url.PathEscape(fileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ========== 个人中心接口 ==========

// GetProfile 获取个人信息
// GET /api/v1/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	resp, err := h.profileService.GetProfile(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.Success(c, resp)
}

// UpdateProfile 更新个人信息
// PUT /api/v1/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := handlerctx.BuildServiceContext(c)
	if err := h.profileService.UpdateProfile(c.Request.Context(), sc, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "更新成功", nil)
}
