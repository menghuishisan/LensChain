// helper.go
// 路由辅助函数
// 提供路由注册阶段使用的占位处理器等工具函数

package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// todo 占位处理器
// 在 handler 层实现之前，所有路由暂时绑定此函数
// 返回 501 Not Implemented，表示接口尚未实现
func todo(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"code":    501,
		"message": "接口尚未实现",
		"data":    nil,
	})
}
