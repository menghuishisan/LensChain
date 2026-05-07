// experiment.go
// 模块04 — 实验环境路由注册
// 注册镜像管理、实验模板、实例生命周期、分组协作、监控统计、资源配额与管理员监控等路由
// WebSocket 路由在 router.go 中统一注册

package router

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/middleware"
)

// RegisterExperimentRoutes 注册实验环境模块路由。
func RegisterExperimentRoutes(rg *gin.RouterGroup, eh *ExperimentHandlers) {
	// ========== 0. 实验文件上传 ==========
	experimentFiles := rg.Group("/experiment-files")
	experimentFiles.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	experimentFiles.Use(middleware.RequireRoles(middleware.RoleTeacher, middleware.RoleSchoolAdmin, middleware.RoleStudent, middleware.RoleSuperAdmin))
	{
		experimentFiles.POST("/upload", eh.InstanceHandler.UploadExperimentFile)
	}

	// ========== 1. 镜像管理 ==========
	images := rg.Group("/images")
	images.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		images.GET("", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.ListImages)
		images.POST("", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.CreateImage)
		images.GET("/:id", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.GetImage)
		images.PUT("/:id", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.UpdateImage)
		images.DELETE("/:id", middleware.RequireSuperAdmin(), eh.TemplateHandler.DeleteImage)
		images.POST("/:id/review", middleware.RequireSuperAdmin(), eh.TemplateHandler.ReviewImage)
		images.GET("/:id/versions", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.ListImageVersions)
		images.POST("/:id/versions", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.CreateImageVersion)
		images.GET("/:id/config-template", middleware.RequireTeacher(), eh.TemplateHandler.GetImageConfigTemplate)
		images.GET("/:id/documentation", middleware.RequireTeacher(), eh.TemplateHandler.GetImageDocumentation)
	}

	imageCategories := rg.Group("/image-categories")
	imageCategories.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		imageCategories.GET("", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.ListImageCategories)
	}

	imageVersions := rg.Group("/image-versions")
	imageVersions.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher))
	{
		imageVersions.PUT("/:id", eh.TemplateHandler.UpdateImageVersion)
		imageVersions.DELETE("/:id", middleware.RequireSuperAdmin(), eh.TemplateHandler.DeleteImageVersion)
		imageVersions.PATCH("/:id/default", eh.TemplateHandler.SetDefaultImageVersion)
	}

	// ========== 2. 实验模板 ==========
	templates := rg.Group("/experiment-templates")
	templates.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		templates.POST("", middleware.RequireTeacher(), eh.TemplateHandler.CreateTemplate)
		templates.GET("", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplates)
		templates.GET("/:id", middleware.RequireTeacher(), eh.TemplateHandler.GetTemplate)
		templates.PUT("/:id", middleware.RequireTeacher(), eh.TemplateHandler.UpdateTemplate)
		templates.DELETE("/:id", middleware.RequireTeacher(), eh.TemplateHandler.DeleteTemplate)
		templates.POST("/:id/publish", middleware.RequireTeacher(), eh.TemplateHandler.PublishTemplate)
		templates.POST("/:id/clone", middleware.RequireTeacher(), eh.TemplateHandler.CloneTemplate)
		templates.PATCH("/:id/share", middleware.RequireTeacher(), eh.TemplateHandler.ShareTemplate)
		templates.GET("/:id/k8s-config", middleware.RequireTeacher(), eh.TemplateHandler.GetTemplateK8sConfig)
		templates.POST("/:id/k8s-config", middleware.RequireTeacher(), eh.TemplateHandler.SetTemplateK8sConfig)
		templates.POST("/:id/validate", middleware.RequireTeacher(), eh.TemplateHandler.ValidateTemplate)
		templates.GET("/:id/containers", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplateContainers)
		templates.POST("/:id/containers", middleware.RequireTeacher(), eh.TemplateHandler.CreateTemplateContainer)
		templates.PUT("/:id/containers/sort", middleware.RequireTeacher(), eh.TemplateHandler.SortTemplateContainers)
		templates.GET("/:id/checkpoints", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplateCheckpoints)
		templates.POST("/:id/checkpoints", middleware.RequireTeacher(), eh.TemplateHandler.CreateTemplateCheckpoint)
		templates.PUT("/:id/checkpoints/sort", middleware.RequireTeacher(), eh.TemplateHandler.SortTemplateCheckpoints)
		templates.GET("/:id/init-scripts", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplateInitScripts)
		templates.POST("/:id/init-scripts", middleware.RequireTeacher(), eh.TemplateHandler.CreateTemplateInitScript)
		templates.GET("/:id/sim-scenes", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplateSimScenes)
		templates.POST("/:id/sim-scenes", middleware.RequireTeacher(), eh.TemplateHandler.CreateTemplateSimScene)
		templates.PUT("/:id/sim-scenes/layout", middleware.RequireTeacher(), eh.TemplateHandler.UpdateTemplateSimSceneLayout)
		templates.PUT("/:id/tags", middleware.RequireTeacher(), eh.TemplateHandler.SetTemplateTags)
		templates.GET("/:id/tags", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplateTags)
		templates.GET("/:id/roles", middleware.RequireTeacher(), eh.TemplateHandler.ListTemplateRoles)
		templates.POST("/:id/roles", middleware.RequireTeacher(), eh.TemplateHandler.CreateTemplateRole)
	}

	templateContainers := rg.Group("/template-containers")
	templateContainers.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		templateContainers.PUT("/:id", eh.TemplateHandler.UpdateTemplateContainer)
		templateContainers.DELETE("/:id", eh.TemplateHandler.DeleteTemplateContainer)
	}

	templateCheckpoints := rg.Group("/template-checkpoints")
	templateCheckpoints.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		templateCheckpoints.PUT("/:id", eh.TemplateHandler.UpdateTemplateCheckpoint)
		templateCheckpoints.DELETE("/:id", eh.TemplateHandler.DeleteTemplateCheckpoint)
	}

	templateInitScripts := rg.Group("/template-init-scripts")
	templateInitScripts.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		templateInitScripts.PUT("/:id", eh.TemplateHandler.UpdateTemplateInitScript)
		templateInitScripts.DELETE("/:id", eh.TemplateHandler.DeleteTemplateInitScript)
	}

	templateSimScenes := rg.Group("/template-sim-scenes")
	templateSimScenes.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		templateSimScenes.PUT("/:id", eh.TemplateHandler.UpdateTemplateSimScene)
		templateSimScenes.DELETE("/:id", eh.TemplateHandler.DeleteTemplateSimScene)
	}

	templateRoles := rg.Group("/template-roles")
	templateRoles.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		templateRoles.PUT("/:id", eh.TemplateHandler.UpdateTemplateRole)
		templateRoles.DELETE("/:id", eh.TemplateHandler.DeleteTemplateRole)
	}

	// ========== 3. 仿真场景库与标签 ==========
	simScenarios := rg.Group("/sim-scenarios")
	simScenarios.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		simScenarios.GET("", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.ListScenarios)
		simScenarios.POST("", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.CreateScenario)
		simScenarios.GET("/:id", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.GetScenario)
		simScenarios.PUT("/:id", middleware.RequireRoles(middleware.RoleSuperAdmin, middleware.RoleTeacher), eh.TemplateHandler.UpdateScenario)
		simScenarios.DELETE("/:id", middleware.RequireSuperAdmin(), eh.TemplateHandler.DeleteScenario)
		simScenarios.POST("/:id/review", middleware.RequireSuperAdmin(), eh.TemplateHandler.ReviewScenario)
	}

	simLinkGroups := rg.Group("/sim-link-groups")
	simLinkGroups.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		simLinkGroups.GET("", eh.TemplateHandler.ListLinkGroups)
		simLinkGroups.GET("/:id", eh.TemplateHandler.GetLinkGroup)
	}

	tags := rg.Group("/tags")
	tags.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		tags.GET("", eh.TemplateHandler.ListTags)
		tags.POST("", eh.TemplateHandler.CreateTag)
	}

	sharedTemplates := rg.Group("/shared-experiment-templates")
	sharedTemplates.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		sharedTemplates.GET("", eh.TemplateHandler.ListSharedTemplates)
		sharedTemplates.GET("/:id", eh.TemplateHandler.GetSharedTemplate)
	}

	// ========== 3.5 学生端模板只读 ==========
	studentTemplates := rg.Group("/student/experiment-templates")
	studentTemplates.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireStudent())
	{
		studentTemplates.GET("", eh.TemplateHandler.StudentListTemplates)
		studentTemplates.GET("/:id", eh.TemplateHandler.StudentGetTemplate)
	}

	// ========== 4. 实验实例 ==========
	instances := rg.Group("/experiment-instances")
	instances.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		instances.POST("", middleware.RequireStudent(), eh.InstanceHandler.CreateInstance)
		instances.GET("", middleware.RequireStudent(), eh.InstanceHandler.ListInstances)
		instances.GET("/:id", eh.InstanceHandler.GetInstance)
		instances.POST("/:id/pause", eh.InstanceHandler.PauseInstance)
		instances.POST("/:id/resume", eh.InstanceHandler.ResumeInstance)
		instances.POST("/:id/restart", eh.InstanceHandler.RestartInstance)
		instances.POST("/:id/submit", eh.InstanceHandler.SubmitInstance)
		instances.POST("/:id/destroy", eh.InstanceHandler.DestroyInstance)
		instances.GET("/:id/terminal", middleware.RequireStudent(), eh.InstanceHandler.ServeStudentTerminalWS)
		instances.POST("/:id/heartbeat", eh.InstanceHandler.Heartbeat)
		instances.POST("/:id/checkpoints/verify", eh.InstanceHandler.VerifyCheckpoints)
		instances.GET("/:id/checkpoints", eh.InstanceHandler.ListCheckpointResults)
		instances.GET("/:id/snapshots", eh.InstanceHandler.ListSnapshots)
		instances.POST("/:id/snapshots", eh.InstanceHandler.CreateSnapshot)
		instances.POST("/:id/snapshots/:snapshot_id/restore", eh.InstanceHandler.RestoreSnapshot)
		instances.DELETE("/:id/snapshots/:snapshot_id", eh.InstanceHandler.DeleteSnapshot)
		instances.GET("/:id/operation-logs", eh.InstanceHandler.ListOperationLogs)
		instances.POST("/:id/report", eh.InstanceHandler.CreateReport)
		instances.GET("/:id/report", eh.InstanceHandler.GetReport)
		instances.PUT("/:id/report", eh.InstanceHandler.UpdateReport)
		instances.GET("/:id/terminal-stream", middleware.RequireTeacher(), eh.InstanceHandler.ServeTerminalStreamWS)
		instances.POST("/:id/message", middleware.RequireTeacher(), eh.InstanceHandler.SendGuidance)
		instances.POST("/:id/force-destroy", middleware.RequireTeacher(), eh.InstanceHandler.ForceDestroyInstance)
		instances.POST("/:id/manual-grade", middleware.RequireTeacher(), eh.InstanceHandler.ManualGradeInstance)
	}

	checkpointResults := rg.Group("/checkpoint-results")
	checkpointResults.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		checkpointResults.POST("/:id/grade", eh.InstanceHandler.GradeCheckpoint)
	}

	// ========== 5. 分组协作 ==========
	groups := rg.Group("/experiment-groups")
	groups.Use(middleware.JWTAuth(), middleware.TenantIsolation())
	{
		groups.POST("", middleware.RequireTeacher(), eh.InstanceHandler.CreateGroup)
		groups.GET("", eh.InstanceHandler.ListGroups)
		groups.GET("/:id", eh.InstanceHandler.GetGroup)
		groups.PUT("/:id", middleware.RequireTeacher(), eh.InstanceHandler.UpdateGroup)
		groups.DELETE("/:id", middleware.RequireTeacher(), eh.InstanceHandler.DeleteGroup)
		groups.POST("/auto-assign", middleware.RequireTeacher(), eh.InstanceHandler.AutoAssignGroups)
		groups.POST("/:id/join", middleware.RequireStudent(), eh.InstanceHandler.JoinGroup)
		groups.DELETE("/:id/members/:student_id", middleware.RequireTeacher(), eh.InstanceHandler.RemoveGroupMember)
		groups.GET("/:id/members/:student_id/terminal-stream", middleware.RequireStudent(), eh.InstanceHandler.ServeGroupMemberTerminalStreamWS)
		groups.GET("/:id/members", eh.InstanceHandler.ListGroupMembers)
		groups.GET("/:id/progress", eh.InstanceHandler.GetGroupProgress)
		groups.POST("/:id/messages", eh.InstanceHandler.SendGroupMessage)
		groups.GET("/:id/messages", eh.InstanceHandler.ListGroupMessages)
	}

	// ========== 6. 教师监控 ==========
	courseMonitor := rg.Group("/courses")
	courseMonitor.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireTeacher())
	{
		courseMonitor.GET("/:id/experiment-monitor", eh.InstanceHandler.GetCourseMonitor)
		courseMonitor.GET("/:id/experiment-statistics", eh.InstanceHandler.GetCourseStatistics)
	}

	// ========== 7. 配额与学校管理员视角 ==========
	quotas := rg.Group("/resource-quotas")
	quotas.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireSchoolAdminOrSuperAdmin())
	{
		quotas.GET("", eh.InstanceHandler.ListQuotas)
		quotas.POST("", middleware.RequireSuperAdmin(), eh.InstanceHandler.CreateQuota)
		quotas.PUT("/:id", eh.InstanceHandler.UpdateQuota)
		quotas.GET("/:id", eh.InstanceHandler.GetQuota)
	}

	schoolResource := rg.Group("/schools")
	schoolResource.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireSchoolAdminOrSuperAdmin())
	{
		schoolResource.GET("/:id/resource-usage", eh.InstanceHandler.GetSchoolUsage)
	}

	schoolAdmin := rg.Group("/school")
	schoolAdmin.Use(middleware.JWTAuth(), middleware.TenantIsolation(), middleware.RequireSchoolAdmin())
	{
		schoolAdmin.GET("/images", eh.TemplateHandler.ListSchoolImages)
		schoolAdmin.GET("/experiment-monitor", eh.InstanceHandler.GetSchoolMonitor)
		schoolAdmin.PUT("/course-quotas/:course_id", eh.InstanceHandler.AssignCourseQuota)
	}

	// ========== 8. 超管监控 ==========
	adminExperiment := rg.Group("/admin")
	adminExperiment.Use(middleware.JWTAuth(), middleware.RequireSuperAdmin())
	{
		adminExperiment.GET("/experiment-overview", eh.InstanceHandler.GetExperimentOverview)
		adminExperiment.GET("/container-resources", eh.InstanceHandler.GetContainerResources)
		adminExperiment.GET("/k8s-cluster-status", eh.InstanceHandler.GetK8sClusterStatus)
		adminExperiment.GET("/experiment-instances", eh.InstanceHandler.ListAdminInstances)
		adminExperiment.POST("/experiment-instances/:id/force-destroy", eh.InstanceHandler.ForceDestroyAdminInstance)
		adminExperiment.GET("/image-pull-status", eh.InstanceHandler.GetImagePullStatus)
		adminExperiment.POST("/image-pull", eh.InstanceHandler.TriggerImagePull)
	}
}
