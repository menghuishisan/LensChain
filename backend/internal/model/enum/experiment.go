// experiment.go
// 模块04 — 实验环境：枚举常量定义
// 对照 docs/modules/04-实验环境/02-数据库设计.md
// 包含镜像来源、镜像状态、实验类型、拓扑模式、判题模式、成绩策略、模板状态、
// 检查点类型、场景来源、场景状态、数据源模式、实例状态、容器状态、快照类型、
// 分组方式、分组状态、消息类型、配额级别、交付阶段、操作动作、场景领域、
// 时间控制模式、链生态、标签分类等枚举

package enum

// ========== 镜像来源类型（experiment_images.source_type） ==========

const (
	ImageSourceTypeOfficial = 1 // 平台官方
	ImageSourceTypeCustom   = 2 // 教师自定义
)

// ImageSourceTypeText 镜像来源类型文本映射
var ImageSourceTypeText = map[int]string{
	ImageSourceTypeOfficial: "平台官方",
	ImageSourceTypeCustom:   "教师自定义",
}

// GetImageSourceTypeText 获取镜像来源类型文本
func GetImageSourceTypeText(t int) string {
	if text, ok := ImageSourceTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidImageSourceType 校验镜像来源类型是否合法
func IsValidImageSourceType(t int) bool {
	_, ok := ImageSourceTypeText[t]
	return ok
}

// ========== 镜像状态（experiment_images.status） ==========

const (
	ImageStatusNormal   = 1 // 正常
	ImageStatusPending  = 2 // 待审核
	ImageStatusOffShelf = 3 // 已下架
	ImageStatusRejected = 4 // 审核拒绝
)

// ImageStatusText 镜像状态文本映射
var ImageStatusText = map[int]string{
	ImageStatusNormal:   "正常",
	ImageStatusPending:  "待审核",
	ImageStatusOffShelf: "已下架",
	ImageStatusRejected: "审核拒绝",
}

// GetImageStatusText 获取镜像状态文本
func GetImageStatusText(s int) string {
	if text, ok := ImageStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidImageStatus 校验镜像状态是否合法
func IsValidImageStatus(s int) bool {
	_, ok := ImageStatusText[s]
	return ok
}

// ========== 镜像版本状态（image_versions.status） ==========

const (
	ImageVersionStatusNormal     = 1 // 正常
	ImageVersionStatusDeprecated = 2 // 已废弃
)

// ImageVersionStatusText 镜像版本状态文本映射
var ImageVersionStatusText = map[int]string{
	ImageVersionStatusNormal:     "正常",
	ImageVersionStatusDeprecated: "已废弃",
}

// GetImageVersionStatusText 获取镜像版本状态文本
func GetImageVersionStatusText(s int) string {
	if text, ok := ImageVersionStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidImageVersionStatus 校验镜像版本状态是否合法
func IsValidImageVersionStatus(s int) bool {
	_, ok := ImageVersionStatusText[s]
	return ok
}

// ========== 实验类型（experiment_templates.experiment_type） ==========

const (
	ExperimentTypeSimulation = 1 // 纯仿真
	ExperimentTypeReal       = 2 // 真实环境
	ExperimentTypeMixed      = 3 // 混合
)

// ExperimentTypeText 实验类型文本映射
var ExperimentTypeText = map[int]string{
	ExperimentTypeSimulation: "纯仿真",
	ExperimentTypeReal:       "真实环境",
	ExperimentTypeMixed:      "混合",
}

// GetExperimentTypeText 获取实验类型文本
func GetExperimentTypeText(t int) string {
	if text, ok := ExperimentTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidExperimentType 校验实验类型是否合法
func IsValidExperimentType(t int) bool {
	_, ok := ExperimentTypeText[t]
	return ok
}

// ========== 拓扑模式（experiment_templates.topology_mode） ==========

const (
	TopologyModeSingleNode  = 1 // 单人单节点
	TopologyModeMultiNode   = 2 // 单人多节点
	TopologyModeCollaborate = 3 // 多人协作组网
	TopologyModeShared      = 4 // 共享基础设施
)

// TopologyModeText 拓扑模式文本映射
var TopologyModeText = map[int]string{
	TopologyModeSingleNode:  "单人单节点",
	TopologyModeMultiNode:   "单人多节点",
	TopologyModeCollaborate: "多人协作组网",
	TopologyModeShared:      "共享基础设施",
}

// GetTopologyModeText 获取拓扑模式文本
func GetTopologyModeText(m int) string {
	if text, ok := TopologyModeText[m]; ok {
		return text
	}
	return "未知"
}

// IsValidTopologyMode 校验拓扑模式是否合法
func IsValidTopologyMode(m int) bool {
	_, ok := TopologyModeText[m]
	return ok
}

// ========== 判题模式（experiment_templates.judge_mode） ==========

const (
	JudgeModeAuto   = 1 // 纯自动
	JudgeModeManual = 2 // 纯手动
	JudgeModeMixed  = 3 // 混合
)

// JudgeModeText 判题模式文本映射
var JudgeModeText = map[int]string{
	JudgeModeAuto:   "纯自动",
	JudgeModeManual: "纯手动",
	JudgeModeMixed:  "混合",
}

// GetJudgeModeText 获取判题模式文本
func GetJudgeModeText(m int) string {
	if text, ok := JudgeModeText[m]; ok {
		return text
	}
	return "未知"
}

// IsValidJudgeMode 校验判题模式是否合法
func IsValidJudgeMode(m int) bool {
	_, ok := JudgeModeText[m]
	return ok
}

// ========== 成绩策略（experiment_templates.score_strategy） ==========

const (
	ScoreStrategyLast    = 1 // 取最后一次
	ScoreStrategyHighest = 2 // 取最高分
)

// ScoreStrategyText 成绩策略文本映射
var ScoreStrategyText = map[int]string{
	ScoreStrategyLast:    "取最后一次",
	ScoreStrategyHighest: "取最高分",
}

// GetScoreStrategyText 获取成绩策略文本
func GetScoreStrategyText(s int) string {
	if text, ok := ScoreStrategyText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidScoreStrategy 校验成绩策略是否合法
func IsValidScoreStrategy(s int) bool {
	_, ok := ScoreStrategyText[s]
	return ok
}

// ========== 模板状态（experiment_templates.status） ==========

const (
	TemplateStatusDraft     = 1 // 草稿
	TemplateStatusPublished = 2 // 已发布
	TemplateStatusOffShelf  = 3 // 已下架
)

// TemplateStatusText 模板状态文本映射
var TemplateStatusText = map[int]string{
	TemplateStatusDraft:     "草稿",
	TemplateStatusPublished: "已发布",
	TemplateStatusOffShelf:  "已下架",
}

// GetTemplateStatusText 获取模板状态文本
func GetTemplateStatusText(s int) string {
	if text, ok := TemplateStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidTemplateStatus 校验模板状态是否合法
func IsValidTemplateStatus(s int) bool {
	_, ok := TemplateStatusText[s]
	return ok
}

// ========== 检查点类型（experiment_checkpoints.check_type） ==========

const (
	CheckTypeScript    = 1 // 后端脚本验证
	CheckTypeManual    = 2 // 手动评分
	CheckTypeSimAssert = 3 // SimEngine状态断言
)

// CheckTypeText 检查点类型文本映射
var CheckTypeText = map[int]string{
	CheckTypeScript:    "后端脚本验证",
	CheckTypeManual:    "手动评分",
	CheckTypeSimAssert: "SimEngine状态断言",
}

// GetCheckTypeText 获取检查点类型文本
func GetCheckTypeText(t int) string {
	if text, ok := CheckTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidCheckType 校验检查点类型是否合法
func IsValidCheckType(t int) bool {
	_, ok := CheckTypeText[t]
	return ok
}

// ========== 检查点范围（experiment_checkpoints.scope） ==========

const (
	CheckpointScopePersonal = 1 // 个人
	CheckpointScopeGroup    = 2 // 组级
)

// CheckpointScopeText 检查点范围文本映射
var CheckpointScopeText = map[int]string{
	CheckpointScopePersonal: "个人",
	CheckpointScopeGroup:    "组级",
}

// GetCheckpointScopeText 获取检查点范围文本
func GetCheckpointScopeText(s int) string {
	if text, ok := CheckpointScopeText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidCheckpointScope 校验检查点范围是否合法
func IsValidCheckpointScope(s int) bool {
	_, ok := CheckpointScopeText[s]
	return ok
}

// ========== 场景来源（sim_scenarios.source_type） ==========

const (
	ScenarioSourceTypeBuiltIn = 1 // 平台内置
	ScenarioSourceTypeCustom  = 2 // 教师自定义
)

// ScenarioSourceTypeText 场景来源文本映射
var ScenarioSourceTypeText = map[int]string{
	ScenarioSourceTypeBuiltIn: "平台内置",
	ScenarioSourceTypeCustom:  "教师自定义",
}

// GetScenarioSourceTypeText 获取场景来源文本
func GetScenarioSourceTypeText(t int) string {
	if text, ok := ScenarioSourceTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidScenarioSourceType 校验场景来源是否合法
func IsValidScenarioSourceType(t int) bool {
	_, ok := ScenarioSourceTypeText[t]
	return ok
}

// ========== 场景状态（sim_scenarios.status） ==========

const (
	ScenarioStatusNormal   = 1 // 正常
	ScenarioStatusPending  = 2 // 待审核
	ScenarioStatusOffShelf = 3 // 已下架
	ScenarioStatusRejected = 4 // 审核拒绝
)

// ScenarioStatusText 场景状态文本映射
var ScenarioStatusText = map[int]string{
	ScenarioStatusNormal:   "正常",
	ScenarioStatusPending:  "待审核",
	ScenarioStatusOffShelf: "已下架",
	ScenarioStatusRejected: "审核拒绝",
}

// GetScenarioStatusText 获取场景状态文本
func GetScenarioStatusText(s int) string {
	if text, ok := ScenarioStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidScenarioStatus 校验场景状态是否合法
func IsValidScenarioStatus(s int) bool {
	_, ok := ScenarioStatusText[s]
	return ok
}

// ========== 场景分类（sim_scenarios.category） ==========

const (
	ScenarioCategoryNodeNetwork    = "node_network"
	ScenarioCategoryConsensus      = "consensus"
	ScenarioCategoryCryptography   = "cryptography"
	ScenarioCategoryDataStructure  = "data_structure"
	ScenarioCategoryTransaction    = "transaction"
	ScenarioCategorySmartContract  = "smart_contract"
	ScenarioCategoryAttackSecurity = "attack_security"
	ScenarioCategoryEconomic       = "economic"
)

// ScenarioCategoryText 场景分类文本映射。
var ScenarioCategoryText = map[string]string{
	ScenarioCategoryNodeNetwork:    "节点网络",
	ScenarioCategoryConsensus:      "共识过程",
	ScenarioCategoryCryptography:   "密码学",
	ScenarioCategoryDataStructure:  "数据结构",
	ScenarioCategoryTransaction:    "交易流程",
	ScenarioCategorySmartContract:  "智能合约",
	ScenarioCategoryAttackSecurity: "攻击安全",
	ScenarioCategoryEconomic:       "经济模型",
}

// GetScenarioCategoryText 获取场景分类文本。
func GetScenarioCategoryText(category string) string {
	if text, ok := ScenarioCategoryText[category]; ok {
		return text
	}
	return category
}

// ========== 数据源模式（sim_scenarios.data_source_mode） ==========

const (
	DataSourceModeSim  = 1 // 仿真模式
	DataSourceModeReal = 2 // 采集模式
	DataSourceModeDual = 3 // 双模式
)

// DataSourceModeText 数据源模式文本映射
var DataSourceModeText = map[int]string{
	DataSourceModeSim:  "仿真模式",
	DataSourceModeReal: "采集模式",
	DataSourceModeDual: "双模式",
}

// GetDataSourceModeText 获取数据源模式文本
func GetDataSourceModeText(m int) string {
	if text, ok := DataSourceModeText[m]; ok {
		return text
	}
	return "未知"
}

// IsValidDataSourceMode 校验数据源模式是否合法
func IsValidDataSourceMode(m int) bool {
	_, ok := DataSourceModeText[m]
	return ok
}

// ========== 实例状态（experiment_instances.status） ==========

const (
	InstanceStatusCreating  = 1 // 创建中
	InstanceStatusRunning   = 2 // 运行中
	InstanceStatusPaused    = 3 // 暂停
	InstanceStatusSubmitted = 4 // 已提交
	InstanceStatusTimeout   = 5 // 已超时
	InstanceStatusDestroyed = 6 // 已销毁
	InstanceStatusError     = 7 // 错误
	InstanceStatusQueued    = 8 // 排队中
	InstanceStatusRestoring = 9 // 恢复中
)

// InstanceStatusText 实例状态文本映射
var InstanceStatusText = map[int]string{
	InstanceStatusCreating:  "创建中",
	InstanceStatusRunning:   "运行中",
	InstanceStatusPaused:    "暂停",
	InstanceStatusSubmitted: "已提交",
	InstanceStatusTimeout:   "已超时",
	InstanceStatusDestroyed: "已销毁",
	InstanceStatusError:     "错误",
	InstanceStatusQueued:    "排队中",
	InstanceStatusRestoring: "恢复中",
}

// GetInstanceStatusText 获取实例状态文本
func GetInstanceStatusText(s int) string {
	if text, ok := InstanceStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidInstanceStatus 校验实例状态是否合法
func IsValidInstanceStatus(s int) bool {
	_, ok := InstanceStatusText[s]
	return ok
}

// ========== 容器状态（instance_containers.status） ==========

const (
	ContainerStatusCreating = 1 // 创建中
	ContainerStatusRunning  = 2 // 运行中
	ContainerStatusStopped  = 3 // 已停止
	ContainerStatusError    = 4 // 错误
)

// ContainerStatusText 容器状态文本映射
var ContainerStatusText = map[int]string{
	ContainerStatusCreating: "创建中",
	ContainerStatusRunning:  "运行中",
	ContainerStatusStopped:  "已停止",
	ContainerStatusError:    "错误",
}

// GetContainerStatusText 获取容器状态文本
func GetContainerStatusText(s int) string {
	if text, ok := ContainerStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidContainerStatus 校验容器状态是否合法
func IsValidContainerStatus(s int) bool {
	_, ok := ContainerStatusText[s]
	return ok
}

// ========== 快照类型（instance_snapshots.snapshot_type） ==========

const (
	SnapshotTypeScheduled = 1 // 定时
	SnapshotTypePause     = 2 // 暂停
	SnapshotTypeTimeout   = 3 // 超时
	SnapshotTypeSimEngine = 4 // SimEngine
)

// SnapshotTypeText 快照类型文本映射
var SnapshotTypeText = map[int]string{
	SnapshotTypeScheduled: "定时",
	SnapshotTypePause:     "暂停",
	SnapshotTypeTimeout:   "超时",
	SnapshotTypeSimEngine: "SimEngine",
}

// GetSnapshotTypeText 获取快照类型文本
func GetSnapshotTypeText(t int) string {
	if text, ok := SnapshotTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidSnapshotType 校验快照类型是否合法
func IsValidSnapshotType(t int) bool {
	_, ok := SnapshotTypeText[t]
	return ok
}

// ========== 分组方式（experiment_groups.group_method） ==========

const (
	GroupMethodManual = 1 // 手动
	GroupMethodSelf   = 2 // 自选
	GroupMethodRandom = 3 // 随机
)

// GroupMethodText 分组方式文本映射
var GroupMethodText = map[int]string{
	GroupMethodManual: "手动",
	GroupMethodSelf:   "自选",
	GroupMethodRandom: "随机",
}

// GetGroupMethodText 获取分组方式文本
func GetGroupMethodText(m int) string {
	if text, ok := GroupMethodText[m]; ok {
		return text
	}
	return "未知"
}

// IsValidGroupMethod 校验分组方式是否合法
func IsValidGroupMethod(m int) bool {
	_, ok := GroupMethodText[m]
	return ok
}

// ========== 分组状态（experiment_groups.status） ==========

const (
	GroupStatusForming   = 1 // 组建中
	GroupStatusReady     = 2 // 已就绪
	GroupStatusRunning   = 3 // 实验中
	GroupStatusCompleted = 4 // 已完成
)

// GroupStatusText 分组状态文本映射
var GroupStatusText = map[int]string{
	GroupStatusForming:   "组建中",
	GroupStatusReady:     "已就绪",
	GroupStatusRunning:   "实验中",
	GroupStatusCompleted: "已完成",
}

// GetGroupStatusText 获取分组状态文本
func GetGroupStatusText(s int) string {
	if text, ok := GroupStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidGroupStatus 校验分组状态是否合法
func IsValidGroupStatus(s int) bool {
	_, ok := GroupStatusText[s]
	return ok
}

// ========== 消息类型（group_messages.message_type） ==========

const (
	MessageTypeText   = 1 // 文本
	MessageTypeSystem = 2 // 系统通知
)

// MessageTypeText_ 消息类型文本映射（加下划线避免与 MessageTypeText 常量名冲突）
var MessageTypeText_ = map[int]string{
	MessageTypeText:   "文本",
	MessageTypeSystem: "系统通知",
}

// GetMessageTypeText 获取消息类型文本
func GetMessageTypeText(t int) string {
	if text, ok := MessageTypeText_[t]; ok {
		return text
	}
	return "未知"
}

// IsValidMessageType 校验消息类型是否合法
func IsValidMessageType(t int) bool {
	_, ok := MessageTypeText_[t]
	return ok
}

// ========== 配额级别（resource_quotas.level） ==========

const (
	QuotaLevelSchool = 1 // 学校
	QuotaLevelCourse = 2 // 课程
)

// QuotaLevelText 配额级别文本映射
var QuotaLevelText = map[int]string{
	QuotaLevelSchool: "学校",
	QuotaLevelCourse: "课程",
}

// GetQuotaLevelText 获取配额级别文本
func GetQuotaLevelText(l int) string {
	if text, ok := QuotaLevelText[l]; ok {
		return text
	}
	return "未知"
}

// IsValidQuotaLevel 校验配额级别是否合法
func IsValidQuotaLevel(l int) bool {
	_, ok := QuotaLevelText[l]
	return ok
}

// ========== 交付阶段（experiment_templates.delivery_phase） ==========

const (
	DeliveryPhaseOne   = 1 // 第一阶段
	DeliveryPhaseTwo   = 2 // 第二阶段
	DeliveryPhaseThree = 3 // 第三阶段
)

// DeliveryPhaseText 交付阶段文本映射
var DeliveryPhaseText = map[int]string{
	DeliveryPhaseOne:   "第一阶段",
	DeliveryPhaseTwo:   "第二阶段",
	DeliveryPhaseThree: "第三阶段",
}

// GetDeliveryPhaseText 获取交付阶段文本
func GetDeliveryPhaseText(p int) string {
	if text, ok := DeliveryPhaseText[p]; ok {
		return text
	}
	return "未知"
}

// IsValidDeliveryPhase 校验交付阶段是否合法
func IsValidDeliveryPhase(p int) bool {
	_, ok := DeliveryPhaseText[p]
	return ok
}

// ========== 操作动作（instance_operation_logs.action） ==========

const (
	ActionStart           = "start"            // 启动
	ActionPause           = "pause"            // 暂停
	ActionResume          = "resume"           // 恢复
	ActionRestart         = "restart"          // 重启
	ActionSubmit          = "submit"           // 提交
	ActionDestroy         = "destroy"          // 销毁
	ActionForceDestroy    = "force_destroy"    // 强制销毁
	ActionCheckpoint      = "checkpoint_check" // 检查点验证
	ActionTerminalCommand = "terminal_command" // 终端命令
	ActionSimInteraction  = "sim_interaction"  // 仿真交互
	ActionSimTimeControl  = "sim_time_control" // 仿真时间控制
	ActionManualGrade     = "manual_grade"     // 手动评分
	ActionSnapshotCreate  = "snapshot_create"  // 创建快照
	ActionSnapshotRestore = "snapshot_restore" // 恢复快照
	ActionReportSubmit    = "report_submit"    // 提交报告
	ActionReportUpdate    = "report_update"    // 更新报告
	ActionGuidanceMessage = "guidance_message" // 教师指导消息
)

// OperationActionOptions 操作动作合法值列表
var OperationActionOptions = []string{
	ActionStart,
	ActionPause,
	ActionResume,
	ActionRestart,
	ActionSubmit,
	ActionDestroy,
	ActionForceDestroy,
	ActionCheckpoint,
	ActionTerminalCommand,
	ActionSimInteraction,
	ActionSimTimeControl,
	ActionManualGrade,
	ActionSnapshotCreate,
	ActionSnapshotRestore,
	ActionReportSubmit,
	ActionReportUpdate,
	ActionGuidanceMessage,
}

// IsValidOperationAction 校验操作动作是否合法
func IsValidOperationAction(s string) bool {
	for _, v := range OperationActionOptions {
		if v == s {
			return true
		}
	}
	return false
}

// ========== 场景领域（sim_scenarios.scene_category） ==========

const (
	SceneCategoryNodeNetwork    = "node_network"    // 节点网络
	SceneCategoryConsensus      = "consensus"       // 共识过程
	SceneCategoryCryptography   = "cryptography"    // 密码学
	SceneCategoryDataStructure  = "data_structure"  // 数据结构
	SceneCategoryTransaction    = "transaction"     // 交易流程
	SceneCategorySmartContract  = "smart_contract"  // 智能合约
	SceneCategoryAttackSecurity = "attack_security" // 攻击安全
	SceneCategoryEconomic       = "economic"        // 经济模型
)

// SceneCategoryOptions 场景领域合法值列表
var SceneCategoryOptions = []string{
	SceneCategoryNodeNetwork,
	SceneCategoryConsensus,
	SceneCategoryCryptography,
	SceneCategoryDataStructure,
	SceneCategoryTransaction,
	SceneCategorySmartContract,
	SceneCategoryAttackSecurity,
	SceneCategoryEconomic,
}

// IsValidSceneCategory 校验场景领域是否合法
func IsValidSceneCategory(s string) bool {
	for _, v := range SceneCategoryOptions {
		if v == s {
			return true
		}
	}
	return false
}

// ========== 时间控制模式（sim_scenarios.time_control_mode） ==========

const (
	TimeControlProcess    = "process"    // 流程驱动
	TimeControlReactive   = "reactive"   // 响应式
	TimeControlContinuous = "continuous" // 连续推进
)

// TimeControlModeOptions 时间控制模式合法值列表
var TimeControlModeOptions = []string{
	TimeControlProcess,
	TimeControlReactive,
	TimeControlContinuous,
}

// IsValidTimeControlMode 校验时间控制模式是否合法
func IsValidTimeControlMode(s string) bool {
	for _, v := range TimeControlModeOptions {
		if v == s {
			return true
		}
	}
	return false
}

// ========== 链生态（experiment_images.ecosystem / sim_scenarios.ecosystem） ==========

const (
	EcosystemBitcoin    = "bitcoin"    // 比特币
	EcosystemEthereum   = "ethereum"   // 以太坊
	EcosystemFabric     = "fabric"     // Hyperledger Fabric
	EcosystemChainmaker = "chainmaker" // 长安链
	EcosystemFisco      = "fisco"      // FISCO BCOS
	EcosystemSolana     = "solana"     // Solana
	EcosystemPolkadot   = "polkadot"   // Polkadot
	EcosystemCosmos     = "cosmos"     // Cosmos
	EcosystemMove       = "move"       // Move 系
	EcosystemGeneral    = "general"    // 通用
)

// EcosystemOptions 链生态合法值列表
var EcosystemOptions = []string{
	EcosystemBitcoin,
	EcosystemEthereum,
	EcosystemFabric,
	EcosystemChainmaker,
	EcosystemFisco,
	EcosystemSolana,
	EcosystemPolkadot,
	EcosystemCosmos,
	EcosystemMove,
	EcosystemGeneral,
}

// IsValidEcosystem 校验链生态是否合法
func IsValidEcosystem(s string) bool {
	for _, v := range EcosystemOptions {
		if v == s {
			return true
		}
	}
	return false
}

// IsValidCollectorEcosystem 校验 Collector 内置采集生态是否合法。
func IsValidCollectorEcosystem(s string) bool {
	switch s {
	case EcosystemEthereum, EcosystemFabric, EcosystemChainmaker, EcosystemFisco:
		return true
	default:
		return false
	}
}

// ========== 标签分类（experiment_tags.category） ==========

const (
	TagCategoryEcosystem  = "ecosystem"  // 链生态
	TagCategoryType       = "type"       // 类型
	TagCategoryDifficulty = "difficulty" // 难度
	TagCategoryCustom     = "custom"     // 自定义
)

// TagCategoryOptions 标签分类合法值列表
var TagCategoryOptions = []string{
	TagCategoryEcosystem,
	TagCategoryType,
	TagCategoryDifficulty,
	TagCategoryCustom,
}

// IsValidTagCategory 校验标签分类是否合法
func IsValidTagCategory(s string) bool {
	for _, v := range TagCategoryOptions {
		if v == s {
			return true
		}
	}
	return false
}
