// school_lifecycle.go
// 模块02 — 学校与租户管理：学校生命周期管理
// 负责冻结/解冻、注销/恢复等状态变更操作，以及实体转 DTO 辅助方法
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"context"
	"math"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/mask"
)

// Freeze 冻结学校
// 只有已激活或缓冲期的学校可以冻结
func (s *schoolService) Freeze(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.FreezeSchoolReq) error {
	sch, err := s.schoolRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrSchoolNotFound
	}
	// 只有已激活或缓冲期的学校可以冻结
	if sch.Status == enum.SchoolStatusFrozen {
		return errcode.ErrSchoolAlreadyFrozen
	}
	if sch.Status != enum.SchoolStatusActive && sch.Status != enum.SchoolStatusBuffering {
		return errcode.ErrForbidden.WithMessage("当前学校状态不允许冻结")
	}

	now := time.Now()
	if err := s.schoolRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status":        enum.SchoolStatusFrozen,
		"frozen_at":     now,
		"frozen_reason": req.Reason,
		"updated_at":    now,
	}); err != nil {
		return errcode.ErrInternal.WithMessage("冻结学校失败")
	}

	// 清除缓存 + 踢出所有用户
	deleteSchoolStatusCache(ctx, id)
	if s.sessionKicker != nil {
		_ = s.sessionKicker.KickSchoolUsers(ctx, id)
	}

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "freeze_school", "school", id, map[string]interface{}{
		"reason": req.Reason,
	})
	return nil
}

// Unfreeze 解冻学校
func (s *schoolService) Unfreeze(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	sch, err := s.schoolRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrSchoolNotFound
	}
	if sch.Status != enum.SchoolStatusFrozen {
		return errcode.ErrSchoolNotFrozen
	}

	if err := s.schoolRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status":        enum.SchoolStatusActive,
		"frozen_at":     nil,
		"frozen_reason": nil,
		"updated_at":    time.Now(),
	}); err != nil {
		return errcode.ErrInternal.WithMessage("解冻学校失败")
	}

	// 刷新缓存
	refreshSchoolStatusCache(ctx, id, enum.SchoolStatusActive, sch.LicenseEndAt)

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "unfreeze_school", "school", id, nil)
	return nil
}

// Cancel 注销学校（软删除学校 + 软删除所有用户）
// 需要二次确认（confirm: true），防止误操作
func (s *schoolService) Cancel(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.CancelSchoolReq) error {
	// 二次确认校验
	if !req.Confirm {
		return errcode.ErrCancelNotConfirmed
	}

	sch, err := s.schoolRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrSchoolNotFound
	}
	if sch.Status == enum.SchoolStatusCancelled {
		return errcode.ErrSchoolAlreadyCancelled
	}

	now := time.Now()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 更新学校状态 + 软删除
		if err := tx.Model(&entity.School{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":     enum.SchoolStatusCancelled,
			"deleted_at": now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		// 2. 软删除该校所有用户
		return tx.Model(&entity.User{}).
			Where("school_id = ? AND deleted_at IS NULL", id).
			Updates(map[string]interface{}{"deleted_at": now}).Error
	})
	if err != nil {
		return errcode.ErrInternal.WithMessage("注销学校失败")
	}

	// 清除缓存 + 踢出所有用户
	deleteSchoolStatusCache(ctx, id)
	if s.sessionKicker != nil {
		_ = s.sessionKicker.KickSchoolUsers(ctx, id)
	}

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "cancel_school", "school", id, nil)
	return nil
}

// Restore 恢复已注销学校
func (s *schoolService) Restore(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	// 使用 Unscoped 查询已软删除的学校
	var sch entity.School
	if err := s.db.WithContext(ctx).Unscoped().First(&sch, id).Error; err != nil {
		return errcode.ErrSchoolNotFound
	}
	if sch.Status != enum.SchoolStatusCancelled {
		return errcode.ErrSchoolNotCancelled
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 恢复学校
		if err := tx.Unscoped().Model(&entity.School{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":     enum.SchoolStatusActive,
			"deleted_at": nil,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		// 2. 恢复该校所有用户
		return tx.Unscoped().Model(&entity.User{}).
			Where("school_id = ? AND deleted_at IS NOT NULL", id).
			Updates(map[string]interface{}{"deleted_at": nil}).Error
	})
	if err != nil {
		return errcode.ErrInternal.WithMessage("恢复学校失败")
	}

	// 刷新缓存
	refreshSchoolStatusCache(ctx, id, enum.SchoolStatusActive, sch.LicenseEndAt)

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "restore_school", "school", id, nil)
	return nil
}

// ========== 实体转 DTO 辅助方法 ==========

// schoolToListItem 学校实体转列表项
func schoolToListItem(sch *entity.School) *dto.SchoolListItem {
	item := &dto.SchoolListItem{
		ID:           strconv.FormatInt(sch.ID, 10),
		Name:         sch.Name,
		Code:         sch.Code,
		LogoURL:      sch.LogoURL,
		Status:       sch.Status,
		StatusText:   enum.GetSchoolStatusText(sch.Status),
		ContactName:  sch.ContactName,
		ContactPhone: mask.Phone(sch.ContactPhone),
		CreatedAt:    sch.CreatedAt.Format(time.RFC3339),
	}
	if sch.LicenseStartAt != nil {
		t := sch.LicenseStartAt.Format(time.RFC3339)
		item.LicenseStartAt = &t
	}
	if sch.LicenseEndAt != nil {
		t := sch.LicenseEndAt.Format(time.RFC3339)
		item.LicenseEndAt = &t
		remaining := int(math.Ceil(time.Until(*sch.LicenseEndAt).Hours() / 24))
		item.LicenseRemainingDays = &remaining
	}
	return item
}

// schoolToDetailResp 学校实体转详情响应
func schoolToDetailResp(sch *entity.School) *dto.SchoolDetailResp {
	resp := &dto.SchoolDetailResp{
		ID:           strconv.FormatInt(sch.ID, 10),
		Name:         sch.Name,
		Code:         sch.Code,
		LogoURL:      sch.LogoURL,
		Address:      sch.Address,
		Website:      sch.Website,
		Description:  sch.Description,
		Status:       sch.Status,
		StatusText:   enum.GetSchoolStatusText(sch.Status),
		FrozenReason: sch.FrozenReason,
		ContactName:  sch.ContactName,
		ContactPhone: sch.ContactPhone,
		ContactEmail: sch.ContactEmail,
		ContactTitle: sch.ContactTitle,
		CreatedAt:    sch.CreatedAt.Format(time.RFC3339),
	}
	if sch.LicenseStartAt != nil {
		t := sch.LicenseStartAt.Format(time.RFC3339)
		resp.LicenseStartAt = &t
	}
	if sch.LicenseEndAt != nil {
		t := sch.LicenseEndAt.Format(time.RFC3339)
		resp.LicenseEndAt = &t
	}
	if sch.FrozenAt != nil {
		t := sch.FrozenAt.Format(time.RFC3339)
		resp.FrozenAt = &t
	}
	if sch.CreatedBy != nil {
		cb := strconv.FormatInt(*sch.CreatedBy, 10)
		resp.CreatedBy = &cb
	}
	return resp
}
