package simcore

import "fmt"

// TimeControlMode 定义仿真时钟控制模式。
type TimeControlMode string

const (
	// TimeControlModeProcess 表示支持播放、单步、回退的过程化模式。
	TimeControlModeProcess TimeControlMode = "process"
	// TimeControlModeReactive 表示输入即响应的交互响应式模式。
	TimeControlModeReactive TimeControlMode = "reactive"
	// TimeControlModeContinuous 表示持续演化观察的连续运行模式。
	TimeControlModeContinuous TimeControlMode = "continuous"
)

// Clock 管理单个仿真会话的 tick、速度和播放状态。
type Clock struct {
	mode    TimeControlMode
	tick    int64
	speed   float64
	running bool
}

// NewClock 创建仿真时钟。
func NewClock(mode TimeControlMode) *Clock {
	return &Clock{
		mode:  mode,
		speed: 1,
	}
}

// Tick 返回当前 tick。
func (c *Clock) Tick() int64 {
	return c.tick
}

// Speed 返回当前仿真速度。
func (c *Clock) Speed() float64 {
	return c.speed
}

// Mode 返回当前时钟的时间控制模式。
func (c *Clock) Mode() TimeControlMode {
	return c.mode
}

// IsRunning 返回当前时钟是否处于自动推进状态。
func (c *Clock) IsRunning() bool {
	return c.running
}

// Play 启动过程化时钟播放。
func (c *Clock) Play() error {
	if c.mode != TimeControlModeProcess {
		return unsupportedControl(c.mode, "play")
	}
	c.running = true
	return nil
}

// Resume 恢复持续运行式时钟的自动推进。
func (c *Clock) Resume() error {
	if c.mode != TimeControlModeContinuous {
		return unsupportedControl(c.mode, "resume")
	}
	c.running = true
	return nil
}

// Pause 暂停过程化或持续运行式时钟。
func (c *Clock) Pause() error {
	if c.mode != TimeControlModeProcess && c.mode != TimeControlModeContinuous {
		return unsupportedControl(c.mode, "pause")
	}
	c.running = false
	return nil
}

// Step 对过程化场景执行单步推进。
func (c *Clock) Step() error {
	if c.mode != TimeControlModeProcess {
		return unsupportedControl(c.mode, "step")
	}
	c.tick++
	return nil
}

// Advance 在自动推进状态下前进一步。
func (c *Clock) Advance() error {
	if !c.running {
		return nil
	}
	if c.mode != TimeControlModeProcess && c.mode != TimeControlModeContinuous {
		return unsupportedControl(c.mode, "advance")
	}
	c.tick++
	return nil
}

// SetSpeed 设置仿真速度，只允许文档规定的四档。
func (c *Clock) SetSpeed(speed float64) error {
	if speed != 0.5 && speed != 1 && speed != 1.5 && speed != 2 {
		return fmt.Errorf("unsupported sim speed: %v", speed)
	}
	c.speed = speed
	return nil
}

// Reset 将过程化时钟重置到初始 tick。
func (c *Clock) Reset() {
	c.tick = 0
	c.running = false
}

// StepBack 单步回退 1 个 tick（对齐 06.md §7.4 step_back：仅单场景 process 模式有效）。
// 上层（如 LinkGroup / 多场景对照 / 混合实验）的可用性限制在会话层校验；
// 本方法只负责 process 模式本身的可用性与 tick 边界校验。
func (c *Clock) StepBack() error {
	if c.mode != TimeControlModeProcess {
		return unsupportedControl(c.mode, "step_back")
	}
	if c.tick <= 0 {
		return fmt.Errorf("step_back 越过起始 tick")
	}
	c.tick--
	c.running = false
	return nil
}

// unsupportedControl 构造当前时间模式不支持该命令的错误。
func unsupportedControl(mode TimeControlMode, command string) error {
	return fmt.Errorf("time control %q is unsupported for mode %q", command, mode)
}
