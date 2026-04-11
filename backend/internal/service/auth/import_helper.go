// import_helper.go
// 模块01 — 用户与认证：用户导入内部辅助方法
// 包含行解析、数据校验、用户创建/覆盖等导入流程的底层操作
// 从 import_service.go 拆分，保持单文件不超过 500 行

package auth

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// parseRow 解析 Excel 行数据
func (s *importService) parseRow(rowNum int, cells []string) *importRow {
	row := &importRow{
		Row:    rowNum,
		Status: enum.ImportRowValid,
	}

	getCell := func(idx int) string {
		if idx < len(cells) {
			return cells[idx]
		}
		return ""
	}

	row.Name = getCell(0)
	row.Phone = getCell(1)
	row.StudentNo = getCell(2)
	row.PasswordHash = getCell(3) // 暂存明文，validateRow 后在 Preview 中 hash
	row.College = getCell(4)
	row.Major = getCell(5)
	row.ClassName = getCell(6)
	row.EnrollmentYear = getCell(7)
	row.EducationLevel = getCell(8)
	row.Grade = getCell(9)
	row.Email = getCell(10)
	row.Remark = getCell(11)

	return row
}

// validateRow 校验行数据
func (s *importService) validateRow(ctx context.Context, schoolID int64, row *importRow) {
	var errors []string

	// 必填校验
	if row.Name == "" {
		errors = append(errors, "姓名不能为空")
	}
	if row.Phone == "" {
		errors = append(errors, "手机号不能为空")
	} else if !phoneRegex.MatchString(row.Phone) {
		errors = append(errors, "手机号格式不正确")
	}
	if row.StudentNo == "" {
		errors = append(errors, "学号不能为空")
	}
	if row.PasswordHash == "" {
		errors = append(errors, "初始密码不能为空")
	}
	if row.Email != "" {
		if _, err := mail.ParseAddress(row.Email); err != nil {
			errors = append(errors, "邮箱格式不正确")
		}
	}
	if len(strings.TrimSpace(row.Name)) > 50 {
		errors = append(errors, "姓名长度不能超过50")
	}
	if len(strings.TrimSpace(row.StudentNo)) > 50 {
		errors = append(errors, "学号长度不能超过50")
	}

	if len(errors) > 0 {
		row.Status = enum.ImportRowInvalid
		row.Message = joinErrors(errors)
		return
	}

	// 冲突检测：手机号
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

	// 冲突检测：学号（校内唯一）
	existingByNo, err := s.userRepo.GetBySchoolAndStudentNo(ctx, schoolID, row.StudentNo)
	if err == nil && existingByNo != nil {
		row.Status = enum.ImportRowConflict
		row.Message = fmt.Sprintf("学号已存在，当前用户：%s（手机号%s）", existingByNo.Name, existingByNo.Phone)
		return
	}
}

// createImportUser 创建导入用户
// P3-8 修复：使用已 hash 的密码，不再在此处 hash
func (s *importService) createImportUser(_ context.Context, tx *gorm.DB, sc *svcctx.ServiceContext, row *importRow, roleID int64) error {
	userID := snowflake.Generate()
	studentNo := row.StudentNo

	user := &entity.User{
		ID:           userID,
		Phone:        row.Phone,
		PasswordHash: row.PasswordHash, // 已在 Preview 阶段 hash
		Name:         row.Name,
		SchoolID:     sc.SchoolID,
		StudentNo:    &studentNo,
		Status:       enum.UserStatusActive,
		IsFirstLogin: true,
		CreatedBy:    &sc.UserID,
	}
	if err := tx.Create(user).Error; err != nil {
		return err
	}

	// 创建扩展信息
	profile := &entity.UserProfile{
		ID:     snowflake.Generate(),
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
			profile.EnrollmentYear = &year
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
			profile.Grade = &grade
		}
	}
	if row.Email != "" {
		profile.Email = &row.Email
	}
	if row.Remark != "" {
		profile.Remark = &row.Remark
	}
	if err := tx.Create(profile).Error; err != nil {
		return err
	}

	// 分配角色
	userRole := &entity.UserRole{
		ID:     snowflake.Generate(),
		UserID: userID,
		RoleID: roleID,
	}
	return tx.Create(userRole).Error
}

// overwriteUser 覆盖已有用户
// P0-5 修复：校验目标用户属于同一学校，防止跨租户覆盖
// 修复 ctx/tx 不一致：读写均通过事务 tx 执行，保证隔离性
func (s *importService) overwriteUser(_ context.Context, tx *gorm.DB, sc *svcctx.ServiceContext, row *importRow) error {
	// 通过事务 tx 读取，确保与后续写入在同一事务中（避免脏读/丢失更新）
	var existing entity.User
	if err := tx.Where("phone = ? AND deleted_at IS NULL", row.Phone).First(&existing).Error; err != nil {
		return err
	}

	// P0-5 修复：跨租户校验，只能覆盖本校用户
	if existing.SchoolID != sc.SchoolID {
		return fmt.Errorf("目标用户不属于当前学校，禁止跨租户覆盖")
	}

	// 更新用户主表（使用已 hash 的密码）
	studentNo := row.StudentNo
	updates := map[string]interface{}{
		"name":           row.Name,
		"password_hash":  row.PasswordHash, // 已在 Preview 阶段 hash
		"student_no":     &studentNo,
		"is_first_login": true,
	}
	if err := tx.Model(&entity.User{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
		return err
	}

	// 更新扩展信息
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
	if len(profileFields) > 0 {
		tx.Model(&entity.UserProfile{}).Where("user_id = ?", existing.ID).Updates(profileFields)
	}

	return nil
}

// joinErrors 拼接错误信息
func joinErrors(errors []string) string {
	result := ""
	for i, e := range errors {
		if i > 0 {
			result += "；"
		}
		result += e
	}
	return result
}
