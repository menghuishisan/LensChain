// import_service.go
// 模块01 — 用户与认证：用户导入业务逻辑
// 负责导入模板下载、文件预览解析、确认执行导入、失败明细下载
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	excelpkg "github.com/lenschain/backend/internal/pkg/excel"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/repository/auth"
)

// ImportService 用户导入服务接口
type ImportService interface {
	BuildTemplate(importType string) (*bytes.Buffer, string, error)
	ParseFile(filename string, reader io.Reader) ([][]string, error)
	Preview(ctx context.Context, sc *svcctx.ServiceContext, importType string, rows [][]string) (*dto.ImportPreviewResp, error)
	Execute(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImportExecuteReq) (*dto.ImportExecuteResp, error)
	GetImportFailures(ctx context.Context, sc *svcctx.ServiceContext, importID string) ([]*dto.ImportFailureRow, error)
	BuildFailureFile(rows []*dto.ImportFailureRow) (*bytes.Buffer, string, error)
}

// importService 用户导入服务实现
type importService struct {
	db               *gorm.DB
	userRepo         authrepo.UserRepository
	profileRepo      authrepo.ProfileRepository
	roleRepo         authrepo.RoleRepository
	policyProvider   runtimePolicyProvider
	importCacheStore importCacheStore
}

// NewImportService 创建用户导入服务实例
func NewImportService(
	db *gorm.DB,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	roleRepo authrepo.RoleRepository,
) ImportService {
	return &importService{
		db:               db,
		userRepo:         userRepo,
		profileRepo:      profileRepo,
		roleRepo:         roleRepo,
		policyProvider:   &cacheRuntimePolicyProvider{},
		importCacheStore: &redisImportCacheStore{},
	}
}

var userImportHeaders = []string{"姓名", "手机号", "学号/工号", "初始密码", "学院", "专业", "班级", "入学年份", "学业层次", "年级", "邮箱", "备注"}

var userImportFailureHeaders = []string{"行号", "姓名", "手机号", "学号/工号", "学院", "专业", "班级", "入学年份", "学业层次", "年级", "邮箱", "备注", "失败原因"}

// BuildTemplate 生成用户导入模板文件
func (s *importService) BuildTemplate(importType string) (*bytes.Buffer, string, error) {
	if importType != enum.ImportTypeStudent && importType != enum.ImportTypeTeacher {
		return nil, "", errcode.ErrInvalidParams.WithMessage("type 必须为 student 或 teacher")
	}
	fileName := "学生导入模板.xlsx"
	if importType == enum.ImportTypeTeacher {
		fileName = "教师导入模板.xlsx"
	}
	buf, err := excelpkg.CreateTemplate("Sheet1", userImportHeaders, nil)
	if err != nil {
		return nil, "", errcode.ErrInternal.WithMessage("生成模板失败")
	}
	return buf, fileName, nil
}

// ParseFile 解析用户导入文件
// 文件格式解析统一复用 pkg/excel，业务校验由 Preview 负责。
func (s *importService) ParseFile(filename string, reader io.Reader) ([][]string, error) {
	return excelpkg.ImportRawRows(filename, reader)
}

// BuildFailureFile 生成用户导入失败明细文件
func (s *importService) BuildFailureFile(rows []*dto.ImportFailureRow) (*bytes.Buffer, string, error) {
	data := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		data = append(data, []interface{}{
			row.Row,
			row.Name,
			row.Phone,
			row.StudentNo,
			row.College,
			row.Major,
			row.ClassName,
			row.EnrollmentYear,
			row.EducationLevel,
			row.Grade,
			row.Email,
			row.Remark,
			row.FailReason,
		})
	}
	buf, err := excelpkg.Export(&excelpkg.ExportConfig{
		SheetName: "Sheet1",
		Headers:   userImportFailureHeaders,
	}, data)
	if err != nil {
		return nil, "", errcode.ErrInternal.WithMessage("生成失败明细文件失败")
	}
	return buf, "导入失败明细.xlsx", nil
}

// importRow 导入行数据（内部使用）
// P3-8 修复：不再缓存明文密码，缓存 bcrypt hash
type importRow struct {
	Row            int    `json:"row"`
	Name           string `json:"name"`
	Phone          string `json:"phone"`
	StudentNo      string `json:"student_no"`
	PasswordHash   string `json:"password_hash"` // P3-8 修复：存储 bcrypt hash 而非明文
	College        string `json:"college"`
	Major          string `json:"major"`
	ClassName      string `json:"class_name"`
	EnrollmentYear string `json:"enrollment_year"`
	EducationLevel string `json:"education_level"`
	Grade          string `json:"grade"`
	Email          string `json:"email"`
	Remark         string `json:"remark"`
	Status         string `json:"status"`  // valid / invalid / conflict
	Message        string `json:"message"` // 错误信息
}

// importCache 导入缓存数据（存储在 Redis 中）
type importCache struct {
	ImportID   string       `json:"import_id"`
	SchoolID   int64        `json:"school_id"`
	ImportType string       `json:"import_type"`
	Rows       []*importRow `json:"rows"`
	CreatedAt  time.Time    `json:"created_at"`
}

// 手机号正则
var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Preview 上传文件预览
// 解析 Excel 数据，校验每行数据，返回预览结果
func (s *importService) Preview(ctx context.Context, sc *svcctx.ServiceContext, importType string, rows [][]string) (*dto.ImportPreviewResp, error) {
	if len(rows) == 0 {
		return nil, errcode.ErrInvalidParams.WithMessage("文件内容为空")
	}
	if len(rows) > 5000 {
		return nil, errcode.ErrInvalidParams.WithMessage("单次导入最多支持5000行数据")
	}

	importID := fmt.Sprintf("imp_%d", snowflake.Generate())
	parsedRows := make([]*importRow, 0, len(rows))
	validCount, invalidCount, conflictCount := 0, 0, 0

	for i, row := range rows {
		parsed := s.parseRow(i+2, row) // 行号从2开始（第1行是表头）

		// 校验数据
		s.validateRow(ctx, sc.SchoolID, parsed)

		// P3-8 修复：对有效行的密码进行 bcrypt hash，不缓存明文
		if parsed.Status == enum.ImportRowValid || parsed.Status == enum.ImportRowConflict {
			if parsed.PasswordHash != "" {
				hash, err := crypto.HashPassword(parsed.PasswordHash)
				if err == nil {
					parsed.PasswordHash = hash
				} else {
					parsed.Status = enum.ImportRowInvalid
					parsed.Message = "密码加密失败"
				}
			}
		}

		switch parsed.Status {
		case enum.ImportRowValid:
			validCount++
		case enum.ImportRowInvalid:
			invalidCount++
		case enum.ImportRowConflict:
			conflictCount++
		}

		parsedRows = append(parsedRows, parsed)
	}

	// 缓存导入数据到 Redis（30分钟有效）
	cacheData := &importCache{
		ImportID:   importID,
		SchoolID:   sc.SchoolID,
		ImportType: importType,
		Rows:       parsedRows,
		CreatedAt:  time.Now(),
	}
	if s.importCacheStore != nil {
		_ = s.importCacheStore.SetImport(ctx, cacheData, 30*time.Minute)
	}

	// 构建预览列表（前100条 + 所有冲突和无效记录）
	previewList := make([]dto.ImportPreviewRow, 0)
	normalCount := 0
	for _, row := range parsedRows {
		if row.Status == enum.ImportRowValid && normalCount >= 100 {
			continue
		}
		if row.Status == enum.ImportRowValid {
			normalCount++
		}
		var msg *string
		if row.Message != "" {
			msg = &row.Message
		}
		previewList = append(previewList, dto.ImportPreviewRow{
			Row:       row.Row,
			Name:      row.Name,
			Phone:     row.Phone,
			StudentNo: row.StudentNo,
			Status:    row.Status,
			Message:   msg,
		})
	}

	return &dto.ImportPreviewResp{
		ImportID:    importID,
		Total:       len(parsedRows),
		Valid:       validCount,
		Invalid:     invalidCount,
		Conflict:    conflictCount,
		PreviewList: previewList,
	}, nil
}

// Execute 确认执行导入
func (s *importService) Execute(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImportExecuteReq) (*dto.ImportExecuteResp, error) {
	// 从 Redis 获取缓存的导入数据
	if s.importCacheStore == nil {
		return nil, errcode.ErrInternal.WithMessage("导入缓存服务未初始化")
	}

	cacheData, err := s.importCacheStore.GetImport(ctx, req.ImportID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("导入批次不存在或已过期")
	}

	// P0-3 修复：校验学校ID一致，防止跨租户操作
	if cacheData.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden.WithMessage("无权操作其他学校的导入批次")
	}

	// 构建覆盖手机号集合
	overrideSet := make(map[string]bool)
	for _, phone := range req.ConflictOverrides {
		overrideSet[phone] = true
	}

	// 获取角色
	roleCode := enum.RoleStudent
	if cacheData.ImportType == enum.ImportTypeTeacher {
		roleCode = enum.RoleTeacher
	}
	role, err := s.roleRepo.GetByCode(ctx, roleCode)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("角色不存在")
	}

	successCount, failCount, skipCount, overwriteCount := 0, 0, 0, 0
	failedRows := make([]*importRow, 0)

	// 逐行处理（事务）
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		for _, row := range cacheData.Rows {
			// 跳过无效行
			if row.Status == enum.ImportRowInvalid {
				failCount++
				failedRows = append(failedRows, row)
				continue
			}

			// 处理冲突行
			if row.Status == enum.ImportRowConflict {
				shouldOverwrite := false
				if overrideSet[row.Phone] {
					shouldOverwrite = true
				} else if req.ConflictStrategy == enum.ConflictStrategyOverwrite {
					shouldOverwrite = true
				}

				if !shouldOverwrite {
					skipCount++
					continue
				}

				// P0-5 修复：覆盖时校验目标用户属于同一学校
				if err := s.overwriteUser(ctx, tx, sc, row); err != nil {
					failCount++
					row.Message = "覆盖失败：" + err.Error()
					failedRows = append(failedRows, row)
					continue
				}
				overwriteCount++
				continue
			}

			// 创建新用户
			if err := s.createImportUser(ctx, tx, sc, row, role.ID); err != nil {
				failCount++
				row.Message = "创建失败：" + err.Error()
				failedRows = append(failedRows, row)
				continue
			}
			successCount++
		}
		return nil
	})
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("执行导入失败")
	}

	// 清除导入预览缓存
	_ = s.importCacheStore.DeleteImport(ctx, req.ImportID)

	// 持久化失败明细到 Redis（24小时有效，供下载失败明细使用）
	if len(failedRows) > 0 {
		_ = s.importCacheStore.SetImportFailures(ctx, req.ImportID, failedRows, &importFailureMeta{
			ImportID: req.ImportID,
			SchoolID: sc.SchoolID,
		}, 24*time.Hour)
	}

	// 记录操作日志（使用 pkg/audit 公共包）
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "import_users", "user", 0, map[string]interface{}{
		"import_id":   req.ImportID,
		"import_type": cacheData.ImportType,
		"total":       len(cacheData.Rows),
		"success":     successCount,
		"fail":        failCount,
		"skip":        skipCount,
		"overwrite":   overwriteCount,
	})

	return &dto.ImportExecuteResp{
		ImportID:       req.ImportID,
		SuccessCount:   successCount,
		FailCount:      failCount,
		SkipCount:      skipCount,
		OverwriteCount: overwriteCount,
	}, nil
}

// GetImportFailures 获取导入失败明细
// 从 Redis 中读取 Execute 阶段缓存的失败行数据，转换为 DTO
func (s *importService) GetImportFailures(ctx context.Context, sc *svcctx.ServiceContext, importID string) ([]*dto.ImportFailureRow, error) {
	if s.importCacheStore == nil {
		return nil, errcode.ErrInternal.WithMessage("导入缓存服务未初始化")
	}

	rows, meta, err := s.importCacheStore.GetImportFailures(ctx, importID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("失败明细不存在或已过期（24小时内有效）")
	}
	if meta == nil {
		return nil, errcode.ErrInternal.WithMessage("失败明细元数据缺失")
	}
	if meta.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden.WithMessage("无权访问其他学校的失败明细")
	}

	// 转换为 DTO（不暴露内部 importRow 结构）
	result := make([]*dto.ImportFailureRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, &dto.ImportFailureRow{
			Row:            row.Row,
			Name:           row.Name,
			Phone:          row.Phone,
			StudentNo:      row.StudentNo,
			College:        row.College,
			Major:          row.Major,
			ClassName:      row.ClassName,
			EnrollmentYear: row.EnrollmentYear,
			EducationLevel: row.EducationLevel,
			Grade:          row.Grade,
			Email:          row.Email,
			Remark:         row.Remark,
			FailReason:     row.Message,
		})
	}

	return result, nil
}
