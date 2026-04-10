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
func RegisterCTFRoutes(rg *gin.RouterGroup) {
	ctf := rg.Group("/ctf")
	ctf.Use(middleware.JWTAuth(), middleware.TenantIsolation())

	// ========== 1. 竞赛管理 ==========
	competitions := ctf.Group("/competitions")
	{
		competitions.POST("", middleware.RequireSchoolAdminOrSuperAdmin(), todo)      // 创建竞赛
		competitions.GET("", todo)                                                    // 竞赛列表
		competitions.GET("/:id", todo)                                                // 竞赛详情
		competitions.PUT("/:id", todo)                                                // 编辑竞赛
		competitions.DELETE("/:id", todo)                                             // 删除竞赛
		competitions.POST("/:id/publish", todo)                                       // 发布竞赛
		competitions.POST("/:id/archive", todo)                                       // 归档竞赛
		competitions.POST("/:id/terminate", middleware.RequireSuperAdmin(), todo)      // 强制终止竞赛

		// 竞赛题目配置
		competitions.GET("/:id/challenges", todo)                                     // 竞赛题目列表
		competitions.POST("/:id/challenges", todo)                                    // 添加题目到竞赛
		competitions.PUT("/:id/challenges/sort", todo)                                // 竞赛题目排序

		// 报名
		competitions.POST("/:id/register", middleware.RequireStudent(), todo)          // 报名竞赛
		competitions.DELETE("/:id/register", middleware.RequireStudent(), todo)        // 取消报名
		competitions.GET("/:id/registrations", todo)                                  // 报名列表
		competitions.GET("/:id/my-registration", middleware.RequireStudent(), todo)    // 我的报名状态

		// 解题赛提交
		competitions.POST("/:id/submissions", todo)                                   // 提交Flag/攻击交易
		competitions.GET("/:id/submissions", todo)                                    // 团队提交记录
		competitions.GET("/:id/submissions/statistics", todo)                         // 提交统计

		// 攻防赛分组
		competitions.POST("/:id/ad-groups", todo)                                     // 创建攻防赛分组
		competitions.GET("/:id/ad-groups", todo)                                      // 分组列表
		competitions.POST("/:id/ad-groups/auto-assign", todo)                         // 自动分组

		// 排行榜
		competitions.GET("/:id/leaderboard", todo)                                    // 实时排行榜
		competitions.GET("/:id/leaderboard/history", todo)                            // 排行榜历史快照
		competitions.GET("/:id/leaderboard/final", todo)                              // 最终排名

		// 公告
		competitions.POST("/:id/announcements", todo)                                 // 发布公告
		competitions.GET("/:id/announcements", todo)                                  // 公告列表

		// 资源配额
		competitions.GET("/:id/resource-quota", todo)                                 // 竞赛资源配额详情
		competitions.PUT("/:id/resource-quota", middleware.RequireSuperAdmin(), todo)  // 设置竞赛资源配额

		// 题目环境
		competitions.POST("/:comp_id/challenges/:challenge_id/environment", todo)     // 启动题目环境
		competitions.GET("/:id/my-environments", todo)                                // 我的所有题目环境

		// 竞赛监控
		competitions.GET("/:id/monitor", todo)                                        // 竞赛运行监控
		competitions.GET("/:id/environments", todo)                                   // 竞赛环境资源列表

		// Token流水
		competitions.GET("/:id/token-ledger", todo)                                   // Token流水记录

		// 竞赛统计
		competitions.GET("/:id/statistics", todo)                                     // 竞赛统计数据
		competitions.GET("/:id/results", todo)                                        // 竞赛最终结果
	}

	// ========== 2. 题目管理 ==========
	challenges := ctf.Group("/challenges")
	{
		challenges.POST("", middleware.RequireTeacher(), todo)                         // 创建题目
		challenges.GET("", middleware.RequireAdminOrTeacher(), todo)                   // 题目列表/题库
		challenges.GET("/:id", todo)                                                  // 题目详情
		challenges.PUT("/:id", todo)                                                  // 编辑题目
		challenges.DELETE("/:id", todo)                                               // 删除题目
		challenges.POST("/:id/submit-review", todo)                                   // 提交审核

		// 合约管理
		challenges.GET("/:id/contracts", todo)                                        // 合约列表
		challenges.POST("/:id/contracts", todo)                                       // 添加合约

		// 断言管理
		challenges.GET("/:id/assertions", todo)                                       // 断言列表
		challenges.POST("/:id/assertions", todo)                                      // 添加断言
		challenges.PUT("/:id/assertions/sort", todo)                                  // 断言排序

		// 预验证
		challenges.POST("/:id/verify", todo)                                          // 发起预验证
		challenges.GET("/:id/verifications", todo)                                    // 预验证记录列表

		// 审核
		challenges.POST("/:id/review", middleware.RequireSuperAdmin(), todo)           // 审核题目
		challenges.GET("/:id/reviews", todo)                                          // 题目审核记录
	}

	// 题目合约（独立路径）
	challengeContracts := ctf.Group("/challenge-contracts")
	{
		challengeContracts.PUT("/:id", todo)            // 编辑合约
		challengeContracts.DELETE("/:id", todo)         // 删除合约
	}

	// 题目断言（独立路径）
	challengeAssertions := ctf.Group("/challenge-assertions")
	{
		challengeAssertions.PUT("/:id", todo)           // 编辑断言
		challengeAssertions.DELETE("/:id", todo)        // 删除断言
	}

	// 竞赛题目（独立路径）
	competitionChallenges := ctf.Group("/competition-challenges")
	{
		competitionChallenges.DELETE("/:id", todo)      // 移除竞赛题目
	}

	// 预验证详情（独立路径）
	challengeVerifications := ctf.Group("/challenge-verifications")
	{
		challengeVerifications.GET("/:id", todo)        // 预验证详情
	}

	// ========== 3. 漏洞转化 ==========
	ctf.GET("/swc-registry", middleware.RequireTeacher(), todo)                        // SWC Registry 列表
	ctf.POST("/challenges/import-swc", middleware.RequireTeacher(), todo)              // 从SWC导入生成题目
	ctf.GET("/challenge-templates", middleware.RequireTeacher(), todo)                 // 参数化模板列表
	ctf.GET("/challenge-templates/:id", middleware.RequireTeacher(), todo)             // 模板详情

	// 从模板生成题目
	ctf.POST("/challenges/generate-from-template", middleware.RequireTeacher(), todo)  // 从模板生成题目

	// ========== 4. 题目审核（超管） ==========
	challengeReviews := ctf.Group("/challenge-reviews")
	challengeReviews.Use(middleware.RequireSuperAdmin())
	{
		challengeReviews.GET("/pending", todo)           // 待审核题目列表
	}

	// ========== 5. 团队管理 ==========
	teams := ctf.Group("/teams")
	{
		teams.GET("/:id", todo)                                                       // 团队详情
		teams.PUT("/:id", todo)                                                       // 编辑团队信息
		teams.POST("/:id/disband", todo)                                              // 解散团队
		teams.POST("/join", middleware.RequireStudent(), todo)                         // 通过邀请码加入团队
		teams.DELETE("/:id/members/:student_id", todo)                                // 移除队员
		teams.POST("/:id/leave", todo)                                                // 退出团队
		teams.GET("/:id/token-ledger", todo)                                          // 团队Token流水
		teams.GET("/:id/chain", todo)                                                 // 队伍链信息
	}

	// ========== 6. 攻防赛 ==========
	adGroups := ctf.Group("/ad-groups")
	{
		adGroups.GET("/:id", todo)                                                    // 分组详情
		adGroups.GET("/:id/rounds", todo)                                             // 回合列表
		adGroups.GET("/:id/current-round", todo)                                      // 当前回合状态
		adGroups.GET("/:id/attacks", todo)                                            // 分组全部攻击记录
		adGroups.GET("/:id/chains", todo)                                             // 分组所有队伍链
	}

	adRounds := ctf.Group("/ad-rounds")
	{
		adRounds.GET("/:id", todo)                                                    // 回合详情
		adRounds.POST("/:id/attacks", todo)                                           // 提交攻击交易
		adRounds.GET("/:id/attacks", todo)                                            // 本回合攻击记录
		adRounds.POST("/:id/defenses", todo)                                          // 提交补丁合约
		adRounds.GET("/:id/defenses", todo)                                           // 本回合防守记录
	}

	// ========== 7. 题目环境（独立路径） ==========
	challengeEnvs := ctf.Group("/challenge-environments")
	{
		challengeEnvs.GET("/:id", todo)                                               // 环境详情
		challengeEnvs.POST("/:id/reset", todo)                                        // 重置题目环境
		challengeEnvs.POST("/:id/destroy", todo)                                      // 销毁题目环境
		challengeEnvs.POST("/:id/force-destroy", middleware.RequireSuperAdmin(), todo) // 强制回收环境
	}

	// ========== 8. CTF公告（独立路径） ==========
	ctfAnnouncements := ctf.Group("/announcements")
	{
		ctfAnnouncements.GET("/:id", todo)              // 公告详情
		ctfAnnouncements.DELETE("/:id", todo)           // 删除公告
	}

	// ========== 9. 全平台竞赛概览（超管） ==========
	ctfAdmin := ctf.Group("/admin")
	ctfAdmin.Use(middleware.RequireSuperAdmin())
	{
		ctfAdmin.GET("/competitions/overview", todo)     // 全平台竞赛概览
	}
}
