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

// userToListItem 用户实体转列表项 DTO
func userToListItem(user *entity.User) *dto.UserListItem {
	item := &dto.UserListItem{
		ID:         strconv.FormatInt(user.ID, 10),
		Phone:      user.Phone,
		Name:       user.Name,
		StudentNo:  user.StudentNo,
		Status:     user.Status,
		StatusText: enum.GetUserStatusText(user.Status),
		Roles:      make([]string, 0),
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
	}

	// 角色
	for _, ur := range user.Roles {
		if ur.Role != nil {
			item.Roles = append(item.Roles, ur.Role.Code)
		}
	}

	// 最后登录时间
	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Format(time.RFC3339)
		item.LastLoginAt = &t
	}

	// 扩展信息
	if user.Profile != nil {
		item.College = user.Profile.College
		item.Major = user.Profile.Major
		item.ClassName = user.Profile.ClassName
		item.EducationLevel = user.Profile.EducationLevel
		if user.Profile.EducationLevel != nil {
			text := enum.GetEduLevelText(*user.Profile.EducationLevel)
			item.EducationLevelText = &text
		}
	}

	return item
}

// userToDetailResp 用户实体转详情 DTO
func userToDetailResp(user *entity.User) *dto.UserDetailResp {
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
		Roles:         make([]string, 0),
		CreatedAt:     user.CreatedAt.Format(time.RFC3339),
	}

	// 角色
	for _, ur := range user.Roles {
		if ur.Role != nil {
			resp.Roles = append(resp.Roles, ur.Role.Code)
		}
	}

	// 最后登录时间
	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Format(time.RFC3339)
		resp.LastLoginAt = &t
	}

	// 扩展信息
	if user.Profile != nil {
		resp.AvatarURL = user.Profile.AvatarURL
		resp.Nickname = user.Profile.Nickname
		resp.Email = user.Profile.Email
		resp.College = user.Profile.College
		resp.Major = user.Profile.Major
		resp.ClassName = user.Profile.ClassName
		resp.EnrollmentYear = user.Profile.EnrollmentYear
		resp.EducationLevel = user.Profile.EducationLevel
		resp.Grade = user.Profile.Grade
		resp.Remark = user.Profile.Remark
		if user.Profile.EducationLevel != nil {
			text := enum.GetEduLevelText(*user.Profile.EducationLevel)
			resp.EducationLevelText = &text
		}
	}

	return resp
}
