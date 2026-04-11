// handlerctx.go
// Handler 公共上下文辅助工具
// 统一从 gin.Context 构建 service 层上下文并处理业务错误响应

package handlerctx

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
)

// BuildServiceContext 从 gin.Context 构建 ServiceContext
// handler 层提取 HTTP 信息后传给 service 层，service 层不依赖 gin.Context。
func BuildServiceContext(c *gin.Context) *svcctx.ServiceContext {
	return svcctx.NewServiceContext(
		c.Request.Context(),
		middleware.GetUserID(c),
		middleware.GetSchoolID(c),
		middleware.GetRoles(c),
	).WithClientIP(c.ClientIP())
}

// HandleError 统一处理 handler 层错误响应
// 业务错误按 errcode 返回，非业务错误统一返回内部错误。
func HandleError(c *gin.Context, err error) {
	if appErr, ok := errcode.IsAppError(err); ok {
		response.Error(c, appErr)
		return
	}
	response.Error(c, errcode.ErrInternal)
}
