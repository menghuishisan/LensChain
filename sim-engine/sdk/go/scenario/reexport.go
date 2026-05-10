// 模块：sim-engine/sdk/go/scenario
// 文件职责：把 framework 的原语构造器、几何工具、协议字段解码工具通过 var 引用方式重导出，
//          让教师写场景时仅 import 本包即可使用全部 framework 工具。
//
// Go 不支持函数 alias，因此用 `var X = framework.X` 把函数指针重新绑定到 sdk 包符号。
// 因为指针指向同一函数，sdk.X(...) 的调用与直接 framework.X(...) 完全等价。
//
// 命名约定（详 sim-engine/AGENTS.md §0.3 + §12.1.6）：
//   - 默认构造器（如 PrimNode / PrimRingLayout / PrimMatrixLayout）输出**无绝对坐标**版本，
//     由前端按父布局原语推导位置——这是 90%+ 场景的正确路径；
//   - *At 后缀构造器（如 PrimNodeAt / PrimRingLayoutAt）输出带逻辑坐标 0~1 版本，
//     **仅用于绝对定位**（自由分布、对照展示、画布角落 HUD）。

package scenario

import fw "github.com/lenschain/sim-engine/framework"

// =====================================================================
// 47 原语构造器（详 framework/primitive.go）
// =====================================================================

// 几何类（8 个）：默认无坐标 + *At 绝对定位变体（适用 node / bar / ring）。
var (
	PrimNode      = fw.PrimNode
	PrimNodeAt    = fw.PrimNodeAt
	PrimEdge      = fw.PrimEdge
	PrimBar       = fw.PrimBar
	PrimBarAt     = fw.PrimBarAt
	PrimPolygon   = fw.PrimPolygon
	PrimGridCell  = fw.PrimGridCell
	PrimRing      = fw.PrimRing
	PrimRingAt    = fw.PrimRingAt
	PrimCurve     = fw.PrimCurve
	PrimArea      = fw.PrimArea
)

// 动效类（7 个）：均通过 anchor_id 锚定，自身不含坐标。
var (
	PrimParticleStream = fw.PrimParticleStream
	PrimBurst          = fw.PrimBurst
	PrimPulse          = fw.PrimPulse
	PrimTrail          = fw.PrimTrail
	PrimGlow           = fw.PrimGlow
	PrimShake          = fw.PrimShake
	PrimShiftAnimation = fw.PrimShiftAnimation
)

// 布局类（6 个）：默认无坐标 + *At 绝对定位变体。
var (
	PrimHorizontalLane    = fw.PrimHorizontalLane
	PrimHorizontalLaneAt  = fw.PrimHorizontalLaneAt
	PrimStack             = fw.PrimStack
	PrimStackAt           = fw.PrimStackAt
	PrimRingLayout        = fw.PrimRingLayout
	PrimRingLayoutAt      = fw.PrimRingLayoutAt
	PrimTreeLayout        = fw.PrimTreeLayout
	PrimGraphLayout       = fw.PrimGraphLayout
	PrimMatrixLayout      = fw.PrimMatrixLayout
	PrimMatrixLayoutAt    = fw.PrimMatrixLayoutAt
)

// 数据展示类（7 个）：默认无坐标 / 通过 anchor_id 锚定 + *At 绝对定位变体。
var (
	PrimLabel           = fw.PrimLabel
	PrimLabelAt         = fw.PrimLabelAt
	PrimTooltip         = fw.PrimTooltip
	PrimAnnotation      = fw.PrimAnnotation
	PrimRegisterRow     = fw.PrimRegisterRow
	PrimRegisterRowAt   = fw.PrimRegisterRowAt
	PrimMathPipeline    = fw.PrimMathPipeline
	PrimMathPipelineAt  = fw.PrimMathPipelineAt
	PrimCodeBlock       = fw.PrimCodeBlock
	PrimCodeBlockAt     = fw.PrimCodeBlockAt
	PrimMathFormula     = fw.PrimMathFormula
	PrimMathFormulaAt   = fw.PrimMathFormulaAt
)

// 状态指示类（8 个）：默认无坐标 + *At 绝对定位变体（适用 progress_bar / link_indicator）。
var (
	PrimPhaseProgress       = fw.PrimPhaseProgress
	PrimProgressBar         = fw.PrimProgressBar
	PrimProgressBarAt       = fw.PrimProgressBarAt
	PrimTargetZone          = fw.PrimTargetZone
	PrimLinkIndicator       = fw.PrimLinkIndicator
	PrimLinkIndicatorAt     = fw.PrimLinkIndicatorAt
	PrimExternalEventMarker = fw.PrimExternalEventMarker
	PrimErrorOverlay        = fw.PrimErrorOverlay
	PrimVerifyPathHighlight = fw.PrimVerifyPathHighlight
	PrimRiskGauge           = fw.PrimRiskGauge
)

// 领域复合类（11 个）：默认无坐标 / 无单元尺寸 + *At 绝对定位 / 含尺寸变体。
var (
	PrimVoteMatrix    = fw.PrimVoteMatrix
	PrimVoteMatrixAt  = fw.PrimVoteMatrixAt
	PrimDualTrack     = fw.PrimDualTrack
	PrimTimeWheel     = fw.PrimTimeWheel
	PrimTimeWheelAt   = fw.PrimTimeWheelAt
	PrimPieChart      = fw.PrimPieChart
	PrimPieChartAt    = fw.PrimPieChartAt
	PrimSankeyFlow    = fw.PrimSankeyFlow
	PrimHeatMap       = fw.PrimHeatMap
	PrimHeatMapAt     = fw.PrimHeatMapAt
	PrimMempoolSlot   = fw.PrimMempoolSlot
	PrimBridgeTrack   = fw.PrimBridgeTrack
	PrimCodeMarker    = fw.PrimCodeMarker
	PrimPartitionZone = fw.PrimPartitionZone
	PrimCurvePoint    = fw.PrimCurvePoint
)

// =====================================================================
// 几何工具（详 framework/layout.go）
//
// 仅服务于少数绝对定位场景；参与父布局原语的节点不应使用本工具，应通过默认构造器装入布局原语。
// =====================================================================

var (
	CircleLayout       = fw.CircleLayout
	LinearLayout       = fw.LinearLayout
	VerticalLayout     = fw.VerticalLayout
	GridLayout         = fw.GridLayout
	MidPoint           = fw.MidPoint
	Distance           = fw.Distance
	PointsToParamSlice = fw.PointsToParamSlice
)

// =====================================================================
// 协议字段解码工具（详 framework/jsonparam.go）
// =====================================================================

var (
	MapStr          = fw.MapStr
	MapInt          = fw.MapInt
	MapBool         = fw.MapBool
	StringValue     = fw.StringValue
	NumberValue     = fw.NumberValue
	BoolValue       = fw.BoolValue
	ToStringSliceOr = fw.ToStringSliceOr
	ToBoolSliceOr   = fw.ToBoolSliceOr
	ToIntSliceOr    = fw.ToIntSliceOr
	ToIntMapOr      = fw.ToIntMapOr
	MergeMap        = fw.MergeMap
	CloneMap        = fw.CloneMap
)
