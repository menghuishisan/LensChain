package framework

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"
)

// NewEvent 创建一个带统一标识的时间线事件。
func NewEvent(sceneCode string, tick int64, title string, description string, tone string) TimelineEvent {
	return TimelineEvent{
		ID:          fmt.Sprintf("%s-%d-%d", sceneCode, tick, time.Now().UTC().UnixNano()),
		Tick:        tick,
		Title:       title,
		Description: description,
		Tone:        tone,
	}
}

// MetricValue 将数值转成适合前端展示的文本。
func MetricValue(value float64, suffix string) string {
	return strconv.FormatFloat(value, 'f', 1, 64) + suffix
}

// Clamp 将数值限制在指定范围。
func Clamp(value float64, min float64, max float64) float64 {
	return math.Min(max, math.Max(min, value))
}

// NextProgress 根据当前 tick 和总 tick 计算标准化进度。
func NextProgress(tick int64, totalTicks int64) float64 {
	if totalTicks <= 0 {
		return 0
	}
	return Clamp(float64(tick%totalTicks)/float64(totalTicks), 0, 1)
}

// DeterministicRand 创建基于种子的确定性随机源。
func DeterministicRand(seed int64) *rand.Rand {
	if seed == 0 {
		seed = time.Now().UTC().UnixNano()
	}
	return rand.New(rand.NewSource(seed))
}
