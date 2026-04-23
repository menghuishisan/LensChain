// ctf.go
// 模块05 — CTF竞赛：数据库实体结构体。
// 该文件按竞赛、题目、团队、攻防赛、资源与环境五个子域组织模块05全部表映射。

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Competition 竞赛主表。
type Competition struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Title               string         `gorm:"type:varchar(200);not null" json:"title"`
	Description         *string        `gorm:"type:text" json:"description,omitempty"`
	BannerURL           *string        `gorm:"type:varchar(500)" json:"banner_url,omitempty"`
	CompetitionType     int16          `gorm:"column:competition_type;type:smallint;not null" json:"competition_type"`
	Scope               int16          `gorm:"column:scope;type:smallint;not null;default:1" json:"scope"`
	SchoolID            *int64         `gorm:"index" json:"school_id,omitempty,string"`
	CreatedBy           int64          `gorm:"not null;index" json:"created_by,string"`
	TeamMode            int16          `gorm:"column:team_mode;type:smallint;not null;default:1" json:"team_mode"`
	MaxTeamSize         int            `gorm:"not null;default:1" json:"max_team_size"`
	MinTeamSize         int            `gorm:"not null;default:1" json:"min_team_size"`
	MaxTeams            *int           `gorm:"" json:"max_teams,omitempty"`
	Status              int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	RegistrationStartAt *time.Time     `gorm:"" json:"registration_start_at,omitempty"`
	RegistrationEndAt   *time.Time     `gorm:"" json:"registration_end_at,omitempty"`
	StartAt             *time.Time     `gorm:"index" json:"start_at,omitempty"`
	EndAt               *time.Time     `gorm:"" json:"end_at,omitempty"`
	FreezeAt            *time.Time     `gorm:"" json:"freeze_at,omitempty"`
	JeopardyConfig      datatypes.JSON `gorm:"column:jeopardy_config;type:jsonb" json:"jeopardy_config,omitempty"`
	AdConfig            datatypes.JSON `gorm:"column:ad_config;type:jsonb" json:"ad_config,omitempty"`
	Rules               *string        `gorm:"type:text" json:"rules,omitempty"`
	CreatedAt           time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定竞赛主表表名。
func (Competition) TableName() string {
	return "competitions"
}

// Challenge 题目主表。
type Challenge struct {
	ID                int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Title             string         `gorm:"type:varchar(200);not null" json:"title"`
	Description       string         `gorm:"type:text;not null" json:"description"`
	Category          string         `gorm:"type:varchar(20);not null;index" json:"category"`
	Difficulty        int16          `gorm:"column:difficulty;type:smallint;not null;index" json:"difficulty"`
	BaseScore         int            `gorm:"not null" json:"base_score"`
	FlagType          int16          `gorm:"column:flag_type;type:smallint;not null;default:1;index" json:"flag_type"`
	StaticFlag        *string        `gorm:"type:varchar(500)" json:"static_flag,omitempty"`
	DynamicFlagSecret *string        `gorm:"type:varchar(200)" json:"dynamic_flag_secret,omitempty"`
	RuntimeMode       int16          `gorm:"column:runtime_mode;type:smallint;not null;default:1;index" json:"runtime_mode"`
	ChainConfig       datatypes.JSON `gorm:"column:chain_config;type:jsonb" json:"chain_config,omitempty"`
	SetupTransactions datatypes.JSON `gorm:"column:setup_transactions;type:jsonb" json:"setup_transactions,omitempty"`
	SourcePath        *int16         `gorm:"column:source_path;type:smallint" json:"source_path,omitempty"`
	SwcID             *string        `gorm:"type:varchar(20)" json:"swc_id,omitempty"`
	TemplateID        *int64         `gorm:"" json:"template_id,omitempty,string"`
	TemplateParams    datatypes.JSON `gorm:"column:template_params;type:jsonb" json:"template_params,omitempty"`
	EnvironmentConfig datatypes.JSON `gorm:"column:environment_config;type:jsonb" json:"environment_config,omitempty"`
	AttachmentURLs    datatypes.JSON `gorm:"column:attachment_urls;type:jsonb" json:"attachment_urls,omitempty"`
	AuthorID          int64          `gorm:"not null;index" json:"author_id,string"`
	SchoolID          int64          `gorm:"not null;index" json:"school_id,string"`
	Status            int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	IsPublic          bool           `gorm:"not null;default:false" json:"is_public"`
	UsageCount        int            `gorm:"not null;default:0" json:"usage_count"`
	CreatedAt         time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定题目主表表名。
func (Challenge) TableName() string {
	return "challenges"
}

// ChallengeContract 题目合约表。
type ChallengeContract struct {
	ID              int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ChallengeID     int64          `gorm:"not null;index" json:"challenge_id,string"`
	Name            string         `gorm:"type:varchar(100);not null" json:"name"`
	SourceCode      string         `gorm:"type:text;not null" json:"source_code"`
	ABI             datatypes.JSON `gorm:"column:abi;type:jsonb;not null" json:"abi"`
	Bytecode        string         `gorm:"type:text;not null" json:"bytecode"`
	ConstructorArgs datatypes.JSON `gorm:"column:constructor_args;type:jsonb" json:"constructor_args,omitempty"`
	DeployOrder     int            `gorm:"not null;default:1" json:"deploy_order"`
	CreatedAt       time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定题目合约表表名。
func (ChallengeContract) TableName() string {
	return "challenge_contracts"
}

// ChallengeAssertion 题目断言表。
type ChallengeAssertion struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ChallengeID   int64          `gorm:"not null;index" json:"challenge_id,string"`
	AssertionType string         `gorm:"type:varchar(30);not null" json:"assertion_type"`
	Target        string         `gorm:"type:varchar(200);not null" json:"target"`
	Operator      string         `gorm:"type:varchar(10);not null" json:"operator"`
	ExpectedValue string         `gorm:"type:text;not null" json:"expected_value"`
	Description   *string        `gorm:"type:varchar(500)" json:"description,omitempty"`
	ExtraParams   datatypes.JSON `gorm:"column:extra_params;type:jsonb" json:"extra_params,omitempty"`
	SortOrder     int            `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定题目断言表表名。
func (ChallengeAssertion) TableName() string {
	return "challenge_assertions"
}

// ChallengeTemplate 参数化模板库表。
type ChallengeTemplate struct {
	ID                    int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	Name                  string         `gorm:"type:varchar(200);not null" json:"name"`
	Code                  string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"code"`
	Description           *string        `gorm:"type:text" json:"description,omitempty"`
	VulnerabilityType     string         `gorm:"type:varchar(100);not null;index" json:"vulnerability_type"`
	BaseSourceCode        string         `gorm:"type:text;not null" json:"base_source_code"`
	BaseAssertions        datatypes.JSON `gorm:"column:base_assertions;type:jsonb;not null" json:"base_assertions"`
	BaseSetupTransactions datatypes.JSON `gorm:"column:base_setup_transactions;type:jsonb;not null" json:"base_setup_transactions"`
	Parameters            datatypes.JSON `gorm:"column:parameters;type:jsonb;not null" json:"parameters"`
	Variants              datatypes.JSON `gorm:"column:variants;type:jsonb" json:"variants,omitempty"`
	ReferenceEvents       datatypes.JSON `gorm:"column:reference_events;type:jsonb" json:"reference_events,omitempty"`
	DifficultyRange       datatypes.JSON `gorm:"column:difficulty_range;type:jsonb;not null" json:"difficulty_range"`
	UsageCount            int            `gorm:"not null;default:0" json:"usage_count"`
	CreatedAt             time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定参数化模板库表表名。
func (ChallengeTemplate) TableName() string {
	return "challenge_templates"
}

// ChallengeReview 题目审核记录表。
type ChallengeReview struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ChallengeID int64     `gorm:"not null;index" json:"challenge_id,string"`
	ReviewerID  int64     `gorm:"not null;index" json:"reviewer_id,string"`
	Action      int16     `gorm:"column:action;type:smallint;not null" json:"action"`
	Comment     *string   `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定题目审核记录表表名。
func (ChallengeReview) TableName() string {
	return "challenge_reviews"
}

// ChallengeVerification 题目预验证记录表。
type ChallengeVerification struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ChallengeID   int64          `gorm:"not null;index" json:"challenge_id,string"`
	InitiatedBy   int64          `gorm:"not null" json:"initiated_by,string"`
	Status        int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	StepResults   datatypes.JSON `gorm:"column:step_results;type:jsonb;not null;default:'[]'" json:"step_results"`
	PocContent    *string        `gorm:"type:text" json:"poc_content,omitempty"`
	PocLanguage   *string        `gorm:"type:varchar(20)" json:"poc_language,omitempty"`
	EnvironmentID *string        `gorm:"type:varchar(200)" json:"environment_id,omitempty"`
	ErrorMessage  *string        `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt     time.Time      `gorm:"not null;default:now()" json:"started_at"`
	CompletedAt   *time.Time     `gorm:"" json:"completed_at,omitempty"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定题目预验证记录表表名。
func (ChallengeVerification) TableName() string {
	return "challenge_verifications"
}

// CompetitionChallenge 竞赛题目关联表。
type CompetitionChallenge struct {
	ID               int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID    int64      `gorm:"not null;uniqueIndex:uk_competition_challenges" json:"competition_id,string"`
	ChallengeID      int64      `gorm:"not null;uniqueIndex:uk_competition_challenges;index" json:"challenge_id,string"`
	SortOrder        int        `gorm:"not null;default:0" json:"sort_order"`
	CurrentScore     *int       `gorm:"" json:"current_score,omitempty"`
	SolveCount       int        `gorm:"not null;default:0" json:"solve_count"`
	FirstBloodTeamID *int64     `gorm:"" json:"first_blood_team_id,omitempty,string"`
	FirstBloodAt     *time.Time `gorm:"" json:"first_blood_at,omitempty"`
	CreatedAt        time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定竞赛题目关联表表名。
func (CompetitionChallenge) TableName() string {
	return "competition_challenges"
}

// Team 参赛团队表。
type Team struct {
	ID            int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID int64     `gorm:"not null;index" json:"competition_id,string"`
	Name          string    `gorm:"type:varchar(100);not null" json:"name"`
	CaptainID     int64     `gorm:"not null;index" json:"captain_id,string"`
	InviteCode    *string   `gorm:"type:varchar(20);uniqueIndex:uk_teams_invite_code,where:invite_code IS NOT NULL" json:"invite_code,omitempty"`
	Status        int16     `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	AdGroupID     *int64    `gorm:"index" json:"ad_group_id,omitempty,string"`
	TokenBalance  *int      `gorm:"" json:"token_balance,omitempty"`
	FinalRank     *int      `gorm:"" json:"final_rank,omitempty"`
	TotalScore    *int      `gorm:"" json:"total_score,omitempty"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定参赛团队表表名。
func (Team) TableName() string {
	return "teams"
}

// TeamMember 团队成员表。
type TeamMember struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TeamID    int64     `gorm:"not null;uniqueIndex:uk_team_members" json:"team_id,string"`
	StudentID int64     `gorm:"not null;uniqueIndex:uk_team_members;index" json:"student_id,string"`
	Role      int16     `gorm:"column:role;type:smallint;not null;default:2" json:"role"`
	JoinedAt  time.Time `gorm:"not null;default:now()" json:"joined_at"`
}

// TableName 指定团队成员表表名。
func (TeamMember) TableName() string {
	return "team_members"
}

// CompetitionRegistration 竞赛报名表。
type CompetitionRegistration struct {
	ID            int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID int64     `gorm:"not null;uniqueIndex:uk_competition_registrations" json:"competition_id,string"`
	TeamID        int64     `gorm:"not null;uniqueIndex:uk_competition_registrations;index" json:"team_id,string"`
	RegisteredBy  int64     `gorm:"not null" json:"registered_by,string"`
	Status        int16     `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定竞赛报名表表名。
func (CompetitionRegistration) TableName() string {
	return "competition_registrations"
}

// Submission 提交记录表。
type Submission struct {
	ID               int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID    int64          `gorm:"not null;index" json:"competition_id,string"`
	ChallengeID      int64          `gorm:"not null;index" json:"challenge_id,string"`
	TeamID           int64          `gorm:"not null;index" json:"team_id,string"`
	StudentID        int64          `gorm:"not null;index" json:"student_id,string"`
	SubmissionType   int16          `gorm:"column:submission_type;type:smallint;not null" json:"submission_type"`
	Content          string         `gorm:"type:text;not null" json:"content"`
	IsCorrect        bool           `gorm:"not null;default:false;index:,where:is_correct = TRUE" json:"is_correct"`
	ScoreAwarded     *int           `gorm:"" json:"score_awarded,omitempty"`
	IsFirstBlood     bool           `gorm:"not null;default:false" json:"is_first_blood"`
	AssertionResults datatypes.JSON `gorm:"column:assertion_results;type:jsonb" json:"assertion_results,omitempty"`
	ErrorMessage     *string        `gorm:"type:text" json:"error_message,omitempty"`
	Namespace        *string        `gorm:"type:varchar(100)" json:"namespace,omitempty"`
	CreatedAt        time.Time      `gorm:"not null;default:now();index" json:"created_at"`
}

// TableName 指定提交记录表表名。
func (Submission) TableName() string {
	return "submissions"
}

// AdGroup 攻防赛分组表。
type AdGroup struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID        int64     `gorm:"not null;index" json:"competition_id,string"`
	GroupName            string    `gorm:"type:varchar(100);not null" json:"group_name"`
	Namespace            *string   `gorm:"type:varchar(100)" json:"namespace,omitempty"`
	JudgeChainURL        *string   `gorm:"type:varchar(500)" json:"judge_chain_url,omitempty"`
	JudgeContractAddress *string   `gorm:"type:varchar(100)" json:"judge_contract_address,omitempty"`
	Status               int16     `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	CreatedAt            time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt            time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定攻防赛分组表表名。
func (AdGroup) TableName() string {
	return "ad_groups"
}

// AdRound 攻防赛回合表。
type AdRound struct {
	ID                int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID     int64          `gorm:"not null;uniqueIndex:uk_ad_rounds" json:"competition_id,string"`
	GroupID           int64          `gorm:"not null;uniqueIndex:uk_ad_rounds;index" json:"group_id,string"`
	RoundNumber       int            `gorm:"not null;uniqueIndex:uk_ad_rounds" json:"round_number"`
	Phase             int16          `gorm:"column:phase;type:smallint;not null;default:1;index" json:"phase"`
	AttackStartAt     *time.Time     `gorm:"" json:"attack_start_at,omitempty"`
	AttackEndAt       *time.Time     `gorm:"" json:"attack_end_at,omitempty"`
	DefenseStartAt    *time.Time     `gorm:"" json:"defense_start_at,omitempty"`
	DefenseEndAt      *time.Time     `gorm:"" json:"defense_end_at,omitempty"`
	SettlementStartAt *time.Time     `gorm:"" json:"settlement_start_at,omitempty"`
	SettlementEndAt   *time.Time     `gorm:"" json:"settlement_end_at,omitempty"`
	SettlementResult  datatypes.JSON `gorm:"column:settlement_result;type:jsonb" json:"settlement_result,omitempty"`
	CreatedAt         time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定攻防赛回合表表名。
func (AdRound) TableName() string {
	return "ad_rounds"
}

// AdAttack 攻防赛攻击记录表。
type AdAttack struct {
	ID               int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID    int64          `gorm:"not null;index" json:"competition_id,string"`
	RoundID          int64          `gorm:"not null;index" json:"round_id,string"`
	AttackerTeamID   int64          `gorm:"not null;index" json:"attacker_team_id,string"`
	TargetTeamID     int64          `gorm:"not null;index" json:"target_team_id,string"`
	ChallengeID      int64          `gorm:"not null;index" json:"challenge_id,string"`
	AttackTxData     string         `gorm:"type:text;not null" json:"attack_tx_data"`
	IsSuccessful     bool           `gorm:"not null;default:false;index:,where:is_successful = TRUE" json:"is_successful"`
	AssertionResults datatypes.JSON `gorm:"column:assertion_results;type:jsonb" json:"assertion_results,omitempty"`
	TokenReward      *int           `gorm:"" json:"token_reward,omitempty"`
	ExploitCount     *int           `gorm:"" json:"exploit_count,omitempty"`
	IsFirstBlood     bool           `gorm:"not null;default:false" json:"is_first_blood"`
	ErrorMessage     *string        `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt        time.Time      `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定攻防赛攻击记录表表名。
func (AdAttack) TableName() string {
	return "ad_attacks"
}

// AdDefense 攻防赛防守记录表。
type AdDefense struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID       int64     `gorm:"not null;index" json:"competition_id,string"`
	RoundID             int64     `gorm:"not null;index" json:"round_id,string"`
	TeamID              int64     `gorm:"not null;index" json:"team_id,string"`
	ChallengeID         int64     `gorm:"not null;index" json:"challenge_id,string"`
	PatchSourceCode     string    `gorm:"type:text;not null" json:"patch_source_code"`
	IsAccepted          bool      `gorm:"not null;default:false" json:"is_accepted"`
	FunctionalityPassed *bool     `gorm:"" json:"functionality_passed,omitempty"`
	VulnerabilityFixed  *bool     `gorm:"" json:"vulnerability_fixed,omitempty"`
	IsFirstPatch        bool      `gorm:"not null;default:false" json:"is_first_patch"`
	TokenReward         *int      `gorm:"" json:"token_reward,omitempty"`
	RejectionReason     *string   `gorm:"type:text" json:"rejection_reason,omitempty"`
	CreatedAt           time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定攻防赛防守记录表表名。
func (AdDefense) TableName() string {
	return "ad_defenses"
}

// AdTokenLedger Token流水账表。
type AdTokenLedger struct {
	ID               int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID    int64     `gorm:"not null;index" json:"competition_id,string"`
	GroupID          int64     `gorm:"not null;index" json:"group_id,string"`
	RoundID          *int64    `gorm:"index" json:"round_id,omitempty,string"`
	TeamID           int64     `gorm:"not null;index" json:"team_id,string"`
	ChangeType       int16     `gorm:"column:change_type;type:smallint;not null" json:"change_type"`
	Amount           int       `gorm:"not null" json:"amount"`
	BalanceAfter     int       `gorm:"not null" json:"balance_after"`
	RelatedAttackID  *int64    `gorm:"" json:"related_attack_id,omitempty,string"`
	RelatedDefenseID *int64    `gorm:"" json:"related_defense_id,omitempty,string"`
	Description      *string   `gorm:"type:varchar(500)" json:"description,omitempty"`
	CreatedAt        time.Time `gorm:"not null;default:now();index" json:"created_at"`
}

// TableName 指定 Token 流水账表表名。
func (AdTokenLedger) TableName() string {
	return "ad_token_ledger"
}

// LeaderboardSnapshot 排行榜快照表。
type LeaderboardSnapshot struct {
	ID            int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID int64      `gorm:"not null;index" json:"competition_id,string"`
	TeamID        int64      `gorm:"not null;index" json:"team_id,string"`
	Rank          int        `gorm:"not null" json:"rank"`
	Score         int        `gorm:"not null" json:"score"`
	SolveCount    *int       `gorm:"" json:"solve_count,omitempty"`
	LastSolveAt   *time.Time `gorm:"" json:"last_solve_at,omitempty"`
	IsFrozen      bool       `gorm:"not null;default:false" json:"is_frozen"`
	SnapshotAt    time.Time  `gorm:"not null;default:now();index" json:"snapshot_at"`
	CreatedAt     time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定排行榜快照表表名。
func (LeaderboardSnapshot) TableName() string {
	return "leaderboard_snapshots"
}

// CtfAnnouncement 竞赛公告表。
type CtfAnnouncement struct {
	ID               int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID    int64     `gorm:"not null;index" json:"competition_id,string"`
	ChallengeID      *int64    `gorm:"index" json:"challenge_id,omitempty,string"`
	Title            string    `gorm:"type:varchar(200);not null" json:"title"`
	Content          string    `gorm:"type:text;not null" json:"content"`
	AnnouncementType int16     `gorm:"column:announcement_type;type:smallint;not null;default:1" json:"announcement_type"`
	PublishedBy      int64     `gorm:"not null" json:"published_by,string"`
	CreatedAt        time.Time `gorm:"not null;default:now();index" json:"created_at"`
}

// TableName 指定竞赛公告表表名。
func (CtfAnnouncement) TableName() string {
	return "announcements"
}

// CtfResourceQuota CTF资源配额表。
type CtfResourceQuota struct {
	ID                int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID     int64     `gorm:"not null;uniqueIndex:uk_ctf_resource_quotas_competition" json:"competition_id,string"`
	MaxCPU            *string   `gorm:"type:varchar(20)" json:"max_cpu,omitempty"`
	MaxMemory         *string   `gorm:"type:varchar(20)" json:"max_memory,omitempty"`
	MaxStorage        *string   `gorm:"type:varchar(20)" json:"max_storage,omitempty"`
	MaxNamespaces     *int      `gorm:"" json:"max_namespaces,omitempty"`
	UsedCPU           string    `gorm:"type:varchar(20);not null;default:'0'" json:"used_cpu"`
	UsedMemory        string    `gorm:"type:varchar(20);not null;default:'0'" json:"used_memory"`
	UsedStorage       string    `gorm:"type:varchar(20);not null;default:'0'" json:"used_storage"`
	CurrentNamespaces int       `gorm:"not null;default:0" json:"current_namespaces"`
	CreatedAt         time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定 CTF 资源配额表表名。
func (CtfResourceQuota) TableName() string {
	return "ctf_resource_quotas"
}

// ChallengeEnvironment 题目环境实例表。
type ChallengeEnvironment struct {
	ID              int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID   int64          `gorm:"not null;index" json:"competition_id,string"`
	ChallengeID     int64          `gorm:"not null;index" json:"challenge_id,string"`
	TeamID          int64          `gorm:"not null;index" json:"team_id,string"`
	Namespace       string         `gorm:"type:varchar(100);not null" json:"namespace"`
	ChainRPCURL     *string        `gorm:"column:chain_rpc_url;type:varchar(500)" json:"chain_rpc_url,omitempty"`
	ContainerStatus datatypes.JSON `gorm:"column:container_status;type:jsonb" json:"container_status,omitempty"`
	Status          int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	StartedAt       *time.Time     `gorm:"" json:"started_at,omitempty"`
	DestroyedAt     *time.Time     `gorm:"" json:"destroyed_at,omitempty"`
	CreatedAt       time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定题目环境实例表表名。
func (ChallengeEnvironment) TableName() string {
	return "challenge_environments"
}

// AdTeamChain 攻防赛队伍链表。
type AdTeamChain struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CompetitionID       int64          `gorm:"not null;uniqueIndex:uk_ad_team_chains" json:"competition_id,string"`
	GroupID             int64          `gorm:"not null;index" json:"group_id,string"`
	TeamID              int64          `gorm:"not null;uniqueIndex:uk_ad_team_chains" json:"team_id,string"`
	ChainRPCURL         *string        `gorm:"column:chain_rpc_url;type:varchar(500)" json:"chain_rpc_url,omitempty"`
	ChainWSURL          *string        `gorm:"column:chain_ws_url;type:varchar(500)" json:"chain_ws_url,omitempty"`
	DeployedContracts   datatypes.JSON `gorm:"column:deployed_contracts;type:jsonb" json:"deployed_contracts,omitempty"`
	CurrentPatchVersion int            `gorm:"not null;default:0" json:"current_patch_version"`
	Status              int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	CreatedAt           time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定攻防赛队伍链表表名。
func (AdTeamChain) TableName() string {
	return "ad_team_chains"
}
