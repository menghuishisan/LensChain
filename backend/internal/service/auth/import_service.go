// import_service.go
// 模块01 — 用户与认证：用户导入业务逻辑
// 负责导入模板下载、文件预览解析、确认执行导入、失败明细下载
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
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
	sampleRows := studentImportSampleRows()
	if importType == enum.ImportTypeTeacher {
		fileName = "教师导入模板.xlsx"
		sampleRows = teacherImportSampleRows()
	}
	buf, err := excelpkg.CreateTemplate("Sheet1", userImportHeaders, sampleRows)
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
	if !enum.IsValidImportType(importType) {
		return nil, errcode.ErrInvalidParams.WithMessage("导入类型不合法")
	}
	if len(rows) > 5000 {
		return nil, errcode.ErrInvalidParams.WithMessage("单次导入不超过5000条")
	}

	importID := fmt.Sprintf("imp_%d", snowflake.Generate())
	parsedRows := make([]*importRow, 0, len(rows))
	validCount, invalidCount, conflictCount := 0, 0, 0

	for i, row := range rows {
		parsed := s.parseRow(i+2, row) // 行号从2开始（第1行是表头）
		if parsed == nil {
			continue
		}

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

	acquired, err := s.importCacheStore.AcquireExecution(ctx, req.ImportID, 24*time.Hour)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("导入执行状态初始化失败")
	}
	if !acquired {
		return nil, errcode.ErrInvalidParams.WithMessage("该导入批次已执行或正在处理中")
	}

	cacheData, err := s.importCacheStore.GetImport(ctx, req.ImportID)
	if err != nil {
		_ = s.importCacheStore.ReleaseExecution(ctx, req.ImportID)
		return nil, errcode.ErrInvalidParams.WithMessage("导入批次不存在或已过期")
	}

	// P0-3 修复：校验学校ID一致，防止跨租户操作
	if cacheData.SchoolID != sc.SchoolID {
		_ = s.importCacheStore.ReleaseExecution(ctx, req.ImportID)
		return nil, errcode.ErrForbidden.WithMessage("无权操作其他学校的导入批次")
	}
	if !enum.IsValidImportType(cacheData.ImportType) {
		_ = s.importCacheStore.ReleaseExecution(ctx, req.ImportID)
		return nil, errcode.ErrInvalidParams.WithMessage("导入批次类型不合法")
	}
	if !enum.IsValidConflictStrategy(req.ConflictStrategy) {
		_ = s.importCacheStore.ReleaseExecution(ctx, req.ImportID)
		return nil, errcode.ErrInvalidParams.WithMessage("冲突处理策略不合法")
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
		_ = s.importCacheStore.ReleaseExecution(ctx, req.ImportID)
		return nil, errcode.ErrInternal.WithMessage("角色不存在")
	}

	successCount, failCount, skipCount, overwriteCount := 0, 0, 0, 0
	failedRows := make([]*importRow, 0)
	overwrittenUserIDs := make([]int64, 0)

	// 逐行处理（事务）
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txUserRepo := authrepo.NewUserRepository(tx)
		txProfileRepo := authrepo.NewProfileRepository(tx)
		txRoleRepo := authrepo.NewRoleRepository(tx)
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
				overwrittenUserID, err := s.overwriteUser(ctx, txUserRepo, txProfileRepo, sc, row)
				if err != nil {
					failCount++
					row.Message = "覆盖失败：" + err.Error()
					failedRows = append(failedRows, row)
					continue
				}
				overwrittenUserIDs = append(overwrittenUserIDs, overwrittenUserID)
				overwriteCount++
				continue
			}

			// 创建新用户
			if err := s.createImportUser(ctx, txUserRepo, txProfileRepo, txRoleRepo, sc, row, role.ID); err != nil {
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
		_ = s.importCacheStore.ReleaseExecution(ctx, req.ImportID)
		return nil, errcode.ErrInternal.WithMessage("执行导入失败")
	}
	for _, userID := range overwrittenUserIDs {
		invalidateUserSession(ctx, s.userRepo, userID, resolveAccessTokenTTLByProvider(ctx, s.policyProvider))
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

// parseRow 解析 Excel 行数据，并统一裁剪字段空白。
func (s *importService) parseRow(rowNum int, cells []string) *importRow {
	row := &importRow{
		Row:    rowNum,
		Status: enum.ImportRowValid,
	}

	getCell := func(idx int) string {
		if idx >= len(cells) {
			return ""
		}
		return strings.TrimSpace(cells[idx])
	}

	row.Name = getCell(0)
	row.Phone = getCell(1)
	row.StudentNo = getCell(2)
	row.PasswordHash = getCell(3)
	row.College = getCell(4)
	row.Major = getCell(5)
	row.ClassName = getCell(6)
	row.EnrollmentYear = getCell(7)
	row.EducationLevel = getCell(8)
	row.Grade = getCell(9)
	row.Email = getCell(10)
	row.Remark = getCell(11)

	if row.Name == "" &&
		row.Phone == "" &&
		row.StudentNo == "" &&
		row.PasswordHash == "" &&
		row.College == "" &&
		row.Major == "" &&
		row.ClassName == "" &&
		row.EnrollmentYear == "" &&
		row.EducationLevel == "" &&
		row.Grade == "" &&
		row.Email == "" &&
		row.Remark == "" {
		return nil
	}

	return row
}

// validateRow 校验导入行数据。
func (s *importService) validateRow(ctx context.Context, schoolID int64, row *importRow) {
	var errors []string
	policy := defaultRuntimeSecurityPolicy()
	if s.policyProvider != nil {
		if runtimePolicy, err := s.policyProvider.GetRuntimeSecurityPolicy(ctx); err == nil && runtimePolicy != nil {
			policy = runtimePolicy
		}
	}

	if row.Name == "" {
		errors = append(errors, "姓名不能为空")
	}
	if row.Phone == "" {
		errors = append(errors, "手机号不能为空")
	} else if !phoneRegex.MatchString(row.Phone) {
		errors = append(errors, "手机号格式不正确")
	}
	if row.StudentNo == "" {
		errors = append(errors, "学号/工号不能为空")
	}
	if row.PasswordHash == "" {
		errors = append(errors, "初始密码不能为空")
	} else if err := validatePasswordWithPolicy(row.PasswordHash, policy); err != nil {
		errors = append(errors, "初始密码不满足当前密码复杂度要求")
	}
	if row.Email != "" {
		if _, err := mail.ParseAddress(row.Email); err != nil {
			errors = append(errors, "邮箱格式不正确")
		}
	}
	if len(row.Name) > 50 {
		errors = append(errors, "姓名长度不能超过50")
	}
	if len(row.StudentNo) > 50 {
		errors = append(errors, "学号/工号长度不能超过50")
	}
	if row.EnrollmentYear != "" {
		year, err := strconv.Atoi(row.EnrollmentYear)
		currentYear := time.Now().Year()
		if err != nil || year < 2000 || year > currentYear {
			errors = append(errors, "入学年份不在有效范围内")
		}
	}
	if row.EducationLevel != "" && enum.ParseEduLevel(row.EducationLevel) == 0 {
		errors = append(errors, "学业层次仅支持专科/本科/硕士/博士")
	}

	if len(errors) > 0 {
		row.Status = enum.ImportRowInvalid
		row.Message = joinErrors(errors)
		return
	}

	existingByPhone, err := s.userRepo.GetByPhone(ctx, row.Phone)
	if err == nil && existingByPhone != nil {
		row.Status = enum.ImportRowConflict
		existName := existingByPhone.Name
		existNo := ""
		if existingByPhone.StudentNo != nil {
			existNo = *existingByPhone.StudentNo
		}
		row.Message = fmt.Sprintf("手机号已存在，当前用户：%s（学号%s）", existName, existNo)
		return
	}

	existingByNo, err := s.userRepo.GetBySchoolAndStudentNo(ctx, schoolID, row.StudentNo)
	if err == nil && existingByNo != nil {
		row.Status = enum.ImportRowConflict
		row.Message = fmt.Sprintf("学号已存在，当前用户：%s（手机号%s）", existingByNo.Name, existingByNo.Phone)
	}
}

// createImportUser 创建导入用户。
func (s *importService) createImportUser(
	ctx context.Context,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	roleRepo authrepo.RoleRepository,
	sc *svcctx.ServiceContext,
	row *importRow,
	roleID int64,
) error {
	userID := snowflake.Generate()
	studentNo := row.StudentNo

	user := &entity.User{
		ID:           userID,
		Phone:        row.Phone,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		SchoolID:     sc.SchoolID,
		StudentNo:    &studentNo,
		Status:       enum.UserStatusActive,
		IsFirstLogin: true,
		CreatedBy:    &sc.UserID,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		return err
	}

	profile := &entity.UserProfile{
		UserID: userID,
	}
	if row.College != "" {
		profile.College = &row.College
	}
	if row.Major != "" {
		profile.Major = &row.Major
	}
	if row.ClassName != "" {
		profile.ClassName = &row.ClassName
	}
	if row.EnrollmentYear != "" {
		if year, err := strconv.Atoi(row.EnrollmentYear); err == nil {
			enrollmentYear := int16(year)
			profile.EnrollmentYear = &enrollmentYear
		}
	}
	if row.EducationLevel != "" {
		level := enum.ParseEduLevel(row.EducationLevel)
		if level > 0 {
			profile.EducationLevel = &level
		}
	}
	if row.Grade != "" {
		if grade, err := strconv.Atoi(row.Grade); err == nil {
			levelGrade := int16(grade)
			profile.Grade = &levelGrade
		}
	}
	if row.Email != "" {
		profile.Email = &row.Email
	}
	if row.Remark != "" {
		profile.Remark = &row.Remark
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		return err
	}

	return roleRepo.AssignRole(ctx, userID, roleID)
}

// overwriteUser 覆盖已有用户。
func (s *importService) overwriteUser(
	ctx context.Context,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	sc *svcctx.ServiceContext,
	row *importRow,
) (int64, error) {
	existing, err := userRepo.GetByPhone(ctx, row.Phone)
	if err != nil {
		return 0, err
	}
	if existing.SchoolID != sc.SchoolID {
		return 0, fmt.Errorf("目标用户不属于当前学校，禁止跨租户覆盖")
	}

	studentNo := row.StudentNo
	if err := userRepo.UpdateFields(ctx, existing.ID, map[string]interface{}{
		"name":           row.Name,
		"password_hash":  row.PasswordHash,
		"student_no":     &studentNo,
		"is_first_login": true,
	}); err != nil {
		return 0, err
	}

	profileFields := make(map[string]interface{})
	if row.College != "" {
		profileFields["college"] = row.College
	}
	if row.Major != "" {
		profileFields["major"] = row.Major
	}
	if row.ClassName != "" {
		profileFields["class_name"] = row.ClassName
	}
	if row.Email != "" {
		profileFields["email"] = row.Email
	}
	if row.Remark != "" {
		profileFields["remark"] = row.Remark
	}
	if row.EnrollmentYear != "" {
		if year, err := strconv.Atoi(row.EnrollmentYear); err == nil {
			profileFields["enrollment_year"] = int16(year)
		}
	}
	if row.EducationLevel != "" {
		if level := enum.ParseEduLevel(row.EducationLevel); level > 0 {
			profileFields["education_level"] = level
		}
	}
	if row.Grade != "" {
		if grade, err := strconv.Atoi(row.Grade); err == nil {
			profileFields["grade"] = int16(grade)
		}
	}
	if len(profileFields) == 0 {
		return existing.ID, nil
	}

	if _, err := profileRepo.GetByUserID(ctx, existing.ID); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
		profile := &entity.UserProfile{UserID: existing.ID}
		applyImportProfileFields(profile, profileFields)
		if err := profileRepo.Create(ctx, profile); err != nil {
			return 0, err
		}
		return existing.ID, nil
	}
	if err := profileRepo.UpdateFields(ctx, existing.ID, profileFields); err != nil {
		return 0, err
	}
	return existing.ID, nil
}

// joinErrors 拼接多条校验错误。
func joinErrors(errors []string) string {
	return strings.Join(errors, "；")
}

// applyImportProfileFields 将导入字段映射到扩展资料实体。
func applyImportProfileFields(profile *entity.UserProfile, fields map[string]interface{}) {
	if value, ok := fields["college"].(string); ok {
		profile.College = &value
	}
	if value, ok := fields["major"].(string); ok {
		profile.Major = &value
	}
	if value, ok := fields["class_name"].(string); ok {
		profile.ClassName = &value
	}
	if value, ok := fields["email"].(string); ok {
		profile.Email = &value
	}
	if value, ok := fields["remark"].(string); ok {
		profile.Remark = &value
	}
	if value, ok := fields["enrollment_year"].(int16); ok {
		profile.EnrollmentYear = &value
	}
	if value, ok := fields["education_level"].(int16); ok {
		profile.EducationLevel = &value
	}
	if value, ok := fields["grade"].(int16); ok {
		profile.Grade = &value
	}
}

// studentImportSampleRows 返回学生导入模板的说明行和示例行。
func studentImportSampleRows() [][]interface{} {
	return [][]interface{}{
		{"必填，1-50字", "必填，11位手机号", "必填，校内唯一", "必填，满足密码复杂度", "选填", "选填", "选填", "选填，如2024", "选填：专科/本科/硕士/博士", "选填", "选填，邮箱格式", "选填"},
		{"张三", "13800138000", "2024001", "InitPass123", "计算机学院", "软件工程", "软工2401", "2024", "本科", "1", "zhangsan@example.com", "示例学生"},
	}
}

// teacherImportSampleRows 返回教师导入模板的说明行和示例行。
func teacherImportSampleRows() [][]interface{} {
	return [][]interface{}{
		{"必填，1-50字", "必填，11位手机号", "必填，工号", "必填，满足密码复杂度", "选填", "通常留空", "通常留空", "通常留空", "通常留空", "通常留空", "选填，邮箱格式", "选填"},
		{"李老师", "13900139000", "T2024001", "InitPass123", "计算机学院", "", "", "", "", "", "li@example.com", "示例教师"},
	}
}
