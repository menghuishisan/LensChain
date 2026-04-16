package simcore

// Snapshot 表示某个 tick 的状态快照。
type Snapshot struct {
	Tick      int64
	StateJSON []byte
	DiffJSON  []byte
	Keyframe  bool
}

// SnapshotStack 保存最近一段 tick 的关键帧和增量快照。
type SnapshotStack struct {
	maxTicks         int
	keyframeInterval int64
	items            []Snapshot
}

// NewSnapshotStack 创建快照栈。
func NewSnapshotStack(maxTicks int, keyframeInterval int64) *SnapshotStack {
	if maxTicks <= 0 {
		maxTicks = 1000
	}
	if keyframeInterval <= 0 {
		keyframeInterval = 50
	}
	return &SnapshotStack{
		maxTicks:         maxTicks,
		keyframeInterval: keyframeInterval,
	}
}

// Save 保存指定 tick 的完整状态或增量状态。
func (s *SnapshotStack) Save(tick int64, stateJSON []byte, diffJSON []byte) {
	snapshot := Snapshot{
		Tick:      tick,
		StateJSON: cloneSnapshotBytes(stateJSON),
		DiffJSON:  cloneSnapshotBytes(diffJSON),
		Keyframe:  tick%s.keyframeInterval == 0 && stateJSON != nil,
	}
	s.items = append(s.items, snapshot)
	if len(s.items) > s.maxTicks {
		s.items = s.items[len(s.items)-s.maxTicks:]
	}
}

// NearestKeyframe 返回不晚于目标 tick 的最近关键帧。
func (s *SnapshotStack) NearestKeyframe(targetTick int64) (Snapshot, bool) {
	for i := len(s.items) - 1; i >= 0; i-- {
		item := s.items[i]
		if item.Keyframe && item.Tick <= targetTick {
			return cloneSnapshot(item), true
		}
	}
	return Snapshot{}, false
}

// DiffsAfter 返回 fromTick 之后、toTick 之前或等于 toTick 的增量快照。
func (s *SnapshotStack) DiffsAfter(fromTick int64, toTick int64) []Snapshot {
	result := make([]Snapshot, 0)
	for _, item := range s.items {
		if item.Tick > fromTick && item.Tick <= toTick && item.DiffJSON != nil {
			result = append(result, cloneSnapshot(item))
		}
	}
	return result
}

// cloneSnapshot 深复制快照内容。
func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.StateJSON = cloneSnapshotBytes(snapshot.StateJSON)
	snapshot.DiffJSON = cloneSnapshotBytes(snapshot.DiffJSON)
	return snapshot
}

// cloneSnapshotBytes 复制快照字节切片。
func cloneSnapshotBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}
