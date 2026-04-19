// school_status.go
// 该文件提供租户学校状态的统一访问校验逻辑，负责从缓存读取学校快照并判断当前学校是否
// 处于可访问状态。中间件通过它完成冻结、注销、授权过期等拦截，而不是自己解析缓存字段。

package tenantguard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

// SchoolStatusSnapshot 表示学校状态缓存的最小快照。
type SchoolStatusSnapshot struct {
	Status       int    `json:"status"`
	LicenseEndAt string `json:"license_end_at"`
}

// CheckSchoolAccess 校验学校当前是否允许访问。
// 缓存缺失或缓存异常时降级放行，由上层业务继续做更精确的校验。
func CheckSchoolAccess(ctx context.Context, schoolID int64, now time.Time) error {
	snapshot, ok, err := loadSchoolStatusSnapshot(ctx, schoolID)
	if err != nil || !ok {
		return nil
	}

	switch snapshot.Status {
	case enum.SchoolStatusFrozen:
		return errcode.ErrSchoolFrozen
	case enum.SchoolStatusCancelled:
		return errcode.ErrForbidden.WithMessage("学校已注销")
	}

	if snapshot.LicenseEndAt == "" || snapshot.Status != enum.SchoolStatusActive {
		return nil
	}

	licenseEndAt, err := time.Parse(time.RFC3339, snapshot.LicenseEndAt)
	if err != nil {
		return nil
	}
	if now.UTC().After(licenseEndAt.UTC()) {
		return errcode.ErrSchoolExpired
	}

	return nil
}

// loadSchoolStatusSnapshot 从缓存读取学校状态快照。
func loadSchoolStatusSnapshot(ctx context.Context, schoolID int64) (*SchoolStatusSnapshot, bool, error) {
	value, err := cache.GetString(ctx, cache.KeySchoolStatus+fmt.Sprintf("%d", schoolID))
	if err != nil {
		return nil, false, err
	}

	var snapshot SchoolStatusSnapshot
	if err := json.Unmarshal([]byte(value), &snapshot); err != nil {
		return nil, false, err
	}
	return &snapshot, true, nil
}
