// ctf.ts
// 模块05 CTF竞赛 service：竞赛、题目、漏洞转化、团队、环境、排行榜、公告和攻防赛接口。

import { apiClient } from "@/lib/api-client";
import type { ID, PaginatedData, QueryParams } from "@/types/api";
import type {
  AdAttackListItem,
  AdAttackResponse,
  AdDefenseListItem,
  AdDefenseResponse,
  AutoAssignCtfAdGroupsRequest,
  AutoAssignCtfAdGroupsResponse,
  CreateCtfAdGroupRequest,
  CreateCtfChallengeRequest,
  CreateCtfCompetitionRequest,
  CtfAdGroup,
  CtfAnnouncement,
  CtfChallengeAssertion,
  CtfChallengeContract,
  CtfChallengeDetail,
  CtfChallengeListParams,
  CtfChallengeListResponse,
  CtfChallengeReviewListItem,
  CtfChallengeStatusResponse,
  CtfChallengeTemplateDetail,
  CtfChallengeTemplateListItem,
  CtfCompetitionChallengeListItem,
  CtfCompetitionCreateResponse,
  CtfCompetitionDetail,
  CtfCompetitionListParams,
  CtfCompetitionListResponse,
  CtfCompetitionMonitor,
  CtfCompetitionResults,
  CtfCompetitionStatistics,
  CtfCurrentRound,
  CtfEnvironmentListParams,
  CtfChallengeEnvironment,
  CtfTeamChain,
  CtfJsonObject,
  CtfLeaderboard,
  CtfLeaderboardHistorySnapshot,
  CtfMyRegistration,
  CtfRegistration,
  CtfResourceQuota,
  CtfSubmissionListItem,
  CtfSubmissionResponse,
  CtfTeam,
  CtfTeamListItem,
  CtfTokenLedgerItem,
  CtfVerificationDetail,
  CtfVerificationListItem,
  ForceDestroyCtfEnvironmentResponse,
  GenerateChallengeFromTemplateRequest,
  ImportSwcChallengeRequest,
  ImportExternalVulnerabilityRequest,
  ReviewCtfChallengeRequest,
  ReviewCtfChallengeResponse,
  StartCtfEnvironmentResponse,
  SubmitAdAttackRequest,
  SubmitAdDefenseRequest,
  SubmitCtfChallengeRequest,
  SwcRegistryItem,
  UpdateCtfChallengeRequest,
  UpdateCtfCompetitionRequest,
  VerifyCtfChallengeRequest,
  VerifyCtfChallengeResponse,
} from "@/types/ctf";

/**
 * createCtfCompetition 对应 POST /api/v1/ctf/competitions，用于创建 CTF 竞赛。
 */
export function createCtfCompetition(payload: CreateCtfCompetitionRequest) {
  return apiClient.post<CtfCompetitionCreateResponse>("/ctf/competitions", payload);
}

/**
 * listCtfCompetitions 对应 GET /api/v1/ctf/competitions，用于竞赛大厅和管理列表。
 */
export function listCtfCompetitions(params: CtfCompetitionListParams) {
  return apiClient.get<CtfCompetitionListResponse>("/ctf/competitions", { query: params });
}

/**
 * getCtfCompetition 对应 GET /api/v1/ctf/competitions/:id，用于竞赛详情。
 */
export function getCtfCompetition(competitionID: ID) {
  return apiClient.get<CtfCompetitionDetail>(`/ctf/competitions/${competitionID}`);
}

/**
 * updateCtfCompetition 对应 PUT /api/v1/ctf/competitions/:id，用于编辑竞赛。
 */
export function updateCtfCompetition(competitionID: ID, payload: UpdateCtfCompetitionRequest) {
  return apiClient.put<null>(`/ctf/competitions/${competitionID}`, payload);
}

/**
 * deleteCtfCompetition 对应 DELETE /api/v1/ctf/competitions/:id，用于删除草稿竞赛。
 */
export function deleteCtfCompetition(competitionID: ID) {
  return apiClient.delete<null>(`/ctf/competitions/${competitionID}`);
}

/**
 * publishCtfCompetition 对应 POST /api/v1/ctf/competitions/:id/publish，用于发布竞赛。
 */
export function publishCtfCompetition(competitionID: ID) {
  return apiClient.post<{ id: ID; status: number; status_text: string }>(`/ctf/competitions/${competitionID}/publish`);
}

/**
 * archiveCtfCompetition 对应 POST /api/v1/ctf/competitions/:id/archive，用于归档竞赛。
 */
export function archiveCtfCompetition(competitionID: ID) {
  return apiClient.post<{ id: ID; status: number; status_text: string }>(`/ctf/competitions/${competitionID}/archive`);
}

/**
 * terminateCtfCompetition 对应 POST /api/v1/ctf/competitions/:id/terminate，用于强制终止竞赛。
 */
export function terminateCtfCompetition(competitionID: ID, reason: string) {
  return apiClient.post<{ id: ID; status: number; status_text: string; environments_destroyed: number }>(`/ctf/competitions/${competitionID}/terminate`, { reason });
}

/**
 * createCtfChallenge 对应 POST /api/v1/ctf/challenges，用于教师创建题目。
 */
export function createCtfChallenge(payload: CreateCtfChallengeRequest) {
  return apiClient.post<CtfChallengeStatusResponse>("/ctf/challenges", payload);
}

/**
 * listCtfChallenges 对应 GET /api/v1/ctf/challenges，用于题库列表。
 */
export function listCtfChallenges(params: CtfChallengeListParams) {
  return apiClient.get<CtfChallengeListResponse>("/ctf/challenges", { query: params });
}

/**
 * getCtfChallenge 对应 GET /api/v1/ctf/challenges/:id，用于题目详情。
 */
export function getCtfChallenge(challengeID: ID) {
  return apiClient.get<CtfChallengeDetail>(`/ctf/challenges/${challengeID}`);
}

/**
 * updateCtfChallenge 对应 PUT /api/v1/ctf/challenges/:id，用于编辑题目。
 */
export function updateCtfChallenge(challengeID: ID, payload: UpdateCtfChallengeRequest) {
  return apiClient.put<null>(`/ctf/challenges/${challengeID}`, payload);
}

/**
 * deleteCtfChallenge 对应 DELETE /api/v1/ctf/challenges/:id，用于删除题目。
 */
export function deleteCtfChallenge(challengeID: ID) {
  return apiClient.delete<null>(`/ctf/challenges/${challengeID}`);
}

/**
 * submitCtfChallengeReview 对应 POST /api/v1/ctf/challenges/:id/submit-review，用于提交审核。
 */
export function submitCtfChallengeReview(challengeID: ID) {
  return apiClient.post<{ id: ID; status: number; status_text: string }>(`/ctf/challenges/${challengeID}/submit-review`);
}

/**
 * createChallengeContract 对应 POST /api/v1/ctf/challenges/:id/contracts，用于添加题目合约。
 */
export function createChallengeContract(challengeID: ID, payload: { name: string; source_code: string; abi: CtfJsonObject[]; bytecode: string; constructor_args: unknown[]; deploy_order: number }) {
  return apiClient.post<{ id: ID; challenge_id: ID; name: string; deploy_order: number }>(`/ctf/challenges/${challengeID}/contracts`, payload);
}

/**
 * listChallengeContracts 对应 GET /api/v1/ctf/challenges/:id/contracts，用于题目合约列表。
 */
export function listChallengeContracts(challengeID: ID) {
  return apiClient.get<{ list: CtfChallengeContract[] }>(`/ctf/challenges/${challengeID}/contracts`);
}

/**
 * createChallengeAssertion 对应 POST /api/v1/ctf/challenges/:id/assertions，用于添加验证断言。
 */
export function createChallengeAssertion(challengeID: ID, payload: { assertion_type: string; target: string; operator: string; expected_value: string; description?: string | null; extra_params?: CtfJsonObject | null; sort_order?: number }) {
  return apiClient.post<{ id: ID; challenge_id: ID; assertion_type: string; target: string; operator: string; expected_value: string; sort_order: number }>(`/ctf/challenges/${challengeID}/assertions`, payload);
}

/**
 * listChallengeAssertions 对应 GET /api/v1/ctf/challenges/:id/assertions，用于题目断言列表。
 */
export function listChallengeAssertions(challengeID: ID) {
  return apiClient.get<{ list: CtfChallengeAssertion[] }>(`/ctf/challenges/${challengeID}/assertions`);
}

/**
 * listSwcRegistry 对应 GET /api/v1/ctf/swc-registry，用于 SWC Registry 漏洞源列表。
 */
export function listSwcRegistry(params: QueryParams = {}) {
  return apiClient.get<{ list: SwcRegistryItem[] }>("/ctf/swc-registry", { query: params });
}

/**
 * importSwcChallenge 对应 POST /api/v1/ctf/challenges/import-swc，用于从 SWC 生成题目草稿。
 */
export function importSwcChallenge(payload: ImportSwcChallengeRequest) {
  return apiClient.post<CtfChallengeStatusResponse>("/ctf/challenges/import-swc", payload);
}

/**
 * listCtfChallengeTemplates 对应 GET /api/v1/ctf/challenge-templates，用于参数化模板库。
 */
export function listCtfChallengeTemplates(params: QueryParams = {}) {
  return apiClient.get<{ list: CtfChallengeTemplateListItem[] }>("/ctf/challenge-templates", { query: params });
}

/**
 * getCtfChallengeTemplate 对应 GET /api/v1/ctf/challenge-templates/:id，用于模板详情。
 */
export function getCtfChallengeTemplate(templateID: ID) {
  return apiClient.get<CtfChallengeTemplateDetail>(`/ctf/challenge-templates/${templateID}`);
}

/**
 * generateChallengeFromTemplate 对应 POST /api/v1/ctf/challenges/generate-from-template，用于从模板生成题目。
 */
export function generateChallengeFromTemplate(payload: GenerateChallengeFromTemplateRequest) {
  return apiClient.post<CtfChallengeStatusResponse>("/ctf/challenges/generate-from-template", payload);
}

/**
 * importExternalVulnerability 对应 POST /api/v1/ctf/challenges/import-external，用于外部真实漏洞源导入。
 */
export function importExternalVulnerability(payload: ImportExternalVulnerabilityRequest) {
  return apiClient.post<CtfChallengeStatusResponse>("/ctf/challenges/import-external", payload);
}

/**
 * verifyCtfChallenge 对应 POST /api/v1/ctf/challenges/:id/verify，用于发起六步预验证。
 */
export function verifyCtfChallenge(challengeID: ID, payload: VerifyCtfChallengeRequest) {
  return apiClient.post<VerifyCtfChallengeResponse>(`/ctf/challenges/${challengeID}/verify`, payload);
}

/**
 * listCtfVerifications 对应 GET /api/v1/ctf/challenges/:id/verifications，用于预验证记录列表。
 */
export function listCtfVerifications(challengeID: ID) {
  return apiClient.get<{ list: CtfVerificationListItem[] }>(`/ctf/challenges/${challengeID}/verifications`);
}

/**
 * getCtfVerification 对应 GET /api/v1/ctf/challenge-verifications/:id，用于预验证详情。
 */
export function getCtfVerification(verificationID: ID) {
  return apiClient.get<CtfVerificationDetail>(`/ctf/challenge-verifications/${verificationID}`);
}

/**
 * listPendingCtfChallengeReviews 对应 GET /api/v1/ctf/challenge-reviews/pending，用于待审核题目列表。
 */
export function listPendingCtfChallengeReviews() {
  return apiClient.get<{ list: CtfChallengeReviewListItem[] }>("/ctf/challenge-reviews/pending");
}

/**
 * reviewCtfChallenge 对应 POST /api/v1/ctf/challenges/:id/review，用于超级管理员审核题目。
 */
export function reviewCtfChallenge(challengeID: ID, payload: ReviewCtfChallengeRequest) {
  return apiClient.post<ReviewCtfChallengeResponse>(`/ctf/challenges/${challengeID}/review`, payload);
}

/**
 * listCompetitionChallenges 对应 GET /api/v1/ctf/competitions/:id/challenges，用于竞赛题目列表。
 */
export function listCompetitionChallenges(competitionID: ID) {
  return apiClient.get<{ list: CtfCompetitionChallengeListItem[] }>(`/ctf/competitions/${competitionID}/challenges`);
}

/**
 * addCompetitionChallenges 对应 POST /api/v1/ctf/competitions/:id/challenges，用于添加题目到竞赛。
 */
export function addCompetitionChallenges(competitionID: ID, challengeIDs: ID[]) {
  return apiClient.post<{ added_count: number; competition_id: ID }>(`/ctf/competitions/${competitionID}/challenges`, { challenge_ids: challengeIDs });
}

/**
 * removeCompetitionChallenge 对应 DELETE /api/v1/ctf/competition-challenges/:id，用于移除竞赛题目。
 */
export function removeCompetitionChallenge(competitionChallengeID: ID) {
  return apiClient.delete<null>(`/ctf/competition-challenges/${competitionChallengeID}`);
}

/**
 * sortCompetitionChallenges 对应 PUT /api/v1/ctf/competitions/:id/challenges/sort，用于竞赛题目排序。
 */
export function sortCompetitionChallenges(competitionID: ID, items: Array<{ id: ID; sort_order: number }>) {
  return apiClient.put<null>(`/ctf/competitions/${competitionID}/challenges/sort`, { items });
}

/**
 * createCtfTeam 对应 POST /api/v1/ctf/competitions/:id/teams，用于创建队伍。
 */
export function createCtfTeam(competitionID: ID, payload: { name: string }) {
  return apiClient.post<CtfTeam>(`/ctf/competitions/${competitionID}/teams`, payload);
}

/**
 * listCtfTeams 对应 GET /api/v1/ctf/competitions/:id/teams，用于竞赛团队列表。
 */
export function listCtfTeams(competitionID: ID) {
  return apiClient.get<{ list: CtfTeamListItem[] }>(`/ctf/competitions/${competitionID}/teams`);
}

/**
 * getCtfTeam 对应 GET /api/v1/ctf/teams/:id，用于团队详情。
 */
export function getCtfTeam(teamID: ID) {
  return apiClient.get<CtfTeam>(`/ctf/teams/${teamID}`);
}

/**
 * updateCtfTeam 对应 PUT /api/v1/ctf/teams/:id，用于编辑团队名称。
 */
export function updateCtfTeam(teamID: ID, name: string) {
  return apiClient.put<null>(`/ctf/teams/${teamID}`, { name });
}

/**
 * disbandCtfTeam 对应 POST /api/v1/ctf/teams/:id/disband，用于队长解散团队。
 */
export function disbandCtfTeam(teamID: ID) {
  return apiClient.post<null>(`/ctf/teams/${teamID}/disband`);
}

/**
 * leaveCtfTeam 对应 POST /api/v1/ctf/teams/:id/leave，用于队员退出团队。
 */
export function leaveCtfTeam(teamID: ID) {
  return apiClient.post<null>(`/ctf/teams/${teamID}/leave`);
}

/**
 * getCtfTeamTokenLedger 对应 GET /api/v1/ctf/teams/:id/token-ledger，用于团队 Token 流水。
 */
export function getCtfTeamTokenLedger(teamID: ID, params: QueryParams = {}) {
  return apiClient.get<PaginatedData<CtfTokenLedgerItem>>(`/ctf/teams/${teamID}/token-ledger`, { query: params });
}

/**
 * getCtfTeamChain 对应 GET /api/v1/ctf/teams/:id/chain，用于攻防赛队伍链信息。
 */
export function getCtfTeamChain(teamID: ID) {
  return apiClient.get<CtfTeamChain>(`/ctf/teams/${teamID}/chain`);
}

/**
 * joinCtfTeam 对应 POST /api/v1/ctf/teams/join，用于邀请码加入团队。
 */
export function joinCtfTeam(inviteCode: string) {
  return apiClient.post<{ team_id: ID; team_name: string; competition_id: ID; role: number; role_text: string; current_members: number; max_team_size: number }>("/ctf/teams/join", { invite_code: inviteCode });
}

/**
 * registerCtfCompetition 对应 POST /api/v1/ctf/competitions/:id/register，用于报名竞赛。
 */
export function registerCtfCompetition(competitionID: ID, teamID?: ID | null) {
  return apiClient.post<CtfRegistration>(`/ctf/competitions/${competitionID}/register`, { team_id: teamID ?? null });
}

/**
 * getMyCtfRegistration 对应 GET /api/v1/ctf/competitions/:id/my-registration，用于我的报名状态。
 */
export function getMyCtfRegistration(competitionID: ID) {
  return apiClient.get<CtfMyRegistration>(`/ctf/competitions/${competitionID}/my-registration`);
}

/**
 * submitCtfChallenge 对应 POST /api/v1/ctf/competitions/:id/submissions，用于提交 Flag 或攻击交易。
 */
export function submitCtfChallenge(competitionID: ID, payload: SubmitCtfChallengeRequest) {
  return apiClient.post<CtfSubmissionResponse>(`/ctf/competitions/${competitionID}/submissions`, payload);
}

/**
 * listCtfSubmissions 对应 GET /api/v1/ctf/competitions/:id/submissions，用于团队提交记录。
 */
export function listCtfSubmissions(competitionID: ID, params: QueryParams = {}) {
  return apiClient.get<PaginatedData<CtfSubmissionListItem>>(`/ctf/competitions/${competitionID}/submissions`, { query: params });
}

/**
 * startCtfChallengeEnvironment 对应 POST /api/v1/ctf/competitions/:comp_id/challenges/:challenge_id/environment，用于启动题目环境。
 */
export function startCtfChallengeEnvironment(competitionID: ID, challengeID: ID) {
  return apiClient.post<StartCtfEnvironmentResponse>(`/ctf/competitions/${competitionID}/challenges/${challengeID}/environment`);
}

/**
 * getCtfChallengeEnvironment 对应 GET /api/v1/ctf/challenge-environments/:id，用于环境详情。
 */
export function getCtfChallengeEnvironment(environmentID: ID) {
  return apiClient.get<import("@/types/ctf").CtfChallengeEnvironment>(`/ctf/challenge-environments/${environmentID}`);
}

/**
 * resetCtfChallengeEnvironment 对应 POST /api/v1/ctf/challenge-environments/:id/reset，用于重置环境。
 */
export function resetCtfChallengeEnvironment(environmentID: ID) {
  return apiClient.post<{ environment_id: ID; status: number; status_text: string }>(`/ctf/challenge-environments/${environmentID}/reset`);
}

/**
 * destroyCtfChallengeEnvironment 对应 POST /api/v1/ctf/challenge-environments/:id/destroy，用于销毁环境。
 */
export function destroyCtfChallengeEnvironment(environmentID: ID) {
  return apiClient.post<null>(`/ctf/challenge-environments/${environmentID}/destroy`);
}

/**
 * forceDestroyCtfChallengeEnvironment 对应 POST /api/v1/ctf/challenge-environments/:id/force-destroy，用于超级管理员强制回收题目环境。
 */
export function forceDestroyCtfChallengeEnvironment(environmentID: ID, reason?: string) {
  return reason
    ? apiClient.post<ForceDestroyCtfEnvironmentResponse>(`/ctf/challenge-environments/${environmentID}/force-destroy`, { reason })
    : apiClient.post<ForceDestroyCtfEnvironmentResponse>(`/ctf/challenge-environments/${environmentID}/force-destroy`);
}

/**
 * listMyCtfEnvironments 对应 GET /api/v1/ctf/competitions/:id/my-environments，用于我的题目环境列表。
 */
export function listMyCtfEnvironments(competitionID: ID) {
  return apiClient.get<{ list: Array<{ id: ID; challenge_id: ID; challenge_title: string; namespace: string; status: number; status_text: string; chain_rpc_url: string | null; created_at: string }> }>(`/ctf/competitions/${competitionID}/my-environments`);
}

/**
 * listCtfCompetitionEnvironments 对应 GET /api/v1/ctf/competitions/:id/environments，用于竞赛环境资源列表。
 */
export function listCtfCompetitionEnvironments(competitionID: ID, params: CtfEnvironmentListParams = {}) {
  return apiClient.get<PaginatedData<CtfChallengeEnvironment>>(`/ctf/competitions/${competitionID}/environments`, { query: params });
}

/**
 * getCtfLeaderboard 对应 GET /api/v1/ctf/competitions/:id/leaderboard，用于实时排行榜。
 */
export function getCtfLeaderboard(competitionID: ID, params: QueryParams = {}) {
  return apiClient.get<CtfLeaderboard>(`/ctf/competitions/${competitionID}/leaderboard`, { query: params });
}

/**
 * getCtfFinalLeaderboard 对应 GET /api/v1/ctf/competitions/:id/leaderboard/final，用于最终排名。
 */
export function getCtfFinalLeaderboard(competitionID: ID) {
  return apiClient.get<{ competition_id: ID; competition_type: number; ended_at: string; rankings: CtfLeaderboard["rankings"]; total_teams: number; total_solves: number }>(`/ctf/competitions/${competitionID}/leaderboard/final`);
}

/**
 * getCtfLeaderboardHistory 对应 GET /api/v1/ctf/competitions/:id/leaderboard/history，用于排行榜历史快照。
 */
export function getCtfLeaderboardHistory(competitionID: ID, params: QueryParams = {}) {
  return apiClient.get<PaginatedData<CtfLeaderboardHistorySnapshot>>(`/ctf/competitions/${competitionID}/leaderboard/history`, { query: params });
}

/**
 * listCtfAnnouncements 对应 GET /api/v1/ctf/competitions/:id/announcements，用于公告列表。
 */
export function listCtfAnnouncements(competitionID: ID) {
  return apiClient.get<{ list: CtfAnnouncement[] }>(`/ctf/competitions/${competitionID}/announcements`);
}

/**
 * createCtfAnnouncement 对应 POST /api/v1/ctf/competitions/:id/announcements，用于发布公告。
 */
export function createCtfAnnouncement(competitionID: ID, payload: { title: string; content: string; announcement_type: 1 | 2 | 3; challenge_id?: ID | null }) {
  return apiClient.post<CtfAnnouncement>(`/ctf/competitions/${competitionID}/announcements`, payload);
}

/**
 * listCtfCompetitionTokenLedger 对应 GET /api/v1/ctf/competitions/:id/token-ledger，用于竞赛创建者查看 Token 流水。
 */
export function listCtfCompetitionTokenLedger(competitionID: ID, params: QueryParams = {}) {
  return apiClient.get<PaginatedData<CtfTokenLedgerItem>>(`/ctf/competitions/${competitionID}/token-ledger`, { query: params });
}

/**
 * getCtfCompetitionMonitor 对应 GET /api/v1/ctf/competitions/:id/monitor，用于竞赛运行监控。
 */
export function getCtfCompetitionMonitor(competitionID: ID) {
  return apiClient.get<CtfCompetitionMonitor>(`/ctf/competitions/${competitionID}/monitor`);
}

/**
 * getCtfCompetitionStatistics 对应 GET /api/v1/ctf/competitions/:id/statistics，用于竞赛统计。
 */
export function getCtfCompetitionStatistics(competitionID: ID) {
  return apiClient.get<CtfCompetitionStatistics>(`/ctf/competitions/${competitionID}/statistics`);
}

/**
 * getCtfCompetitionResults 对应 GET /api/v1/ctf/competitions/:id/results，用于竞赛最终结果。
 */
export function getCtfCompetitionResults(competitionID: ID) {
  return apiClient.get<CtfCompetitionResults>(`/ctf/competitions/${competitionID}/results`);
}

/**
 * getCtfResourceQuota 对应 GET /api/v1/ctf/competitions/:id/resource-quota，用于竞赛资源配额详情。
 */
export function getCtfResourceQuota(competitionID: ID) {
  return apiClient.get<CtfResourceQuota>(`/ctf/competitions/${competitionID}/resource-quota`);
}

/**
 * updateCtfResourceQuota 对应 PUT /api/v1/ctf/competitions/:id/resource-quota，用于设置竞赛资源配额。
 */
export function updateCtfResourceQuota(competitionID: ID, payload: { max_cpu: string; max_memory: string; max_storage: string; max_namespaces: number }) {
  return apiClient.put<CtfResourceQuota>(`/ctf/competitions/${competitionID}/resource-quota`, payload);
}

/**
 * getCtfAdminOverview 对应 GET /api/v1/ctf/admin/competitions/overview，用于平台 CTF 概览。
 */
export function getCtfAdminOverview() {
  return apiClient.get<{ total_competitions: number; running_competitions: number; upcoming_competitions: number; total_participants: number; total_resource_usage: { cpu_used: string; memory_used: string; namespaces_active: number }; running_competitions_list: Array<{ id: ID; title: string; competition_type: number; competition_type_text: string; status: number; status_text: string; teams: number; environments_running: number; start_at: string | null; end_at: string | null }>; alerts: Array<{ type: string; message: string; competition_id: ID; created_at: string }> }>("/ctf/admin/competitions/overview");
}

/**
 * createCtfAdGroup 对应 POST /api/v1/ctf/competitions/:id/ad-groups，用于创建攻防赛分组。
 */
export function createCtfAdGroup(competitionID: ID, payload: CreateCtfAdGroupRequest) {
  return apiClient.post<CtfAdGroup>(`/ctf/competitions/${competitionID}/ad-groups`, payload);
}

/**
 * listAdGroups 对应 GET /api/v1/ctf/competitions/:id/ad-groups，用于攻防赛分组。
 */
export function listAdGroups(competitionID: ID) {
  return apiClient.get<{ list: CtfAdGroup[] }>(`/ctf/competitions/${competitionID}/ad-groups`);
}

/**
 * autoAssignCtfAdGroups 对应 POST /api/v1/ctf/competitions/:id/ad-groups/auto-assign，用于自动划分攻防赛分组。
 */
export function autoAssignCtfAdGroups(competitionID: ID, payload: AutoAssignCtfAdGroupsRequest) {
  return apiClient.post<AutoAssignCtfAdGroupsResponse>(`/ctf/competitions/${competitionID}/ad-groups/auto-assign`, payload);
}

/**
 * getCurrentAdRound 对应 GET /api/v1/ctf/ad-groups/:id/current-round，用于当前回合状态。
 */
export function getCurrentAdRound(groupID: ID) {
  return apiClient.get<CtfCurrentRound>(`/ctf/ad-groups/${groupID}/current-round`);
}

/**
 * submitAdAttack 对应 POST /api/v1/ctf/ad-rounds/:id/attacks，用于提交攻击交易。
 */
export function submitAdAttack(roundID: ID, payload: SubmitAdAttackRequest) {
  return apiClient.post<AdAttackResponse>(`/ctf/ad-rounds/${roundID}/attacks`, payload);
}

/**
 * listAdAttacks 对应 GET /api/v1/ctf/ad-rounds/:id/attacks，用于本回合攻击记录。
 */
export function listAdAttacks(roundID: ID, params: QueryParams = {}) {
  return apiClient.get<PaginatedData<AdAttackListItem>>(`/ctf/ad-rounds/${roundID}/attacks`, { query: params });
}

/**
 * submitAdDefense 对应 POST /api/v1/ctf/ad-rounds/:id/defenses，用于提交防守补丁。
 */
export function submitAdDefense(roundID: ID, payload: SubmitAdDefenseRequest) {
  return apiClient.post<AdDefenseResponse>(`/ctf/ad-rounds/${roundID}/defenses`, payload);
}

/**
 * listAdDefenses 对应 GET /api/v1/ctf/ad-rounds/:id/defenses，用于本回合防守记录。
 */
export function listAdDefenses(roundID: ID, params: QueryParams = {}) {
  return apiClient.get<PaginatedData<AdDefenseListItem>>(`/ctf/ad-rounds/${roundID}/defenses`, { query: params });
}
