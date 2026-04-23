// ctf.go
// 模块05 — CTF竞赛：请求/响应 DTO 定义。
// 该文件对齐 docs/modules/05-CTF竞赛/03-API接口设计.md，覆盖竞赛、题目、团队、攻防赛、资源、监控等接口。

package dto

// ========== 通用子结构 ==========

// CTFUserBrief 用户摘要信息。
type CTFUserBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CTFTeamBrief 团队摘要信息。
type CTFTeamBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// JeopardyScoringConfig 解题赛动态计分配置。
type JeopardyScoringConfig struct {
	DecayFactor   float64 `json:"decay_factor"`
	MinScoreRatio float64 `json:"min_score_ratio"`
	// FirstBloodBonus 表示额外奖励比例，例如 0.1 表示额外奖励当前分值的 10%。
	FirstBloodBonus float64 `json:"first_blood_bonus"`
}

// JeopardySubmissionLimitConfig 解题赛提交限流配置。
type JeopardySubmissionLimitConfig struct {
	MaxPerMinute      int `json:"max_per_minute"`
	CooldownThreshold int `json:"cooldown_threshold"`
	CooldownMinutes   int `json:"cooldown_minutes"`
}

// JeopardyCompetitionConfig 解题赛竞赛配置。
type JeopardyCompetitionConfig struct {
	Scoring         JeopardyScoringConfig         `json:"scoring"`
	SubmissionLimit JeopardySubmissionLimitConfig `json:"submission_limit"`
}

// ADCompetitionConfig 攻防赛竞赛配置。
type ADCompetitionConfig struct {
	TotalRounds              int     `json:"total_rounds"`
	AttackDurationMinutes    int     `json:"attack_duration_minutes"`
	DefenseDurationMinutes   int     `json:"defense_duration_minutes"`
	InitialToken             int     `json:"initial_token"`
	AttackBonusRatio         float64 `json:"attack_bonus_ratio"`
	DefenseRewardPerRound    int     `json:"defense_reward_per_round"`
	FirstPatchBonus          int     `json:"first_patch_bonus"`
	FirstBloodBonusRatio     float64 `json:"first_blood_bonus_ratio"`
	VulnerabilityDecayFactor float64 `json:"vulnerability_decay_factor"`
	MaxTeamsPerGroup         int     `json:"max_teams_per_group"`
	JudgeChainImage          string  `json:"judge_chain_image"`
	TeamChainImage           string  `json:"team_chain_image"`
}

// ChallengeChainAccount 题目链预置账户。
type ChallengeChainAccount struct {
	Name    string `json:"name"`
	Balance string `json:"balance"`
}

// ChallengePinnedContract Fork 题目允许绑定的真实合约。
type ChallengePinnedContract struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// ChallengeChainForkConfig Fork 模式链配置。
type ChallengeChainForkConfig struct {
	RPCURL               string                    `json:"rpc_url"`
	ChainID              int64                     `json:"chain_id"`
	BlockNumber          int64                     `json:"block_number"`
	ImpersonatedAccounts []string                  `json:"impersonated_accounts,omitempty"`
	PinnedContracts      []ChallengePinnedContract `json:"pinned_contracts,omitempty"`
}

// ChallengeChainConfig 智能合约题目链配置。
type ChallengeChainConfig struct {
	ChainType    string                    `json:"chain_type"`
	ChainVersion string                    `json:"chain_version"`
	BlockNumber  int64                     `json:"block_number"`
	Fork         *ChallengeChainForkConfig `json:"fork,omitempty"`
	Accounts     []ChallengeChainAccount   `json:"accounts"`
}

// ChallengeSetupTransaction 题目初始化交易定义。
type ChallengeSetupTransaction struct {
	From     string        `json:"from"`
	To       string        `json:"to"`
	Function string        `json:"function"`
	Args     []interface{} `json:"args"`
	Value    string        `json:"value"`
}

// ChallengeEnvironmentPort 非合约题目环境端口配置。
type ChallengeEnvironmentPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// ChallengeEnvironmentEnvVar 非合约题目环境变量配置。
type ChallengeEnvironmentEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ChallengeEnvironmentImageConfig 非合约题目镜像配置。
type ChallengeEnvironmentImageConfig struct {
	Image       string                       `json:"image"`
	Version     string                       `json:"version"`
	Ports       []ChallengeEnvironmentPort   `json:"ports"`
	EnvVars     []ChallengeEnvironmentEnvVar `json:"env_vars"`
	CPULimit    string                       `json:"cpu_limit"`
	MemoryLimit string                       `json:"memory_limit"`
}

// ChallengeEnvironmentConfig 非合约题目环境配置。
type ChallengeEnvironmentConfig struct {
	Images     []ChallengeEnvironmentImageConfig `json:"images"`
	InitScript *string                           `json:"init_script"`
}

// DifficultyRange 模板适用难度范围。
type DifficultyRange struct {
	Min int16 `json:"min"`
	Max int16 `json:"max"`
}

// ChallengeTemplateParameterDef 模板参数定义。
type ChallengeTemplateParameterDef struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`
	DefaultValue any      `json:"default"`
	Options      []string `json:"options,omitempty"`
}

// ChallengeTemplateParameters 模板参数元数据。
type ChallengeTemplateParameters struct {
	Params []ChallengeTemplateParameterDef `json:"params"`
}

// ChallengeTemplateVariant 模板预设变体。
type ChallengeTemplateVariant struct {
	Name                string                 `json:"name"`
	Params              map[string]interface{} `json:"params"`
	SuggestedDifficulty int16                  `json:"suggested_difficulty"`
}

// ChallengeReferenceEvent 模板参考安全事件。
type ChallengeReferenceEvent struct {
	Name string `json:"name"`
	Date string `json:"date"`
	Loss string `json:"loss"`
}

// VerificationAssertionResult 断言验证结果项。
type VerificationAssertionResult struct {
	Type     string `json:"type"`
	Target   string `json:"target,omitempty"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

// VerificationAssertionResults 断言验证结果汇总。
type VerificationAssertionResults struct {
	AllPassed       bool                          `json:"all_passed"`
	Results         []VerificationAssertionResult `json:"results"`
	ExecutionTimeMS *int                          `json:"execution_time_ms,omitempty"`
	TxHash          *string                       `json:"tx_hash,omitempty"`
}

// VerificationStepResult 题目预验证步骤结果。
type VerificationStepResult struct {
	Step       int                           `json:"step"`
	Name       string                        `json:"name"`
	Status     string                        `json:"status"`
	Detail     string                        `json:"detail"`
	Assertions []VerificationAssertionResult `json:"assertions,omitempty"`
	DurationMS *int                          `json:"duration_ms,omitempty"`
}

// ChallengeEnvironmentContainerState 题目环境内单个容器状态。
type ChallengeEnvironmentContainerState struct {
	Status string `json:"status"`
	Image  string `json:"image"`
}

// TeamChainContractItem 攻防赛队伍链已部署合约信息。
type TeamChainContractItem struct {
	ChallengeID  string `json:"challenge_id"`
	ContractName string `json:"contract_name"`
	Address      string `json:"address"`
	PatchVersion int    `json:"patch_version"`
	IsPatched    bool   `json:"is_patched"`
}

// CompetitionMonitorOverview 竞赛监控概览数据。
type CompetitionMonitorOverview struct {
	RegisteredTeams     int `json:"registered_teams"`
	ActiveTeams         int `json:"active_teams"`
	TotalSubmissions    int `json:"total_submissions"`
	CorrectSubmissions  int `json:"correct_submissions"`
	TotalEnvironments   int `json:"total_environments"`
	RunningEnvironments int `json:"running_environments"`
}

// CompetitionMonitorResourceUsage 竞赛监控资源占用数据。
type CompetitionMonitorResourceUsage struct {
	CPUUsed        string `json:"cpu_used"`
	CPUMax         string `json:"cpu_max"`
	MemoryUsed     string `json:"memory_used"`
	MemoryMax      string `json:"memory_max"`
	NamespacesUsed int    `json:"namespaces_used"`
	NamespacesMax  int    `json:"namespaces_max"`
}

// CompetitionMonitorChallengeStat 竞赛监控题目统计项。
type CompetitionMonitorChallengeStat struct {
	ChallengeID         string  `json:"challenge_id"`
	Title               string  `json:"title"`
	Category            string  `json:"category"`
	SolveCount          int     `json:"solve_count"`
	AttemptCount        int     `json:"attempt_count"`
	SolveRate           float64 `json:"solve_rate"`
	CurrentScore        *int    `json:"current_score"`
	EnvironmentsRunning int     `json:"environments_running"`
}

// CompetitionMonitorRecentSubmission 竞赛监控最近提交项。
type CompetitionMonitorRecentSubmission struct {
	TeamName       string `json:"team_name"`
	ChallengeTitle string `json:"challenge_title"`
	IsCorrect      bool   `json:"is_correct"`
	SubmittedAt    string `json:"submitted_at"`
}

// CompetitionStatisticsSummary 竞赛统计摘要。
type CompetitionStatisticsSummary struct {
	TotalTeams        int     `json:"total_teams"`
	TotalParticipants int     `json:"total_participants"`
	TotalSubmissions  int     `json:"total_submissions"`
	TotalCorrect      int     `json:"total_correct"`
	OverallSolveRate  float64 `json:"overall_solve_rate"`
	AverageScore      int     `json:"average_score"`
	HighestScore      int     `json:"highest_score"`
	LowestScore       int     `json:"lowest_score"`
}

// CompetitionStatisticsChallengeItem 竞赛统计题目项。
type CompetitionStatisticsChallengeItem struct {
	ChallengeID             string  `json:"challenge_id"`
	Title                   string  `json:"title"`
	Category                string  `json:"category"`
	Difficulty              int16   `json:"difficulty"`
	DifficultyText          string  `json:"difficulty_text"`
	SolveCount              int     `json:"solve_count"`
	AttemptCount            int     `json:"attempt_count"`
	SolveRate               float64 `json:"solve_rate"`
	FirstBloodTeam          *string `json:"first_blood_team"`
	FirstBloodTimeMinutes   *int    `json:"first_blood_time_minutes"`
	AverageSolveTimeMinutes *int    `json:"average_solve_time_minutes"`
}

// CompetitionScoreDistributionRange 分数分布区间。
type CompetitionScoreDistributionRange struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// CompetitionScoreDistribution 竞赛分数分布。
type CompetitionScoreDistribution struct {
	Ranges []CompetitionScoreDistributionRange `json:"ranges"`
}

// CompetitionTimelinePoint 竞赛时间线数据点。
type CompetitionTimelinePoint struct {
	Hour  string `json:"hour"`
	Count int    `json:"count"`
}

// CompetitionTimeline 竞赛时间线统计。
type CompetitionTimeline struct {
	SubmissionsPerHour []CompetitionTimelinePoint `json:"submissions_per_hour"`
}

// LeaderboardRankingItem 排行榜项。
type LeaderboardRankingItem struct {
	Rank               int     `json:"rank"`
	TeamID             string  `json:"team_id"`
	TeamName           string  `json:"team_name"`
	Score              *int    `json:"score,omitempty"`
	SolveCount         *int    `json:"solve_count,omitempty"`
	LastSolveAt        *string `json:"last_solve_at,omitempty"`
	TokenBalance       *int    `json:"token_balance,omitempty"`
	AttacksSuccessful  *int    `json:"attacks_successful,omitempty"`
	DefensesSuccessful *int    `json:"defenses_successful,omitempty"`
	PatchesAccepted    *int    `json:"patches_accepted,omitempty"`
}

// FinalLeaderboardMember 最终排名中的团队成员。
type FinalLeaderboardMember struct {
	Name     string `json:"name"`
	RoleText string `json:"role_text"`
}

// FinalLeaderboardSolvedChallenge 最终排名中的已解题目项。
type FinalLeaderboardSolvedChallenge struct {
	ChallengeID  string `json:"challenge_id"`
	Title        string `json:"title"`
	Score        int    `json:"score"`
	SolvedAt     string `json:"solved_at"`
	IsFirstBlood bool   `json:"is_first_blood"`
}

// FinalLeaderboardItem 最终排名项。
type FinalLeaderboardItem struct {
	Rank               int                               `json:"rank"`
	TeamID             string                            `json:"team_id"`
	TeamName           string                            `json:"team_name"`
	Score              *int                              `json:"score,omitempty"`
	SolveCount         *int                              `json:"solve_count,omitempty"`
	LastSolveAt        *string                           `json:"last_solve_at,omitempty"`
	TokenBalance       *int                              `json:"token_balance,omitempty"`
	AttacksSuccessful  *int                              `json:"attacks_successful,omitempty"`
	DefensesSuccessful *int                              `json:"defenses_successful,omitempty"`
	PatchesAccepted    *int                              `json:"patches_accepted,omitempty"`
	Members            []FinalLeaderboardMember          `json:"members,omitempty"`
	SolvedChallenges   []FinalLeaderboardSolvedChallenge `json:"solved_challenges,omitempty"`
}

// CompetitionResultsSummary 竞赛最终结果摘要。
type CompetitionResultsSummary struct {
	TotalTeams        int     `json:"total_teams"`
	TotalParticipants int     `json:"total_participants"`
	TotalSubmissions  int     `json:"total_submissions"`
	TotalCorrect      int     `json:"total_correct"`
	OverallSolveRate  float64 `json:"overall_solve_rate"`
	AverageScore      int     `json:"average_score"`
	HighestScore      int     `json:"highest_score"`
	LowestScore       int     `json:"lowest_score"`
}

// AdminTotalResourceUsage 平台级竞赛总资源占用。
type AdminTotalResourceUsage struct {
	CPUUsed          string `json:"cpu_used"`
	MemoryUsed       string `json:"memory_used"`
	NamespacesActive int    `json:"namespaces_active"`
}

// RunningCompetitionOverviewItem 运行中竞赛概览项。
type RunningCompetitionOverviewItem struct {
	ID                  string  `json:"id"`
	Title               string  `json:"title"`
	CompetitionType     int16   `json:"competition_type"`
	CompetitionTypeText string  `json:"competition_type_text"`
	Status              int16   `json:"status"`
	StatusText          string  `json:"status_text"`
	Teams               int     `json:"teams"`
	EnvironmentsRunning int     `json:"environments_running"`
	StartAt             *string `json:"start_at"`
	EndAt               *string `json:"end_at"`
}

// AdminCompetitionAlert 平台级竞赛告警项。
type AdminCompetitionAlert struct {
	Type          string `json:"type"`
	Message       string `json:"message"`
	CompetitionID string `json:"competition_id"`
	CreatedAt     string `json:"created_at"`
}

// ========== 竞赛管理 ==========

// CreateCompetitionReq 创建竞赛请求。
type CreateCompetitionReq struct {
	Title               string                     `json:"title" binding:"required,max=200"`
	Description         *string                    `json:"description"`
	BannerURL           *string                    `json:"banner_url" binding:"omitempty,url,max=500"`
	CompetitionType     int16                      `json:"competition_type" binding:"required,oneof=1 2"`
	Scope               int16                      `json:"scope" binding:"required,oneof=1 2"`
	SchoolID            *string                    `json:"school_id"`
	TeamMode            int16                      `json:"team_mode" binding:"required,oneof=1 2 3"`
	MaxTeamSize         int                        `json:"max_team_size" binding:"required,min=1"`
	MinTeamSize         int                        `json:"min_team_size" binding:"required,min=1"`
	MaxTeams            *int                       `json:"max_teams" binding:"omitempty,min=1"`
	RegistrationStartAt *string                    `json:"registration_start_at"`
	RegistrationEndAt   *string                    `json:"registration_end_at"`
	StartAt             *string                    `json:"start_at"`
	EndAt               *string                    `json:"end_at"`
	FreezeAt            *string                    `json:"freeze_at"`
	Rules               *string                    `json:"rules"`
	JeopardyConfig      *JeopardyCompetitionConfig `json:"jeopardy_config"`
	AdConfig            *ADCompetitionConfig       `json:"ad_config"`
}

// UpdateCompetitionReq 编辑竞赛请求。
type UpdateCompetitionReq struct {
	Title               *string                    `json:"title" binding:"omitempty,max=200"`
	Description         *string                    `json:"description"`
	BannerURL           *string                    `json:"banner_url" binding:"omitempty,url,max=500"`
	Scope               *int16                     `json:"scope" binding:"omitempty,oneof=1 2"`
	SchoolID            *string                    `json:"school_id"`
	TeamMode            *int16                     `json:"team_mode" binding:"omitempty,oneof=1 2 3"`
	MaxTeamSize         *int                       `json:"max_team_size" binding:"omitempty,min=1"`
	MinTeamSize         *int                       `json:"min_team_size" binding:"omitempty,min=1"`
	MaxTeams            *int                       `json:"max_teams" binding:"omitempty,min=1"`
	RegistrationStartAt *string                    `json:"registration_start_at"`
	RegistrationEndAt   *string                    `json:"registration_end_at"`
	StartAt             *string                    `json:"start_at"`
	EndAt               *string                    `json:"end_at"`
	FreezeAt            *string                    `json:"freeze_at"`
	Rules               *string                    `json:"rules"`
	JeopardyConfig      *JeopardyCompetitionConfig `json:"jeopardy_config"`
	AdConfig            *ADCompetitionConfig       `json:"ad_config"`
}

// CompetitionListReq 竞赛列表查询参数。
type CompetitionListReq struct {
	Page            int    `form:"page" binding:"omitempty,min=1"`
	PageSize        int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	CompetitionType int16  `form:"competition_type" binding:"omitempty,oneof=1 2"`
	Scope           int16  `form:"scope" binding:"omitempty,oneof=1 2"`
	Status          int16  `form:"status" binding:"omitempty,oneof=1 2 3 4 5"`
	Keyword         string `form:"keyword"`
}

// CompetitionListItem 竞赛列表项。
type CompetitionListItem struct {
	ID                  string  `json:"id"`
	Title               string  `json:"title"`
	BannerURL           *string `json:"banner_url"`
	CompetitionType     int16   `json:"competition_type"`
	CompetitionTypeText string  `json:"competition_type_text"`
	Scope               int16   `json:"scope"`
	ScopeText           string  `json:"scope_text"`
	TeamMode            int16   `json:"team_mode"`
	TeamModeText        string  `json:"team_mode_text"`
	MaxTeamSize         int     `json:"max_team_size"`
	Status              int16   `json:"status"`
	StatusText          string  `json:"status_text"`
	RegisteredTeams     int     `json:"registered_teams"`
	MaxTeams            *int    `json:"max_teams"`
	ChallengeCount      int     `json:"challenge_count"`
	RegistrationStartAt *string `json:"registration_start_at"`
	RegistrationEndAt   *string `json:"registration_end_at"`
	StartAt             *string `json:"start_at"`
	EndAt               *string `json:"end_at"`
	CreatedByName       string  `json:"created_by_name"`
}

// CompetitionDetailResp 竞赛详情响应。
type CompetitionDetailResp struct {
	ID                  string                     `json:"id"`
	Title               string                     `json:"title"`
	Description         *string                    `json:"description"`
	BannerURL           *string                    `json:"banner_url"`
	CompetitionType     int16                      `json:"competition_type"`
	CompetitionTypeText string                     `json:"competition_type_text"`
	Scope               int16                      `json:"scope"`
	ScopeText           string                     `json:"scope_text"`
	TeamMode            int16                      `json:"team_mode"`
	TeamModeText        string                     `json:"team_mode_text"`
	MaxTeamSize         int                        `json:"max_team_size"`
	MinTeamSize         int                        `json:"min_team_size"`
	MaxTeams            *int                       `json:"max_teams"`
	Status              int16                      `json:"status"`
	StatusText          string                     `json:"status_text"`
	RegistrationStartAt *string                    `json:"registration_start_at"`
	RegistrationEndAt   *string                    `json:"registration_end_at"`
	StartAt             *string                    `json:"start_at"`
	EndAt               *string                    `json:"end_at"`
	FreezeAt            *string                    `json:"freeze_at"`
	Rules               *string                    `json:"rules"`
	JeopardyConfig      *JeopardyCompetitionConfig `json:"jeopardy_config"`
	AdConfig            *ADCompetitionConfig       `json:"ad_config"`
	RegisteredTeams     int                        `json:"registered_teams"`
	ChallengeCount      int                        `json:"challenge_count"`
	CreatedBy           *CTFUserBrief              `json:"created_by"`
	CreatedAt           string                     `json:"created_at"`
}

// CompetitionCreateResp 创建竞赛响应。
type CompetitionCreateResp struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	CompetitionType     int16  `json:"competition_type"`
	CompetitionTypeText string `json:"competition_type_text"`
	Scope               int16  `json:"scope"`
	ScopeText           string `json:"scope_text"`
	TeamMode            int16  `json:"team_mode"`
	TeamModeText        string `json:"team_mode_text"`
	Status              int16  `json:"status"`
	StatusText          string `json:"status_text"`
	CreatedAt           string `json:"created_at"`
}

// CompetitionStatusResp 竞赛状态响应。
type CompetitionStatusResp struct {
	ID         string `json:"id"`
	Status     int16  `json:"status"`
	StatusText string `json:"status_text"`
}

// TerminateCompetitionReq 强制终止竞赛请求。
type TerminateCompetitionReq struct {
	Reason string `json:"reason" binding:"required"`
}

// TerminateCompetitionResp 强制终止竞赛响应。
type TerminateCompetitionResp struct {
	ID                    string `json:"id"`
	Status                int16  `json:"status"`
	StatusText            string `json:"status_text"`
	EnvironmentsDestroyed int    `json:"environments_destroyed"`
}

// ========== 题目管理 ==========

// CreateChallengeReq 创建题目请求。
type CreateChallengeReq struct {
	Title             string                      `json:"title" binding:"required,max=200"`
	Description       string                      `json:"description" binding:"required"`
	Category          string                      `json:"category" binding:"required"`
	Difficulty        int16                       `json:"difficulty" binding:"required,oneof=1 2 3 4 5"`
	BaseScore         int                         `json:"base_score" binding:"required,min=1"`
	FlagType          int16                       `json:"flag_type" binding:"required,oneof=1 2 3"`
	StaticFlag        *string                     `json:"static_flag"`
	DynamicFlagSecret *string                     `json:"dynamic_flag_secret"`
	RuntimeMode       *int16                      `json:"runtime_mode" binding:"omitempty,oneof=1 2"`
	ChainConfig       *ChallengeChainConfig       `json:"chain_config"`
	SetupTransactions []ChallengeSetupTransaction `json:"setup_transactions"`
	SourcePath        *int16                      `json:"source_path" binding:"omitempty,oneof=1 2 3"`
	SwcID             *string                     `json:"swc_id"`
	TemplateID        *string                     `json:"template_id"`
	TemplateParams    map[string]interface{}      `json:"template_params"`
	EnvironmentConfig *ChallengeEnvironmentConfig `json:"environment_config"`
	AttachmentURLs    []string                    `json:"attachment_urls"`
}

// UpdateChallengeReq 编辑题目请求。
type UpdateChallengeReq struct {
	Title             *string                     `json:"title" binding:"omitempty,max=200"`
	Description       *string                     `json:"description"`
	Category          *string                     `json:"category"`
	Difficulty        *int16                      `json:"difficulty" binding:"omitempty,oneof=1 2 3 4 5"`
	BaseScore         *int                        `json:"base_score" binding:"omitempty,min=1"`
	FlagType          *int16                      `json:"flag_type" binding:"omitempty,oneof=1 2 3"`
	StaticFlag        *string                     `json:"static_flag"`
	DynamicFlagSecret *string                     `json:"dynamic_flag_secret"`
	RuntimeMode       *int16                      `json:"runtime_mode" binding:"omitempty,oneof=1 2"`
	ChainConfig       *ChallengeChainConfig       `json:"chain_config"`
	SetupTransactions []ChallengeSetupTransaction `json:"setup_transactions"`
	EnvironmentConfig *ChallengeEnvironmentConfig `json:"environment_config"`
	AttachmentURLs    []string                    `json:"attachment_urls"`
	IsPublic          *bool                       `json:"is_public"`
}

// ChallengeListReq 题目列表查询参数。
type ChallengeListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Category   string `form:"category"`
	Difficulty int16  `form:"difficulty" binding:"omitempty,oneof=1 2 3 4 5"`
	FlagType   int16  `form:"flag_type" binding:"omitempty,oneof=1 2 3"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
	IsPublic   *bool  `form:"is_public"`
	Keyword    string `form:"keyword"`
	AuthorID   string `form:"author_id"`
}

// ChallengeAuthorBrief 题目作者摘要。
type ChallengeAuthorBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ChallengeListItem 题目列表项。
type ChallengeListItem struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	Category        string                `json:"category"`
	CategoryText    string                `json:"category_text"`
	Difficulty      int16                 `json:"difficulty"`
	DifficultyText  string                `json:"difficulty_text"`
	BaseScore       int                   `json:"base_score"`
	FlagType        int16                 `json:"flag_type"`
	FlagTypeText    string                `json:"flag_type_text"`
	RuntimeMode     *int16                `json:"runtime_mode"`
	RuntimeModeText *string               `json:"runtime_mode_text"`
	SourcePath      *int16                `json:"source_path"`
	SourcePathText  *string               `json:"source_path_text"`
	Status          int16                 `json:"status"`
	StatusText      string                `json:"status_text"`
	IsPublic        bool                  `json:"is_public"`
	UsageCount      int                   `json:"usage_count"`
	Author          *ChallengeAuthorBrief `json:"author"`
	CreatedAt       string                `json:"created_at"`
}

// ChallengeContractItem 题目合约项。
type ChallengeContractItem struct {
	ID              string                   `json:"id"`
	ChallengeID     *string                  `json:"challenge_id,omitempty"`
	Name            string                   `json:"name"`
	SourceCode      *string                  `json:"source_code,omitempty"`
	ABI             []map[string]interface{} `json:"abi,omitempty"`
	Bytecode        *string                  `json:"bytecode,omitempty"`
	ConstructorArgs []interface{}            `json:"constructor_args,omitempty"`
	DeployOrder     int                      `json:"deploy_order"`
}

// ChallengeAssertionItem 题目断言项。
type ChallengeAssertionItem struct {
	ID            string                 `json:"id"`
	ChallengeID   *string                `json:"challenge_id,omitempty"`
	AssertionType string                 `json:"assertion_type"`
	Target        string                 `json:"target"`
	Operator      string                 `json:"operator"`
	ExpectedValue string                 `json:"expected_value"`
	Description   *string                `json:"description"`
	ExtraParams   map[string]interface{} `json:"extra_params"`
	SortOrder     int                    `json:"sort_order"`
}

// VerificationSummary 预验证摘要。
type VerificationSummary struct {
	ID          string  `json:"id"`
	Status      int16   `json:"status"`
	StatusText  string  `json:"status_text"`
	CompletedAt *string `json:"completed_at"`
}

// ChallengeDetailResp 题目详情响应。
type ChallengeDetailResp struct {
	ID                 string                      `json:"id"`
	Title              string                      `json:"title"`
	Description        string                      `json:"description"`
	Category           string                      `json:"category"`
	CategoryText       string                      `json:"category_text"`
	Difficulty         int16                       `json:"difficulty"`
	DifficultyText     string                      `json:"difficulty_text"`
	BaseScore          int                         `json:"base_score"`
	FlagType           int16                       `json:"flag_type"`
	FlagTypeText       string                      `json:"flag_type_text"`
	RuntimeMode        *int16                      `json:"runtime_mode"`
	RuntimeModeText    *string                     `json:"runtime_mode_text"`
	ChainConfig        *ChallengeChainConfig       `json:"chain_config"`
	SetupTransactions  []ChallengeSetupTransaction `json:"setup_transactions"`
	SourcePath         *int16                      `json:"source_path"`
	SourcePathText     *string                     `json:"source_path_text"`
	SwcID              *string                     `json:"swc_id"`
	TemplateID         *string                     `json:"template_id"`
	EnvironmentConfig  *ChallengeEnvironmentConfig `json:"environment_config"`
	AttachmentURLs     []string                    `json:"attachment_urls"`
	Status             int16                       `json:"status"`
	StatusText         string                      `json:"status_text"`
	IsPublic           bool                        `json:"is_public"`
	UsageCount         int                         `json:"usage_count"`
	Contracts          []ChallengeContractItem     `json:"contracts"`
	Assertions         []ChallengeAssertionItem    `json:"assertions"`
	LatestVerification *VerificationSummary        `json:"latest_verification"`
	Author             *ChallengeAuthorBrief       `json:"author"`
	CreatedAt          string                      `json:"created_at"`
	UpdatedAt          string                      `json:"updated_at"`
}

// ChallengeStatusResp 题目状态响应。
type ChallengeStatusResp struct {
	ID                  string  `json:"id"`
	Title               string  `json:"title"`
	Category            string  `json:"category,omitempty"`
	CategoryText        *string `json:"category_text,omitempty"`
	Difficulty          int16   `json:"difficulty,omitempty"`
	DifficultyText      *string `json:"difficulty_text,omitempty"`
	BaseScore           int     `json:"base_score,omitempty"`
	FlagType            int16   `json:"flag_type,omitempty"`
	FlagTypeText        *string `json:"flag_type_text,omitempty"`
	RuntimeMode         *int16  `json:"runtime_mode,omitempty"`
	RuntimeModeText     *string `json:"runtime_mode_text,omitempty"`
	SourcePath          *int16  `json:"source_path,omitempty"`
	SourcePathText      *string `json:"source_path_text,omitempty"`
	SwcID               *string `json:"swc_id,omitempty"`
	TemplateID          *string `json:"template_id,omitempty"`
	Status              int16   `json:"status"`
	StatusText          string  `json:"status_text"`
	CreatedAt           *string `json:"created_at,omitempty"`
	ContractsGenerated  *int    `json:"contracts_generated,omitempty"`
	AssertionsGenerated *int    `json:"assertions_generated,omitempty"`
}

// SubmitChallengeReviewResp 提交题目审核响应。
type SubmitChallengeReviewResp struct {
	ID         string `json:"id"`
	Status     int16  `json:"status"`
	StatusText string `json:"status_text"`
}

// ========== 合约与断言管理 ==========

// CreateChallengeContractReq 添加合约请求。
type CreateChallengeContractReq struct {
	Name            string                   `json:"name" binding:"required,max=100"`
	SourceCode      string                   `json:"source_code" binding:"required"`
	ABI             []map[string]interface{} `json:"abi" binding:"required"`
	Bytecode        string                   `json:"bytecode" binding:"required"`
	ConstructorArgs []interface{}            `json:"constructor_args"`
	DeployOrder     int                      `json:"deploy_order" binding:"omitempty,min=1"`
}

// UpdateChallengeContractReq 编辑合约请求。
type UpdateChallengeContractReq struct {
	Name            *string                  `json:"name" binding:"omitempty,max=100"`
	SourceCode      *string                  `json:"source_code"`
	ABI             []map[string]interface{} `json:"abi"`
	Bytecode        *string                  `json:"bytecode"`
	ConstructorArgs []interface{}            `json:"constructor_args"`
	DeployOrder     *int                     `json:"deploy_order" binding:"omitempty,min=1"`
}

// ChallengeContractResp 合约响应。
type ChallengeContractResp struct {
	ID          string `json:"id"`
	ChallengeID string `json:"challenge_id"`
	Name        string `json:"name"`
	DeployOrder int    `json:"deploy_order"`
}

// CreateChallengeAssertionReq 添加断言请求。
type CreateChallengeAssertionReq struct {
	AssertionType string                 `json:"assertion_type" binding:"required"`
	Target        string                 `json:"target" binding:"required,max=200"`
	Operator      string                 `json:"operator" binding:"required"`
	ExpectedValue string                 `json:"expected_value" binding:"required"`
	Description   *string                `json:"description" binding:"omitempty,max=500"`
	ExtraParams   map[string]interface{} `json:"extra_params"`
	SortOrder     int                    `json:"sort_order" binding:"omitempty,min=0"`
}

// UpdateChallengeAssertionReq 编辑断言请求。
type UpdateChallengeAssertionReq struct {
	AssertionType *string                `json:"assertion_type"`
	Target        *string                `json:"target" binding:"omitempty,max=200"`
	Operator      *string                `json:"operator"`
	ExpectedValue *string                `json:"expected_value"`
	Description   *string                `json:"description" binding:"omitempty,max=500"`
	ExtraParams   map[string]interface{} `json:"extra_params"`
	SortOrder     *int                   `json:"sort_order" binding:"omitempty,min=0"`
}

// ChallengeAssertionResp 断言响应。
type ChallengeAssertionResp struct {
	ID            string `json:"id"`
	ChallengeID   string `json:"challenge_id"`
	AssertionType string `json:"assertion_type"`
	Target        string `json:"target"`
	Operator      string `json:"operator"`
	ExpectedValue string `json:"expected_value"`
	SortOrder     int    `json:"sort_order"`
}

// SortChallengeAssertionReq 断言排序请求。
type SortChallengeAssertionReq struct {
	Items []SortItemReq `json:"items" binding:"required,min=1,dive"`
}

// ========== 漏洞转化 ==========

// SWCRegistryListReq SWC Registry 列表查询参数。
type SWCRegistryListReq struct {
	Keyword string `form:"keyword"`
}

// SWCRegistryItem SWC Registry 列表项。
type SWCRegistryItem struct {
	SwcID               string `json:"swc_id"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	Severity            string `json:"severity"`
	HasExample          bool   `json:"has_example"`
	SuggestedDifficulty int16  `json:"suggested_difficulty"`
}

// ImportSWCChallengeReq 从SWC导入生成题目请求。
type ImportSWCChallengeReq struct {
	SwcID      string `json:"swc_id" binding:"required"`
	Title      string `json:"title" binding:"required,max=200"`
	Difficulty int16  `json:"difficulty" binding:"required,oneof=1 2 3 4 5"`
	BaseScore  int    `json:"base_score" binding:"required,min=1"`
}

// ChallengeTemplateListReq 模板列表查询参数。
type ChallengeTemplateListReq struct {
	VulnerabilityType string `form:"vulnerability_type"`
	Keyword           string `form:"keyword"`
}

// ChallengeTemplateListItem 模板列表项。
type ChallengeTemplateListItem struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Code              string          `json:"code"`
	VulnerabilityType string          `json:"vulnerability_type"`
	Description       string          `json:"description"`
	DifficultyRange   DifficultyRange `json:"difficulty_range"`
	VariantCount      int             `json:"variant_count"`
	UsageCount        int             `json:"usage_count"`
}

// ChallengeTemplateDetailResp 模板详情响应。
type ChallengeTemplateDetailResp struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	Code              string                      `json:"code"`
	Description       string                      `json:"description"`
	VulnerabilityType string                      `json:"vulnerability_type"`
	DifficultyRange   DifficultyRange             `json:"difficulty_range"`
	Parameters        ChallengeTemplateParameters `json:"parameters"`
	Variants          []ChallengeTemplateVariant  `json:"variants"`
	ReferenceEvents   []ChallengeReferenceEvent   `json:"reference_events"`
	UsageCount        int                         `json:"usage_count"`
}

// GenerateChallengeFromTemplateReq 从模板生成题目请求。
type GenerateChallengeFromTemplateReq struct {
	TemplateID     string                 `json:"template_id" binding:"required"`
	Title          string                 `json:"title" binding:"required,max=200"`
	Difficulty     int16                  `json:"difficulty" binding:"required,oneof=1 2 3 4 5"`
	BaseScore      int                    `json:"base_score" binding:"required,min=1"`
	TemplateParams map[string]interface{} `json:"template_params" binding:"required"`
}

// ImportExternalVulnerabilityReq 从外部真实漏洞源导入题目草稿请求。
// POST /api/v1/ctf/challenges/import-external
type ImportExternalVulnerabilityReq struct {
	SourceGrade          string                      `json:"source_grade" binding:"required,oneof=A B C"`
	Title                string                      `json:"title" binding:"required,max=200"`
	VulnerabilityName    string                      `json:"vulnerability_name" binding:"required,max=200"`
	SourceURL            string                      `json:"source_url" binding:"required,max=500"`
	ConfidenceScore      float64                     `json:"confidence_score" binding:"omitempty,min=0,max=1"`
	ReproducibilityScore float64                     `json:"reproducibility_score" binding:"omitempty,min=0,max=1"`
	Category             string                      `json:"category" binding:"required"`
	Difficulty           int16                       `json:"difficulty" binding:"required,oneof=1 2 3 4 5"`
	BaseScore            int                         `json:"base_score" binding:"required,min=1"`
	ChainConfig          *ChallengeChainConfig       `json:"chain_config"`
	SourceCode           *string                     `json:"source_code"`
	PocContent           *string                     `json:"poc_content"`
	SetupTransactions    []ChallengeSetupTransaction `json:"setup_transactions"`
	ReferenceEvent       map[string]interface{}      `json:"reference_event"`
}

// ========== 预验证与审核 ==========

// VerifyChallengeReq 发起预验证请求。
type VerifyChallengeReq struct {
	PocContent  string  `json:"poc_content" binding:"required"`
	PocLanguage *string `json:"poc_language" binding:"omitempty,oneof=solidity javascript python"`
}

// VerifyChallengeResp 发起预验证响应。
type VerifyChallengeResp struct {
	VerificationID string `json:"verification_id"`
	ChallengeID    string `json:"challenge_id"`
	Status         int16  `json:"status"`
	StatusText     string `json:"status_text"`
	StartedAt      string `json:"started_at"`
}

// VerificationListItem 预验证列表项。
type VerificationListItem struct {
	ID           string  `json:"id"`
	Status       int16   `json:"status"`
	StatusText   string  `json:"status_text"`
	PocLanguage  *string `json:"poc_language"`
	StartedAt    string  `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
	ErrorMessage *string `json:"error_message"`
}

// VerificationDetailResp 预验证详情响应。
type VerificationDetailResp struct {
	ID            string                   `json:"id"`
	ChallengeID   string                   `json:"challenge_id"`
	Status        int16                    `json:"status"`
	StatusText    string                   `json:"status_text"`
	StepResults   []VerificationStepResult `json:"step_results"`
	PocContent    *string                  `json:"poc_content"`
	PocLanguage   *string                  `json:"poc_language"`
	EnvironmentID *string                  `json:"environment_id"`
	ErrorMessage  *string                  `json:"error_message"`
	StartedAt     string                   `json:"started_at"`
	CompletedAt   *string                  `json:"completed_at"`
	CreatedAt     string                   `json:"created_at"`
}

// ChallengeReviewListItem 待审核题目列表项。
type ChallengeReviewListItem struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	CategoryText   string `json:"category_text"`
	Difficulty     int16  `json:"difficulty"`
	DifficultyText string `json:"difficulty_text"`
	AuthorName     string `json:"author_name"`
	SchoolName     string `json:"school_name"`
	SubmittedAt    string `json:"submitted_at"`
}

// ReviewChallengeReq 审核题目请求。
type ReviewChallengeReq struct {
	Action  int16   `json:"action" binding:"required,oneof=1 2"`
	Comment *string `json:"comment"`
}

// ChallengeReviewResp 题目审核记录响应。
type ChallengeReviewResp struct {
	ID           string  `json:"id"`
	ChallengeID  string  `json:"challenge_id"`
	ReviewerID   string  `json:"reviewer_id"`
	ReviewerName *string `json:"reviewer_name,omitempty"`
	Action       int16   `json:"action"`
	ActionText   string  `json:"action_text"`
	Comment      *string `json:"comment"`
	CreatedAt    string  `json:"created_at"`
}

// ReviewChallengeActionResp 题目审核动作响应。
type ReviewChallengeActionResp struct {
	ChallengeID string `json:"challenge_id"`
	Status      int16  `json:"status"`
	StatusText  string `json:"status_text"`
	ReviewID    string `json:"review_id"`
}

// ========== 竞赛题目配置 ==========

// CompetitionChallengeSummary 竞赛题目概要信息。
type CompetitionChallengeSummary struct {
	ID                 string                      `json:"id"`
	Title              string                      `json:"title"`
	Description        string                      `json:"description"`
	Category           string                      `json:"category"`
	CategoryText       string                      `json:"category_text"`
	Difficulty         int16                       `json:"difficulty"`
	DifficultyText     string                      `json:"difficulty_text"`
	FlagType           int16                       `json:"flag_type"`
	FlagTypeText       string                      `json:"flag_type_text"`
	AttachmentURLs     []string                    `json:"attachment_urls"`
	ChainConfig        *ChallengeChainConfig       `json:"chain_config,omitempty"`
	SourcePath         *int16                      `json:"source_path,omitempty"`
	SourcePathText     *string                     `json:"source_path_text,omitempty"`
	SwcID              *string                     `json:"swc_id,omitempty"`
	TemplateID         *string                     `json:"template_id,omitempty"`
	EnvironmentConfig  *ChallengeEnvironmentConfig `json:"environment_config,omitempty"`
	Status             *int16                      `json:"status,omitempty"`
	StatusText         *string                     `json:"status_text,omitempty"`
	IsPublic           *bool                       `json:"is_public,omitempty"`
	UsageCount         *int                        `json:"usage_count,omitempty"`
	Contracts          []ChallengeContractItem     `json:"contracts,omitempty"`
	Assertions         []ChallengeAssertionItem    `json:"assertions,omitempty"`
	LatestVerification *VerificationSummary        `json:"latest_verification,omitempty"`
}

// CompetitionChallengeFirstBloodTeam 竞赛题目首解团队信息。
type CompetitionChallengeFirstBloodTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CompetitionChallengeListItem 竞赛题目列表项。
type CompetitionChallengeListItem struct {
	ID                string                              `json:"id"`
	Challenge         CompetitionChallengeSummary         `json:"challenge"`
	BaseScore         int                                 `json:"base_score"`
	CurrentScore      *int                                `json:"current_score"`
	SolveCount        int                                 `json:"solve_count"`
	FirstBloodTeam    *CompetitionChallengeFirstBloodTeam `json:"first_blood_team,omitempty"`
	FirstBloodAt      *string                             `json:"first_blood_at,omitempty"`
	SortOrder         int                                 `json:"sort_order"`
	MyTeamSolved      bool                                `json:"my_team_solved"`
	MyTeamEnvironment *string                             `json:"my_team_environment"`
}

// AddCompetitionChallengeReq 添加题目到竞赛请求。
type AddCompetitionChallengeReq struct {
	ChallengeIDs []string `json:"challenge_ids" binding:"required,min=1"`
}

// AddedCompetitionChallengeItem 添加到竞赛的题目项。
type AddedCompetitionChallengeItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SortOrder int    `json:"sort_order"`
}

// AddCompetitionChallengeResp 添加题目到竞赛响应。
type AddCompetitionChallengeResp struct {
	AddedCount    int                             `json:"added_count"`
	CompetitionID string                          `json:"competition_id"`
	Challenges    []AddedCompetitionChallengeItem `json:"challenges"`
}

// SortCompetitionChallengesReq 竞赛题目排序请求。
type SortCompetitionChallengesReq struct {
	Items []SortItemReq `json:"items" binding:"required,min=1,dive"`
}

// ========== 团队与报名 ==========

// CreateTeamReq 创建团队请求。
type CreateTeamReq struct {
	Name string `json:"name" binding:"required,max=100"`
}

// TeamMemberResp 团队成员响应。
type TeamMemberResp struct {
	StudentID string `json:"student_id"`
	Name      string `json:"name"`
	Role      int16  `json:"role"`
	RoleText  string `json:"role_text"`
	JoinedAt  string `json:"joined_at"`
}

// TeamResp 团队响应。
type TeamResp struct {
	ID            string           `json:"id"`
	CompetitionID string           `json:"competition_id"`
	Name          string           `json:"name"`
	CaptainID     string           `json:"captain_id"`
	InviteCode    *string          `json:"invite_code"`
	Status        int16            `json:"status"`
	StatusText    string           `json:"status_text"`
	Members       []TeamMemberResp `json:"members,omitempty"`
}

// TeamListItem 团队列表项。
type TeamListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CaptainName string `json:"captain_name"`
	MemberCount int    `json:"member_count"`
	Status      int16  `json:"status"`
	StatusText  string `json:"status_text"`
	Registered  bool   `json:"registered"`
	FinalRank   *int   `json:"final_rank"`
	TotalScore  *int   `json:"total_score"`
}

// UpdateTeamReq 编辑团队信息请求。
type UpdateTeamReq struct {
	Name string `json:"name" binding:"required,max=100"`
}

// JoinTeamResp 通过邀请码加入团队响应。
type JoinTeamResp struct {
	TeamID         string `json:"team_id"`
	TeamName       string `json:"team_name"`
	CompetitionID  string `json:"competition_id"`
	Role           int16  `json:"role"`
	RoleText       string `json:"role_text"`
	CurrentMembers int    `json:"current_members"`
	MaxTeamSize    int    `json:"max_team_size"`
}

// JoinTeamReq 通过邀请码加入团队请求。
type JoinTeamReq struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

// RegisterCompetitionReq 报名竞赛请求。
type RegisterCompetitionReq struct {
	TeamID *string `json:"team_id"`
}

// RegistrationResp 报名响应。
type RegistrationResp struct {
	RegistrationID string `json:"registration_id"`
	CompetitionID  string `json:"competition_id"`
	TeamID         string `json:"team_id"`
	TeamName       string `json:"team_name"`
	Status         int16  `json:"status"`
	StatusText     string `json:"status_text"`
	RegisteredAt   string `json:"registered_at"`
}

// MyRegistrationResp 我的报名状态响应。
type MyRegistrationResp struct {
	IsRegistered   bool    `json:"is_registered"`
	RegistrationID *string `json:"registration_id"`
	TeamID         *string `json:"team_id"`
	TeamName       *string `json:"team_name"`
	Status         *int16  `json:"status"`
	StatusText     *string `json:"status_text"`
	RegisteredAt   *string `json:"registered_at"`
}

// RegistrationListItem 报名列表项。
type RegistrationListItem struct {
	ID           string `json:"id"`
	TeamID       string `json:"team_id"`
	TeamName     string `json:"team_name"`
	CaptainName  string `json:"captain_name"`
	MemberCount  int    `json:"member_count"`
	Status       int16  `json:"status"`
	StatusText   string `json:"status_text"`
	RegisteredAt string `json:"registered_at"`
}

// RegistrationListReq 报名列表查询参数。
type RegistrationListReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ========== 提交与验证 ==========

// SubmitCompetitionChallengeReq 提交 Flag/攻击交易请求。
type SubmitCompetitionChallengeReq struct {
	ChallengeID    string `json:"challenge_id" binding:"required"`
	SubmissionType int16  `json:"submission_type" binding:"required,oneof=1 2 3"`
	Content        string `json:"content" binding:"required"`
}

// CompetitionSubmissionResp 提交响应。
type CompetitionSubmissionResp struct {
	SubmissionID      string                        `json:"submission_id"`
	IsCorrect         bool                          `json:"is_correct"`
	ScoreAwarded      *int                          `json:"score_awarded"`
	IsFirstBlood      *bool                         `json:"is_first_blood"`
	ChallengeNewScore *int                          `json:"challenge_new_score"`
	TeamTotalScore    *int                          `json:"team_total_score"`
	TeamRank          *int                          `json:"team_rank"`
	AssertionResults  *VerificationAssertionResults `json:"assertion_results"`
	ErrorMessage      *string                       `json:"error_message"`
	RemainingAttempts *int                          `json:"remaining_attempts"`
	CooldownUntil     *string                       `json:"cooldown_until"`
}

// CompetitionSubmissionListReq 团队提交记录查询参数。
type CompetitionSubmissionListReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// CompetitionSubmissionListItem 团队提交记录列表项。
type CompetitionSubmissionListItem struct {
	ID                 string  `json:"id"`
	ChallengeID        string  `json:"challenge_id"`
	ChallengeTitle     string  `json:"challenge_title"`
	SubmissionType     int16   `json:"submission_type"`
	SubmissionTypeText string  `json:"submission_type_text"`
	IsCorrect          bool    `json:"is_correct"`
	ScoreAwarded       *int    `json:"score_awarded"`
	IsFirstBlood       bool    `json:"is_first_blood"`
	ErrorMessage       *string `json:"error_message"`
	CreatedAt          string  `json:"created_at"`
}

// CompetitionSubmissionStatisticsResp 提交统计响应。
type CompetitionSubmissionStatisticsResp struct {
	TotalSubmissions   int     `json:"total_submissions"`
	CorrectSubmissions int     `json:"correct_submissions"`
	CorrectRate        float64 `json:"correct_rate"`
	FirstBloodCount    int     `json:"first_blood_count"`
	TeamsParticipated  int     `json:"teams_participated"`
}

// ========== 攻防赛分组与回合 ==========

// CreateAdGroupReq 创建攻防赛分组请求。
type CreateAdGroupReq struct {
	GroupName string   `json:"group_name" binding:"required,max=100"`
	TeamIDs   []string `json:"team_ids" binding:"required,min=1"`
}

// AdGroupResp 攻防赛分组响应。
type AdGroupResp struct {
	ID            string         `json:"id"`
	CompetitionID string         `json:"competition_id"`
	GroupName     string         `json:"group_name"`
	Namespace     *string        `json:"namespace"`
	Status        int16          `json:"status"`
	StatusText    string         `json:"status_text"`
	Teams         []CTFTeamBrief `json:"teams,omitempty"`
}

// AutoAssignAdGroupsReq 自动分组请求。
type AutoAssignAdGroupsReq struct {
	TeamsPerGroup int `json:"teams_per_group" binding:"required,min=1"`
}

// AutoAssignAdGroupsResp 自动分组响应。
type AutoAssignAdGroupsResp struct {
	Groups      []AutoAssignedGroupItem `json:"groups"`
	TotalTeams  int                     `json:"total_teams"`
	TotalGroups int                     `json:"total_groups"`
}

// AutoAssignedGroupItem 自动分组结果项。
type AutoAssignedGroupItem struct {
	ID        string `json:"id"`
	GroupName string `json:"group_name"`
	TeamCount int    `json:"team_count"`
}

// AdRoundListItem 回合列表项。
type AdRoundListItem struct {
	ID          string `json:"id"`
	RoundNumber int    `json:"round_number"`
	Phase       int16  `json:"phase"`
	PhaseText   string `json:"phase_text"`
}

// AdRoundDetailResp 回合详情响应。
type AdRoundDetailResp struct {
	ID                string  `json:"id"`
	CompetitionID     string  `json:"competition_id"`
	GroupID           string  `json:"group_id"`
	RoundNumber       int     `json:"round_number"`
	Phase             int16   `json:"phase"`
	PhaseText         string  `json:"phase_text"`
	AttackStartAt     *string `json:"attack_start_at"`
	AttackEndAt       *string `json:"attack_end_at"`
	DefenseStartAt    *string `json:"defense_start_at"`
	DefenseEndAt      *string `json:"defense_end_at"`
	SettlementStartAt *string `json:"settlement_start_at"`
	SettlementEndAt   *string `json:"settlement_end_at"`
}

// CurrentRoundResp 当前回合状态响应。
type CurrentRoundResp struct {
	GroupID          string                `json:"group_id"`
	RoundID          string                `json:"round_id"`
	RoundNumber      int                   `json:"round_number"`
	TotalRounds      int                   `json:"total_rounds"`
	Phase            int16                 `json:"phase"`
	PhaseText        string                `json:"phase_text"`
	PhaseStartAt     string                `json:"phase_start_at"`
	PhaseEndAt       string                `json:"phase_end_at"`
	RemainingSeconds int                   `json:"remaining_seconds"`
	MyTeam           *CurrentRoundTeamInfo `json:"my_team"`
}

// CurrentRoundTeamInfo 当前回合我的团队信息。
type CurrentRoundTeamInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TokenBalance int    `json:"token_balance"`
	Rank         int    `json:"rank"`
}

// ========== 攻击与防守 ==========

// SubmitAdAttackReq 提交攻击交易请求。
type SubmitAdAttackReq struct {
	TargetTeamID string `json:"target_team_id" binding:"required"`
	ChallengeID  string `json:"challenge_id" binding:"required"`
	AttackTxData string `json:"attack_tx_data" binding:"required"`
}

// AdAttackResp 攻击结果响应。
type AdAttackResp struct {
	AttackID             string                        `json:"attack_id"`
	IsSuccessful         bool                          `json:"is_successful"`
	TokenReward          *int                          `json:"token_reward"`
	IsFirstBlood         *bool                         `json:"is_first_blood"`
	ExploitCount         *int                          `json:"exploit_count"`
	AssertionResults     *VerificationAssertionResults `json:"assertion_results"`
	AttackerBalanceAfter *int                          `json:"attacker_balance_after"`
	TargetBalanceAfter   *int                          `json:"target_balance_after"`
	ErrorMessage         *string                       `json:"error_message"`
}

// AdAttackListReq 攻击记录查询参数。
type AdAttackListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	ChallengeID string `form:"challenge_id"`
	TeamID      string `form:"team_id"`
}

// AdAttackListItem 攻击记录列表项。
type AdAttackListItem struct {
	ID               string `json:"id"`
	AttackerTeamID   string `json:"attacker_team_id"`
	AttackerTeamName string `json:"attacker_team_name"`
	TargetTeamID     string `json:"target_team_id"`
	TargetTeamName   string `json:"target_team_name"`
	ChallengeID      string `json:"challenge_id"`
	ChallengeTitle   string `json:"challenge_title"`
	IsSuccessful     bool   `json:"is_successful"`
	TokenReward      *int   `json:"token_reward"`
	IsFirstBlood     bool   `json:"is_first_blood"`
	CreatedAt        string `json:"created_at"`
}

// SubmitAdDefenseReq 提交补丁合约请求。
type SubmitAdDefenseReq struct {
	ChallengeID     string `json:"challenge_id" binding:"required"`
	PatchSourceCode string `json:"patch_source_code" binding:"required"`
}

// AdDefenseResp 防守结果响应。
type AdDefenseResp struct {
	DefenseID           string  `json:"defense_id"`
	IsAccepted          bool    `json:"is_accepted"`
	FunctionalityPassed *bool   `json:"functionality_passed"`
	VulnerabilityFixed  *bool   `json:"vulnerability_fixed"`
	IsFirstPatch        *bool   `json:"is_first_patch"`
	TokenReward         *int    `json:"token_reward"`
	TeamBalanceAfter    *int    `json:"team_balance_after"`
	RejectionReason     *string `json:"rejection_reason"`
}

// AdDefenseListReq 防守记录查询参数。
type AdDefenseListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	ChallengeID string `form:"challenge_id"`
	TeamID      string `form:"team_id"`
}

// AdDefenseListItem 防守记录列表项。
type AdDefenseListItem struct {
	ID                  string `json:"id"`
	TeamID              string `json:"team_id"`
	TeamName            string `json:"team_name"`
	ChallengeID         string `json:"challenge_id"`
	ChallengeTitle      string `json:"challenge_title"`
	IsAccepted          bool   `json:"is_accepted"`
	FunctionalityPassed *bool  `json:"functionality_passed"`
	VulnerabilityFixed  *bool  `json:"vulnerability_fixed"`
	IsFirstPatch        bool   `json:"is_first_patch"`
	TokenReward         *int   `json:"token_reward"`
	CreatedAt           string `json:"created_at"`
}

// ========== Token 流水与排行榜 ==========

// TokenLedgerListReq Token流水查询参数。
type TokenLedgerListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	RoundID    string `form:"round_id"`
	ChangeType int16  `form:"change_type" binding:"omitempty,min=1"`
}

// TokenLedgerListItem Token流水列表项。
type TokenLedgerListItem struct {
	ID              string  `json:"id"`
	RoundNumber     *int    `json:"round_number"`
	ChangeType      int16   `json:"change_type"`
	ChangeTypeText  string  `json:"change_type_text"`
	Amount          int     `json:"amount"`
	BalanceAfter    int     `json:"balance_after"`
	Description     *string `json:"description"`
	RelatedAttackID *string `json:"related_attack_id"`
	CreatedAt       string  `json:"created_at"`
}

// LeaderboardReq 排行榜查询参数。
type LeaderboardReq struct {
	GroupID string `form:"group_id"`
	Top     int    `form:"top" binding:"omitempty,min=1,max=100"`
}

// LeaderboardResp 排行榜响应。
type LeaderboardResp struct {
	CompetitionID   string                   `json:"competition_id"`
	CompetitionType int16                    `json:"competition_type"`
	GroupID         *string                  `json:"group_id"`
	GroupName       *string                  `json:"group_name"`
	CurrentRound    *int                     `json:"current_round"`
	TotalRounds     *int                     `json:"total_rounds"`
	IsFrozen        bool                     `json:"is_frozen"`
	FrozenAt        *string                  `json:"frozen_at"`
	UpdatedAt       *string                  `json:"updated_at"`
	Rankings        []LeaderboardRankingItem `json:"rankings"`
}

// FinalLeaderboardResp 最终排名响应。
type FinalLeaderboardResp struct {
	CompetitionID   string                 `json:"competition_id"`
	CompetitionType int16                  `json:"competition_type"`
	EndedAt         string                 `json:"ended_at"`
	Rankings        []FinalLeaderboardItem `json:"rankings"`
	TotalTeams      int                    `json:"total_teams"`
	TotalSolves     int                    `json:"total_solves"`
}

// LeaderboardHistoryReq 排行榜历史快照查询参数。
type LeaderboardHistoryReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	GroupID  string `form:"group_id"`
}

// LeaderboardHistorySnapshotItem 排行榜历史快照项。
type LeaderboardHistorySnapshotItem struct {
	SnapshotAt string                   `json:"snapshot_at"`
	Rankings   []LeaderboardRankingItem `json:"rankings"`
}

// LeaderboardHistoryResp 排行榜历史快照响应。
type LeaderboardHistoryResp struct {
	List       []LeaderboardHistorySnapshotItem `json:"list"`
	Pagination PaginationResp                   `json:"pagination"`
}

// ========== 公告 ==========

// CreateCtfAnnouncementReq 发布公告请求。
type CreateCtfAnnouncementReq struct {
	Title            string  `json:"title" binding:"required,max=200"`
	Content          string  `json:"content" binding:"required"`
	AnnouncementType int16   `json:"announcement_type" binding:"required,oneof=1 2 3"`
	ChallengeID      *string `json:"challenge_id"`
}

// CtfAnnouncementItem 竞赛公告列表项。
type CtfAnnouncementItem struct {
	ID                   string  `json:"id"`
	Title                string  `json:"title"`
	Content              string  `json:"content"`
	AnnouncementType     int16   `json:"announcement_type"`
	AnnouncementTypeText string  `json:"announcement_type_text"`
	ChallengeID          *string `json:"challenge_id"`
	ChallengeTitle       *string `json:"challenge_title"`
	PublishedByName      string  `json:"published_by_name"`
	CreatedAt            string  `json:"created_at"`
}

// CtfAnnouncementResp 竞赛公告详情/发布响应。
type CtfAnnouncementResp struct {
	ID                   string  `json:"id"`
	Title                string  `json:"title"`
	Content              string  `json:"content,omitempty"`
	AnnouncementType     int16   `json:"announcement_type"`
	AnnouncementTypeText string  `json:"announcement_type_text"`
	ChallengeID          *string `json:"challenge_id"`
	ChallengeTitle       *string `json:"challenge_title,omitempty"`
	PublishedByName      string  `json:"published_by_name"`
	CreatedAt            string  `json:"created_at"`
}

// ========== 资源配额与环境 ==========

// ResourceQuotaResp 资源配额详情响应。
type ResourceQuotaResp struct {
	CompetitionID     string `json:"competition_id"`
	MaxCPU            string `json:"max_cpu"`
	MaxMemory         string `json:"max_memory"`
	MaxStorage        string `json:"max_storage"`
	MaxNamespaces     int    `json:"max_namespaces"`
	UsedCPU           string `json:"used_cpu"`
	UsedMemory        string `json:"used_memory"`
	UsedStorage       string `json:"used_storage"`
	CurrentNamespaces int    `json:"current_namespaces"`
}

// UpdateResourceQuotaReq 设置竞赛资源配额请求。
type UpdateResourceQuotaReq struct {
	MaxCPU        string `json:"max_cpu" binding:"required"`
	MaxMemory     string `json:"max_memory" binding:"required"`
	MaxStorage    string `json:"max_storage" binding:"required"`
	MaxNamespaces int    `json:"max_namespaces" binding:"required,min=1"`
}

// StartChallengeEnvironmentResp 启动题目环境响应。
type StartChallengeEnvironmentResp struct {
	EnvironmentID string  `json:"environment_id"`
	CompetitionID string  `json:"competition_id"`
	ChallengeID   string  `json:"challenge_id"`
	TeamID        string  `json:"team_id"`
	Namespace     string  `json:"namespace"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
	ChainRPCURL   *string `json:"chain_rpc_url"`
	CreatedAt     string  `json:"created_at"`
}

// ChallengeEnvironmentResp 题目环境详情响应。
type ChallengeEnvironmentResp struct {
	ID              string                                        `json:"id"`
	CompetitionID   string                                        `json:"competition_id"`
	ChallengeID     string                                        `json:"challenge_id"`
	TeamID          string                                        `json:"team_id"`
	Namespace       string                                        `json:"namespace"`
	Status          int16                                         `json:"status"`
	StatusText      string                                        `json:"status_text"`
	ChainRPCURL     *string                                       `json:"chain_rpc_url"`
	ContainerStatus map[string]ChallengeEnvironmentContainerState `json:"container_status"`
	StartedAt       *string                                       `json:"started_at"`
	CreatedAt       string                                        `json:"created_at"`
}

// ResetChallengeEnvironmentResp 重置题目环境响应。
type ResetChallengeEnvironmentResp struct {
	EnvironmentID string `json:"environment_id"`
	Status        int16  `json:"status"`
	StatusText    string `json:"status_text"`
}

// CompetitionEnvironmentListReq 竞赛环境资源列表查询参数。
type CompetitionEnvironmentListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status      int16  `form:"status" binding:"omitempty,oneof=1 2 3 4 5"`
	ChallengeID string `form:"challenge_id"`
	TeamID      string `form:"team_id"`
}

// CompetitionEnvironmentListItem 竞赛环境资源列表项。
type CompetitionEnvironmentListItem struct {
	ID             string  `json:"id"`
	CompetitionID  string  `json:"competition_id"`
	ChallengeID    string  `json:"challenge_id"`
	ChallengeTitle string  `json:"challenge_title"`
	TeamID         string  `json:"team_id"`
	TeamName       string  `json:"team_name"`
	Namespace      string  `json:"namespace"`
	Status         int16   `json:"status"`
	StatusText     string  `json:"status_text"`
	ChainRPCURL    *string `json:"chain_rpc_url"`
	StartedAt      *string `json:"started_at"`
	CreatedAt      string  `json:"created_at"`
}

// CompetitionEnvironmentListResp 竞赛环境资源列表响应。
type CompetitionEnvironmentListResp struct {
	List       []CompetitionEnvironmentListItem `json:"list"`
	Pagination PaginationResp                   `json:"pagination"`
}

// ForceDestroyChallengeEnvironmentReq 强制回收环境请求。
type ForceDestroyChallengeEnvironmentReq struct {
	Reason string `json:"reason" binding:"required"`
}

// ForceDestroyChallengeEnvironmentResp 强制回收环境响应。
type ForceDestroyChallengeEnvironmentResp struct {
	EnvironmentID string `json:"environment_id"`
	Status        int16  `json:"status"`
	StatusText    string `json:"status_text"`
}

// MyChallengeEnvironmentItem 我的题目环境列表项。
type MyChallengeEnvironmentItem struct {
	ID             string  `json:"id"`
	ChallengeID    string  `json:"challenge_id"`
	ChallengeTitle string  `json:"challenge_title"`
	Namespace      string  `json:"namespace"`
	Status         int16   `json:"status"`
	StatusText     string  `json:"status_text"`
	ChainRPCURL    *string `json:"chain_rpc_url"`
	CreatedAt      string  `json:"created_at"`
}

// ========== 队伍链与监控统计 ==========

// TeamChainResp 队伍链信息响应。
type TeamChainResp struct {
	ID                  string                  `json:"id"`
	CompetitionID       string                  `json:"competition_id"`
	GroupID             string                  `json:"group_id"`
	TeamID              string                  `json:"team_id"`
	ChainRPCURL         *string                 `json:"chain_rpc_url"`
	ChainWSURL          *string                 `json:"chain_ws_url"`
	DeployedContracts   []TeamChainContractItem `json:"deployed_contracts"`
	CurrentPatchVersion int                     `json:"current_patch_version"`
	Status              int16                   `json:"status"`
	StatusText          string                  `json:"status_text"`
}

// CompetitionMonitorResp 竞赛运行监控响应。
type CompetitionMonitorResp struct {
	CompetitionID     string                               `json:"competition_id"`
	CompetitionType   int16                                `json:"competition_type"`
	Status            int16                                `json:"status"`
	StatusText        string                               `json:"status_text"`
	Overview          CompetitionMonitorOverview           `json:"overview"`
	ResourceUsage     CompetitionMonitorResourceUsage      `json:"resource_usage"`
	ChallengeStats    []CompetitionMonitorChallengeStat    `json:"challenge_stats"`
	RecentSubmissions []CompetitionMonitorRecentSubmission `json:"recent_submissions"`
}

// CompetitionStatisticsResp 竞赛统计数据响应。
type CompetitionStatisticsResp struct {
	CompetitionID       string                               `json:"competition_id"`
	Summary             CompetitionStatisticsSummary         `json:"summary"`
	ChallengeStatistics []CompetitionStatisticsChallengeItem `json:"challenge_statistics"`
	ScoreDistribution   CompetitionScoreDistribution         `json:"score_distribution"`
	Timeline            CompetitionTimeline                  `json:"timeline"`
}

// CompetitionResultsResp 竞赛最终结果响应。
type CompetitionResultsResp struct {
	CompetitionID string                    `json:"competition_id"`
	Summary       CompetitionResultsSummary `json:"summary"`
	Rankings      []FinalLeaderboardItem    `json:"rankings"`
}

// CtfAdminOverviewResp 全平台竞赛概览响应。
type CtfAdminOverviewResp struct {
	TotalCompetitions       int                              `json:"total_competitions"`
	RunningCompetitions     int                              `json:"running_competitions"`
	UpcomingCompetitions    int                              `json:"upcoming_competitions"`
	TotalParticipants       int                              `json:"total_participants"`
	TotalResourceUsage      AdminTotalResourceUsage          `json:"total_resource_usage"`
	RunningCompetitionsList []RunningCompetitionOverviewItem `json:"running_competitions_list"`
	Alerts                  []AdminCompetitionAlert          `json:"alerts"`
}

// CompetitionListResp 竞赛列表响应。
type CompetitionListResp struct {
	List       []CompetitionListItem `json:"list"`
	Pagination PaginationResp        `json:"pagination"`
}

// ChallengeListResp 题目列表响应。
type ChallengeListResp struct {
	List       []ChallengeListItem `json:"list"`
	Pagination PaginationResp      `json:"pagination"`
}

// ChallengeTemplateListResp 模板列表响应。
type ChallengeTemplateListResp struct {
	List []ChallengeTemplateListItem `json:"list"`
}

// VerificationListResp 预验证记录列表响应。
type VerificationListResp struct {
	List []VerificationListItem `json:"list"`
}

// ChallengeReviewListResp 待审核题目列表响应。
type ChallengeReviewListResp struct {
	List []ChallengeReviewListItem `json:"list"`
}

// ChallengeReviewHistoryResp 题目审核记录列表响应。
type ChallengeReviewHistoryResp struct {
	List []ChallengeReviewResp `json:"list"`
}

// ChallengeContractListResp 题目合约列表响应。
type ChallengeContractListResp struct {
	List []ChallengeContractItem `json:"list"`
}

// ChallengeAssertionListResp 题目断言列表响应。
type ChallengeAssertionListResp struct {
	List []ChallengeAssertionItem `json:"list"`
}

// CompetitionChallengeListResp 竞赛题目列表响应。
type CompetitionChallengeListResp struct {
	List []CompetitionChallengeListItem `json:"list"`
}

// TeamListResp 团队列表响应。
type TeamListResp struct {
	List []TeamListItem `json:"list"`
}

// AdGroupListResp 攻防赛分组列表响应。
type AdGroupListResp struct {
	List []AdGroupResp `json:"list"`
}

// RegistrationListResp 报名列表响应。
type RegistrationListResp struct {
	List []RegistrationListItem `json:"list"`
}

// CompetitionSubmissionListResp 团队提交记录列表响应。
type CompetitionSubmissionListResp struct {
	List       []CompetitionSubmissionListItem `json:"list"`
	Pagination PaginationResp                  `json:"pagination"`
}

// AdRoundListResp 回合列表响应。
type AdRoundListResp struct {
	List []AdRoundListItem `json:"list"`
}

// AdAttackListResp 攻击记录列表响应。
type AdAttackListResp struct {
	List       []AdAttackListItem `json:"list"`
	Pagination PaginationResp     `json:"pagination"`
}

// AdDefenseListResp 防守记录列表响应。
type AdDefenseListResp struct {
	List       []AdDefenseListItem `json:"list"`
	Pagination PaginationResp      `json:"pagination"`
}

// TokenLedgerListResp Token流水列表响应。
type TokenLedgerListResp struct {
	List       []TokenLedgerListItem `json:"list"`
	Pagination PaginationResp        `json:"pagination"`
}

// CtfAnnouncementListResp 公告列表响应。
type CtfAnnouncementListResp struct {
	List []CtfAnnouncementItem `json:"list"`
}

// MyChallengeEnvironmentListResp 我的题目环境列表响应。
type MyChallengeEnvironmentListResp struct {
	List []MyChallengeEnvironmentItem `json:"list"`
}

// TeamChainListResp 攻防赛分组队伍链列表响应。
type TeamChainListResp struct {
	List []TeamChainResp `json:"list"`
}
