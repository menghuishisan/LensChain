// common.go
// 模块01 — 用户与认证：Handler 层公共辅助函数
// 提取各 handler 文件中重复的 ServiceContext 构建、错误处理等公共逻辑
// 消除 P2-6 代码重复问题

package auth

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
)

// buildServiceContext 从 gin.Context 构建 ServiceContext
// handler 层提取 HTTP 信息后传给 service 层，service 层不依赖 gin.Context
func buildServiceContext(c *gin.Context) *svcctx.ServiceContext {
	return svcctx.NewServiceContext(
		c.Request.Context(),
		middleware.GetUserID(c),
		middleware.GetSchoolID(c),
		middleware.GetRoles(c),
	).WithClientIP(c.ClientIP())
}

// handleError 统一错误处理
// 将业务错误码转为 HTTP 响应，非业务错误返回 500
func handleError(c *gin.Context, err error) {
	if appErr, ok := errcode.IsAppError(err); ok {
		response.Error(c, appErr)
		return
	}
	response.Error(c, errcode.ErrInternal)
}
