// instance_service_queue.go
// 模块04 — 实验环境：课程并发排队补位逻辑
// 在课程并发释放后自动拉起下一条排队实例，满足验收中的自动补位要求

package experiment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

// activateNextQueuedInstance 在课程释放并发后自动启动下一条排队实例。
func (s *instanceService) activateNextQueuedInstance(ctx context.Context, courseID int64) {
	if courseID == 0 || cache.Get() == nil {
		return
	}

	queueKey := cache.KeyExpQueue + strconv.FormatInt(courseID, 10)
	for {
		instanceIDStr, err := cache.Get().LPop(ctx, queueKey).Result()
		if err != nil || instanceIDStr == "" {
			return
		}

		instanceID, parseErr := strconv.ParseInt(instanceIDStr, 10, 64)
		if parseErr != nil {
			continue
		}
		instance, loadErr := s.instanceRepo.GetByID(ctx, instanceID)
		if loadErr != nil || instance == nil || instance.Status != enum.InstanceStatusQueued {
			continue
		}

		templateAggregate, templateErr := loadTemplateAggregate(
			ctx,
			s.templateRepo,
			s.templateContainerRepo,
			s.checkpointRepo,
			s.initScriptRepo,
			s.simSceneRepo,
			nil,
			nil,
			nil,
			instance.TemplateID,
		)
		if templateErr != nil {
			_ = s.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
				"status":        enum.InstanceStatusError,
				"error_message": "排队补位失败：实验模板不存在",
				"updated_at":    time.Now(),
			})
			continue
		}

		now := time.Now()
		if updateErr := s.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
			"status":         enum.InstanceStatusCreating,
			"started_at":     now,
			"last_active_at": now,
			"updated_at":     now,
		}); updateErr != nil {
			continue
		}

		instance.Status = enum.InstanceStatusCreating
		instance.StartedAt = &now
		instance.LastActiveAt = &now
		instance.UpdatedAt = now

		if err := s.quotaRepo.IncrUsedConcurrency(ctx, instance.SchoolID, 1); err != nil {
			_ = s.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
				"status":        enum.InstanceStatusQueued,
				"error_message": errcode.ErrResourceQuotaExceeded.Message,
				"updated_at":    time.Now(),
			})
			return
		}

		s.pushCourseMonitorStatusChange(instance, int(enum.InstanceStatusQueued), int(enum.InstanceStatusCreating))
		snapshotID := ""
		if cache.Get() != nil {
			snapshotKey := fmt.Sprintf("%s%d", cache.KeyExpQueueSnapshot, instance.ID)
			if value, getErr := cache.Get().Get(ctx, snapshotKey).Result(); getErr == nil {
				snapshotID = value
				_ = cache.Del(ctx, snapshotKey)
			}
		}
		cronpkg.RunAsync("模块04排队实例自动启动", func(asyncCtx context.Context) {
			s.provisionEnvironment(detachContext(ctx), instance, templateAggregate, snapshotID, true)
		})
		return
	}
}
