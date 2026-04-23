// ctf.ts
// 模块05 CTF竞赛类型定义：竞赛、题目、漏洞转化、预验证、团队、环境、排行榜、攻防赛和实时推送。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/** JSON 对象类型，用于链配置、模板参数和断言附加参数。 */
export type CtfJsonObject = Record<string, unknown>;

/** 竞赛类型：1解题赛 2攻防赛。 */
export type CtfCompetitionType = 1 | 2;

/** 竞赛范围：1平台级 2校级。 */
export type CtfCompetitionScope = 1 | 2;

/** 团队模式：1个人赛 2自由组队 3指定组队。 */
export type CtfTeamMode = 1 | 2 | 3;

/** 竞赛状态：1草稿 2报名中 3进行中 4已结束 5已归档。 */
export type CtfCompetitionStatus = 1 | 2 | 3 | 4 | 5;

/** 题目分类。 */
export type CtfChallengeCategory = "web" | "crypto" | "contract" | "blockchain" | "reverse" | "misc";

/** 题目难度：1Warmup 2Easy 3Medium 4Hard 5Insane。 */
export type CtfDifficulty = 1 | 2 | 3 | 4 | 5;

/** Flag 类型：1静态 2动态 3链上状态验证。 */
export type CtfFlagType = 1 | 2 | 3;

/** 链上题运行时模式：1 isolated 2 forked。 */
export type CtfRuntimeMode = 1 | 2;

/** 题目来源路径：1 SWC 2模板 3自定义。 */
export type CtfSourcePath = 1 | 2 | 3;

/** 题目状态：1草稿 2待审核 3已通过 4已拒绝。 */
export type CtfChallengeStatus = 1 | 2 | 3 | 4;

/** 攻防赛阶段：1攻击 2防守 3结算。 */
export type CtfBattlePhase = 1 | 2 | 3;

/** CTF 用户摘要。 */
export interface CtfUserBrief {
  id: ID;
  name: string;
}

/** CTF 团队摘要。 */
export interface CtfTeamBrief {
  id: ID;
  name: string;
}

/** 解题赛动态计分配置。 */
export interface JeopardyScoringConfig {
  decay_factor: number;
  min_score_ratio: number;
  first_blood_bonus: number;
}

/** 解题赛提交限流配置。 */
export interface JeopardySubmissionLimitConfig {
  max_per_minute: number;
  cooldown_threshold: number;
  cooldown_minutes: number;
}

/** 解题赛配置。 */
export interface JeopardyCompetitionConfig {
  scoring: JeopardyScoringConfig;
  submission_limit: JeopardySubmissionLimitConfig;
}

/** 攻防赛配置。 */
export interface AdCompetitionConfig {
  total_rounds: number;
  attack_duration_minutes: number;
  defense_duration_minutes: number;
  initial_token: number;
  attack_bonus_ratio: number;
  defense_reward_per_round: number;
  first_patch_bonus: number;
  first_blood_bonus_ratio: number;
  vulnerability_decay_factor: number;
  max_teams_per_group: number;
  judge_chain_image: string;
  team_chain_image: string;
}

/** 创建竞赛请求。 */
export interface CreateCtfCompetitionRequest {
  title: string;
  description?: string | null;
  banner_url?: string | null;
  competition_type: CtfCompetitionType;
  scope: CtfCompetitionScope;
  school_id?: ID | null;
  team_mode: CtfTeamMode;
  max_team_size: number;
  min_team_size: number;
  max_teams?: number | null;
  registration_start_at?: string | null;
  registration_end_at?: string | null;
  start_at?: string | null;
  end_at?: string | null;
  freeze_at?: string | null;
  rules?: string | null;
  jeopardy_config?: JeopardyCompetitionConfig | null;
  ad_config?: AdCompetitionConfig | null;
}

/** 编辑竞赛请求。 */
export type UpdateCtfCompetitionRequest = Partial<Omit<CreateCtfCompetitionRequest, "competition_type">>;

/** 竞赛列表查询参数。 */
export interface CtfCompetitionListParams extends QueryParams {
  page?: number;
  page_size?: number;
  competition_type?: CtfCompetitionType;
  scope?: CtfCompetitionScope;
  status?: CtfCompetitionStatus;
  keyword?: string;
}

/** 竞赛列表项。 */
export interface CtfCompetitionListItem {
  id: ID;
  title: string;
  banner_url: string | null;
  competition_type: CtfCompetitionType;
  competition_type_text: string;
  scope: CtfCompetitionScope;
  scope_text: string;
  team_mode: CtfTeamMode;
  team_mode_text: string;
  max_team_size: number;
  status: CtfCompetitionStatus;
  status_text: string;
  registered_teams: number;
  max_teams: number | null;
  challenge_count: number;
  registration_start_at: string | null;
  registration_end_at: string | null;
  start_at: string | null;
  end_at: string | null;
  created_by_name: string;
}

/** 竞赛详情。 */
export interface CtfCompetitionDetail extends CtfCompetitionListItem {
  description: string | null;
  min_team_size: number;
  freeze_at: string | null;
  rules: string | null;
  jeopardy_config: JeopardyCompetitionConfig | null;
  ad_config: AdCompetitionConfig | null;
  created_by: CtfUserBrief | null;
  created_at: string;
}

/** 竞赛列表响应。 */
export type CtfCompetitionListResponse = PaginatedData<CtfCompetitionListItem>;

/** 竞赛创建响应。 */
export interface CtfCompetitionCreateResponse {
  id: ID;
  title: string;
  competition_type: CtfCompetitionType;
  competition_type_text: string;
  scope: CtfCompetitionScope;
  scope_text: string;
  team_mode: CtfTeamMode;
  team_mode_text: string;
  status: CtfCompetitionStatus;
  status_text: string;
  created_at: string;
}

/** 题目链预置账户。 */
export interface CtfChainAccount {
  name: string;
  balance: string;
}

/** Fork 固定合约。 */
export interface CtfPinnedContract {
  name: string;
  address: string;
}

/** Fork 模式链配置。 */
export interface CtfChainForkConfig {
  rpc_url: string;
  chain_id: number;
  block_number: number;
  impersonated_accounts?: string[];
  pinned_contracts?: CtfPinnedContract[];
}

/** 题目链配置。 */
export interface CtfChainConfig {
  chain_type: string;
  chain_version: string;
  block_number: number;
  fork?: CtfChainForkConfig | null;
  accounts: CtfChainAccount[];
}

/** 题目初始化交易。 */
export interface CtfSetupTransaction {
  from: string;
  to: string;
  function: string;
  args: unknown[];
  value: string;
}

/** 非合约题目环境配置。 */
export interface CtfChallengeEnvironmentConfig {
  images: Array<{
    image: string;
    version: string;
    ports: Array<{ port: number; protocol: string }>;
    env_vars: Array<{ key: string; value: string }>;
    cpu_limit: string;
    memory_limit: string;
  }>;
  init_script?: string | null;
}

/** 创建题目请求。 */
export interface CreateCtfChallengeRequest {
  title: string;
  description: string;
  category: CtfChallengeCategory;
  difficulty: CtfDifficulty;
  base_score: number;
  flag_type: CtfFlagType;
  static_flag?: string | null;
  dynamic_flag_secret?: string | null;
  runtime_mode?: CtfRuntimeMode | null;
  chain_config?: CtfChainConfig | null;
  setup_transactions: CtfSetupTransaction[];
  source_path?: CtfSourcePath | null;
  swc_id?: string | null;
  template_id?: ID | null;
  template_params?: CtfJsonObject | null;
  environment_config?: CtfChallengeEnvironmentConfig | null;
  attachment_urls: string[];
}

/** 编辑题目请求。 */
export type UpdateCtfChallengeRequest = Partial<CreateCtfChallengeRequest> & { is_public?: boolean };

/** 题目列表查询参数。 */
export interface CtfChallengeListParams extends QueryParams {
  page?: number;
  page_size?: number;
  category?: CtfChallengeCategory;
  difficulty?: CtfDifficulty;
  flag_type?: CtfFlagType;
  status?: CtfChallengeStatus;
  is_public?: boolean;
  keyword?: string;
  author_id?: ID;
}

/** 题目列表项。 */
export interface CtfChallengeListItem {
  id: ID;
  title: string;
  category: CtfChallengeCategory;
  category_text: string;
  difficulty: CtfDifficulty;
  difficulty_text: string;
  base_score: number;
  flag_type: CtfFlagType;
  flag_type_text: string;
  runtime_mode: CtfRuntimeMode | null;
  runtime_mode_text: string | null;
  source_path: CtfSourcePath | null;
  source_path_text: string | null;
  status: CtfChallengeStatus;
  status_text: string;
  is_public: boolean;
  usage_count: number;
  author: CtfUserBrief | null;
  created_at: string;
}

/** 题目合约。 */
export interface CtfChallengeContract {
  id: ID;
  challenge_id?: ID | null;
  name: string;
  source_code?: string | null;
  abi?: CtfJsonObject[];
  bytecode?: string | null;
  constructor_args?: unknown[];
  deploy_order: number;
}

/** 题目断言。 */
export interface CtfChallengeAssertion {
  id: ID;
  challenge_id?: ID | null;
  assertion_type: string;
  target: string;
  operator: string;
  expected_value: string;
  description: string | null;
  extra_params: CtfJsonObject | null;
  sort_order: number;
}

/** 预验证摘要。 */
export interface CtfVerificationSummary {
  id: ID;
  status: number;
  status_text: string;
  completed_at: string | null;
}

/** 题目详情。 */
export interface CtfChallengeDetail extends CtfChallengeListItem {
  description: string;
  chain_config: CtfChainConfig | null;
  setup_transactions: CtfSetupTransaction[];
  swc_id: string | null;
  template_id: ID | null;
  environment_config: CtfChallengeEnvironmentConfig | null;
  attachment_urls: string[];
  contracts: CtfChallengeContract[];
  assertions: CtfChallengeAssertion[];
  latest_verification: CtfVerificationSummary | null;
  updated_at: string;
}

/** 题目列表响应。 */
export type CtfChallengeListResponse = PaginatedData<CtfChallengeListItem>;

/** 题目创建/状态响应。 */
export interface CtfChallengeStatusResponse {
  id: ID;
  title: string;
  category?: CtfChallengeCategory;
  category_text?: string | null;
  difficulty?: CtfDifficulty;
  difficulty_text?: string | null;
  base_score?: number;
  flag_type?: CtfFlagType;
  flag_type_text?: string | null;
  runtime_mode?: CtfRuntimeMode | null;
  runtime_mode_text?: string | null;
  source_path?: CtfSourcePath | null;
  source_path_text?: string | null;
  swc_id?: string | null;
  template_id?: ID | null;
  status: CtfChallengeStatus;
  status_text: string;
  created_at?: string | null;
  contracts_generated?: number | null;
  assertions_generated?: number | null;
}

/** SWC Registry 项。 */
export interface SwcRegistryItem {
  swc_id: string;
  title: string;
  description: string;
  severity: string;
  has_example: boolean;
  suggested_difficulty: CtfDifficulty;
}

/** SWC 导入请求。 */
export interface ImportSwcChallengeRequest {
  swc_id: string;
  title: string;
  difficulty: CtfDifficulty;
  base_score: number;
}

/** 模板库列表项。 */
export interface CtfChallengeTemplateListItem {
  id: ID;
  name: string;
  code: string;
  vulnerability_type: string;
  description: string;
  difficulty_range: { min: CtfDifficulty; max: CtfDifficulty };
  variant_count: number;
  usage_count: number;
}

/** 模板详情。 */
export interface CtfChallengeTemplateDetail extends CtfChallengeTemplateListItem {
  parameters: { params: Array<{ key: string; label: string; type: string; default: unknown; options?: string[] }> };
  variants: Array<{ name: string; params: CtfJsonObject; suggested_difficulty: CtfDifficulty }>;
  reference_events: Array<{ name: string; date: string; loss: string }>;
}

/** 从模板生成题目请求。 */
export interface GenerateChallengeFromTemplateRequest {
  template_id: ID;
  title: string;
  difficulty: CtfDifficulty;
  base_score: number;
  template_params: CtfJsonObject;
}

/** 外部真实漏洞源导入请求。 */
export interface ImportExternalVulnerabilityRequest {
  source_grade: "A" | "B" | "C";
  title: string;
  vulnerability_name: string;
  source_url: string;
  confidence_score: number;
  reproducibility_score: number;
  category: CtfChallengeCategory;
  difficulty: CtfDifficulty;
  base_score: number;
  chain_config?: CtfChainConfig | null;
  source_code?: string | null;
  poc_content?: string | null;
  setup_transactions: CtfSetupTransaction[];
  reference_event?: CtfJsonObject | null;
}

/** 预验证步骤断言结果。 */
export interface CtfVerificationAssertionResult {
  type: string;
  target?: string;
  expected: string;
  actual: string;
  passed: boolean;
}

/** 断言结果汇总。 */
export interface CtfVerificationAssertionResults {
  all_passed: boolean;
  results: CtfVerificationAssertionResult[];
  execution_time_ms?: number | null;
  tx_hash?: string | null;
}

/** 预验证步骤结果。 */
export interface CtfVerificationStepResult {
  step: number;
  name: string;
  status: string;
  detail: string;
  assertions?: CtfVerificationAssertionResult[];
  duration_ms?: number | null;
}

/** 发起预验证请求。 */
export interface VerifyCtfChallengeRequest {
  poc_content: string;
  poc_language?: "solidity" | "javascript" | "python" | null;
}

/** 发起预验证响应。 */
export interface VerifyCtfChallengeResponse {
  verification_id: ID;
  challenge_id: ID;
  status: number;
  status_text: string;
  started_at: string;
}

/** 预验证记录。 */
export interface CtfVerificationListItem {
  id: ID;
  status: number;
  status_text: string;
  poc_language: string | null;
  started_at: string;
  completed_at: string | null;
  error_message: string | null;
}

/** 预验证详情。 */
export interface CtfVerificationDetail {
  id: ID;
  challenge_id: ID;
  status: number;
  status_text: string;
  step_results: CtfVerificationStepResult[];
  poc_content: string | null;
  poc_language: string | null;
  environment_id: string | null;
  error_message: string | null;
  started_at: string;
  completed_at: string | null;
  created_at: string;
}

/** 题目审核列表项。 */
export interface CtfChallengeReviewListItem {
  id: ID;
  title: string;
  category: CtfChallengeCategory;
  category_text: string;
  difficulty: CtfDifficulty;
  difficulty_text: string;
  author_name: string;
  school_name: string;
  submitted_at: string;
}

/** 审核题目请求。 */
export interface ReviewCtfChallengeRequest {
  action: 1 | 2;
  comment?: string | null;
}

/** 审核题目响应。 */
export interface ReviewCtfChallengeResponse {
  challenge_id: ID;
  status: CtfChallengeStatus;
  status_text: string;
  review_id: ID;
}

/** 竞赛题目摘要。 */
export interface CtfCompetitionChallengeSummary {
  id: ID;
  title: string;
  description: string;
  category: CtfChallengeCategory;
  category_text: string;
  difficulty: CtfDifficulty;
  difficulty_text: string;
  flag_type: CtfFlagType;
  flag_type_text: string;
  attachment_urls: string[];
  chain_config?: CtfChainConfig | null;
  source_path?: CtfSourcePath | null;
  source_path_text?: string | null;
  swc_id?: string | null;
  template_id?: ID | null;
  environment_config?: CtfChallengeEnvironmentConfig | null;
}

/** 竞赛题目列表项。 */
export interface CtfCompetitionChallengeListItem {
  id: ID;
  challenge: CtfCompetitionChallengeSummary;
  base_score: number;
  current_score: number | null;
  solve_count: number;
  first_blood_team?: CtfTeamBrief | null;
  first_blood_at?: string | null;
  sort_order: number;
  my_team_solved: boolean;
  my_team_environment: ID | null;
}

/** 团队成员。 */
export interface CtfTeamMember {
  student_id: ID;
  name: string;
  role: 1 | 2;
  role_text: string;
  joined_at: string;
}

/** 团队详情。 */
export interface CtfTeam {
  id: ID;
  competition_id: ID;
  name: string;
  captain_id: ID;
  invite_code: string | null;
  status: number;
  status_text: string;
  members?: CtfTeamMember[];
}

/** 团队列表项。 */
export interface CtfTeamListItem {
  id: ID;
  name: string;
  captain_name: string;
  member_count: number;
  status: number;
  status_text: string;
  registered: boolean;
  final_rank: number | null;
  total_score: number | null;
}

/** 我的报名状态。 */
export interface CtfMyRegistration {
  is_registered: boolean;
  registration_id: ID | null;
  team_id: ID | null;
  team_name: string | null;
  status: number | null;
  status_text: string | null;
  registered_at: string | null;
}

/** 报名响应。 */
export interface CtfRegistration {
  registration_id: ID;
  competition_id: ID;
  team_id: ID;
  team_name: string;
  status: number;
  status_text: string;
  registered_at: string;
}

/** 提交请求。 */
export interface SubmitCtfChallengeRequest {
  challenge_id: ID;
  submission_type: 1 | 2 | 3;
  content: string;
}

/** 提交响应。 */
export interface CtfSubmissionResponse {
  submission_id: ID;
  is_correct: boolean;
  score_awarded: number | null;
  is_first_blood: boolean | null;
  challenge_new_score: number | null;
  team_total_score: number | null;
  team_rank: number | null;
  assertion_results: CtfVerificationAssertionResults | null;
  error_message: string | null;
  remaining_attempts: number | null;
  cooldown_until: string | null;
}

/** 提交记录。 */
export interface CtfSubmissionListItem {
  id: ID;
  challenge_id: ID;
  challenge_title: string;
  submission_type: number;
  submission_type_text: string;
  is_correct: boolean;
  score_awarded: number | null;
  is_first_blood: boolean;
  error_message: string | null;
  created_at: string;
}

/** 题目环境。 */
export interface CtfChallengeEnvironment {
  id: ID;
  competition_id: ID;
  challenge_id: ID;
  challenge_title?: string;
  team_id: ID;
  team_name?: string;
  namespace: string;
  status: number;
  status_text: string;
  chain_rpc_url: string | null;
  container_status?: Record<string, { status: string; image: string }>;
  started_at?: string | null;
  created_at: string;
}

/** CTF 环境列表查询参数。 */
export interface CtfEnvironmentListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: number;
  challenge_id?: ID;
  team_id?: ID;
}

/** CTF 环境强制回收响应。 */
export interface ForceDestroyCtfEnvironmentResponse {
  environment_id: ID;
  status: number;
  status_text: string;
}

/** 启动环境响应。 */
export interface StartCtfEnvironmentResponse {
  environment_id: ID;
  competition_id: ID;
  challenge_id: ID;
  team_id: ID;
  namespace: string;
  status: number;
  status_text: string;
  chain_rpc_url: string | null;
  created_at: string;
}

/** 攻防分组。 */
export interface CtfAdGroup {
  id: ID;
  competition_id: ID;
  group_name: string;
  namespace: string | null;
  status: number;
  status_text: string;
  teams?: CtfTeamBrief[];
}

/** 创建攻防分组请求。 */
export interface CreateCtfAdGroupRequest {
  group_name: string;
  team_ids: ID[];
}

/** 自动分配攻防分组请求。 */
export interface AutoAssignCtfAdGroupsRequest {
  teams_per_group?: number;
}

/** 自动分配攻防分组响应。 */
export interface AutoAssignCtfAdGroupsResponse {
  groups: Array<{ id: ID; group_name: string; team_count: number }>;
  total_teams: number;
  total_groups: number;
}

/** 攻防赛 Token 流水项。 */
export interface CtfTokenLedgerItem {
  id: ID;
  round_number: number | null;
  change_type: number;
  change_type_text: string;
  amount: number;
  balance_after: number;
  description: string;
  related_attack_id: ID | null;
  created_at: string;
}

/** 攻防赛队伍链信息。 */
export interface CtfTeamChain {
  id: ID;
  competition_id: ID;
  group_id: ID;
  team_id: ID;
  chain_rpc_url: string;
  chain_ws_url: string;
  deployed_contracts: Array<{ challenge_id: ID; contract_name: string; address: string; patch_version: number; is_patched: boolean }>;
  current_patch_version: number;
  status: number;
  status_text: string;
}

/** 当前回合。 */
export interface CtfCurrentRound {
  group_id: ID;
  round_id: ID;
  round_number: number;
  total_rounds: number;
  phase: CtfBattlePhase;
  phase_text: string;
  phase_start_at: string;
  phase_end_at: string;
  remaining_seconds: number;
  my_team: { id: ID; name: string; token_balance: number; rank: number } | null;
}

/** 攻击请求。 */
export interface SubmitAdAttackRequest {
  target_team_id: ID;
  challenge_id: ID;
  attack_tx_data: string;
}

/** 攻击响应。 */
export interface AdAttackResponse {
  attack_id: ID;
  is_successful: boolean;
  token_reward: number | null;
  is_first_blood: boolean | null;
  exploit_count: number | null;
  assertion_results: CtfVerificationAssertionResults | null;
  attacker_balance_after: number | null;
  target_balance_after: number | null;
  error_message: string | null;
}

/** 防守请求。 */
export interface SubmitAdDefenseRequest {
  challenge_id: ID;
  patch_source_code: string;
}

/** 防守响应。 */
export interface AdDefenseResponse {
  defense_id: ID;
  is_accepted: boolean;
  functionality_passed: boolean | null;
  vulnerability_fixed: boolean | null;
  is_first_patch: boolean | null;
  token_reward: number | null;
  team_balance_after: number | null;
  rejection_reason: string | null;
}

/** 攻击记录。 */
export interface AdAttackListItem {
  id: ID;
  attacker_team_id: ID;
  attacker_team_name: string;
  target_team_id: ID;
  target_team_name: string;
  challenge_id: ID;
  challenge_title: string;
  is_successful: boolean;
  token_reward: number | null;
  is_first_blood: boolean;
  created_at: string;
}

/** 防守记录。 */
export interface AdDefenseListItem {
  id: ID;
  team_id: ID;
  team_name: string;
  challenge_id: ID;
  challenge_title: string;
  is_accepted: boolean;
  functionality_passed: boolean | null;
  vulnerability_fixed: boolean | null;
  is_first_patch: boolean;
  token_reward: number | null;
  created_at: string;
}

/** 排行榜排名项。 */
export interface CtfLeaderboardRanking {
  rank: number;
  team_id: ID;
  team_name: string;
  score?: number | null;
  solve_count?: number | null;
  last_solve_at?: string | null;
  token_balance?: number | null;
  attacks_successful?: number | null;
  defenses_successful?: number | null;
  patches_accepted?: number | null;
}

/** 排行榜响应。 */
export interface CtfLeaderboard {
  competition_id: ID;
  competition_type: CtfCompetitionType;
  group_id: ID | null;
  group_name: string | null;
  current_round: number | null;
  total_rounds: number | null;
  is_frozen: boolean;
  frozen_at: string | null;
  updated_at: string | null;
  rankings: CtfLeaderboardRanking[];
}

/** 排行榜历史快照项。 */
export interface CtfLeaderboardHistorySnapshot {
  snapshot_at: string;
  rankings: CtfLeaderboardRanking[];
}

/** 公告。 */
export interface CtfAnnouncement {
  id: ID;
  title: string;
  content: string;
  announcement_type: 1 | 2 | 3;
  announcement_type_text: string;
  challenge_id: ID | null;
  challenge_title: string | null;
  published_by_name: string;
  created_at: string;
}

/** 竞赛监控。 */
export interface CtfCompetitionMonitor {
  competition_id: ID;
  competition_type: CtfCompetitionType;
  status: CtfCompetitionStatus;
  status_text: string;
  overview: { registered_teams: number; active_teams: number; total_submissions: number; correct_submissions: number; total_environments: number; running_environments: number };
  resource_usage: { cpu_used: string; cpu_max: string; memory_used: string; memory_max: string; namespaces_used: number; namespaces_max: number };
  challenge_stats: Array<{ challenge_id: ID; title: string; category: string; solve_count: number; attempt_count: number; solve_rate: number; current_score: number | null; environments_running: number }>;
  recent_submissions: Array<{ team_name: string; challenge_title: string; is_correct: boolean; submitted_at: string }>;
}

/** 竞赛统计。 */
export interface CtfCompetitionStatistics {
  competition_id: ID;
  summary: { total_teams: number; total_participants: number; total_submissions: number; total_correct: number; overall_solve_rate: number; average_score: number; highest_score: number; lowest_score: number };
  challenge_statistics: Array<{ challenge_id: ID; title: string; category: string; difficulty: CtfDifficulty; difficulty_text: string; solve_count: number; attempt_count: number; solve_rate: number; first_blood_team: string | null; first_blood_time_minutes: number | null; average_solve_time_minutes: number | null }>;
  score_distribution: { ranges: Array<{ label: string; count: number }> };
  timeline: { submissions_per_hour: Array<{ hour: string; count: number }> };
}

/** 竞赛结果。 */
export interface CtfCompetitionResults {
  competition_id: ID;
  summary: CtfCompetitionStatistics["summary"];
  rankings: Array<CtfLeaderboardRanking & { members?: Array<{ name: string; role_text: string }>; solved_challenges?: Array<{ challenge_id: ID; title: string; score: number; solved_at: string; is_first_blood: boolean }> }>;
}

/** CTF 竞赛资源配额。 */
export interface CtfResourceQuota {
  competition_id: ID;
  max_cpu: string;
  max_memory: string;
  max_storage: string;
  max_namespaces: number;
  used_cpu: string;
  used_memory: string;
  used_storage: string;
  current_namespaces: number;
}

/** CTF WebSocket 消息。 */
export interface CtfRealtimeMessage {
  type: "message" | "pong" | "snapshot";
  channel?: "leaderboard" | "announcement" | "round" | "attacks";
  data?: CtfJsonObject;
  timestamp?: string;
}
