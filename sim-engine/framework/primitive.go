// 模块：sim-engine/framework
// 文件职责：47 原语的纯构造器（仅按 06.md §3.2 schema 填字段，不含业务逻辑）。
//
// 职责约束：
//   - 仅提供"按 schema 填 Primitive struct"的构造便利；
//   - 不组合多原语为业务级 widget（如"PBFT 投票矩阵"应在场景包内组装，不在此层）；
//   - 不引用任何场景代码。
//
// 命名约定（详 sim-engine/AGENTS.md §0.3 + §12.1.6）：
//   - 默认构造器（如 PrimNode / PrimRingLayout / PrimMatrixLayout / PrimGraphLayout）输出**无绝对坐标**版本，
//     由前端 PrimitiveBasedRenderer 按父布局原语推导位置，参与布局是 90%+ 场景的正确路径；
//   - *At 后缀构造器（如 PrimNodeAt / PrimRingLayoutAt）输出带逻辑坐标 0~1 版本，**仅用于绝对定位**
//     （如自由分布攻击节点群、对照展示、画布角落 HUD）。
//
// 与 06.md §3.2 v0.5 决议一致：所有坐标字段（x / y / center_x / center_y / radius / cell_w / cell_h）
// 在原语 schema 中的语义为：参与父布局时省略 / 绝对定位时填逻辑坐标 0~1。

package framework

// =====================================================================
// 几何类（8 个）
// =====================================================================

// PrimNode 构造默认（无绝对坐标）的 node 原语。
//
// 用法：场景输出 graph_layout / ring_layout 等布局原语 + 一组 PrimNode 节点；
// 前端 PrimitiveBasedRenderer 按布局算法推导每个节点位置。
// 这样场景层不承担"画布尺寸 / 中心点 / 半径 / 节点坐标"等前端布局职责。
//
// 90%+ 场景应使用本默认构造器。需要画布绝对定位时改用 PrimNodeAt。
func PrimNode(id, label, status, role string) Primitive {
	params := map[string]any{"id": id, "label": label, "status": status}
	if role != "" {
		params["role"] = role
	}
	return Primitive{ID: id, Type: PrimGeometryNode, Layer: LayerContent, Params: params}
}

// PrimNodeAt 构造带画布逻辑坐标（0~1）的 node 原语，仅用于不参与任何父布局的绝对定位场景
// （自由分布攻击节点群、对照展示等）。size 为 0 时省略。
func PrimNodeAt(id, label, status, role string, x, y, size float64) Primitive {
	params := map[string]any{"id": id, "label": label, "status": status, "x": x, "y": y}
	if role != "" {
		params["role"] = role
	}
	if size > 0 {
		params["size"] = size
	}
	return Primitive{ID: id, Type: PrimGeometryNode, Layer: LayerContent, Params: params}
}

// PrimEdge 构造 edge 原语；style/animation 为空时省略。edge 通过 from_id/to_id 引用节点，
// 自身不含坐标，与布局机制天然兼容。
func PrimEdge(id, fromID, toID, style, animation string) Primitive {
	params := map[string]any{"id": id, "from_id": fromID, "to_id": toID}
	if style != "" {
		params["style"] = style
	}
	if animation != "" {
		params["animation"] = animation
	}
	return Primitive{ID: id, Type: PrimGeometryEdge, Layer: LayerContent, Params: params}
}

// PrimBar 构造默认（无绝对坐标）的 bar 原语，用于由父布局原语（如 horizontal_lane）推导位置。
// height 为柱高（语义值），width 为可选柱宽；color_role 必填（语义槽位）。
func PrimBar(id string, height, width float64, colorRole, label string) Primitive {
	params := map[string]any{"id": id, "height": height, "color_role": colorRole}
	if width > 0 {
		params["width"] = width
	}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimGeometryBar, Layer: LayerContent, Params: params}
}

// PrimBarAt 构造带画布逻辑坐标（0~1）的 bar 原语，用于绝对定位场景。
func PrimBarAt(id string, x, y, height, width float64, colorRole, label string) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "height": height, "color_role": colorRole}
	if width > 0 {
		params["width"] = width
	}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimGeometryBar, Layer: LayerContent, Params: params}
}

// PrimPolygon 构造 polygon 原语；vertices 为 [{x,y}, ...]，作为整体形状的相对/绝对坐标列表
// （由场景或父布局决定）。原语 schema 本身不强制独立的 (x, y) 锚点。
func PrimPolygon(id string, vertices []map[string]float64, fill, stroke string) Primitive {
	params := map[string]any{"id": id, "vertices": vertices}
	if fill != "" {
		params["fill"] = fill
	}
	if stroke != "" {
		params["stroke"] = stroke
	}
	return Primitive{ID: id, Type: PrimGeometryPolygon, Layer: LayerContent, Params: params}
}

// PrimGridCell 构造 grid_cell 原语，用 row/col 索引嵌入父 matrix_layout / vote_matrix / heat_map，
// 不含坐标，前端按父布局推导位置。
func PrimGridCell(id string, row, col int, value any, colorRole string) Primitive {
	return Primitive{
		ID:    id,
		Type:  PrimGeometryGridCell,
		Layer: LayerContent,
		Params: map[string]any{
			"id": id, "row": row, "col": col, "value": value, "color_role": colorRole,
		},
	}
}

// PrimRing 构造默认（无绝对坐标）的 ring 原语（进度环 / Epoch 时间轮），由父布局推导位置。
// total / current 为进度数据（语义值），label 可选。
func PrimRing(id string, total, current int, label string) Primitive {
	params := map[string]any{"id": id, "total": total, "current": current}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimGeometryRing, Layer: LayerContent, Params: params}
}

// PrimRingAt 构造带画布逻辑坐标（0~1）的 ring 原语，常用于画布角落显示固定进度环或 HUD。
func PrimRingAt(id string, x, y, radius float64, total, current int, label string) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "radius": radius, "total": total, "current": current}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimGeometryRing, Layer: LayerContent, Params: params}
}

// PrimCurve 构造 curve 原语；points 为曲线坐标系上的点列表（如椭圆曲线、AMM 曲线），
// 由场景按算法语义提供（不是画布坐标）。
func PrimCurve(id, equation string, points []map[string]float64, style string) Primitive {
	params := map[string]any{"id": id, "points": points}
	if equation != "" {
		params["equation"] = equation
	}
	if style != "" {
		params["style"] = style
	}
	return Primitive{ID: id, Type: PrimGeometryCurve, Layer: LayerContent, Params: params}
}

// PrimArea 构造 area 原语；points 为面积图坐标点（语义坐标，由前端归一化到画布）。
func PrimArea(id string, points []map[string]float64, gradient string) Primitive {
	params := map[string]any{"id": id, "points": points}
	if gradient != "" {
		params["gradient"] = gradient
	}
	return Primitive{ID: id, Type: PrimGeometryArea, Layer: LayerContent, Params: params}
}

// =====================================================================
// 动效类（7 个）
// 动效原语全部通过 anchor_id 锚定到内容层原语，自身不含坐标。
// =====================================================================

// PrimParticleStream 构造持续粒子流（详 06.md §3.10.1）。
func PrimParticleStream(id, anchorID string, rate int, lifetimeMs int, colorRole, direction string) Primitive {
	params := map[string]any{"id": id, "anchor_id": anchorID, "color_role": colorRole}
	if rate > 0 {
		params["rate"] = rate
	}
	if lifetimeMs > 0 {
		params["lifetime_ms"] = lifetimeMs
	}
	if direction != "" {
		params["direction"] = direction
	}
	return Primitive{ID: id, Type: PrimEffectParticleStream, Layer: LayerEffect, Params: params}
}

// PrimBurst 构造一次性爆炸特效；fired_at_tick 控制幂等。
func PrimBurst(id, anchorID, color string, firedAtTick int64, durationMs int) Primitive {
	params := map[string]any{"id": id, "anchor_id": anchorID, "color": color, "fired_at_tick": firedAtTick}
	if durationMs > 0 {
		params["duration_ms"] = durationMs
	}
	return Primitive{ID: id, Type: PrimEffectBurst, Layer: LayerEffect, Params: params}
}

// PrimPulse 构造周期脉冲特效。
func PrimPulse(id, anchorID, colorRole string, periodMs int) Primitive {
	params := map[string]any{"id": id, "anchor_id": anchorID, "color_role": colorRole}
	if periodMs > 0 {
		params["period_ms"] = periodMs
	}
	return Primitive{ID: id, Type: PrimEffectPulse, Layer: LayerEffect, Params: params}
}

// PrimTrail 构造路径轨迹原语。points 为轨迹点序列（语义坐标，前端归一化到画布）。
func PrimTrail(id, anchorID string, points []map[string]float64, style string, durationMs, fadeMs int) Primitive {
	params := map[string]any{"id": id, "anchor_id": anchorID, "points": points, "style": style}
	if durationMs > 0 {
		params["duration_ms"] = durationMs
	}
	if fadeMs > 0 {
		params["fade_ms"] = fadeMs
	}
	return Primitive{ID: id, Type: PrimEffectTrail, Layer: LayerEffect, Params: params}
}

// PrimGlow 构造持续高亮原语。
func PrimGlow(id, anchorID, colorRole string, intensity float64) Primitive {
	return Primitive{
		ID: id, Type: PrimEffectGlow, Layer: LayerEffect,
		Params: map[string]any{"id": id, "anchor_id": anchorID, "color_role": colorRole, "intensity": intensity},
	}
}

// PrimShake 构造抖动特效。
func PrimShake(id, anchorID string, magnitude float64, durationMs int) Primitive {
	return Primitive{
		ID: id, Type: PrimEffectShake, Layer: LayerEffect,
		Params: map[string]any{"id": id, "anchor_id": anchorID, "magnitude": magnitude, "duration_ms": durationMs},
	}
}

// PrimShiftAnimation 构造平滑位移特效。
func PrimShiftAnimation(id, targetID, direction string, distance float64, durationMs int) Primitive {
	return Primitive{
		ID: id, Type: PrimEffectShiftAnimation, Layer: LayerEffect,
		Params: map[string]any{"id": id, "target_id": targetID, "direction": direction, "distance": distance, "duration_ms": durationMs},
	}
}

// =====================================================================
// 布局类（6 个）
// 默认构造器输出"教学语义"版本（仅给布局结构信息，不给坐标 / 尺寸）；
// *At 后缀构造器输出"绝对定位 + 尺寸建议"版本，仅在画布角落或对照场景使用。
// =====================================================================

// PrimHorizontalLane 构造默认（无绝对坐标）的水平泳道，仅声明该泳道的语义存在。
// 前端按画布尺寸与同画布泳道数响应式分配 Y 位置。
func PrimHorizontalLane(id, label string) Primitive {
	params := map[string]any{"id": id}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimLayoutHorizontalLane, Layer: LayerBackground, Params: params}
}

// PrimHorizontalLaneAt 构造带画布逻辑 Y 坐标（0~1）的水平泳道，用于场景需要在指定纵向位置
// 锁定泳道的少数情况。
func PrimHorizontalLaneAt(id string, y float64, label string) Primitive {
	params := map[string]any{"id": id, "y": y}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimLayoutHorizontalLane, Layer: LayerBackground, Params: params}
}

// PrimStack 构造默认（无绝对坐标）的栈布局；direction = "horizontal" | "vertical"。
// items 为子原语 ID 列表，前端按方向 + 画布响应式排列。
func PrimStack(id string, items []string, direction string) Primitive {
	return Primitive{
		ID: id, Type: PrimLayoutStack, Layer: LayerBackground,
		Params: map[string]any{"id": id, "items": items, "direction": direction},
	}
}

// PrimStackAt 构造带画布逻辑 X 坐标（0~1）的栈布局，用于绝对定位。
func PrimStackAt(id string, x float64, items []string, direction string) Primitive {
	return Primitive{
		ID: id, Type: PrimLayoutStack, Layer: LayerBackground,
		Params: map[string]any{"id": id, "x": x, "items": items, "direction": direction},
	}
}

// PrimRingLayout 构造默认（无绝对坐标 / 半径）的环形节点布局。
//
// 协议依据：06.md §3.2.3。`nodes` 显式声明环上成员 ID 列表，渲染器按列表顺序从 12 点
// 钟方向顺时针均分 N 个 slot；语义与同级 PrimGraphLayout 的 nodes[] 完全一致。
// slots 数量由 len(nodes) 推导，不再单独承载 slots 字段（避免 slots 与实际 node 数漂移）。
//
// 用法：场景先按业务顺序构造节点 ID 列表，把列表传入 PrimRingLayout，再为每个 ID
// 输出对应的 PrimNode（坐标省略），渲染器自动把 PrimNode 落到 ring 的对应 slot。
func PrimRingLayout(id string, nodes []string) Primitive {
	// 复制切片，避免调用方后续修改污染已发出的 envelope。
	cp := append([]string(nil), nodes...)
	return Primitive{
		ID: id, Type: PrimLayoutRing, Layer: LayerBackground,
		Params: map[string]any{"id": id, "nodes": cp},
	}
}

// PrimRingLayoutAt 构造带画布逻辑坐标的环形布局（中心点 + 半径），用于场景需要在画布
// 特定位置（如对照展示左/右半侧）锁定环形布局的少数情况。
//
// nodes 语义同 PrimRingLayout。
func PrimRingLayoutAt(id string, nodes []string, centerX, centerY, radius float64) Primitive {
	cp := append([]string(nil), nodes...)
	return Primitive{
		ID: id, Type: PrimLayoutRing, Layer: LayerBackground,
		Params: map[string]any{"id": id, "center_x": centerX, "center_y": centerY, "radius": radius, "nodes": cp},
	}
}

// PrimTreeLayout 构造树布局；layoutAlgorithm = "top-down" | "bottom-up"。
// 树布局自身不含坐标，前端按算法 + 画布尺寸推导每个节点位置。
func PrimTreeLayout(id, rootID, layoutAlgorithm string) Primitive {
	return Primitive{
		ID: id, Type: PrimLayoutTree, Layer: LayerBackground,
		Params: map[string]any{"id": id, "root_id": rootID, "layout_algorithm": layoutAlgorithm},
	}
}

// PrimGraphLayout 构造图布局；algorithm = "force" | "circular" | "grid"。
// nodes / edges 为参与本布局的节点/边 ID 列表，前端按 algorithm 推导坐标。
func PrimGraphLayout(id, algorithm string, nodes []string, edges []string) Primitive {
	return Primitive{
		ID: id, Type: PrimLayoutGraph, Layer: LayerBackground,
		Params: map[string]any{"id": id, "algorithm": algorithm, "nodes": nodes, "edges": edges},
	}
}

// PrimMatrixLayout 构造默认（无单元格尺寸）的矩阵布局。仅声明 rows / cols
// （行列数，教学决策），前端按画布尺寸 + 类目皮肤密度响应式计算 cell_w / cell_h。
func PrimMatrixLayout(id string, rows, cols int) Primitive {
	return Primitive{
		ID: id, Type: PrimLayoutMatrix, Layer: LayerBackground,
		Params: map[string]any{"id": id, "rows": rows, "cols": cols},
	}
}

// PrimMatrixLayoutAt 构造带单元格尺寸的矩阵布局，用于场景需要锁定矩阵单元尺寸的少数对照情形。
func PrimMatrixLayoutAt(id string, rows, cols int, cellW, cellH float64) Primitive {
	return Primitive{
		ID: id, Type: PrimLayoutMatrix, Layer: LayerBackground,
		Params: map[string]any{"id": id, "rows": rows, "cols": cols, "cell_w": cellW, "cell_h": cellH},
	}
}

// =====================================================================
// 数据展示类（7 个）
// =====================================================================

// PrimLabel 构造文本标签；anchorID 与 (x,y) 二选一，建议 anchorID 优先。
// anchorID 非空时锚定到对应原语；为空时用 PrimLabelAt 给绝对坐标。
func PrimLabel(id, anchorID, text, style string) Primitive {
	params := map[string]any{"id": id, "text": text, "anchor_id": anchorID}
	if style != "" {
		params["style"] = style
	}
	return Primitive{ID: id, Type: PrimDataLabel, Layer: LayerContent, Params: params}
}

// PrimLabelAt 构造带画布逻辑坐标（0~1）的文本标签，用于无锚点的独立标注。
func PrimLabelAt(id string, x, y float64, text, style string) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "text": text}
	if style != "" {
		params["style"] = style
	}
	return Primitive{ID: id, Type: PrimDataLabel, Layer: LayerContent, Params: params}
}

// PrimTooltip 构造悬停 tooltip 原语；trigger = "hover" | "click"。
// tooltip 通过 anchor_id 锚定，自身不含坐标。
func PrimTooltip(id, anchorID string, content []map[string]string, trigger string) Primitive {
	params := map[string]any{"id": id, "anchor_id": anchorID, "content": content}
	if trigger != "" {
		params["trigger"] = trigger
	}
	return Primitive{ID: id, Type: PrimDataTooltip, Layer: LayerContent, Params: params}
}

// PrimAnnotation 构造教师课堂标注原语（详 06.md §11.3）。geometry 由教师标注工具
// 直接采集（含画布坐标），属合理的绝对定位用例。
func PrimAnnotation(id, shape, ownerRole string, expiresMs int, geometry, style map[string]any, text string) Primitive {
	params := map[string]any{"id": id, "shape": shape, "owner_role": ownerRole, "geometry": geometry}
	if expiresMs > 0 {
		params["expires_ms"] = expiresMs
	}
	if len(style) > 0 {
		params["style"] = style
	}
	if text != "" {
		params["text"] = text
	}
	return Primitive{ID: id, Type: PrimDataAnnotation, Layer: LayerOverlay, Params: params}
}

// PrimRegisterRow 构造默认（无绝对坐标）的寄存器组（如 SHA-256 a~h、EVM 栈），
// 由父布局或类目皮肤决定位置。highlight_index 为高亮的寄存器索引（-1 表示无高亮）。
func PrimRegisterRow(id string, labels []string, values []string, highlightIndex int) Primitive {
	params := map[string]any{"id": id, "labels": labels, "values": values}
	if highlightIndex >= 0 {
		params["highlight_index"] = highlightIndex
	}
	return Primitive{ID: id, Type: PrimDataRegisterRow, Layer: LayerContent, Params: params}
}

// PrimRegisterRowAt 构造带画布逻辑坐标的寄存器组，用于绝对定位。
func PrimRegisterRowAt(id string, x, y float64, labels []string, values []string, highlightIndex int) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "labels": labels, "values": values}
	if highlightIndex >= 0 {
		params["highlight_index"] = highlightIndex
	}
	return Primitive{ID: id, Type: PrimDataRegisterRow, Layer: LayerContent, Params: params}
}

// PrimMathPipeline 构造默认（无绝对坐标）的算式管道；steps 为 [{op, result, animate?, highlight?}]。
func PrimMathPipeline(id string, steps []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDataMathPipeline, Layer: LayerContent,
		Params: map[string]any{"id": id, "steps": steps},
	}
}

// PrimMathPipelineAt 构造带画布逻辑坐标的算式管道，用于绝对定位。
func PrimMathPipelineAt(id string, x, y float64, steps []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDataMathPipeline, Layer: LayerContent,
		Params: map[string]any{"id": id, "x": x, "y": y, "steps": steps},
	}
}

// PrimCodeBlock 构造默认（无绝对坐标）的代码块；language / highlight_lines / max_lines 可选。
func PrimCodeBlock(id, content, language string, highlightLines []int, maxLines int) Primitive {
	params := map[string]any{"id": id, "content": content}
	if language != "" {
		params["language"] = language
	}
	if len(highlightLines) > 0 {
		params["highlight_lines"] = highlightLines
	}
	if maxLines > 0 {
		params["max_lines"] = maxLines
	}
	return Primitive{ID: id, Type: PrimDataCodeBlock, Layer: LayerContent, Params: params}
}

// PrimCodeBlockAt 构造带画布逻辑坐标的代码块，用于绝对定位。
func PrimCodeBlockAt(id string, x, y float64, content, language string, highlightLines []int, maxLines int) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "content": content}
	if language != "" {
		params["language"] = language
	}
	if len(highlightLines) > 0 {
		params["highlight_lines"] = highlightLines
	}
	if maxLines > 0 {
		params["max_lines"] = maxLines
	}
	return Primitive{ID: id, Type: PrimDataCodeBlock, Layer: LayerContent, Params: params}
}

// PrimMathFormula 构造默认（无绝对坐标）的 LaTeX 公式（前端 KaTeX 渲染）。
func PrimMathFormula(id, latex string, inline bool) Primitive {
	return Primitive{
		ID: id, Type: PrimDataMathFormula, Layer: LayerContent,
		Params: map[string]any{"id": id, "latex": latex, "inline": inline},
	}
}

// PrimMathFormulaAt 构造带画布逻辑坐标的 LaTeX 公式，用于绝对定位。
func PrimMathFormulaAt(id string, x, y float64, latex string, inline bool) Primitive {
	return Primitive{
		ID: id, Type: PrimDataMathFormula, Layer: LayerContent,
		Params: map[string]any{"id": id, "x": x, "y": y, "latex": latex, "inline": inline},
	}
}

// =====================================================================
// 状态指示类（8 个）
// =====================================================================

// PrimPhaseProgress 构造阶段进度条；phases / current_index / progress（0-1）。
// 自身不含坐标，由前端 HUD 区域统一布局。
func PrimPhaseProgress(id string, phases []string, currentIndex int, progress float64) Primitive {
	return Primitive{
		ID: id, Type: PrimStatePhaseProgress, Layer: LayerOverlay,
		Params: map[string]any{"id": id, "phases": phases, "current_index": currentIndex, "progress": progress},
	}
}

// PrimProgressBar 构造默认（无绝对坐标）的普通进度条。
func PrimProgressBar(id string, value, max float64, label string) Primitive {
	params := map[string]any{"id": id, "value": value, "max": max}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimStateProgressBar, Layer: LayerOverlay, Params: params}
}

// PrimProgressBarAt 构造带画布逻辑坐标的进度条，用于绝对定位。
func PrimProgressBarAt(id string, x, y, value, max float64, label string) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "value": value, "max": max}
	if label != "" {
		params["label"] = label
	}
	return Primitive{ID: id, Type: PrimStateProgressBar, Layer: LayerOverlay, Params: params}
}

// PrimTargetZone 构造目标线（如 PoW 难度阈值、共识 2f+1）；axis = "x" | "y"。
// 通过逻辑值 + 轴向声明，由前端坐标系映射到画布。
func PrimTargetZone(id string, value float64, label, axis string) Primitive {
	params := map[string]any{"id": id, "value": value, "label": label}
	if axis != "" {
		params["axis"] = axis
	}
	return Primitive{ID: id, Type: PrimStateTargetZone, Layer: LayerOverlay, Params: params}
}

// PrimLinkIndicator 构造默认（无绝对坐标）的画布角落联动徽章（M2）。
// 前端 HUD 区域统一布局多个徽章；status: "idle" | "active" | "recent"。
func PrimLinkIndicator(id, linkGroup, status, lastEvent string) Primitive {
	params := map[string]any{"id": id, "link_group": linkGroup, "status": status}
	if lastEvent != "" {
		params["last_event"] = lastEvent
	}
	return Primitive{ID: id, Type: PrimStateLinkIndicator, Layer: LayerOverlay, Params: params}
}

// PrimLinkIndicatorAt 构造带画布逻辑坐标的联动徽章，用于场景需要在指定位置
// 锁定徽章的少数情况（M8 多场景对照）。
func PrimLinkIndicatorAt(id string, x, y float64, linkGroup, status, lastEvent string) Primitive {
	params := map[string]any{"id": id, "x": x, "y": y, "link_group": linkGroup, "status": status}
	if lastEvent != "" {
		params["last_event"] = lastEvent
	}
	return Primitive{ID: id, Type: PrimStateLinkIndicator, Layer: LayerOverlay, Params: params}
}

// PrimExternalEventMarker 构造临时浮窗（联动外部事件），通过 anchor_id 锚定。
func PrimExternalEventMarker(id, anchorID, sourceScene, label string, fadeMs int) Primitive {
	return Primitive{
		ID: id, Type: PrimStateExternalEventMarker, Layer: LayerOverlay,
		Params: map[string]any{"id": id, "anchor_id": anchorID, "source_scene": sourceScene, "label": label, "fade_ms": fadeMs},
	}
}

// PrimErrorOverlay 构造容器级故障指示（不自动 fade）。
// severity: "warning" | "error" | "fatal"; source: "container" | "collector" | "core" | "scene"。
func PrimErrorOverlay(id, severity, title, message, source, actionHint string, dismissible bool) Primitive {
	params := map[string]any{
		"id": id, "severity": severity, "title": title, "message": message,
		"source": source, "dismissible": dismissible,
	}
	if actionHint != "" {
		params["action_hint"] = actionHint
	}
	return Primitive{ID: id, Type: PrimStateErrorOverlay, Layer: LayerOverlay, Params: params}
}

// PrimVerifyPathHighlight 构造验证路径高亮（Merkle / MPT），通过 node_ids 引用。
func PrimVerifyPathHighlight(id string, nodeIDs []string) Primitive {
	return Primitive{
		ID: id, Type: PrimStateVerifyPathHighlight, Layer: LayerOverlay,
		Params: map[string]any{"id": id, "node_ids": nodeIDs},
	}
}

// PrimRiskGauge 构造风险仪表；ranges 为 [{from, to, color}]。
func PrimRiskGauge(id string, value float64, ranges []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimStateRiskGauge, Layer: LayerOverlay,
		Params: map[string]any{"id": id, "value": value, "ranges": ranges},
	}
}

// =====================================================================
// 领域复合类（11 个）
//
// 这些原语本身就是协议中的"原语糖"，前端基类按底层组合渲染。
// 默认构造器输出无坐标 / 无尺寸版本；*At 后缀构造器输出绝对定位 / 含尺寸版本。
// =====================================================================

// PrimVoteMatrix 构造默认（无单元格尺寸）的投票矩阵（PBFT/Raft）。
// cells 形如 [{row, col, value, color_role}]；前端按画布响应式决定 cell_w / cell_h。
func PrimVoteMatrix(id string, rows, cols int, cells []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainVoteMatrix, Layer: LayerContent,
		Params: map[string]any{"id": id, "rows": rows, "cols": cols, "cells": cells},
	}
}

// PrimVoteMatrixAt 构造带单元格尺寸的投票矩阵，用于对照场景需要锁定单元尺寸的情形。
func PrimVoteMatrixAt(id string, rows, cols int, cellW, cellH float64, cells []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainVoteMatrix, Layer: LayerContent,
		Params: map[string]any{"id": id, "rows": rows, "cols": cols, "cell_w": cellW, "cell_h": cellH, "cells": cells},
	}
}

// PrimDualTrack 构造双轨链对比；tracks 形如 [{lane: "honest"|"attack", blocks: [{id, label}], ...}]。
func PrimDualTrack(id string, tracks []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainDualTrack, Layer: LayerContent,
		Params: map[string]any{"id": id, "tracks": tracks},
	}
}

// PrimTimeWheel 构造默认（无绝对坐标）的 Epoch 时间轮，由父布局或类目皮肤决定位置。
func PrimTimeWheel(id string, slots, currentSlot int) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainTimeWheel, Layer: LayerContent,
		Params: map[string]any{"id": id, "slots": slots, "current_slot": currentSlot},
	}
}

// PrimTimeWheelAt 构造带画布逻辑坐标的时间轮，用于绝对定位（对照展示）。
func PrimTimeWheelAt(id string, slots, currentSlot int, centerX, centerY, radius float64) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainTimeWheel, Layer: LayerContent,
		Params: map[string]any{"id": id, "center_x": centerX, "center_y": centerY, "radius": radius, "slots": slots, "current_slot": currentSlot},
	}
}

// PrimPieChart 构造默认（无绝对坐标）的饼图；segments 形如 [{label, value, color_role}]。
func PrimPieChart(id string, segments []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainPieChart, Layer: LayerContent,
		Params: map[string]any{"id": id, "segments": segments},
	}
}

// PrimPieChartAt 构造带画布逻辑坐标的饼图，用于绝对定位。
func PrimPieChartAt(id string, segments []map[string]any, centerX, centerY, radius float64) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainPieChart, Layer: LayerContent,
		Params: map[string]any{"id": id, "center_x": centerX, "center_y": centerY, "radius": radius, "segments": segments},
	}
}

// PrimSankeyFlow 构造桑基图；flows 形如 [{from, to, value, label?}]。
func PrimSankeyFlow(id string, flows []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainSankeyFlow, Layer: LayerContent,
		Params: map[string]any{"id": id, "flows": flows},
	}
}

// PrimHeatMap 构造默认（无单元格尺寸）的热力图；cells 形如 [{row, col, value}]。
func PrimHeatMap(id string, rows, cols int, cells []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainHeatMap, Layer: LayerContent,
		Params: map[string]any{"id": id, "rows": rows, "cols": cols, "cells": cells},
	}
}

// PrimHeatMapAt 构造带单元格尺寸的热力图。
func PrimHeatMapAt(id string, rows, cols int, cellW, cellH float64, cells []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainHeatMap, Layer: LayerContent,
		Params: map[string]any{"id": id, "rows": rows, "cols": cols, "cell_w": cellW, "cell_h": cellH, "cells": cells},
	}
}

// PrimMempoolSlot 构造内存池槽位，通过 row/col 索引嵌入父矩阵布局。
func PrimMempoolSlot(id string, row, col int, txID, label string) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainMempoolSlot, Layer: LayerContent,
		Params: map[string]any{"id": id, "row": row, "col": col, "tx_id": txID, "label": label},
	}
}

// PrimBridgeTrack 构造跨链桥双链；leftChain / rightChain 内部为链上区块/事件项。
func PrimBridgeTrack(id string, leftChain, rightChain []map[string]any) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainBridgeTrack, Layer: LayerContent,
		Params: map[string]any{"id": id, "left_chain": leftChain, "right_chain": rightChain},
	}
}

// PrimCodeMarker 构造操作码 / PC 指针标记（EVM），通过 code_block_id + line_number 引用。
func PrimCodeMarker(id, codeBlockID string, lineNumber int, label string) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainCodeMarker, Layer: LayerContent,
		Params: map[string]any{"id": id, "code_block_id": codeBlockID, "line_number": lineNumber, "label": label},
	}
}

// PrimPartitionZone 构造网络分区区域；vertices 为分区边界（语义坐标，前端归一化到画布）。
func PrimPartitionZone(id string, vertices []map[string]float64, label string) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainPartitionZone, Layer: LayerContent,
		Params: map[string]any{"id": id, "vertices": vertices, "label": label},
	}
}

// PrimCurvePoint 构造曲线上的点（ECDSA / RSA）；x / y 是曲线坐标系上的点（不是画布坐标），
// 由场景按算法语义提供。
func PrimCurvePoint(id, curveID string, x, y float64, label string) Primitive {
	return Primitive{
		ID: id, Type: PrimDomainCurvePoint, Layer: LayerContent,
		Params: map[string]any{"id": id, "curve_id": curveID, "x": x, "y": y, "label": label},
	}
}
