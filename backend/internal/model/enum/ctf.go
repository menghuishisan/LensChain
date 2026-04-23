// ctf.go
// 模块05 — CTF竞赛：统一维护竞赛、题目、团队、攻防赛等领域枚举。
// 该文件只表达状态值与文本映射，避免在 service/handler 中散落魔法值。

package enum

// ========== competitions ==========

const (
	CompetitionTypeJeopardy      = 1 // 解题赛
	CompetitionTypeAttackDefense = 2 // 攻防对抗赛
)

// CompetitionTypeText 竞赛类型文本映射。
var CompetitionTypeText = map[int16]string{
	CompetitionTypeJeopardy:      "解题赛",
	CompetitionTypeAttackDefense: "攻防对抗赛",
}

// GetCompetitionTypeText 获取竞赛类型文本。
func GetCompetitionTypeText(value int16) string {
	if text, ok := CompetitionTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidCompetitionType 校验竞赛类型是否合法。
func IsValidCompetitionType(value int16) bool {
	_, ok := CompetitionTypeText[value]
	return ok
}

const (
	CompetitionScopePlatform = 1 // 平台级
	CompetitionScopeSchool   = 2 // 校级
)

// CompetitionScopeText 竞赛范围文本映射。
var CompetitionScopeText = map[int16]string{
	CompetitionScopePlatform: "平台级",
	CompetitionScopeSchool:   "校级",
}

// GetCompetitionScopeText 获取竞赛范围文本。
func GetCompetitionScopeText(value int16) string {
	if text, ok := CompetitionScopeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidCompetitionScope 校验竞赛范围是否合法。
func IsValidCompetitionScope(value int16) bool {
	_, ok := CompetitionScopeText[value]
	return ok
}

const (
	CompetitionStatusDraft        = 1 // 草稿
	CompetitionStatusRegistration = 2 // 报名中
	CompetitionStatusRunning      = 3 // 进行中
	CompetitionStatusEnded        = 4 // 已结束
	CompetitionStatusArchived     = 5 // 已归档
)

// CompetitionStatusText 竞赛状态文本映射。
var CompetitionStatusText = map[int16]string{
	CompetitionStatusDraft:        "草稿",
	CompetitionStatusRegistration: "报名中",
	CompetitionStatusRunning:      "进行中",
	CompetitionStatusEnded:        "已结束",
	CompetitionStatusArchived:     "已归档",
}

// GetCompetitionStatusText 获取竞赛状态文本。
func GetCompetitionStatusText(value int16) string {
	if text, ok := CompetitionStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidCompetitionStatus 校验竞赛状态是否合法。
func IsValidCompetitionStatus(value int16) bool {
	_, ok := CompetitionStatusText[value]
	return ok
}

const (
	TeamModeIndividual = 1 // 个人赛
	TeamModeFree       = 2 // 自由组队
	TeamModeAssigned   = 3 // 指定组队
)

// TeamModeText 参赛模式文本映射。
var TeamModeText = map[int16]string{
	TeamModeIndividual: "个人赛",
	TeamModeFree:       "自由组队",
	TeamModeAssigned:   "指定组队",
}

// GetTeamModeText 获取参赛模式文本。
func GetTeamModeText(value int16) string {
	if text, ok := TeamModeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidTeamMode 校验参赛模式是否合法。
func IsValidTeamMode(value int16) bool {
	_, ok := TeamModeText[value]
	return ok
}

// ========== challenges ==========

const (
	ChallengeCategoryWeb        = "web"        // Web安全
	ChallengeCategoryCrypto     = "crypto"     // 密码学
	ChallengeCategoryContract   = "contract"   // 智能合约安全
	ChallengeCategoryBlockchain = "blockchain" // 链上分析
	ChallengeCategoryReverse    = "reverse"    // 逆向
	ChallengeCategoryMisc       = "misc"       // 杂项
)

// ChallengeCategoryText 题目类型文本映射。
var ChallengeCategoryText = map[string]string{
	ChallengeCategoryWeb:        "Web安全",
	ChallengeCategoryCrypto:     "密码学",
	ChallengeCategoryContract:   "智能合约安全",
	ChallengeCategoryBlockchain: "链上分析",
	ChallengeCategoryReverse:    "逆向",
	ChallengeCategoryMisc:       "杂项",
}

// GetChallengeCategoryText 获取题目类型文本。
func GetChallengeCategoryText(value string) string {
	if text, ok := ChallengeCategoryText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidChallengeCategory 校验题目类型是否合法。
func IsValidChallengeCategory(value string) bool {
	_, ok := ChallengeCategoryText[value]
	return ok
}

const (
	CtfDifficultyWarmup = 1 // Warmup
	CtfDifficultyEasy   = 2 // Easy
	CtfDifficultyMedium = 3 // Medium
	CtfDifficultyHard   = 4 // Hard
	CtfDifficultyInsane = 5 // Insane
)

// CtfDifficultyText 题目难度文本映射。
var CtfDifficultyText = map[int16]string{
	CtfDifficultyWarmup: "Warmup",
	CtfDifficultyEasy:   "Easy",
	CtfDifficultyMedium: "Medium",
	CtfDifficultyHard:   "Hard",
	CtfDifficultyInsane: "Insane",
}

// GetCtfDifficultyText 获取题目难度文本。
func GetCtfDifficultyText(value int16) string {
	if text, ok := CtfDifficultyText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidCtfDifficulty 校验题目难度是否合法。
func IsValidCtfDifficulty(value int16) bool {
	_, ok := CtfDifficultyText[value]
	return ok
}

const (
	FlagTypeStatic  = 1 // 静态Flag
	FlagTypeDynamic = 2 // 动态Flag
	FlagTypeOnChain = 3 // 链上状态验证
)

// FlagTypeText Flag类型文本映射。
var FlagTypeText = map[int16]string{
	FlagTypeStatic:  "静态Flag",
	FlagTypeDynamic: "动态Flag",
	FlagTypeOnChain: "链上状态验证",
}

// GetFlagTypeText 获取 Flag 类型文本。
func GetFlagTypeText(value int16) string {
	if text, ok := FlagTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidFlagType 校验 Flag 类型是否合法。
func IsValidFlagType(value int16) bool {
	_, ok := FlagTypeText[value]
	return ok
}

const (
	RuntimeModeIsolated = 1 // 独立链模式
	RuntimeModeForked   = 2 // Fork模式
)

// RuntimeModeText 题目运行时模式文本映射。
var RuntimeModeText = map[int16]string{
	RuntimeModeIsolated: "独立链模式",
	RuntimeModeForked:   "Fork模式",
}

// GetRuntimeModeText 获取题目运行时模式文本。
func GetRuntimeModeText(value int16) string {
	if text, ok := RuntimeModeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidRuntimeMode 校验题目运行时模式是否合法。
func IsValidRuntimeMode(value int16) bool {
	_, ok := RuntimeModeText[value]
	return ok
}

const (
	ChallengeSourceSWC      = 1 // SWC导入
	ChallengeSourceTemplate = 2 // 参数化模板
	ChallengeSourceCustom   = 3 // 完全自定义
)

// ChallengeSourcePathText 题目来源路径文本映射。
var ChallengeSourcePathText = map[int16]string{
	ChallengeSourceSWC:      "SWC导入",
	ChallengeSourceTemplate: "参数化模板",
	ChallengeSourceCustom:   "完全自定义",
}

// GetChallengeSourcePathText 获取题目来源路径文本。
func GetChallengeSourcePathText(value int16) string {
	if text, ok := ChallengeSourcePathText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidChallengeSourcePath 校验题目来源路径是否合法。
func IsValidChallengeSourcePath(value int16) bool {
	_, ok := ChallengeSourcePathText[value]
	return ok
}

const (
	ChallengeStatusDraft    = 1 // 草稿
	ChallengeStatusPending  = 2 // 待审核
	ChallengeStatusApproved = 3 // 已通过
	ChallengeStatusRejected = 4 // 已拒绝
)

// ChallengeStatusText 题目状态文本映射。
var ChallengeStatusText = map[int16]string{
	ChallengeStatusDraft:    "草稿",
	ChallengeStatusPending:  "待审核",
	ChallengeStatusApproved: "已通过",
	ChallengeStatusRejected: "已拒绝",
}

// GetChallengeStatusText 获取题目状态文本。
func GetChallengeStatusText(value int16) string {
	if text, ok := ChallengeStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidChallengeStatus 校验题目状态是否合法。
func IsValidChallengeStatus(value int16) bool {
	_, ok := ChallengeStatusText[value]
	return ok
}

// ========== challenge_assertions ==========

const (
	AssertionTypeBalance      = "balance_check"       // ETH余额检查
	AssertionTypeTokenBalance = "token_balance_check" // ERC20代币余额检查
	AssertionTypeStorage      = "storage_check"       // 存储槽值检查
	AssertionTypeOwner        = "owner_check"         // owner检查
	AssertionTypeEvent        = "event_check"         // 事件检查
	AssertionTypeCode         = "code_check"          // 代码检查
	AssertionTypeCustomScript = "custom_script"       // 自定义脚本验证
)

// AssertionTypeText 断言类型文本映射。
var AssertionTypeText = map[string]string{
	AssertionTypeBalance:      "ETH余额检查",
	AssertionTypeTokenBalance: "ERC20代币余额检查",
	AssertionTypeStorage:      "存储槽值检查",
	AssertionTypeOwner:        "Owner检查",
	AssertionTypeEvent:        "事件检查",
	AssertionTypeCode:         "代码检查",
	AssertionTypeCustomScript: "自定义脚本验证",
}

// GetAssertionTypeText 获取断言类型文本。
func GetAssertionTypeText(value string) string {
	if text, ok := AssertionTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAssertionType 校验断言类型是否合法。
func IsValidAssertionType(value string) bool {
	_, ok := AssertionTypeText[value]
	return ok
}

// AssertionOperatorOptions 支持的断言比较运算符。
var AssertionOperatorOptions = []string{"eq", "ne", "gt", "gte", "lt", "lte", "contains"}

// IsValidAssertionOperator 校验断言运算符是否合法。
func IsValidAssertionOperator(value string) bool {
	for _, item := range AssertionOperatorOptions {
		if item == value {
			return true
		}
	}
	return false
}

// ========== review / verification ==========

const (
	ReviewActionApprove = 1 // 通过
	ReviewActionReject  = 2 // 拒绝
)

// ReviewActionText 审核动作文本映射。
var ReviewActionText = map[int16]string{
	ReviewActionApprove: "通过",
	ReviewActionReject:  "拒绝",
}

// GetReviewActionText 获取审核动作文本。
func GetReviewActionText(value int16) string {
	if text, ok := ReviewActionText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidReviewAction 校验审核动作是否合法。
func IsValidReviewAction(value int16) bool {
	_, ok := ReviewActionText[value]
	return ok
}

const (
	VerificationStatusRunning = 1 // 进行中
	VerificationStatusPassed  = 2 // 通过
	VerificationStatusFailed  = 3 // 失败
)

// VerificationStatusText 预验证状态文本映射。
var VerificationStatusText = map[int16]string{
	VerificationStatusRunning: "进行中",
	VerificationStatusPassed:  "通过",
	VerificationStatusFailed:  "失败",
}

// GetVerificationStatusText 获取预验证状态文本。
func GetVerificationStatusText(value int16) string {
	if text, ok := VerificationStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidVerificationStatus 校验预验证状态是否合法。
func IsValidVerificationStatus(value int16) bool {
	_, ok := VerificationStatusText[value]
	return ok
}

// ========== teams / registrations / submissions ==========

const (
	TeamStatusForming   = 1 // 组建中
	TeamStatusLocked    = 2 // 已锁定
	TeamStatusDisbanded = 3 // 已解散
)

// TeamStatusText 团队状态文本映射。
var TeamStatusText = map[int16]string{
	TeamStatusForming:   "组建中",
	TeamStatusLocked:    "已锁定",
	TeamStatusDisbanded: "已解散",
}

// GetTeamStatusText 获取团队状态文本。
func GetTeamStatusText(value int16) string {
	if text, ok := TeamStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidTeamStatus 校验团队状态是否合法。
func IsValidTeamStatus(value int16) bool {
	_, ok := TeamStatusText[value]
	return ok
}

const (
	TeamMemberRoleCaptain = 1 // 队长
	TeamMemberRoleMember  = 2 // 队员
)

// TeamMemberRoleText 团队成员角色文本映射。
var TeamMemberRoleText = map[int16]string{
	TeamMemberRoleCaptain: "队长",
	TeamMemberRoleMember:  "队员",
}

// GetTeamMemberRoleText 获取团队成员角色文本。
func GetTeamMemberRoleText(value int16) string {
	if text, ok := TeamMemberRoleText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidTeamMemberRole 校验团队成员角色是否合法。
func IsValidTeamMemberRole(value int16) bool {
	_, ok := TeamMemberRoleText[value]
	return ok
}

const (
	RegistrationStatusRegistered = 1 // 已报名
	RegistrationStatusCanceled   = 2 // 已取消
)

// RegistrationStatusText 报名状态文本映射。
var RegistrationStatusText = map[int16]string{
	RegistrationStatusRegistered: "已报名",
	RegistrationStatusCanceled:   "已取消",
}

// GetRegistrationStatusText 获取报名状态文本。
func GetRegistrationStatusText(value int16) string {
	if text, ok := RegistrationStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidRegistrationStatus 校验报名状态是否合法。
func IsValidRegistrationStatus(value int16) bool {
	_, ok := RegistrationStatusText[value]
	return ok
}

const (
	SubmissionTypeStaticFlag  = 1 // 静态Flag
	SubmissionTypeDynamicFlag = 2 // 动态Flag
	SubmissionTypeAttackTx    = 3 // 攻击交易
)

// SubmissionTypeText 提交类型文本映射。
var SubmissionTypeText = map[int16]string{
	SubmissionTypeStaticFlag:  "静态Flag",
	SubmissionTypeDynamicFlag: "动态Flag",
	SubmissionTypeAttackTx:    "攻击交易",
}

// GetSubmissionTypeText 获取提交类型文本。
func GetSubmissionTypeText(value int16) string {
	if text, ok := SubmissionTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidSubmissionType 校验提交类型是否合法。
func IsValidSubmissionType(value int16) bool {
	_, ok := SubmissionTypeText[value]
	return ok
}

// ========== attack-defense ==========

const (
	AdGroupStatusPreparing = 1 // 准备中
	AdGroupStatusRunning   = 2 // 进行中
	AdGroupStatusFinished  = 3 // 已结束
)

// AdGroupStatusText 攻防赛分组状态文本映射。
var AdGroupStatusText = map[int16]string{
	AdGroupStatusPreparing: "准备中",
	AdGroupStatusRunning:   "进行中",
	AdGroupStatusFinished:  "已结束",
}

// GetAdGroupStatusText 获取攻防赛分组状态文本。
func GetAdGroupStatusText(value int16) string {
	if text, ok := AdGroupStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAdGroupStatus 校验攻防赛分组状态是否合法。
func IsValidAdGroupStatus(value int16) bool {
	_, ok := AdGroupStatusText[value]
	return ok
}

const (
	RoundPhaseAttacking = 1 // 攻击阶段
	RoundPhaseDefending = 2 // 防守阶段
	RoundPhaseSettling  = 3 // 结算阶段
	RoundPhaseCompleted = 4 // 已完成
)

// RoundPhaseText 回合阶段文本映射。
var RoundPhaseText = map[int16]string{
	RoundPhaseAttacking: "攻击阶段",
	RoundPhaseDefending: "防守阶段",
	RoundPhaseSettling:  "结算阶段",
	RoundPhaseCompleted: "已完成",
}

// GetRoundPhaseText 获取回合阶段文本。
func GetRoundPhaseText(value int16) string {
	if text, ok := RoundPhaseText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidRoundPhase 校验回合阶段是否合法。
func IsValidRoundPhase(value int16) bool {
	_, ok := RoundPhaseText[value]
	return ok
}

const (
	TokenChangeInit          = 1 // 初始化
	TokenChangeAttackSteal   = 2 // 攻击窃取
	TokenChangeAttackBonus   = 3 // 攻击奖励
	TokenChangeAttackLoss    = 4 // 被攻击扣除
	TokenChangeDefenseReward = 5 // 防守奖励
	TokenChangeFirstPatch    = 6 // 首补丁奖励
	TokenChangeFirstBlood    = 7 // First Blood奖励
)

// TokenChangeTypeText Token 变动类型文本映射。
var TokenChangeTypeText = map[int16]string{
	TokenChangeInit:          "初始化",
	TokenChangeAttackSteal:   "攻击窃取",
	TokenChangeAttackBonus:   "攻击奖励",
	TokenChangeAttackLoss:    "被攻击扣除",
	TokenChangeDefenseReward: "防守奖励",
	TokenChangeFirstPatch:    "首补丁奖励",
	TokenChangeFirstBlood:    "First Blood奖励",
}

// GetTokenChangeTypeText 获取 Token 变动类型文本。
func GetTokenChangeTypeText(value int16) string {
	if text, ok := TokenChangeTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidTokenChangeType 校验 Token 变动类型是否合法。
func IsValidTokenChangeType(value int16) bool {
	_, ok := TokenChangeTypeText[value]
	return ok
}

const (
	AnnouncementTypeInfo   = 1 // 信息通知
	AnnouncementTypeErrata = 2 // 题目勘误
	AnnouncementTypeRule   = 3 // 规则说明
)

// CtfAnnouncementTypeText 公告类型文本映射。
var CtfAnnouncementTypeText = map[int16]string{
	AnnouncementTypeInfo:   "信息通知",
	AnnouncementTypeErrata: "题目勘误",
	AnnouncementTypeRule:   "规则说明",
}

// GetCtfAnnouncementTypeText 获取公告类型文本。
func GetCtfAnnouncementTypeText(value int16) string {
	if text, ok := CtfAnnouncementTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidCtfAnnouncementType 校验公告类型是否合法。
func IsValidCtfAnnouncementType(value int16) bool {
	_, ok := CtfAnnouncementTypeText[value]
	return ok
}

const (
	ChallengeEnvStatusCreating  = 1 // 创建中
	ChallengeEnvStatusRunning   = 2 // 运行中
	ChallengeEnvStatusStopped   = 3 // 已停止
	ChallengeEnvStatusError     = 4 // 异常
	ChallengeEnvStatusDestroyed = 5 // 已销毁
)

// ChallengeEnvStatusText 题目环境状态文本映射。
var ChallengeEnvStatusText = map[int16]string{
	ChallengeEnvStatusCreating:  "创建中",
	ChallengeEnvStatusRunning:   "运行中",
	ChallengeEnvStatusStopped:   "已停止",
	ChallengeEnvStatusError:     "异常",
	ChallengeEnvStatusDestroyed: "已销毁",
}

// GetChallengeEnvStatusText 获取题目环境状态文本。
func GetChallengeEnvStatusText(value int16) string {
	if text, ok := ChallengeEnvStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidChallengeEnvStatus 校验题目环境状态是否合法。
func IsValidChallengeEnvStatus(value int16) bool {
	_, ok := ChallengeEnvStatusText[value]
	return ok
}

const (
	AdTeamChainStatusCreating = 1 // 创建中
	AdTeamChainStatusRunning  = 2 // 运行中
	AdTeamChainStatusStopped  = 3 // 已停止
	AdTeamChainStatusError    = 4 // 异常
)

// AdTeamChainStatusText 队伍链状态文本映射。
var AdTeamChainStatusText = map[int16]string{
	AdTeamChainStatusCreating: "创建中",
	AdTeamChainStatusRunning:  "运行中",
	AdTeamChainStatusStopped:  "已停止",
	AdTeamChainStatusError:    "异常",
}

// GetAdTeamChainStatusText 获取队伍链状态文本。
func GetAdTeamChainStatusText(value int16) string {
	if text, ok := AdTeamChainStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAdTeamChainStatus 校验队伍链状态是否合法。
func IsValidAdTeamChainStatus(value int16) bool {
	_, ok := AdTeamChainStatusText[value]
	return ok
}
