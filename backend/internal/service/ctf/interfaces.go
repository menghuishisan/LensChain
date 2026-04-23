// interfaces.go
// 模块05 — CTF竞赛：service 层接口定义。
// 按竞赛、题目、团队、攻防赛和环境五个功能域拆分接口，确保 handler 只依赖抽象能力，
// 同时让 init 装配层可以按功能域完成依赖注入。

package ctf

import (
	"context"

	svcctx "github.com/lenschain/backend/internal/pkg/context"

	"github.com/lenschain/backend/internal/model/dto"
)

// CompetitionService 竞赛主流程服务接口。
type CompetitionService interface {
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateCompetitionReq) (*dto.CompetitionCreateResp, error)
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CompetitionListReq) (*dto.CompetitionListResp, error)
	Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionDetailResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateCompetitionReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionStatusResp, error)
	Archive(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionStatusResp, error)
	Terminate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.TerminateCompetitionReq) (*dto.TerminateCompetitionResp, error)

	AddChallenges(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.AddCompetitionChallengeReq) (*dto.AddCompetitionChallengeResp, error)
	ListChallenges(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionChallengeListResp, error)
	SortChallenges(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SortCompetitionChallengesReq) error
	RemoveChallenge(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	GetLeaderboard(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.LeaderboardReq) (*dto.LeaderboardResp, error)
	GetFinalLeaderboard(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.FinalLeaderboardResp, error)
	GetLeaderboardHistory(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.LeaderboardHistoryReq) (*dto.LeaderboardHistoryResp, error)

	CreateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CreateCtfAnnouncementReq) (*dto.CtfAnnouncementResp, error)
	ListAnnouncements(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CtfAnnouncementListResp, error)
	GetAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CtfAnnouncementResp, error)
	DeleteAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	GetResourceQuota(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.ResourceQuotaResp, error)
	UpdateResourceQuota(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.UpdateResourceQuotaReq) (*dto.ResourceQuotaResp, error)

	GetMonitor(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionMonitorResp, error)
	GetStatistics(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionStatisticsResp, error)
	GetResults(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionResultsResp, error)
	GetAdminOverview(ctx context.Context, sc *svcctx.ServiceContext) (*dto.CtfAdminOverviewResp, error)
}

// ChallengeService 题目与模板服务接口。
type ChallengeService interface {
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateChallengeReq) (*dto.ChallengeStatusResp, error)
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ChallengeListReq) (*dto.ChallengeListResp, error)
	Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ChallengeDetailResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChallengeReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	SubmitReview(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmitChallengeReviewResp, error)

	CreateContract(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.CreateChallengeContractReq) (*dto.ChallengeContractResp, error)
	UpdateContract(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChallengeContractReq) error
	DeleteContract(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	ListContracts(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.ChallengeContractListResp, error)

	CreateAssertion(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.CreateChallengeAssertionReq) (*dto.ChallengeAssertionResp, error)
	UpdateAssertion(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateChallengeAssertionReq) error
	DeleteAssertion(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	ListAssertions(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.ChallengeAssertionListResp, error)
	SortAssertions(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.SortChallengeAssertionReq) error

	ListSWCRegistry(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SWCRegistryListReq) ([]dto.SWCRegistryItem, error)
	ImportSWC(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImportSWCChallengeReq) (*dto.ChallengeStatusResp, error)
	ListTemplates(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ChallengeTemplateListReq) (*dto.ChallengeTemplateListResp, error)
	GetTemplate(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ChallengeTemplateDetailResp, error)
	GenerateFromTemplate(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GenerateChallengeFromTemplateReq) (*dto.ChallengeStatusResp, error)
	ImportExternalVulnerability(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ImportExternalVulnerabilityReq) (*dto.ChallengeStatusResp, error)

	StartVerification(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.VerifyChallengeReq) (*dto.VerifyChallengeResp, error)
	GetVerification(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.VerificationDetailResp, error)
	ListVerifications(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.VerificationListResp, error)

	ListPendingReviews(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ChallengeListReq) (*dto.ChallengeReviewListResp, error)
	Review(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64, req *dto.ReviewChallengeReq) (*dto.ReviewChallengeActionResp, error)
	ListReviews(ctx context.Context, sc *svcctx.ServiceContext, challengeID int64) (*dto.ChallengeReviewHistoryResp, error)
}

// TeamService 团队、报名和提交服务接口。
type TeamService interface {
	CreateTeam(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CreateTeamReq) (*dto.TeamResp, error)
	GetTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.TeamResp, error)
	UpdateTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateTeamReq) error
	DisbandTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	JoinTeam(ctx context.Context, sc *svcctx.ServiceContext, req *dto.JoinTeamReq) (*dto.JoinTeamResp, error)
	LeaveTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	RemoveMember(ctx context.Context, sc *svcctx.ServiceContext, teamID, studentID int64) error
	ListTeams(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.TeamListResp, error)

	RegisterCompetition(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.RegisterCompetitionReq) (*dto.RegistrationResp, error)
	CancelRegistration(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) error
	ListRegistrations(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.RegistrationListReq) (*dto.RegistrationListResp, error)
	GetMyRegistration(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.MyRegistrationResp, error)

	SubmitChallenge(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.SubmitCompetitionChallengeReq) (*dto.CompetitionSubmissionResp, error)
	ListSubmissions(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CompetitionSubmissionListReq) (*dto.CompetitionSubmissionListResp, error)
	GetSubmissionStatistics(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionSubmissionStatisticsResp, error)
}

// BattleService 攻防赛服务接口。
type BattleService interface {
	CreateAdGroup(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CreateAdGroupReq) (*dto.AdGroupResp, error)
	ListAdGroups(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.AdGroupListResp, error)
	GetAdGroup(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AdGroupResp, error)
	AutoAssignAdGroups(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.AutoAssignAdGroupsReq) (*dto.AutoAssignAdGroupsResp, error)

	ListRounds(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.AdRoundListResp, error)
	GetRound(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AdRoundDetailResp, error)
	GetCurrentRound(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.CurrentRoundResp, error)

	SubmitAttack(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.SubmitAdAttackReq) (*dto.AdAttackResp, error)
	ListRoundAttacks(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.AdAttackListReq) (*dto.AdAttackListResp, error)
	ListGroupAttacks(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.AdAttackListReq) (*dto.AdAttackListResp, error)

	SubmitDefense(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.SubmitAdDefenseReq) (*dto.AdDefenseResp, error)
	ListRoundDefenses(ctx context.Context, sc *svcctx.ServiceContext, roundID int64, req *dto.AdDefenseListReq) (*dto.AdDefenseListResp, error)

	ListTokenLedgerByCompetition(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.TokenLedgerListReq) (*dto.TokenLedgerListResp, error)
	ListTokenLedgerByTeam(ctx context.Context, sc *svcctx.ServiceContext, teamID int64, req *dto.TokenLedgerListReq) (*dto.TokenLedgerListResp, error)
	GetTeamChain(ctx context.Context, sc *svcctx.ServiceContext, teamID int64) (*dto.TeamChainResp, error)
	ListGroupChains(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.TeamChainListResp, error)
}

// EnvironmentService 题目环境服务接口。
type EnvironmentService interface {
	Start(ctx context.Context, sc *svcctx.ServiceContext, competitionID, challengeID int64) (*dto.StartChallengeEnvironmentResp, error)
	Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ChallengeEnvironmentResp, error)
	Reset(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ResetChallengeEnvironmentResp, error)
	Destroy(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	ForceDestroy(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ForceDestroyChallengeEnvironmentReq) (*dto.ForceDestroyChallengeEnvironmentResp, error)
	ListMyEnvironments(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.MyChallengeEnvironmentListResp, error)
	ListCompetitionEnvironments(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CompetitionEnvironmentListReq) (*dto.CompetitionEnvironmentListResp, error)
}
