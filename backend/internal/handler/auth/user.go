// user.go
// 模块01 — 用户与认证：用户管理 HTTP 处理器
// 负责用户 CRUD、状态变更、密码重置、解锁、批量删除、导入、个人中心
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/errcode"
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

	sc := buildServiceContext(c)
	items, total, err := h.userService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	// 规范化分页参数
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	response.Paginated(c, items, total, page, pageSize)
}

// Get 用户详情
// GET /api/v1/users/:id
func (h *UserHandler) Get(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}

	sc := buildServiceContext(c)
	detail, err := h.userService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	resp, err := h.userService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	if err := h.userService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	if err := h.userService.Delete(c.Request.Context(), sc, id); err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	if err := h.userService.UpdateStatus(c.Request.Context(), sc, id, &req); err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	if err := h.userService.ResetPassword(c.Request.Context(), sc, id, req.NewPassword); err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	if err := h.userService.UnlockUser(c.Request.Context(), sc, id); err != nil {
		handleError(c, err)
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

	// 将字符串ID转为int64
	ids := make([]int64, 0, len(req.IDs))
	for _, idStr := range req.IDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.Error(c, errcode.ErrInvalidParams.WithMessage(fmt.Sprintf("无效的ID: %s", idStr)))
			return
		}
		ids = append(ids, id)
	}

	sc := buildServiceContext(c)
	if err := h.userService.BatchDelete(c.Request.Context(), sc, ids); err != nil {
		handleError(c, err)
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

	// 创建 Excel 模板
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 设置表头
	headers := []string{"姓名", "手机号", "学号/工号", "初始密码", "学院", "专业", "班级", "入学年份", "学业层次", "年级", "邮箱", "备注"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		_ = f.SetCellValue(sheet, cell, header)
	}

	// 设置文件名（P1-5 修复：RFC 5987 编码中文文件名）
	fileName := "学生导入模板.xlsx"
	if importType == "teacher" {
		fileName = "教师导入模板.xlsx"
	}
	encodedName := url.PathEscape(fileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))

	if err := f.Write(c.Writer); err != nil {
		response.Error(c, errcode.ErrInternal.WithMessage("生成模板失败"))
		return
	}
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

	dataRows, err := parseImportRows(file.Filename, src)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams.WithMessage(err.Error()))
		return
	}

	if len(dataRows) == 0 {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("文件内容为空（仅有表头）"))
		return
	}

	sc := buildServiceContext(c)
	resp, err := h.importService.Preview(c.Request.Context(), sc, importType, dataRows)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, resp)
}

// parseImportRows 解析导入文件数据行
// 支持 xlsx / csv，两者均跳过首行表头
func parseImportRows(filename string, src io.Reader) ([][]string, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".csv":
		reader := csv.NewReader(src)
		reader.FieldsPerRecord = -1
		rows, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("CSV文件解析失败")
		}
		if len(rows) <= 1 {
			return [][]string{}, nil
		}
		return rows[1:], nil
	case ".xlsx", ".xlsm", ".xltx", ".xltm", "":
		f, err := excelize.OpenReader(src)
		if err != nil {
			return nil, fmt.Errorf("文件格式不正确，请上传 Excel 或 CSV 文件")
		}
		defer f.Close()

		sheetName := f.GetSheetName(0)
		if sheetName == "" {
			return nil, fmt.Errorf("读取文件内容失败")
		}

		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败")
		}
		if len(rows) <= 1 {
			return [][]string{}, nil
		}
		return rows[1:], nil
	default:
		return nil, fmt.Errorf("文件格式不正确，请上传 Excel 或 CSV 文件")
	}
}

// ImportExecute 确认执行导入
// POST /api/v1/user-imports/execute
func (h *UserHandler) ImportExecute(c *gin.Context) {
	var req dto.ImportExecuteReq
	if !validator.BindJSON(c, &req) {
		return
	}

	sc := buildServiceContext(c)
	resp, err := h.importService.Execute(c.Request.Context(), sc, &req)
	if err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	rows, err := h.importService.GetImportFailures(c.Request.Context(), sc, importID)
	if err != nil {
		handleError(c, err)
		return
	}

	if len(rows) == 0 {
		response.Error(c, errcode.ErrInvalidParams.WithMessage("没有失败记录"))
		return
	}

	// 生成 Excel 文件
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 设置表头
	headers := []string{"行号", "姓名", "手机号", "学号/工号", "学院", "专业", "班级", "入学年份", "学业层次", "年级", "邮箱", "备注", "失败原因"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		_ = f.SetCellValue(sheet, cell, header)
	}

	// 填充数据
	for i, row := range rows {
		rowNum := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), row.Row)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), row.Name)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), row.Phone)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), row.StudentNo)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), row.College)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), row.Major)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), row.ClassName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), row.EnrollmentYear)
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), row.EducationLevel)
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", rowNum), row.Grade)
		_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", rowNum), row.Email)
		_ = f.SetCellValue(sheet, fmt.Sprintf("L%d", rowNum), row.Remark)
		_ = f.SetCellValue(sheet, fmt.Sprintf("M%d", rowNum), row.FailReason)
	}

	// P1-5 修复：RFC 5987 编码中文文件名
	fileName := "导入失败明细.xlsx"
	encodedName := url.PathEscape(fileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedName))

	if err := f.Write(c.Writer); err != nil {
		response.Error(c, errcode.ErrInternal.WithMessage("生成失败明细文件失败"))
		return
	}
}

// ========== 个人中心接口 ==========

// GetProfile 获取个人信息
// GET /api/v1/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	sc := buildServiceContext(c)
	resp, err := h.profileService.GetProfile(c.Request.Context(), sc)
	if err != nil {
		handleError(c, err)
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

	sc := buildServiceContext(c)
	if err := h.profileService.UpdateProfile(c.Request.Context(), sc, &req); err != nil {
		handleError(c, err)
		return
	}

	response.SuccessWithMsg(c, "更新成功", nil)
}
