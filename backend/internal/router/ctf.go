// ctf.go
// 模块05 — CTF竞赛 路由注册
// 注册竞赛管理、题目管理、合约、断言、漏洞转化、预验证、审核、
// 竞赛题目配置、团队、报名、解题赛提交、攻防赛、排行榜、公告、
// 资源配额、题目环境、队伍链、竞赛监控、竞赛统计等路由
// 共 89 个 REST 端点 + 1 个 WebSocket 端点

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterCTFRoutes 注册CTF竞赛模块路由
func RegisterCTFRoutes(rg *gin.RouterGroup, ch *CTFHandlers) {
	ctf := rg.Group("/ctf")
	ctf.Use(middleware.JWTAuth(), middleware.TenantIsolation())

	// ========== 1. 竞赛管理 ==========
	competitions := ctf.Group("/competitions")
	{
		competitions.POST("", middleware.RequireSchoolAdminOrSuperAdmin(), ch.CompetitionHandler.CreateCompetition)     // 创建竞赛
		competitions.GET("", ch.CompetitionHandler.ListCompetitions)                                                    // 竞赛列表
		competitions.GET("/:id", ch.CompetitionHandler.GetCompetition)                                                  // 竞赛详情
		competitions.PUT("/:id", ch.CompetitionHandler.UpdateCompetition)                                               // 编辑竞赛
		competitions.DELETE("/:id", ch.CompetitionHandler.DeleteCompetition)                                            // 删除竞赛
		competitions.POST("/:id/publish", ch.CompetitionHandler.PublishCompetition)                                     // 发布竞赛
		competitions.POST("/:id/archive", ch.CompetitionHandler.ArchiveCompetition)                                     // 归档竞赛
		competitions.POST("/:id/terminate", middleware.RequireSuperAdmin(), ch.CompetitionHandler.TerminateCompetition) // 强制终止竞赛

		// 竞赛题目配置
		competitions.GET("/:id/challenges", ch.CompetitionHandler.ListCompetitionChallenges)      // 竞赛题目列表
		competitions.POST("/:id/challenges", ch.CompetitionHandler.AddCompetitionChallenges)      // 添加题目到竞赛
		competitions.PUT("/:id/challenges/sort", ch.CompetitionHandler.SortCompetitionChallenges) // 竞赛题目排序

		// 团队
		competitions.POST("/:id/teams", middleware.RequireStudent(), ch.CompetitionHandler.CreateTeam) // 创建团队
		competitions.GET("/:id/teams", ch.CompetitionHandler.ListTeams)                                // 竞赛团队列表

		// 报名
		competitions.POST("/:id/register", middleware.RequireStudent(), ch.CompetitionHandler.RegisterCompetition)     // 报名竞赛
		competitions.DELETE("/:id/register", middleware.RequireStudent(), ch.CompetitionHandler.CancelRegistration)    // 取消报名
		competitions.GET("/:id/registrations", ch.CompetitionHandler.ListRegistrations)                                // 报名列表
		competitions.GET("/:id/my-registration", middleware.RequireStudent(), ch.CompetitionHandler.GetMyRegistration) // 我的报名状态

		// 解题赛提交
		competitions.POST("/:id/submissions", ch.CompetitionHandler.SubmitCompetitionChallenge)                   // 提交Flag/攻击交易
		competitions.GET("/:id/submissions", ch.CompetitionHandler.ListCompetitionSubmissions)                    // 团队提交记录
		competitions.GET("/:id/submissions/statistics", ch.CompetitionHandler.GetCompetitionSubmissionStatistics) // 提交统计

		// 攻防赛分组
		competitions.POST("/:id/ad-groups", ch.BattleHandler.CreateAdGroup)                  // 创建攻防赛分组
		competitions.GET("/:id/ad-groups", ch.BattleHandler.ListAdGroups)                    // 分组列表
		competitions.POST("/:id/ad-groups/auto-assign", ch.BattleHandler.AutoAssignAdGroups) // 自动分组

		// 排行榜
		competitions.GET("/:id/leaderboard", ch.CompetitionHandler.GetLeaderboard)                // 实时排行榜
		competitions.GET("/:id/leaderboard/history", ch.CompetitionHandler.GetLeaderboardHistory) // 排行榜历史快照
		competitions.GET("/:id/leaderboard/final", ch.CompetitionHandler.GetFinalLeaderboard)     // 最终排名

		// 公告
		competitions.POST("/:id/announcements", ch.CompetitionHandler.CreateAnnouncement) // 发布公告
		competitions.GET("/:id/announcements", ch.CompetitionHandler.ListAnnouncements)   // 公告列表

		// 资源配额
		competitions.GET("/:id/resource-quota", ch.CompetitionHandler.GetResourceQuota)                                    // 竞赛资源配额详情
		competitions.PUT("/:id/resource-quota", middleware.RequireSuperAdmin(), ch.CompetitionHandler.UpdateResourceQuota) // 设置竞赛资源配额

		// 题目环境
		competitions.POST("/:comp_id/challenges/:challenge_id/environment", ch.EnvironmentHandler.StartChallengeEnvironment) // 启动题目环境
		competitions.GET("/:id/my-environments", ch.EnvironmentHandler.ListMyEnvironments)                                   // 我的所有题目环境

		// 竞赛监控
		competitions.GET("/:id/monitor", ch.CompetitionHandler.GetCompetitionMonitor)            // 竞赛运行监控
		competitions.GET("/:id/environments", ch.EnvironmentHandler.ListCompetitionEnvironments) // 竞赛环境资源列表

		// Token流水
		competitions.GET("/:id/token-ledger", ch.BattleHandler.ListCompetitionTokenLedger) // Token流水记录

		// 竞赛统计
		competitions.GET("/:id/statistics", ch.CompetitionHandler.GetCompetitionStatistics) // 竞赛统计数据
		competitions.GET("/:id/results", ch.CompetitionHandler.GetCompetitionResults)       // 竞赛最终结果
	}

	// ========== 2. 题目管理 ==========
	challenges := ctf.Group("/challenges")
	{
		challenges.POST("", middleware.RequireTeacher(), ch.CompetitionHandler.CreateChallenge)      // 创建题目
		challenges.GET("", middleware.RequireAdminOrTeacher(), ch.CompetitionHandler.ListChallenges) // 题目列表/题库
		challenges.GET("/:id", ch.CompetitionHandler.GetChallenge)                                   // 题目详情
		challenges.PUT("/:id", ch.CompetitionHandler.UpdateChallenge)                                // 编辑题目
		challenges.DELETE("/:id", ch.CompetitionHandler.DeleteChallenge)                             // 删除题目
		challenges.POST("/:id/submit-review", ch.CompetitionHandler.SubmitChallengeReview)           // 提交审核

		// 合约管理
		challenges.GET("/:id/contracts", ch.CompetitionHandler.ListChallengeContracts)   // 合约列表
		challenges.POST("/:id/contracts", ch.CompetitionHandler.CreateChallengeContract) // 添加合约

		// 断言管理
		challenges.GET("/:id/assertions", ch.CompetitionHandler.ListChallengeAssertions)      // 断言列表
		challenges.POST("/:id/assertions", ch.CompetitionHandler.CreateChallengeAssertion)    // 添加断言
		challenges.PUT("/:id/assertions/sort", ch.CompetitionHandler.SortChallengeAssertions) // 断言排序

		// 预验证
		challenges.POST("/:id/verify", ch.CompetitionHandler.VerifyChallenge)                  // 发起预验证
		challenges.GET("/:id/verifications", ch.CompetitionHandler.ListChallengeVerifications) // 预验证记录列表

		// 审核
		challenges.POST("/:id/review", middleware.RequireSuperAdmin(), ch.CompetitionHandler.ReviewChallenge) // 审核题目
		challenges.GET("/:id/reviews", ch.CompetitionHandler.ListChallengeReviews)                            // 题目审核记录
	}

	// 题目合约（独立路径）
	challengeContracts := ctf.Group("/challenge-contracts")
	{
		challengeContracts.PUT("/:id", ch.CompetitionHandler.UpdateChallengeContract)    // 编辑合约
		challengeContracts.DELETE("/:id", ch.CompetitionHandler.DeleteChallengeContract) // 删除合约
	}

	// 题目断言（独立路径）
	challengeAssertions := ctf.Group("/challenge-assertions")
	{
		challengeAssertions.PUT("/:id", ch.CompetitionHandler.UpdateChallengeAssertion)    // 编辑断言
		challengeAssertions.DELETE("/:id", ch.CompetitionHandler.DeleteChallengeAssertion) // 删除断言
	}

	// 竞赛题目（独立路径）
	competitionChallenges := ctf.Group("/competition-challenges")
	{
		competitionChallenges.DELETE("/:id", ch.CompetitionHandler.RemoveCompetitionChallenge) // 移除竞赛题目
	}

	// 预验证详情（独立路径）
	challengeVerifications := ctf.Group("/challenge-verifications")
	{
		challengeVerifications.GET("/:id", ch.CompetitionHandler.GetChallengeVerification) // 预验证详情
	}

	// ========== 3. 漏洞转化 ==========
	ctf.GET("/swc-registry", middleware.RequireTeacher(), ch.CompetitionHandler.ListSWCRegistry)                 // SWC Registry 列表
	ctf.POST("/challenges/import-swc", middleware.RequireTeacher(), ch.CompetitionHandler.ImportSWCChallenge)    // 从SWC导入生成题目
	ctf.GET("/challenge-templates", middleware.RequireTeacher(), ch.CompetitionHandler.ListChallengeTemplates)   // 参数化模板列表
	ctf.GET("/challenge-templates/:id", middleware.RequireTeacher(), ch.CompetitionHandler.GetChallengeTemplate) // 模板详情

	// 从模板生成题目
	ctf.POST("/challenges/generate-from-template", middleware.RequireTeacher(), ch.CompetitionHandler.GenerateChallengeFromTemplate) // 从模板生成题目
	ctf.POST("/challenges/import-external", middleware.RequireTeacher(), ch.CompetitionHandler.ImportExternalVulnerability)          // 从外部真实漏洞源导入

	// WebSocket
	ctf.GET("/ws", ch.RealtimeHandler.ServeWS) // CTF实时通信

	// ========== 4. 题目审核（超管） ==========
	challengeReviews := ctf.Group("/challenge-reviews")
	challengeReviews.Use(middleware.RequireSuperAdmin())
	{
		challengeReviews.GET("/pending", ch.CompetitionHandler.ListPendingChallengeReviews) // 待审核题目列表
	}

	// ========== 5. 团队管理 ==========
	teams := ctf.Group("/teams")
	{
		teams.GET("/:id", ch.CompetitionHandler.GetTeam)                                 // 团队详情
		teams.PUT("/:id", ch.CompetitionHandler.UpdateTeam)                              // 编辑团队信息
		teams.POST("/:id/disband", ch.CompetitionHandler.DisbandTeam)                    // 解散团队
		teams.POST("/join", middleware.RequireStudent(), ch.CompetitionHandler.JoinTeam) // 通过邀请码加入团队
		teams.DELETE("/:id/members/:student_id", ch.CompetitionHandler.RemoveTeamMember) // 移除队员
		teams.POST("/:id/leave", ch.CompetitionHandler.LeaveTeam)                        // 退出团队
		teams.GET("/:id/token-ledger", ch.BattleHandler.ListTeamTokenLedger)             // 团队Token流水
		teams.GET("/:id/chain", ch.BattleHandler.GetTeamChain)                           // 队伍链信息
	}

	// ========== 6. 攻防赛 ==========
	adGroups := ctf.Group("/ad-groups")
	{
		adGroups.GET("/:id", ch.BattleHandler.GetAdGroup)                    // 分组详情
		adGroups.GET("/:id/rounds", ch.BattleHandler.ListRounds)             // 回合列表
		adGroups.GET("/:id/current-round", ch.BattleHandler.GetCurrentRound) // 当前回合状态
		adGroups.GET("/:id/attacks", ch.BattleHandler.ListGroupAttacks)      // 分组全部攻击记录
		adGroups.GET("/:id/chains", ch.BattleHandler.ListGroupChains)        // 分组所有队伍链
	}

	adRounds := ctf.Group("/ad-rounds")
	{
		adRounds.GET("/:id", ch.BattleHandler.GetRound)                   // 回合详情
		adRounds.POST("/:id/attacks", ch.BattleHandler.SubmitAttack)      // 提交攻击交易
		adRounds.GET("/:id/attacks", ch.BattleHandler.ListRoundAttacks)   // 本回合攻击记录
		adRounds.POST("/:id/defenses", ch.BattleHandler.SubmitDefense)    // 提交补丁合约
		adRounds.GET("/:id/defenses", ch.BattleHandler.ListRoundDefenses) // 本回合防守记录
	}

	// ========== 7. 题目环境（独立路径） ==========
	challengeEnvs := ctf.Group("/challenge-environments")
	{
		challengeEnvs.GET("/:id", ch.EnvironmentHandler.GetChallengeEnvironment)                                                         // 环境详情
		challengeEnvs.POST("/:id/reset", ch.EnvironmentHandler.ResetChallengeEnvironment)                                                // 重置题目环境
		challengeEnvs.POST("/:id/destroy", ch.EnvironmentHandler.DestroyChallengeEnvironment)                                            // 销毁题目环境
		challengeEnvs.POST("/:id/force-destroy", middleware.RequireSuperAdmin(), ch.EnvironmentHandler.ForceDestroyChallengeEnvironment) // 强制回收环境
	}

	// ========== 8. CTF公告（独立路径） ==========
	ctfAnnouncements := ctf.Group("/announcements")
	{
		ctfAnnouncements.GET("/:id", ch.CompetitionHandler.GetAnnouncement)       // 公告详情
		ctfAnnouncements.DELETE("/:id", ch.CompetitionHandler.DeleteAnnouncement) // 删除公告
	}

	// ========== 9. 全平台竞赛概览（超管） ==========
	ctfAdmin := ctf.Group("/admin")
	ctfAdmin.Use(middleware.RequireSuperAdmin())
	{
		ctfAdmin.GET("/competitions/overview", ch.CompetitionHandler.GetAdminOverview) // 全平台竞赛概览
	}
}
