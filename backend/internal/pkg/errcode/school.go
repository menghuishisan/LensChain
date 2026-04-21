// school.go
// 该文件集中定义模块02“学校与租户管理”的错误码，覆盖入驻申请、短信验证码、学校审核、
// 授权状态和 SSO 配置等场景，用来保证租户模块的错误返回与文档一致。

package errcode

import "net/http"

var (
	// 入驻申请
	ErrDuplicateApplication   = New(40901, http.StatusConflict, "该手机号已有待审核的申请")
	ErrDuplicateSchoolName    = New(40902, http.StatusConflict, "该学校名称已存在")
	ErrDuplicateSchoolCode    = New(40903, http.StatusConflict, "该学校编码已存在")
	ErrSMSCodeSendTooFrequent = New(40014, http.StatusBadRequest, "短信发送过于频繁，请稍后再试")
	ErrApplicationNotPending  = New(40909, http.StatusConflict, "该申请已审核")
	ErrApplicationNotRejected = New(40910, http.StatusConflict, "仅被拒绝的申请可重新提交")

	// 学校管理
	ErrSchoolNotFound         = New(40402, http.StatusNotFound, "学校不存在")
	ErrApplicationNotFound    = New(40403, http.StatusNotFound, "申请记录不存在")
	ErrSchoolAlreadyActive    = New(40906, http.StatusConflict, "学校已处于激活状态")
	ErrSchoolNotFrozen        = New(40907, http.StatusConflict, "仅冻结状态的学校可解冻")
	ErrSchoolNotCancelled     = New(40908, http.StatusConflict, "仅已注销学校可恢复")
	ErrSchoolAlreadyFrozen    = New(40911, http.StatusConflict, "该学校已处于冻结状态")
	ErrSchoolAlreadyCancelled = New(40912, http.StatusConflict, "该学校已注销")
	ErrCancelNotConfirmed     = New(40013, http.StatusBadRequest, "注销操作需要确认")

	// SSO配置
	ErrSSOTestFailed     = New(40010, http.StatusBadRequest, "SSO连接测试失败")
	ErrSSONotTested      = New(40011, http.StatusBadRequest, "SSO配置尚未通过测试")
	ErrSSOConfigNotFound = New(40410, http.StatusNotFound, "SSO配置不存在")
)
