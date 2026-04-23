// init_ctf.go
// 模块05 — CTF竞赛：依赖注入初始化。
// 按照 repository → service → handler 的顺序装配模块05依赖，
// 并在此处统一完成跨模块只读查询与运行时能力注入。

package main

import (
	"context"
	"sort"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	ctfhandler "github.com/lenschain/backend/internal/handler/ctf"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	ctfsvc "github.com/lenschain/backend/internal/service/ctf"
	experimentsvc "github.com/lenschain/backend/internal/service/experiment"
	notificationsvc "github.com/lenschain/backend/internal/service/notification"
)

// initCTFModule 初始化模块05（CTF竞赛）的 Handler。
func initCTFModule(notificationDispatcher notificationsvc.EventDispatcher) *router.CTFHandlers {
	db := database.Get()

	// ========== Repository 层 ==========
	competitionRepo := ctfrepo.NewCompetitionRepository(db)
	challengeRepo := ctfrepo.NewChallengeRepository(db)
	templateRepo := ctfrepo.NewChallengeTemplateRepository(db)
	contractRepo := ctfrepo.NewChallengeContractRepository(db)
	assertionRepo := ctfrepo.NewChallengeAssertionRepository(db)
	reviewRepo := ctfrepo.NewChallengeReviewRepository(db)
	verificationRepo := ctfrepo.NewChallengeVerificationRepository(db)
	compChallengeRepo := ctfrepo.NewCompetitionChallengeRepository(db)
	teamRepo := ctfrepo.NewTeamRepository(db)
	teamMemberRepo := ctfrepo.NewTeamMemberRepository(db)
	registrationRepo := ctfrepo.NewCompetitionRegistrationRepository(db)
	submissionRepo := ctfrepo.NewSubmissionRepository(db)
	leaderboardRepo := ctfrepo.NewLeaderboardSnapshotRepository(db)
	announcementRepo := ctfrepo.NewCtfAnnouncementRepository(db)
	quotaRepo := ctfrepo.NewCtfResourceQuotaRepository(db)
	environmentRepo := ctfrepo.NewChallengeEnvironmentRepository(db)
	adGroupRepo := ctfrepo.NewAdGroupRepository(db)
	adRoundRepo := ctfrepo.NewAdRoundRepository(db)
	adAttackRepo := ctfrepo.NewAdAttackRepository(db)
	adDefenseRepo := ctfrepo.NewAdDefenseRepository(db)
	adLedgerRepo := ctfrepo.NewAdTokenLedgerRepository(db)
	adChainRepo := ctfrepo.NewAdTeamChainRepository(db)

	// ========== 跨模块 Repository ==========
	userRepo := authrepo.NewUserRepository(db)
	schoolRepo := schoolrepo.NewSchoolRepository(db)

	// ========== 跨模块只读查询与运行时注入 ==========
	userQuerier := &ctfUserSummaryQuerierAdapter{userRepo: userRepo}
	schoolQuerier := &ctfSchoolNameQuerierAdapter{schoolRepo: schoolRepo}
	ctfNotificationDispatcher := newCTFNotificationDispatcher(notificationDispatcher)
	competitionAudienceResolver := &ctfCompetitionAudienceResolverAdapter{
		userRepo: userRepo,
	}
	realtimePublisher := ctfsvc.NewRealtimePublisherAdapter()

	var provisioner ctfsvc.NamespaceProvisioner
	var runtimeProvisioner ctfsvc.ADRuntimeProvisioner
	var challengeEnvProvisioner ctfsvc.ChallengeEnvironmentProvisioner
	var submissionExecutor ctfsvc.ChallengeSubmissionExecutor
	var attackExecutor ctfsvc.ADAttackExecutor
	var patchVerifier ctfsvc.ADPatchVerifier
	k8sSvc, err := experimentsvc.NewK8sService(config.Get().K8s)
	if err != nil {
		logger.L.Warn("初始化模块05 K8s 适配失败，环境与链上执行相关能力将不可用", zap.Error(err))
	} else {
		runtimeAdapter := ctfsvc.NewRuntimeProvisionerAdapter(&ctfRuntimeClusterAdapter{k8sSvc: k8sSvc})
		provisioner = runtimeAdapter
		runtimeProvisioner = runtimeAdapter
		challengeEnvProvisioner = runtimeAdapter
		submissionExecutor = runtimeAdapter
		attackExecutor = runtimeAdapter
		patchVerifier = runtimeAdapter
	}

	// ========== Service 层 ==========
	challengeService := ctfsvc.NewChallengeService(
		db,
		challengeRepo,
		templateRepo,
		contractRepo,
		assertionRepo,
		reviewRepo,
		verificationRepo,
		userQuerier,
		schoolQuerier,
		provisioner,
		challengeEnvProvisioner,
		submissionExecutor,
	)
	teamService := ctfsvc.NewTeamService(
		db,
		competitionRepo,
		challengeRepo,
		contractRepo,
		assertionRepo,
		compChallengeRepo,
		teamRepo,
		teamMemberRepo,
		registrationRepo,
		submissionRepo,
		environmentRepo,
		userQuerier,
		submissionExecutor,
		realtimePublisher,
	)
	battleService := ctfsvc.NewBattleService(
		db,
		competitionRepo,
		challengeRepo,
		assertionRepo,
		contractRepo,
		verificationRepo,
		compChallengeRepo,
		teamRepo,
		teamMemberRepo,
		adGroupRepo,
		adRoundRepo,
		adAttackRepo,
		adDefenseRepo,
		adLedgerRepo,
		adChainRepo,
		quotaRepo,
		runtimeProvisioner,
		attackExecutor,
		patchVerifier,
		realtimePublisher,
	)
	environmentService := ctfsvc.NewEnvironmentService(
		db,
		competitionRepo,
		challengeRepo,
		contractRepo,
		compChallengeRepo,
		teamRepo,
		teamMemberRepo,
		environmentRepo,
		quotaRepo,
		userQuerier,
		provisioner,
		challengeEnvProvisioner,
	)
	competitionService := ctfsvc.NewCompetitionService(
		db,
		competitionRepo,
		challengeRepo,
		contractRepo,
		assertionRepo,
		verificationRepo,
		compChallengeRepo,
		teamRepo,
		teamMemberRepo,
		registrationRepo,
		submissionRepo,
		leaderboardRepo,
		announcementRepo,
		quotaRepo,
		environmentRepo,
		adGroupRepo,
		adRoundRepo,
		adAttackRepo,
		adDefenseRepo,
		adLedgerRepo,
		environmentService,
		battleService,
		realtimePublisher,
		userQuerier,
		schoolQuerier,
		ctfNotificationDispatcher,
		competitionAudienceResolver,
	)
	ctfScheduler := ctfsvc.NewCTFScheduler(
		db,
		competitionRepo,
		compChallengeRepo,
		teamRepo,
		leaderboardRepo,
		environmentRepo,
		verificationRepo,
		adGroupRepo,
		adRoundRepo,
		adAttackRepo,
		adDefenseRepo,
		adLedgerRepo,
		adChainRepo,
		competitionService,
		battleService,
		environmentService,
		realtimePublisher,
	)
	realtimePublisher.SetCompetitionService(competitionService)

	// ========== Handler 层 ==========
	competitionHandler := ctfhandler.NewCompetitionHandler(competitionService, challengeService, teamService)
	battleHandler := ctfhandler.NewBattleHandler(battleService)
	environmentHandler := ctfhandler.NewEnvironmentHandler(environmentService)
	realtimeHandler := ctfhandler.NewRealtimeHandler(competitionService, teamService, battleService)

	// ========== 定时任务注册 ==========
	cronpkg.AddTask(cronpkg.CronCTFStatusTransition, "CTF竞赛状态流转", ctfScheduler.RunCompetitionStatusTransition)
	cronpkg.AddTask(cronpkg.CronCTFLeaderboardFreeze, "CTF排行榜冻结标记", ctfScheduler.RunLeaderboardFreeze)
	cronpkg.AddTask(cronpkg.CronCTFLeaderboardSnap, "CTF排行榜快照", ctfScheduler.RunLeaderboardSnapshot)
	cronpkg.AddTask(cronpkg.CronCTFEnvRecycle, "CTF题目环境回收", ctfScheduler.RunEnvironmentRecycle)
	cronpkg.AddTask(cronpkg.CronCTFVerificationCleanup, "CTF预验证环境清理", ctfScheduler.RunVerificationCleanup)
	cronpkg.AddTask(cronpkg.CronCTFADRoundAdvance, "CTF攻防赛回合推进", ctfScheduler.RunADRoundAdvance)
	cronpkg.AddTask(cronpkg.CronCTFTokenSync, "CTF攻防赛Token余额同步", ctfScheduler.RunTokenBalanceSync)
	cronpkg.AddTask(cronpkg.CronCTFDynamicScoreSync, "CTF题目动态分值同步", ctfScheduler.RunDynamicScoreSync)
	cronpkg.AddTask(cronpkg.CronCTFArchiveCleanup, "CTF竞赛归档清理", ctfScheduler.RunArchiveCleanup)

	return &router.CTFHandlers{
		CompetitionHandler: competitionHandler,
		BattleHandler:      battleHandler,
		EnvironmentHandler: environmentHandler,
		RealtimeHandler:    realtimeHandler,
	}
}

// ctfNotificationDispatcherAdapter 跨模块适配器：转发模块05产生的通知事件到模块07。
type ctfNotificationDispatcherAdapter struct {
	dispatcher notificationsvc.EventDispatcher
}

// newCTFNotificationDispatcher 创建模块05使用的通知事件分发器。
func newCTFNotificationDispatcher(dispatcher notificationsvc.EventDispatcher) ctfsvc.NotificationEventDispatcher {
	if dispatcher == nil {
		return nil
	}
	return &ctfNotificationDispatcherAdapter{dispatcher: dispatcher}
}

// DispatchEvent 将模块05内部事件转交给模块07统一生成站内信。
func (a *ctfNotificationDispatcherAdapter) DispatchEvent(ctx context.Context, req *dto.InternalSendNotificationEventReq) error {
	if a == nil || a.dispatcher == nil || req == nil {
		return nil
	}
	return a.dispatcher.DispatchEvent(ctx, req)
}

// ctfCompetitionAudienceResolverAdapter 跨模块适配器：解析竞赛发布时的学生受众集合。
type ctfCompetitionAudienceResolverAdapter struct {
	userRepo authrepo.UserRepository
}

// ListCompetitionPublishStudentIDs 按竞赛范围返回可接收竞赛发布通知的学生 ID 列表。
func (a *ctfCompetitionAudienceResolverAdapter) ListCompetitionPublishStudentIDs(
	ctx context.Context,
	scope int16,
	schoolID *int64,
) ([]int64, error) {
	if a == nil || a.userRepo == nil {
		return nil, nil
	}
	params := &authrepo.UserListParams{
		Role:     enum.RoleStudent,
		Status:   enum.UserStatusActive,
		Page:     1,
		PageSize: 100000,
	}
	if scope == enum.CompetitionScopeSchool && schoolID != nil {
		params.SchoolID = *schoolID
	}
	users, _, err := a.userRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	result := make([]int64, 0, len(users))
	for _, user := range users {
		if user == nil || user.ID == 0 {
			continue
		}
		result = append(result, user.ID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

// ctfUserSummaryQuerierAdapter 跨模块适配器：查询模块05所需的用户摘要。
type ctfUserSummaryQuerierAdapter struct {
	userRepo authrepo.UserRepository
}

// GetUserSummary 根据用户ID查询最小用户摘要。
func (a *ctfUserSummaryQuerierAdapter) GetUserSummary(ctx context.Context, userID int64) *ctfsvc.UserSummary {
	if userID == 0 {
		return nil
	}
	user, err := a.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil
	}
	summary := &ctfsvc.UserSummary{
		UserID: user.ID,
		Name:   user.Name,
	}
	if user.StudentNo != nil {
		summary.StudentNo = *user.StudentNo
	}
	return summary
}

// GetUserName 根据用户ID查询用户名。
func (a *ctfUserSummaryQuerierAdapter) GetUserName(ctx context.Context, userID int64) string {
	summary := a.GetUserSummary(ctx, userID)
	if summary == nil {
		return ""
	}
	return summary.Name
}

// GetUserSummaries 批量查询用户摘要。
func (a *ctfUserSummaryQuerierAdapter) GetUserSummaries(ctx context.Context, userIDs []int64) map[int64]*ctfsvc.UserSummary {
	result := make(map[int64]*ctfsvc.UserSummary, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	users, err := a.userRepo.GetByIDs(ctx, uniqueInt64s(userIDs))
	if err != nil {
		return result
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		summary := &ctfsvc.UserSummary{
			UserID: user.ID,
			Name:   user.Name,
		}
		if user.StudentNo != nil {
			summary.StudentNo = *user.StudentNo
		}
		result[user.ID] = summary
	}
	return result
}

// ctfSchoolNameQuerierAdapter 跨模块适配器：查询学校名称。
type ctfSchoolNameQuerierAdapter struct {
	schoolRepo schoolrepo.SchoolRepository
}

// GetSchoolName 根据学校ID查询学校名称。
func (a *ctfSchoolNameQuerierAdapter) GetSchoolName(ctx context.Context, schoolID int64) string {
	if schoolID == 0 {
		return ""
	}
	school, err := a.schoolRepo.GetByID(ctx, schoolID)
	if err != nil || school == nil {
		return ""
	}
	return school.Name
}

// ctfRuntimeClusterAdapter 跨模块适配器：把模块04 K8s 能力转换为模块05最小运行时契约。
type ctfRuntimeClusterAdapter struct {
	k8sSvc experimentsvc.K8sService
}

// CreateNamespace 创建模块05运行时命名空间。
func (a *ctfRuntimeClusterAdapter) CreateNamespace(ctx context.Context, name string, labels map[string]string) error {
	return a.k8sSvc.CreateNamespace(ctx, name, labels, nil)
}

// DeleteNamespace 删除模块05运行时命名空间。
func (a *ctfRuntimeClusterAdapter) DeleteNamespace(ctx context.Context, name string) error {
	return a.k8sSvc.DeleteNamespace(ctx, name)
}

// DeployPod 部署模块05运行时 Pod。
func (a *ctfRuntimeClusterAdapter) DeployPod(ctx context.Context, req *ctfsvc.RuntimeDeployPodRequest) (*ctfsvc.RuntimeDeployPodResponse, error) {
	if req == nil {
		return nil, nil
	}
	resp, err := a.k8sSvc.DeployPod(ctx, &experimentsvc.DeployPodRequest{
		Namespace:  req.Namespace,
		PodName:    req.PodName,
		Labels:     req.Labels,
		Containers: buildExperimentContainerSpecs(req.Containers),
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return &ctfsvc.RuntimeDeployPodResponse{
		PodName:    resp.PodName,
		Namespace:  resp.Namespace,
		InternalIP: resp.InternalIP,
		Status:     resp.Status,
	}, nil
}

// ExecInPod 在模块05运行时容器中执行命令。
func (a *ctfRuntimeClusterAdapter) ExecInPod(ctx context.Context, namespace, podName, container, command string) (*ctfsvc.RuntimeExecResult, error) {
	result, err := a.k8sSvc.ExecInPod(ctx, namespace, podName, container, command)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return &ctfsvc.RuntimeExecResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

// GetPodStatus 查询模块05运行时 Pod 状态。
func (a *ctfRuntimeClusterAdapter) GetPodStatus(ctx context.Context, namespace, podName string) (*ctfsvc.RuntimePodStatus, error) {
	status, err := a.k8sSvc.GetPodStatus(ctx, namespace, podName)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, nil
	}
	return &ctfsvc.RuntimePodStatus{
		PodName:    status.PodName,
		Namespace:  status.Namespace,
		NodeName:   status.NodeName,
		Status:     status.Status,
		Reason:     status.Reason,
		Message:    status.Message,
		InternalIP: status.InternalIP,
	}, nil
}

// buildExperimentContainerSpecs 将模块05运行时容器规格转换为模块04 K8s 请求模型。
func buildExperimentContainerSpecs(containers []ctfsvc.RuntimeContainerSpec) []experimentsvc.ContainerSpec {
	result := make([]experimentsvc.ContainerSpec, 0, len(containers))
	for _, item := range containers {
		result = append(result, experimentsvc.ContainerSpec{
			Name:        item.Name,
			Image:       item.Image,
			Ports:       buildExperimentPortSpecs(item.Ports),
			EnvVars:     item.EnvVars,
			CPULimit:    item.CPULimit,
			MemoryLimit: item.MemoryLimit,
			Command:     item.Command,
		})
	}
	return result
}

// buildExperimentPortSpecs 将模块05运行时端口规格转换为模块04 K8s 请求模型。
func buildExperimentPortSpecs(ports []ctfsvc.RuntimePortSpec) []experimentsvc.PortSpec {
	result := make([]experimentsvc.PortSpec, 0, len(ports))
	for _, item := range ports {
		result = append(result, experimentsvc.PortSpec{
			ContainerPort: item.ContainerPort,
			Protocol:      item.Protocol,
			ServicePort:   item.ServicePort,
		})
	}
	return result
}

// uniqueInt64s 对用户ID切片去重并排序，避免重复查询。
func uniqueInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	set := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}
