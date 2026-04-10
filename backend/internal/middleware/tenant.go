// tenant.go
// 多租户 school_id 注入中间件
// 以 school_id 为租户标识，业务数据严格隔离
// 超级管理员可跨学校查询（不注入 school_id 过滤）
// 学校管理员/教师/学生 自动注入 school_id 过滤

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
)

// ContextKeyTenantSchoolID 租户隔离的 school_id
const ContextKeyTenantSchoolID = "tenant_school_id"

// TenantIsolation 多租户隔离中间件
// 非超级管理员的请求自动注入 school_id 过滤
// 同时检查学校状态（冻结/过期的学校拒绝访问）
func TenantIsolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 超级管理员不受租户隔离限制
		if IsSuperAdmin(c) {
			c.Set(ContextKeyTenantSchoolID, int64(0))
			c.Next()
			return
		}

		schoolID := GetSchoolID(c)
		if schoolID == 0 {
			response.Abort(c, errcode.ErrForbidden.WithMessage("用户未关联学校"))
			return
		}

		// 检查学校状态缓存
		if err := checkSchoolStatus(c, schoolID); err != nil {
			return
		}

		// 注入租户 school_id
		c.Set(ContextKeyTenantSchoolID, schoolID)
		c.Next()
	}
}

// GetTenantSchoolID 获取租户 school_id
// 返回0表示超级管理员（不限制学校）
func GetTenantSchoolID(c *gin.Context) int64 {
	if v, exists := c.Get(ContextKeyTenantSchoolID); exists {
		return v.(int64)
	}
	return 0
}

// checkSchoolStatus 检查学校状态
// 从 Redis 缓存读取，冻结/过期/注销的学校拒绝访问
func checkSchoolStatus(c *gin.Context, schoolID int64) error {
	ctx := context.Background()
	key := cache.KeySchoolStatus + formatInt64(schoolID)

	val, err := cache.GetString(ctx, key)
	if err != nil {
		// 缓存未命中，放行（由业务层进一步检查）
		return nil
	}

	var status struct {
		Status       int    `json:"status"`
		LicenseEndAt string `json:"license_end_at"` // ISO 8601 格式，与模块02写入一致
	}
	if err := json.Unmarshal([]byte(val), &status); err != nil {
		return nil
	}

	// 状态4=已冻结
	if status.Status == 4 {
		response.Abort(c, errcode.ErrSchoolFrozen)
		return errcode.ErrSchoolFrozen
	}

	// 状态5=已注销
	if status.Status == 5 {
		response.Abort(c, errcode.ErrForbidden.WithMessage("学校已注销"))
		return errcode.ErrForbidden
	}

	// 检查授权是否过期
	if status.LicenseEndAt != "" {
		licenseEnd, err := time.Parse(time.RFC3339, status.LicenseEndAt)
		if err == nil && time.Now().UTC().After(licenseEnd) {
			response.Abort(c, errcode.ErrSchoolExpired)
			return errcode.ErrSchoolExpired
		}
	}

	return nil
}

// formatInt64 将 int64 转为字符串
func formatInt64(n int64) string {
	return fmt.Sprintf("%d", n)
}
