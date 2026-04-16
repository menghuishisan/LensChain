package simcore

import "sort"

// Event 表示写入事件总线的标准过程事件。
type Event struct {
	EventID     string
	EventType   string
	SceneCode   string
	Tick        int64
	TimestampMS int64
	PayloadJSON []byte
}

// EventBus 保存按 tick 排序的仿真事件流。
type EventBus struct {
	events []Event
}

// NewEventBus 创建事件总线。
func NewEventBus() *EventBus {
	return &EventBus{}
}

// Append 写入事件并按 tick、时间戳稳定排序。
func (b *EventBus) Append(events []Event) {
	for _, event := range events {
		b.events = append(b.events, Event{
			EventID:     event.EventID,
			EventType:   event.EventType,
			SceneCode:   event.SceneCode,
			Tick:        event.Tick,
			TimestampMS: event.TimestampMS,
			PayloadJSON: cloneSnapshotBytes(event.PayloadJSON),
		})
	}
	sort.SliceStable(b.events, func(i int, j int) bool {
		if b.events[i].Tick == b.events[j].Tick {
			return b.events[i].TimestampMS < b.events[j].TimestampMS
		}
		return b.events[i].Tick < b.events[j].Tick
	})
}

// Range 返回指定 tick 区间内的事件副本。
func (b *EventBus) Range(fromTick int64, toTick int64) []Event {
	result := make([]Event, 0)
	for _, event := range b.events {
		if event.Tick >= fromTick && event.Tick <= toTick {
			result = append(result, Event{
				EventID:     event.EventID,
				EventType:   event.EventType,
				SceneCode:   event.SceneCode,
				Tick:        event.Tick,
				TimestampMS: event.TimestampMS,
				PayloadJSON: cloneSnapshotBytes(event.PayloadJSON),
			})
		}
	}
	return result
}
