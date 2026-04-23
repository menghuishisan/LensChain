// backup_service.go
// 模块08 — 系统管理与监控：数据备份业务。
// 该文件集中处理手动备份、备份列表、下载与备份策略配置，避免和告警/统计逻辑混放。

package system

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/storage"
	systemrepo "github.com/lenschain/backend/internal/repository/system"
)

// TriggerBackup 手动触发备份。
func (s *service) TriggerBackup(ctx context.Context, sc *svcctx.ServiceContext) (*dto.TriggerBackupResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	running, _, err := s.backupsRunning(ctx)
	if err != nil {
		return nil, err
	}
	if running {
		return nil, errcode.ErrBackupInProgress
	}

	now := time.Now().UTC()
	record := &entity.BackupRecord{
		BackupType:   enum.BackupTypeManual,
		Status:       enum.BackupStatusRunning,
		DatabaseName: s.cfg.Database.DBName,
		StartedAt:    now,
		TriggeredBy:  int64Ptr(sc.UserID),
	}
	if err := s.backupRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	cronpkg.RunAsync("模块08手动备份执行", func(asyncCtx context.Context) {
		s.executeBackup(record.ID, record.BackupType, int64Ptr(sc.UserID))
	})

	return &dto.TriggerBackupResp{
		ID:         int64String(record.ID),
		BackupType: record.BackupType,
		Status:     record.Status,
		StatusText: enum.GetBackupStatusText(record.Status),
		StartedAt:  record.StartedAt.UTC().Format(time.RFC3339),
	}, nil
}

// ListBackups 获取备份列表。
func (s *service) ListBackups(ctx context.Context, sc *svcctx.ServiceContext, req *dto.BackupListReq) (*dto.BackupListResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	items, total, err := s.backupRepo.List(ctx, &systemrepo.BackupRecordListParams{
		Status:   req.Status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	backupConfig, err := s.GetBackupConfig(ctx, sc)
	if err != nil {
		return nil, err
	}
	list := make([]dto.BackupListItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		list = append(list, buildBackupListItem(item))
	}
	return &dto.BackupListResp{
		List:         list,
		Pagination:   buildPaginationResp(page, pageSize, total),
		BackupConfig: *backupConfig,
	}, nil
}

// DownloadBackup 获取备份下载结果。
func (s *service) DownloadBackup(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*ServiceFile, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	record, err := s.backupRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrBackupNotFound
		}
		return nil, err
	}
	if record.Status != enum.BackupStatusSuccess || record.FilePath == nil || *record.FilePath == "" {
		return nil, errcode.ErrInvalidParams.WithMessage("备份文件暂不可下载")
	}
	url, err := storage.GetFileURL(ctx, *record.FilePath, time.Hour)
	if err != nil {
		return nil, err
	}
	return &ServiceFile{RedirectURL: url}, nil
}

// UpdateBackupConfig 更新备份配置。
func (s *service) UpdateBackupConfig(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateBackupConfigReq) (*dto.BackupConfigResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	if err := s.BatchUpdateConfigs(ctx, sc, "backup", &dto.BatchUpdateSystemConfigsReq{
		Configs: []dto.BatchUpdateSystemConfigItem{
			{
				Key:       "auto_backup_enabled",
				Value:     fmt.Sprintf("%t", req.AutoEnabled),
				UpdatedAt: req.AutoEnabledUpdatedAt,
			},
			{
				Key:       "auto_backup_cron",
				Value:     req.Cron,
				UpdatedAt: req.CronUpdatedAt,
			},
			{
				Key:       "backup_retention_count",
				Value:     fmt.Sprintf("%d", req.RetentionCount),
				UpdatedAt: req.RetentionCountUpdatedAt,
			},
		},
	}); err != nil {
		return nil, err
	}
	return s.getBackupConfigNoAuth(ctx)
}

// getBackupConfigNoAuth 读取备份配置，不做权限校验，供调度器复用。
func (s *service) getBackupConfigNoAuth(ctx context.Context) (*dto.BackupConfigResp, error) {
	configs, err := s.loadConfigGroupConfigs(ctx, "backup")
	if err != nil {
		return nil, err
	}
	return buildBackupConfigResp(configs), nil
}

// syncAutoBackupSchedule 在备份配置变更后刷新自动备份任务调度。
func (s *service) syncAutoBackupSchedule(ctx context.Context) error {
	if s == nil || s.backupScheduleSyncer == nil {
		return nil
	}
	return s.backupScheduleSyncer.SyncAutoBackupTask(ctx)
}

// GetBackupConfig 获取备份配置。
func (s *service) GetBackupConfig(ctx context.Context, sc *svcctx.ServiceContext) (*dto.BackupConfigResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	return s.getBackupConfigNoAuth(ctx)
}

// buildBackupListItem 构建备份列表项。
func buildBackupListItem(item *entity.BackupRecord) dto.BackupListItem {
	var durationSeconds *int
	if item.CompletedAt != nil {
		value := int(item.CompletedAt.Sub(item.StartedAt).Seconds())
		durationSeconds = &value
	}
	var fileSizeText *string
	if item.FileSize != nil {
		value := formatFileSize(*item.FileSize)
		fileSizeText = &value
	}
	return dto.BackupListItem{
		ID:              int64String(item.ID),
		BackupType:      item.BackupType,
		BackupTypeText:  enum.GetBackupTypeText(item.BackupType),
		Status:          item.Status,
		StatusText:      enum.GetBackupStatusText(item.Status),
		DatabaseName:    item.DatabaseName,
		FileSize:        item.FileSize,
		FileSizeText:    fileSizeText,
		StartedAt:       item.StartedAt.UTC().Format(time.RFC3339),
		CompletedAt:     formatTime(item.CompletedAt),
		DurationSeconds: durationSeconds,
		ErrorMessage:    item.ErrorMessage,
	}
}

// formatFileSize 格式化文件大小文本。
func formatFileSize(value int64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if value >= gb {
		return fmt.Sprintf("%.1f GB", float64(value)/float64(gb))
	}
	if value >= mb {
		return fmt.Sprintf("%.1f MB", float64(value)/float64(mb))
	}
	return fmt.Sprintf("%d B", value)
}
