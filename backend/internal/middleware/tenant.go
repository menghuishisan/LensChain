// tenant.go
// 该文件实现多租户隔离中间件，负责根据当前登录用户所属学校向请求上下文注入租户边界，
// 并在进入业务处理前检查学校是否被冻结、注销或授权过期。它解决的是平台级租户访问边界，
// 不是具体模块的业务数据筛选逻辑本身。

package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/requestctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/tenantguard"
)

// TenantIsolation 多租户隔离中间件
// 非超级管理员的请求自动注入 school_id 过滤
// 同时检查学校状态（冻结/过期的学校拒绝访问）
func TenantIsolation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 超级管理员不受租户隔离限制
		if IsSuperAdmin(c) {
			requestctx.SetTenantSchoolID(c, 0)
			c.Next()
			return
		}

		schoolID := requestctx.GetSchoolID(c)
		if schoolID == 0 {
			response.Abort(c, errcode.ErrForbidden.WithMessage("用户未关联学校"))
			return
		}

		// 检查学校状态缓存
		if err := tenantguard.CheckSchoolAccess(c.Request.Context(), schoolID, time.Now().UTC()); err != nil {
			response.Abort(c, errcode.FromError(err))
			return
		}

		// 注入租户 school_id
		requestctx.SetTenantSchoolID(c, schoolID)
		c.Next()
	}
}
