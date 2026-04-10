// school.go
// 模块02 — 学校与租户管理模块错误码
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package errcode

import "net/http"

var (
	// 入驻申请
	ErrDuplicateApplication   = New(40901, http.StatusConflict, "该手机号已有待审核的申请")
	ErrDuplicateSchoolName    = New(40902, http.StatusConflict, "该学校名称已存在")
	ErrDuplicateSchoolCode    = New(40903, http.StatusConflict, "该学校编码已存在")
	ErrApplicationNotPending  = New(40909, http.StatusConflict, "申请不在待审核状态")
	ErrApplicationNotRejected = New(40910, http.StatusConflict, "仅被拒绝的申请可重新提交")

	// 学校管理
	ErrSchoolNotFound         = New(40402, http.StatusNotFound, "学校不存在")
	ErrApplicationNotFound    = New(40403, http.StatusNotFound, "申请记录不存在")
	ErrSchoolAlreadyActive    = New(40906, http.StatusConflict, "学校已处于激活状态")
	ErrSchoolNotFrozen        = New(40907, http.StatusConflict, "学校未处于冻结状态")
	ErrSchoolNotCancelled     = New(40908, http.StatusConflict, "学校未处于注销状态")
	ErrSchoolAlreadyFrozen    = New(40911, http.StatusConflict, "学校已处于冻结状态")
	ErrSchoolAlreadyCancelled = New(40912, http.StatusConflict, "学校已处于注销状态")
	ErrCancelNotConfirmed     = New(40013, http.StatusBadRequest, "注销操作需要确认")

	// SSO配置
	ErrSSOTestFailed = New(40010, http.StatusBadRequest, "SSO连接测试失败")
)
