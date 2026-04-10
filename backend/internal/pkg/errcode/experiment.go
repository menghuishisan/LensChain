// experiment.go
// 模块04 — 实验环境模块错误码
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package errcode

import "net/http"

var (
	// 镜像管理
	ErrImageNotFound         = New(40410, http.StatusNotFound, "镜像不存在")
	ErrImageVersionNotFound  = New(40411, http.StatusNotFound, "镜像版本不存在")
	ErrImagePendingReview    = New(40917, http.StatusConflict, "镜像正在审核中")

	// 实验模板
	ErrTemplateNotFound      = New(40412, http.StatusNotFound, "实验模板不存在")
	ErrTemplateNotDraft      = New(40918, http.StatusConflict, "仅草稿状态的模板可编辑")

	// 实验实例
	ErrInstanceNotFound      = New(40413, http.StatusNotFound, "实验实例不存在")
	ErrInstanceNotRunning    = New(40919, http.StatusConflict, "实验实例未在运行中")
	ErrInstanceAlreadyExists = New(40920, http.StatusConflict, "已有运行中的实验实例")
	ErrConcurrencyExceeded   = New(40921, http.StatusConflict, "并发实验数已达上限")
	ErrResourceQuotaExceeded = New(40922, http.StatusConflict, "资源配额已用尽")

	// 实验分组
	ErrGroupNotFound         = New(40414, http.StatusNotFound, "实验分组不存在")
	ErrGroupFull             = New(40923, http.StatusConflict, "分组人数已满")

	// 仿真场景
	ErrScenarioNotFound      = New(40415, http.StatusNotFound, "仿真场景不存在")
	ErrCheckpointNotFound    = New(40416, http.StatusNotFound, "检查点不存在")
)
