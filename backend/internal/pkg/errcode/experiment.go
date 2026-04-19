// experiment.go
// 该文件集中定义模块04“实验环境”相关的错误码，覆盖模板、镜像、实例生命周期、分组协作、
// Web 终端、快照、报告与 SimEngine 会话等实验子域，保证实验模块错误返回口径统一。

package errcode

import "net/http"

var (
	// 镜像管理
	ErrImageNotFound         = New(40410, http.StatusNotFound, "镜像不存在")
	ErrImageVersionNotFound  = New(40411, http.StatusNotFound, "镜像版本不存在")
	ErrImagePendingReview    = New(40917, http.StatusConflict, "镜像正在审核中")
	ErrImageVersionInUse     = New(40924, http.StatusConflict, "镜像版本正在被模板引用")
	ErrImageHasReferences    = New(40925, http.StatusConflict, "镜像存在关联版本，无法删除")
	ErrImageCategoryNotFound = New(40417, http.StatusNotFound, "镜像分类不存在")
	ErrImageCategoryInUse    = New(40926, http.StatusConflict, "镜像分类下存在镜像，无法删除")

	// 实验模板
	ErrTemplateNotFound         = New(40412, http.StatusNotFound, "实验模板不存在")
	ErrTemplateNotDraft         = New(40918, http.StatusConflict, "仅草稿状态的模板可编辑")
	ErrTemplateAlreadyPublished = New(40927, http.StatusConflict, "模板已发布")
	ErrTemplateHasInstances     = New(40928, http.StatusConflict, "模板存在关联实例，无法删除")
	ErrTemplateNotPublished     = New(40929, http.StatusConflict, "模板未发布")
	ErrContainerNotFound        = New(40418, http.StatusNotFound, "容器配置不存在")
	ErrInitScriptNotFound       = New(40419, http.StatusNotFound, "初始化脚本不存在")
	ErrRoleNotFound             = New(40420, http.StatusNotFound, "角色不存在")
	ErrTagNotFound              = New(40421, http.StatusNotFound, "标签不存在")
	ErrTagInUse                 = New(40930, http.StatusConflict, "标签正在被模板引用")

	// 实验实例
	ErrInstanceNotFound         = New(40413, http.StatusNotFound, "实验实例不存在")
	ErrInstanceNotRunning       = New(40919, http.StatusConflict, "实验实例未在运行中")
	ErrInstanceAlreadyExists    = New(40920, http.StatusConflict, "已有运行中的实验实例")
	ErrConcurrencyExceeded      = New(40921, http.StatusConflict, "并发实验数已达上限")
	ErrResourceQuotaExceeded    = New(40922, http.StatusConflict, "资源配额已用尽")
	ErrInstanceAlreadyPaused    = New(40931, http.StatusConflict, "实验实例已暂停")
	ErrInstanceAlreadySubmitted = New(40932, http.StatusConflict, "实验实例已提交")
	ErrInstanceTimeout          = New(40933, http.StatusConflict, "实验实例已超时")

	// 检查点
	ErrCheckpointNotFound      = New(40416, http.StatusNotFound, "检查点不存在")
	ErrCheckpointAlreadyPassed = New(40934, http.StatusConflict, "检查点已通过")
	ErrCheckpointScriptFailed  = New(40935, http.StatusConflict, "检查点脚本执行失败")

	// 实验分组
	ErrGroupNotFound     = New(40414, http.StatusNotFound, "实验分组不存在")
	ErrGroupFull         = New(40923, http.StatusConflict, "分组人数已满")
	ErrGroupAlreadyReady = New(40936, http.StatusConflict, "分组已就绪")
	ErrGroupMemberExists = New(40937, http.StatusConflict, "学生已在分组中")
	ErrGroupNotJoinable  = New(40938, http.StatusConflict, "分组不可加入")

	// 仿真场景
	ErrScenarioNotFound      = New(40415, http.StatusNotFound, "仿真场景不存在")
	ErrSimSceneNotFound      = New(40427, http.StatusNotFound, "模板仿真场景配置不存在")
	ErrScenarioPendingReview = New(40939, http.StatusConflict, "仿真场景正在审核中")
	ErrScenarioHasReferences = New(40940, http.StatusConflict, "仿真场景被模板引用，无法删除")
	ErrLinkGroupNotFound     = New(40422, http.StatusNotFound, "联动组不存在")

	// 资源配额
	ErrQuotaNotFound      = New(40423, http.StatusNotFound, "资源配额不存在")
	ErrQuotaAlreadyExists = New(40941, http.StatusConflict, "资源配额已存在")

	// 实验报告
	ErrReportNotFound      = New(40424, http.StatusNotFound, "实验报告不存在")
	ErrReportAlreadyExists = New(40942, http.StatusConflict, "实验报告已存在")
	ErrHeartbeatRateLimit  = New(42910, http.StatusTooManyRequests, "心跳上报过于频繁，请稍后再试")
	ErrCheckpointRateLimit = New(42911, http.StatusTooManyRequests, "检查点验证过于频繁，请稍后再试")

	// 快照
	ErrSnapshotNotFound = New(40425, http.StatusNotFound, "快照不存在")

	// SimEngine
	ErrSimSessionNotFound     = New(40426, http.StatusNotFound, "仿真会话不存在")
	ErrSimSessionCreateFailed = New(50010, http.StatusInternalServerError, "仿真会话创建失败")

	// K8s
	ErrK8sDeployFailed          = New(50011, http.StatusInternalServerError, "K8s部署失败")
	ErrK8sNamespaceCreateFailed = New(50012, http.StatusInternalServerError, "K8s命名空间创建失败")
)
