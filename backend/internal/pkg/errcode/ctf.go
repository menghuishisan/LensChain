// ctf.go
// 该文件定义模块05“CTF竞赛”所需的错误码，包括竞赛配置、题目审核、报名参赛、提交限流、
// 环境资源和攻防对抗等场景，目的是让 CTF 模块的错误返回与接口文档保持一致。

package errcode

import "net/http"

var (
	// 竞赛管理
	ErrCompetitionNotFound       = New(40417, http.StatusNotFound, "竞赛不存在")
	ErrCompetitionNotDraft       = New(40924, http.StatusConflict, "仅草稿状态的竞赛可编辑")
	ErrCompetitionNotActive      = New(40925, http.StatusConflict, "竞赛不在进行中")
	ErrCompetitionEnded          = New(40926, http.StatusConflict, "竞赛已结束")
	ErrCompetitionStatusInvalid  = New(40946, http.StatusConflict, "竞赛状态不允许当前操作")
	ErrCompetitionTypeInvalid    = New(40026, http.StatusBadRequest, "竞赛类型无效")
	ErrCompetitionScopeInvalid   = New(40027, http.StatusBadRequest, "竞赛范围无效")
	ErrCompetitionTimeInvalid    = New(40028, http.StatusBadRequest, "竞赛时间配置不合法")
	ErrCompetitionConfigRequired = New(40029, http.StatusBadRequest, "竞赛配置缺失")
	ErrCompetitionChallengeEmpty = New(40030, http.StatusBadRequest, "竞赛至少需要配置一道题目")
	ErrCompetitionQuotaExceeded  = New(40947, http.StatusConflict, "竞赛资源配额不足")
	ErrCompetitionResultNotReady = New(40948, http.StatusConflict, "竞赛结果尚未生成")

	// 题目管理
	ErrChallengeNotFound             = New(40418, http.StatusNotFound, "题目不存在")
	ErrChallengeNotApproved          = New(40927, http.StatusConflict, "题目未通过审核")
	ErrChallengePendingReview        = New(40928, http.StatusConflict, "题目正在审核中")
	ErrChallengeStatusInvalid        = New(40949, http.StatusConflict, "题目状态不允许当前操作")
	ErrChallengeCategoryInvalid      = New(40031, http.StatusBadRequest, "题目分类无效")
	ErrChallengeDifficultyInvalid    = New(40032, http.StatusBadRequest, "题目难度无效")
	ErrChallengeScoreInvalid         = New(40033, http.StatusBadRequest, "题目基础分值不合法")
	ErrChallengeFlagTypeInvalid      = New(40034, http.StatusBadRequest, "题目 Flag 类型无效")
	ErrChallengeStaticFlagRequired   = New(40035, http.StatusBadRequest, "静态 Flag 题目必须填写 Flag")
	ErrChallengeRuntimeModeInvalid   = New(40047, http.StatusBadRequest, "题目运行时模式无效")
	ErrChallengeRuntimeConfigInvalid = New(40048, http.StatusBadRequest, "题目运行时配置不合法")
	ErrChallengeContractRequired     = New(40036, http.StatusBadRequest, "链上验证题目至少需要一个合约")
	ErrChallengeAssertionRequired    = New(40037, http.StatusBadRequest, "链上验证题目至少需要一个断言")
	ErrChallengeVerificationRunning  = New(40950, http.StatusConflict, "存在进行中的题目预验证")
	ErrChallengeVerificationAbsent   = New(40951, http.StatusConflict, "题目缺少通过的预验证记录")
	ErrChallengeAlreadyInContest     = New(40952, http.StatusConflict, "题目已添加到竞赛中")

	// 题目合约与断言
	ErrChallengeContractNameRequired = New(40038, http.StatusBadRequest, "合约名称不能为空")
	ErrChallengeContractSourceEmpty  = New(40039, http.StatusBadRequest, "合约源码不能为空")
	ErrChallengeContractInvalid      = New(40046, http.StatusBadRequest, "题目合约配置不合法")
	ErrChallengeContractNotFound     = New(40421, http.StatusNotFound, "题目合约不存在")
	ErrChallengeAssertionTypeInvalid = New(40040, http.StatusBadRequest, "断言类型无效")
	ErrChallengeAssertionOpInvalid   = New(40041, http.StatusBadRequest, "断言比较运算符无效")
	ErrChallengeAssertionNotFound    = New(40422, http.StatusNotFound, "题目断言不存在")

	// 模板与 SWC
	ErrChallengeTemplateNotFound      = New(40423, http.StatusNotFound, "题目模板不存在")
	ErrChallengeTemplateParamMissing  = New(40042, http.StatusBadRequest, "模板参数不完整")
	ErrChallengeTemplateDifficultyErr = New(40043, http.StatusBadRequest, "难度超出模板适用范围")
	ErrSWCEntryInvalid                = New(40424, http.StatusNotFound, "SWC 条目不存在")
	ErrSWCExampleUnavailable          = New(40953, http.StatusConflict, "该 SWC 条目缺少可导入示例")

	// 团队
	ErrTeamNotFound            = New(40419, http.StatusNotFound, "团队不存在")
	ErrTeamFull                = New(40929, http.StatusConflict, "团队人数已满")
	ErrAlreadyInTeam           = New(40930, http.StatusConflict, "已在其他团队中")
	ErrInvalidInviteCode       = New(40016, http.StatusBadRequest, "团队邀请码无效")
	ErrTeamLocked              = New(40954, http.StatusConflict, "团队已锁定")
	ErrTeamCaptainOnly         = New(40304, http.StatusForbidden, "仅队长可执行该操作")
	ErrTeamMemberNotFound      = New(40425, http.StatusNotFound, "团队成员不存在")
	ErrTeamMinMemberNotReached = New(40955, http.StatusConflict, "团队人数未达到报名下限")
	ErrTeamModeNoNeedCreate    = New(40956, http.StatusConflict, "个人赛无需创建团队")
	ErrTeamModeNoNeedJoin      = New(40957, http.StatusConflict, "个人赛无需加入团队")

	// 报名
	ErrRegistrationClosed     = New(40931, http.StatusConflict, "报名已截止")
	ErrAlreadyRegistered      = New(40932, http.StatusConflict, "已报名该竞赛")
	ErrMaxTeamsReached        = New(40933, http.StatusConflict, "参赛队伍数已达上限")
	ErrRegistrationNotAllowed = New(40958, http.StatusConflict, "当前竞赛不允许报名")
	ErrRegistrationNotFound   = New(40426, http.StatusNotFound, "报名记录不存在")

	// 提交
	ErrSubmissionRateLimit     = New(42912, http.StatusTooManyRequests, "提交过于频繁，请稍后再试")
	ErrSubmissionCooldown      = New(42913, http.StatusTooManyRequests, "连续失败过多，冷却中")
	ErrAlreadySolved           = New(40934, http.StatusConflict, "已解出该题目")
	ErrLeaderboardFrozen       = New(40935, http.StatusConflict, "排行榜已冻结")
	ErrSubmissionInvalid       = New(40044, http.StatusBadRequest, "提交内容无效")
	ErrSubmissionChallengeGone = New(40427, http.StatusNotFound, "竞赛题目不存在")
	ErrSubmissionTeamMissing   = New(40428, http.StatusNotFound, "当前用户未加入参赛团队")

	// 攻防赛
	ErrNotInAttackPhase       = New(40936, http.StatusConflict, "当前不在攻击阶段")
	ErrNotInDefensePhase      = New(40937, http.StatusConflict, "当前不在防守阶段")
	ErrEnvironmentNotFound    = New(40420, http.StatusNotFound, "题目环境不存在")
	ErrAdGroupNotFound        = New(40429, http.StatusNotFound, "攻防赛分组不存在")
	ErrAdRoundNotFound        = New(40430, http.StatusNotFound, "攻防赛回合不存在")
	ErrAdSelfAttackForbidden  = New(40959, http.StatusConflict, "不能攻击自己的队伍")
	ErrAdCrossGroupForbidden  = New(40960, http.StatusConflict, "目标队伍不在当前分组")
	ErrAdChallengePatched     = New(40961, http.StatusConflict, "目标漏洞已被修复")
	ErrAdPatchAlreadyAccepted = New(40962, http.StatusConflict, "该漏洞补丁已提交成功")
	ErrAdPatchSourceRequired  = New(40045, http.StatusBadRequest, "补丁源码不能为空")
	ErrAdCompetitionOnly      = New(40963, http.StatusConflict, "仅攻防对抗赛支持该操作")
	ErrAdAttackLocked         = New(40965, http.StatusConflict, "目标漏洞正在被其他队伍攻击，请稍后重试")

	// 公告、资源与环境
	ErrAnnouncementNotFoundCTF  = New(40431, http.StatusNotFound, "竞赛公告不存在")
	ErrResourceQuotaNotFoundCTF = New(40432, http.StatusNotFound, "竞赛资源配额不存在")
	ErrEnvironmentAlreadyExists = New(40964, http.StatusConflict, "题目环境已存在")
)
