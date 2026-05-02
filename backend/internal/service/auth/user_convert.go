// user_convert.go
// 模块01 — 用户与认证：用户实体与 DTO 转换函数
// 从 user_service.go 拆分，保持单文件不超过 500 行
// 负责 entity.User → dto.UserListItem / dto.UserDetailResp 的转换

package auth

import (
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// userToListItem 用户主表、扩展资料和角色编码聚合为列表项 DTO。
func userToListItem(user *entity.User, profile *entity.UserProfile, roleCodes []string, school *entity.School) *dto.UserListItem {
	item := &dto.UserListItem{
		ID:         strconv.FormatInt(user.ID, 10),
		Phone:      user.Phone,
		Name:       user.Name,
		StudentNo:  user.StudentNo,
		Status:     user.Status,
		StatusText: enum.GetUserStatusText(user.Status),
		Roles:      roleCodes,
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
	}

	// 最后登录时间
	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Format(time.RFC3339)
		item.LastLoginAt = &t
	}

	// 扩展信息
	if profile != nil {
		item.College = profile.College
		item.Major = profile.Major
		item.ClassName = profile.ClassName
		item.EducationLevel = profile.EducationLevel
		if profile.EducationLevel != nil {
			text := enum.GetEduLevelText(*profile.EducationLevel)
			item.EducationLevelText = &text
		}
	}

	// 学校信息
	if user.SchoolID > 0 {
		schoolID := strconv.FormatInt(user.SchoolID, 10)
		item.SchoolID = schoolID
		if school != nil {
			item.SchoolName = school.Name
		} else {
			item.SchoolName = ""
		}
	} else {
		// school_id = 0 表示平台级用户（超级管理员）
		item.SchoolID = "0"
		item.SchoolName = "平台"
	}

	return item
}

// userToDetailResp 用户主表、扩展资料和角色编码聚合为详情 DTO。
func userToDetailResp(user *entity.User, profile *entity.UserProfile, roleCodes []string) *dto.UserDetailResp {
	resp := &dto.UserDetailResp{
		ID:            strconv.FormatInt(user.ID, 10),
		Phone:         user.Phone,
		Name:          user.Name,
		StudentNo:     user.StudentNo,
		Status:        user.Status,
		StatusText:    enum.GetUserStatusText(user.Status),
		IsFirstLogin:  user.IsFirstLogin,
		IsSchoolAdmin: user.IsSchoolAdmin,
		SchoolID:      strconv.FormatInt(user.SchoolID, 10),
		Roles:         roleCodes,
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	}

	// 最后登录时间
	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Format(time.RFC3339)
		resp.LastLoginAt = &t
	}

	// 扩展信息
	if profile != nil {
		resp.AvatarURL = profile.AvatarURL
		resp.Nickname = profile.Nickname
		resp.Email = profile.Email
		resp.College = profile.College
		resp.Major = profile.Major
		resp.ClassName = profile.ClassName
		resp.EnrollmentYear = profile.EnrollmentYear
		resp.EducationLevel = profile.EducationLevel
		resp.Grade = profile.Grade
		resp.Remark = profile.Remark
		if profile.EducationLevel != nil {
			text := enum.GetEduLevelText(*profile.EducationLevel)
			resp.EducationLevelText = &text
		}
	}

	return resp
}
