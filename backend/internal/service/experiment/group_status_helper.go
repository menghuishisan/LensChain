// group_status_helper.go
// 模块04 — 实验环境：实验分组状态派生辅助
// 负责统一分组在“组建中 / 已就绪 / 实验中 / 已完成”之间的状态推导规则

package experiment

import (
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// deriveGroupMembershipStatus 根据成员数量推导分组的成员就绪状态。
func deriveGroupMembershipStatus(maxMembers, memberCount int) int16 {
	if maxMembers > 0 && memberCount >= maxMembers {
		return enum.GroupStatusReady
	}
	return enum.GroupStatusForming
}

// deriveGroupAggregateStatus 根据成员与实例最新状态推导分组聚合状态。
func deriveGroupAggregateStatus(group *entity.ExperimentGroup, members []*entity.GroupMember, latestInstances map[int64]*entity.ExperimentInstance) int16 {
	if group == nil {
		return enum.GroupStatusForming
	}

	memberStatus := deriveGroupMembershipStatus(group.MaxMembers, len(members))
	if memberStatus == enum.GroupStatusForming {
		return enum.GroupStatusForming
	}

	allCompleted := true
	allRunningOrCompleted := true
	hasRunning := false

	for _, member := range members {
		instance := latestInstances[member.StudentID]
		if instance == nil {
			return enum.GroupStatusReady
		}
		switch instance.Status {
		case enum.InstanceStatusCompleted:
			continue
		case enum.InstanceStatusRunning:
			hasRunning = true
			allCompleted = false
		default:
			allCompleted = false
			allRunningOrCompleted = false
		}
	}

	if allCompleted {
		return enum.GroupStatusCompleted
	}
	if allRunningOrCompleted && hasRunning {
		return enum.GroupStatusRunning
	}
	return enum.GroupStatusReady
}

// buildLatestInstanceByStudent 提取组内每个学生最新的一次实例。
func buildLatestInstanceByStudent(instances []*entity.ExperimentInstance) map[int64]*entity.ExperimentInstance {
	latest := make(map[int64]*entity.ExperimentInstance, len(instances))
	for _, instance := range instances {
		current := latest[instance.StudentID]
		if current == nil || instance.AttemptNo > current.AttemptNo || (instance.AttemptNo == current.AttemptNo && instance.CreatedAt.After(current.CreatedAt)) {
			latest[instance.StudentID] = instance
		}
	}
	return latest
}
