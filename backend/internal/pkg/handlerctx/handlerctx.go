// handlerctx.go
// 该文件是 handler 层与 service 层之间的公共桥接工具，负责把 Gin 请求上下文转换成
// service 可用的服务上下文，并把业务错误统一映射成标准 HTTP 响应。它解决的是 handler
// 层反复写“取用户信息、转错误码、返回统一响应”的重复劳动。

package handlerctx

import (
	"github.com/gin-gonic/gin"

	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/requestctx"
	"github.com/lenschain/backend/internal/pkg/response"
)

// BuildServiceContext 从 gin.Context 构建 ServiceContext
// handler 层提取 HTTP 信息后传给 service 层，service 层不依赖 gin.Context。
func BuildServiceContext(c *gin.Context) *svcctx.ServiceContext {
	return svcctx.NewServiceContext(
		c.Request.Context(),
		requestctx.GetUserID(c),
		requestctx.GetSchoolID(c),
		requestctx.GetRoles(c),
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
