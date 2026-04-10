// ctf.go
// 模块05 — CTF竞赛模块错误码
// 对照 docs/modules/05-CTF竞赛/03-API接口设计.md

package errcode

import "net/http"

var (
	// 竞赛管理
	ErrCompetitionNotFound    = New(40417, http.StatusNotFound, "竞赛不存在")
	ErrCompetitionNotDraft    = New(40924, http.StatusConflict, "仅草稿状态的竞赛可编辑")
	ErrCompetitionNotActive   = New(40925, http.StatusConflict, "竞赛不在进行中")
	ErrCompetitionEnded       = New(40926, http.StatusConflict, "竞赛已结束")

	// 题目管理
	ErrChallengeNotFound      = New(40418, http.StatusNotFound, "题目不存在")
	ErrChallengeNotApproved   = New(40927, http.StatusConflict, "题目未通过审核")
	ErrChallengePendingReview = New(40928, http.StatusConflict, "题目正在审核中")

	// 团队
	ErrTeamNotFound           = New(40419, http.StatusNotFound, "团队不存在")
	ErrTeamFull               = New(40929, http.StatusConflict, "团队人数已满")
	ErrAlreadyInTeam          = New(40930, http.StatusConflict, "已在其他团队中")
	ErrInvalidInviteCode      = New(40016, http.StatusBadRequest, "团队邀请码无效")

	// 报名
	ErrRegistrationClosed     = New(40931, http.StatusConflict, "报名已截止")
	ErrAlreadyRegistered      = New(40932, http.StatusConflict, "已报名该竞赛")
	ErrMaxTeamsReached        = New(40933, http.StatusConflict, "参赛队伍数已达上限")

	// 提交
	ErrSubmissionRateLimit    = New(40017, http.StatusBadRequest, "提交过于频繁，请稍后再试")
	ErrSubmissionCooldown     = New(40018, http.StatusBadRequest, "连续失败过多，冷却中")
	ErrAlreadySolved          = New(40934, http.StatusConflict, "已解出该题目")
	ErrLeaderboardFrozen      = New(40935, http.StatusConflict, "排行榜已冻结")

	// 攻防赛
	ErrNotInAttackPhase       = New(40936, http.StatusConflict, "当前不在攻击阶段")
	ErrNotInDefensePhase      = New(40937, http.StatusConflict, "当前不在防守阶段")
	ErrEnvironmentNotFound    = New(40420, http.StatusNotFound, "题目环境不存在")
)
