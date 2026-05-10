// 模块：sim-engine/framework
// 文件职责：纯几何布局工具，**仅服务于少数绝对定位场景**。
//
// 使用边界（详 sim-engine/AGENTS.md §0.3 + §3.7）：
//   - 90%+ 场景的节点应装入 ring_layout / graph_layout / matrix_layout / tree_layout / stack /
//     horizontal_lane 等布局原语，由前端 PrimitiveBasedRenderer 按布局算法 + 画布响应式尺寸
//     推导坐标——**这种情况不应使用本工具**；
//   - 仅在场景必须给出绝对画布坐标的少数情形（自由分布攻击节点群、对照展示、画布角落 HUD）
//     使用本工具，并通过 *At 后缀构造器（PrimNodeAt / PrimRingLayoutAt 等）下发坐标；
//   - 教师 L3 自定义场景应优先使用默认（无坐标）构造器，避免与 8 类目皮肤的响应式冲突。
//
// 职责约束：
//   - 仅做坐标计算（圆周分布、网格、水平/垂直排列）；
//   - 不构造 Primitive、不引用业务概念；
//   - 不持有任何状态。

package framework

import "math"

// Point 表示一个二维点。
type Point struct {
	X float64
	Y float64
}

// CircleLayout 在以 (centerX, centerY) 为中心、半径 radius 的圆周上均匀分布 count 个点。
//
// 起始角 startAngleRad 表示第一个点相对于水平正方向（X+ 轴）的弧度（顺时针为正）。
// 返回的切片长度恰为 count；count <= 0 时返回 nil。
func CircleLayout(centerX, centerY, radius float64, count int, startAngleRad float64) []Point {
	if count <= 0 {
		return nil
	}
	points := make([]Point, count)
	step := 2 * math.Pi / float64(count)
	for i := 0; i < count; i++ {
		angle := startAngleRad + step*float64(i)
		points[i] = Point{
			X: centerX + radius*math.Cos(angle),
			Y: centerY + radius*math.Sin(angle),
		}
	}
	return points
}

// LinearLayout 在水平直线 y=anchorY 上从 startX 开始按 spacing 间隔排布 count 个点。
func LinearLayout(startX, anchorY, spacing float64, count int) []Point {
	if count <= 0 {
		return nil
	}
	points := make([]Point, count)
	for i := 0; i < count; i++ {
		points[i] = Point{X: startX + spacing*float64(i), Y: anchorY}
	}
	return points
}

// VerticalLayout 在垂直直线 x=anchorX 上从 startY 开始按 spacing 间隔排布 count 个点。
func VerticalLayout(anchorX, startY, spacing float64, count int) []Point {
	if count <= 0 {
		return nil
	}
	points := make([]Point, count)
	for i := 0; i < count; i++ {
		points[i] = Point{X: anchorX, Y: startY + spacing*float64(i)}
	}
	return points
}

// GridLayout 按行优先排列 rows*cols 个点。
//
// originX / originY 是左上角第一个格子的中心点；cellW / cellH 是格子尺寸。
// 返回长度恰为 rows*cols。
func GridLayout(originX, originY, cellW, cellH float64, rows, cols int) []Point {
	if rows <= 0 || cols <= 0 {
		return nil
	}
	points := make([]Point, 0, rows*cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			points = append(points, Point{
				X: originX + float64(c)*cellW,
				Y: originY + float64(r)*cellH,
			})
		}
	}
	return points
}

// MidPoint 计算两点中点。
func MidPoint(a, b Point) Point {
	return Point{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
}

// Distance 计算两点欧氏距离。
func Distance(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// PointsToParamSlice 将 Point 数组转为可放入 Primitive params 的 map 切片。
//
// 用于 trail / curve / area / polygon 等接受 [{x,y}, ...] 字段的原语。
func PointsToParamSlice(points []Point) []map[string]float64 {
	if len(points) == 0 {
		return nil
	}
	out := make([]map[string]float64, len(points))
	for i, p := range points {
		out[i] = map[string]float64{"x": p.X, "y": p.Y}
	}
	return out
}
