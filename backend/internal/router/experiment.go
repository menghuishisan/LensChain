// experiment.go
// 模块04 — 实验环境 路由注册
// 注册镜像管理、实验模板、容器配置、检查点、初始化脚本、仿真场景、
// 标签、多人角色、实验实例、快照、操作日志、实验报告、分组、
// 组内通信、教师监控、资源配额、全局监控、共享实验库等路由
// 共 114 个 REST 端点 + 4 个 WebSocket 端点

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterExperimentRoutes 注册实验环境模块路由
func RegisterExperimentRoutes(rg *gin.RouterGroup) {
	// ========== 1. 镜像管理 ==========
	images := rg.Group("/images")
	images.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		images.GET("", todo)                                                         // 镜像列表
		images.POST("", middleware.RequireAdminOrTeacher(), todo)                     // 创建/上传镜像
		images.GET("/:id", todo)                                                     // 镜像详情
		images.PUT("/:id", todo)                                                     // 编辑镜像信息
		images.DELETE("/:id", middleware.RequireSuperAdmin(), todo)                   // 删除/下架镜像
		images.POST("/:id/review", middleware.RequireSuperAdmin(), todo)              // 审核镜像

		// 镜像版本
		images.GET("/:id/versions", todo)                                            // 镜像版本列表
		images.POST("/:id/versions", todo)                                           // 添加镜像版本
		images.GET("/:id/config-template", middleware.RequireTeacher(), todo)         // 获取镜像配置模板
		images.GET("/:id/documentation", middleware.RequireTeacher(), todo)           // 获取镜像结构化文档
	}

	// 镜像分类
	imageCategories := rg.Group("/image-categories")
	imageCategories.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		imageCategories.GET("", todo)                   // 镜像分类列表
	}

	// 镜像版本（独立路径）
	imageVersions := rg.Group("/image-versions")
	imageVersions.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		imageVersions.PUT("/:id", todo)                 // 编辑镜像版本
		imageVersions.DELETE("/:id", middleware.RequireSuperAdmin(), todo)            // 删除镜像版本
		imageVersions.PATCH("/:id/default", todo)       // 设为默认版本
	}

	// ========== 2. 实验模板管理 ==========
	templates := rg.Group("/experiment-templates")
	templates.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		templates.POST("", middleware.RequireTeacher(), todo)                         // 创建实验模板
		templates.GET("", middleware.RequireTeacher(), todo)                          // 实验模板列表
		templates.GET("/:id", middleware.RequireTeacher(), todo)                      // 实验模板详情
		templates.PUT("/:id", middleware.RequireTeacher(), todo)                      // 编辑实验模板
		templates.DELETE("/:id", middleware.RequireTeacher(), todo)                   // 删除实验模板
		templates.POST("/:id/publish", middleware.RequireTeacher(), todo)             // 发布实验模板
		templates.POST("/:id/clone", middleware.RequireTeacher(), todo)               // 克隆实验模板
		templates.PATCH("/:id/share", middleware.RequireTeacher(), todo)              // 设置共享状态
		templates.GET("/:id/k8s-config", middleware.RequireTeacher(), todo)           // 查看K8s编排配置
		templates.POST("/:id/k8s-config", middleware.RequireTeacher(), todo)          // 微调K8s编排配置
		templates.POST("/:id/validate", middleware.RequireTeacher(), todo)            // 模板配置验证

		// 容器配置
		templates.GET("/:id/containers", middleware.RequireTeacher(), todo)           // 容器配置列表
		templates.POST("/:id/containers", middleware.RequireTeacher(), todo)          // 添加容器配置
		templates.PUT("/:id/containers/sort", middleware.RequireTeacher(), todo)      // 容器排序

		// 检查点
		templates.GET("/:id/checkpoints", middleware.RequireTeacher(), todo)          // 检查点列表
		templates.POST("/:id/checkpoints", middleware.RequireTeacher(), todo)         // 添加检查点
		templates.PUT("/:id/checkpoints/sort", middleware.RequireTeacher(), todo)     // 检查点排序

		// 初始化脚本
		templates.GET("/:id/init-scripts", middleware.RequireTeacher(), todo)         // 初始化脚本列表
		templates.POST("/:id/init-scripts", middleware.RequireTeacher(), todo)        // 添加初始化脚本

		// 仿真场景配置
		templates.GET("/:id/sim-scenes", middleware.RequireTeacher(), todo)           // 模板仿真场景配置列表
		templates.POST("/:id/sim-scenes", middleware.RequireTeacher(), todo)          // 添加仿真场景到模板
		templates.PUT("/:id/sim-scenes/layout", middleware.RequireTeacher(), todo)    // 更新仿真场景布局

		// 标签
		templates.PUT("/:id/tags", middleware.RequireTeacher(), todo)                 // 设置模板标签
		templates.GET("/:id/tags", middleware.RequireTeacher(), todo)                 // 获取模板标签

		// 多人实验角色
		templates.GET("/:id/roles", middleware.RequireTeacher(), todo)                // 角色列表
		templates.POST("/:id/roles", middleware.RequireTeacher(), todo)               // 添加角色
	}

	// 模板容器配置（独立路径）
	templateContainers := rg.Group("/template-containers")
	templateContainers.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	templateContainers.Use(middleware.RequireTeacher())
	{
		templateContainers.PUT("/:id", todo)            // 编辑容器配置
		templateContainers.DELETE("/:id", todo)         // 删除容器配置
	}

	// 模板检查点（独立路径）
	templateCheckpoints := rg.Group("/template-checkpoints")
	templateCheckpoints.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	templateCheckpoints.Use(middleware.RequireTeacher())
	{
		templateCheckpoints.PUT("/:id", todo)           // 编辑检查点
		templateCheckpoints.DELETE("/:id", todo)        // 删除检查点
	}

	// 模板初始化脚本（独立路径）
	templateInitScripts := rg.Group("/template-init-scripts")
	templateInitScripts.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	templateInitScripts.Use(middleware.RequireTeacher())
	{
		templateInitScripts.PUT("/:id", todo)           // 编辑初始化脚本
		templateInitScripts.DELETE("/:id", todo)        // 删除初始化脚本
	}

	// 模板仿真场景（独立路径）
	templateSimScenes := rg.Group("/template-sim-scenes")
	templateSimScenes.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	templateSimScenes.Use(middleware.RequireTeacher())
	{
		templateSimScenes.PUT("/:id", todo)             // 编辑仿真场景配置
		templateSimScenes.DELETE("/:id", todo)          // 移除仿真场景
	}

	// 模板角色（独立路径）
	templateRoles := rg.Group("/template-roles")
	templateRoles.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	templateRoles.Use(middleware.RequireTeacher())
	{
		templateRoles.PUT("/:id", todo)                 // 编辑角色
		templateRoles.DELETE("/:id", todo)              // 删除角色
	}

	// ========== 3. 仿真场景库 ==========
	simScenarios := rg.Group("/sim-scenarios")
	simScenarios.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		simScenarios.GET("", todo)                                                   // 仿真场景列表
		simScenarios.POST("", middleware.RequireAdminOrTeacher(), todo)               // 上传自定义仿真场景
		simScenarios.GET("/:id", todo)                                               // 场景详情
		simScenarios.PUT("/:id", todo)                                               // 编辑场景信息
		simScenarios.DELETE("/:id", middleware.RequireSuperAdmin(), todo)             // 删除/下架场景
		simScenarios.POST("/:id/review", middleware.RequireSuperAdmin(), todo)        // 审核场景
	}

	// 联动组
	simLinkGroups := rg.Group("/sim-link-groups")
	simLinkGroups.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	simLinkGroups.Use(middleware.RequireTeacher())
	{
		simLinkGroups.GET("", todo)                     // 联动组列表
		simLinkGroups.GET("/:id", todo)                 // 联动组详情
	}

	// ========== 4. 标签管理 ==========
	tags := rg.Group("/tags")
	tags.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	tags.Use(middleware.RequireTeacher())
	{
		tags.GET("", todo)                              // 标签列表
		tags.POST("", todo)                             // 创建自定义标签
	}

	// ========== 5. 实验实例管理 ==========
	instances := rg.Group("/experiment-instances")
	instances.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		instances.POST("", middleware.RequireStudent(), todo)                         // 启动实验环境
		instances.GET("", middleware.RequireStudent(), todo)                          // 我的实验实例列表
		instances.GET("/:id", todo)                                                  // 实验实例详情
		instances.POST("/:id/pause", todo)                                           // 暂停实验
		instances.POST("/:id/resume", todo)                                          // 恢复实验
		instances.POST("/:id/restart", todo)                                         // 重新开始实验
		instances.POST("/:id/submit", todo)                                          // 提交实验
		instances.POST("/:id/destroy", todo)                                         // 销毁实验环境
		instances.POST("/:id/heartbeat", todo)                                       // 心跳上报

		// 检查点验证
		instances.POST("/:id/checkpoints/verify", todo)                              // 触发检查点验证
		instances.GET("/:id/checkpoints", todo)                                      // 检查点结果列表

		// 快照
		instances.GET("/:id/snapshots", todo)                                        // 快照列表
		instances.POST("/:id/snapshots", todo)                                       // 手动创建快照
		instances.POST("/:id/snapshots/:snapshot_id/restore", todo)                  // 从快照恢复

		// 操作日志
		instances.GET("/:id/operation-logs", todo)                                   // 操作日志列表

		// 实验报告
		instances.POST("/:id/report", todo)                                          // 提交实验报告
		instances.GET("/:id/report", todo)                                           // 获取实验报告
		instances.PUT("/:id/report", todo)                                           // 更新实验报告

		// 教师监控相关
		instances.GET("/:id/terminal-stream", middleware.RequireTeacher(), todo)      // 远程查看学生终端（WebSocket）
		instances.POST("/:id/message", middleware.RequireTeacher(), todo)             // 向学生发送指导消息
		instances.POST("/:id/force-destroy", middleware.RequireAdminOrTeacher(), todo) // 强制回收实验环境
		instances.POST("/:id/manual-grade", middleware.RequireTeacher(), todo)        // 教师手动评分
	}

	// 检查点结果评分（独立路径）
	checkpointResults := rg.Group("/checkpoint-results")
	checkpointResults.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	checkpointResults.Use(middleware.RequireTeacher())
	{
		checkpointResults.POST("/:id/grade", todo)      // 手动评分检查点
	}

	// ========== 6. 实验分组 ==========
	groups := rg.Group("/experiment-groups")
	groups.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		groups.POST("", middleware.RequireTeacher(), todo)                            // 创建实验分组
		groups.GET("", todo)                                                         // 分组列表
		groups.GET("/:id", todo)                                                     // 分组详情
		groups.PUT("/:id", middleware.RequireTeacher(), todo)                         // 编辑分组
		groups.DELETE("/:id", middleware.RequireTeacher(), todo)                      // 删除分组
		groups.POST("/auto-assign", middleware.RequireTeacher(), todo)                // 系统随机分组
		groups.POST("/:id/join", middleware.RequireStudent(), todo)                   // 学生加入分组
		groups.DELETE("/:id/members/:student_id", middleware.RequireTeacher(), todo)  // 移除组员
		groups.GET("/:id/members", todo)                                             // 组员列表
		groups.GET("/:id/progress", todo)                                            // 组内进度同步

		// 组内通信
		groups.POST("/:id/messages", todo)                                           // 发送组内消息
		groups.GET("/:id/messages", todo)                                            // 组内消息历史
	}

	// ========== 7. 教师监控 ==========
	// 课程维度的实验监控（挂在 /courses 下）
	courseMonitor := rg.Group("/courses")
	courseMonitor.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	courseMonitor.Use(middleware.RequireTeacher())
	{
		courseMonitor.GET("/:id/experiment-monitor", todo)                            // 课程实验监控面板
		courseMonitor.GET("/:id/experiment-statistics", todo)                         // 实验统计数据
	}

	// ========== 8. 资源配额管理 ==========
	quotas := rg.Group("/resource-quotas")
	quotas.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	quotas.Use(middleware.RequireSchoolAdminOrSuperAdmin())
	{
		quotas.GET("", todo)                            // 资源配额列表
		quotas.POST("", middleware.RequireSuperAdmin(), todo)                         // 创建资源配额
		quotas.PUT("/:id", todo)                        // 编辑资源配额
		quotas.GET("/:id", todo)                        // 资源配额详情
	}

	// 学校资源使用情况
	schoolResource := rg.Group("/schools")
	schoolResource.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	schoolResource.Use(middleware.RequireSchoolAdminOrSuperAdmin())
	{
		schoolResource.GET("/:id/resource-usage", todo) // 学校资源使用情况
	}

	// ========== 9. 全局监控（超管） ==========
	adminExperiment := rg.Group("/admin")
	adminExperiment.Use(middleware.JWTAuth())
	adminExperiment.Use(middleware.RequireSuperAdmin())
	{
		adminExperiment.GET("/experiment-overview", todo)                             // 全平台实验概览
		adminExperiment.GET("/container-resources", todo)                             // 全平台容器资源监控
		adminExperiment.GET("/k8s-cluster-status", todo)                              // K8s集群状态
		adminExperiment.GET("/experiment-instances", todo)                            // 全平台实验实例列表
		adminExperiment.POST("/experiment-instances/:id/force-destroy", todo)         // 强制回收任意实验环境
		adminExperiment.GET("/image-pull-status", todo)                               // 镜像预拉取状态
		adminExperiment.POST("/image-pull", todo)                                     // 触发镜像预拉取
	}

	// ========== 10. 共享实验库 ==========
	sharedTemplates := rg.Group("/shared-experiment-templates")
	sharedTemplates.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	sharedTemplates.Use(middleware.RequireTeacher())
	{
		sharedTemplates.GET("", todo)                   // 共享实验模板列表
		sharedTemplates.GET("/:id", todo)               // 共享实验模板详情
	}

	// ========== 11. 学校管理员视角 ==========
	schoolAdmin := rg.Group("/school")
	schoolAdmin.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	schoolAdmin.Use(middleware.RequireSchoolAdmin())
	{
		schoolAdmin.GET("/images", todo)                                              // 本校镜像列表
		schoolAdmin.GET("/experiment-monitor", todo)                                  // 本校实验监控
		schoolAdmin.PUT("/course-quotas/:course_id", todo)                            // 课程资源配额分配
	}
}
