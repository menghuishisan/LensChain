// common.go
// 公共错误码定义
// 所有模块共用的通用错误码

package errcode

import "net/http"

// 400xx — 参数错误
var (
	ErrInvalidParams   = New(40001, http.StatusBadRequest, "参数校验失败")
	ErrInvalidPhone    = New(40002, http.StatusBadRequest, "手机号格式不正确")
	ErrInvalidPageSize = New(40003, http.StatusBadRequest, "分页参数不合法")
	ErrInvalidFormat   = New(40004, http.StatusBadRequest, "数据格式不正确")
	ErrInvalidID       = New(40005, http.StatusBadRequest, "ID格式不正确")
	ErrFileTooLarge    = New(40006, http.StatusBadRequest, "文件大小超出限制")
	ErrInvalidFileType = New(40007, http.StatusBadRequest, "不支持的文件类型")
)

// 401xx — 认证错误
var (
	ErrUnauthorized    = New(40100, http.StatusUnauthorized, "未登录或Token已过期")
	ErrTokenExpired    = New(40111, http.StatusUnauthorized, "Access Token已过期")
	ErrTokenInvalid    = New(40112, http.StatusUnauthorized, "Access Token无效")
	ErrTokenBlacklist  = New(40110, http.StatusUnauthorized, "Token已被注销")
)

// 403xx — 权限错误
var (
	ErrForbidden       = New(40300, http.StatusForbidden, "无权限访问")
	ErrSchoolFrozen    = New(40302, http.StatusForbidden, "学校已被冻结")
	ErrSchoolExpired   = New(40303, http.StatusForbidden, "学校授权已过期")
)

// 404xx — 资源不存在
var (
	ErrNotFound = New(40400, http.StatusNotFound, "资源不存在")
)

// 409xx — 冲突
var (
	ErrConflict = New(40900, http.StatusConflict, "资源冲突")
)

// 500xx — 服务端错误
var (
	ErrInternal = New(50000, http.StatusInternalServerError, "服务器内部错误")
	ErrDatabase = New(50001, http.StatusInternalServerError, "数据库操作失败")
	ErrRedis    = New(50002, http.StatusInternalServerError, "缓存操作失败")
	ErrMinIO    = New(50003, http.StatusInternalServerError, "文件存储操作失败")
	ErrNATS     = New(50004, http.StatusInternalServerError, "消息队列操作失败")
	ErrSMS      = New(50005, http.StatusInternalServerError, "短信发送失败")
)
