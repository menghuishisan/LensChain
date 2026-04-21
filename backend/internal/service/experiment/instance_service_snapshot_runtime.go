// instance_service_snapshot_runtime.go
// 模块04 — 实验环境：实例快照运行态恢复辅助逻辑
// 负责真实/混合实验容器目录状态的采集、剥离与恢复

package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// runtimeContainerState 表示实例快照中的单个容器运行态。
// 当前保留容器基础元数据与卷归档信息，用于暂停后恢复实验容器状态。
type runtimeContainerState = RuntimeContainerState

// runtimeVolumeArchive 表示实例快照中的单个卷归档内容。
type runtimeVolumeArchive = RuntimeVolumeArchive

// stripRuntimeArchiveData 去除容器快照中的归档内容，避免将大体积二进制冗余写入数据库。
func stripRuntimeArchiveData(states []runtimeContainerState) []runtimeContainerState {
	if len(states) == 0 {
		return nil
	}
	stripped := make([]runtimeContainerState, 0, len(states))
	for _, state := range states {
		item := runtimeContainerState{
			ContainerName: state.ContainerName,
			PodName:       state.PodName,
			InternalIP:    state.InternalIP,
			Status:        state.Status,
		}
		if len(state.VolumeArchives) > 0 {
			item.VolumeArchives = make([]runtimeVolumeArchive, 0, len(state.VolumeArchives))
			for _, archive := range state.VolumeArchives {
				item.VolumeArchives = append(item.VolumeArchives, runtimeVolumeArchive{
					MountPath: archive.MountPath,
				})
			}
		}
		stripped = append(stripped, item)
	}
	return stripped
}

// resolveSnapshotIDForResume 解析恢复实验时应使用的快照 ID。
func resolveSnapshotIDForResume(requested *string, snapshots []*entity.InstanceSnapshot) int64 {
	if requested != nil && strings.TrimSpace(*requested) != "" {
		snapshotID, err := snowflake.ParseString(*requested)
		if err == nil {
			return snapshotID
		}
	}
	if len(snapshots) == 0 || snapshots[0] == nil {
		return 0
	}
	return snapshots[0].ID
}

// captureInstanceRuntimeState 采集真实/混合实验实例的容器运行态。
func (s *instanceService) captureInstanceRuntimeState(ctx context.Context, instance *entity.ExperimentInstance) ([]runtimeContainerState, error) {
	if instance == nil || s.k8sSvc == nil || instance.Namespace == nil || *instance.Namespace == "" {
		return nil, nil
	}
	if instance.ExperimentType != enum.ExperimentTypeReal && instance.ExperimentType != enum.ExperimentTypeMixed {
		return nil, nil
	}

	templateAggregate, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.templateContainerRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		instance.TemplateID,
	)
	if err != nil {
		return nil, err
	}
	containers, err := s.containerRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return nil, err
	}

	templateMap := make(map[int64]entity.TemplateContainer, len(templateAggregate.Containers))
	for _, container := range templateAggregate.Containers {
		templateMap[container.ID] = *container
	}

	states := make([]runtimeContainerState, 0, len(containers))
	for _, container := range containers {
		if container == nil {
			continue
		}
		state := runtimeContainerState{
			ContainerName: container.ContainerName,
			Status:        int(container.Status),
		}
		if container.PodName != nil {
			state.PodName = *container.PodName
		}
		if container.InternalIP != nil {
			state.InternalIP = *container.InternalIP
		}

		templateContainer, ok := templateMap[container.TemplateContainerID]
		if !ok || state.PodName == "" {
			states = append(states, state)
			continue
		}
		spec, _, buildErr := s.buildContainerSpec(ctx, templateContainer, templateAggregate)
		if buildErr != nil {
			return nil, buildErr
		}
		mountPaths := collectVolumeMountPaths(spec.Volumes)
		if len(mountPaths) == 0 {
			states = append(states, state)
			continue
		}

		runtimeState, captureErr := s.k8sSvc.CaptureContainerRuntimeState(ctx, *instance.Namespace, state.PodName, container.ContainerName, mountPaths)
		if captureErr != nil {
			return nil, captureErr
		}
		if runtimeState != nil {
			runtimeState.ContainerName = container.ContainerName
			runtimeState.PodName = state.PodName
			runtimeState.InternalIP = state.InternalIP
			runtimeState.Status = int(container.Status)
			states = append(states, *runtimeState)
			continue
		}
		states = append(states, state)
	}

	sort.SliceStable(states, func(i, j int) bool {
		return states[i].ContainerName < states[j].ContainerName
	})
	return states, nil
}

// restoreInstanceRuntimeState 将对象存储中的容器运行态恢复到新建实验环境。
func (s *instanceService) restoreInstanceRuntimeState(ctx context.Context, instance *entity.ExperimentInstance, snapshot *entity.InstanceSnapshot) error {
	if instance == nil || snapshot == nil || s.k8sSvc == nil {
		return nil
	}
	if instance.ExperimentType != enum.ExperimentTypeReal && instance.ExperimentType != enum.ExperimentTypeMixed {
		return nil
	}
	if instance.Namespace == nil || *instance.Namespace == "" {
		return fmt.Errorf("实例命名空间为空，无法恢复容器状态")
	}

	payload, err := s.loadSnapshotArchive(ctx, snapshot)
	if err != nil {
		return err
	}
	states, err := decodeRuntimeContainerStates(payload.ContainerStates)
	if err != nil {
		return err
	}
	if len(states) == 0 {
		return nil
	}

	containers, err := s.containerRepo.ListByInstanceID(ctx, instance.ID)
	if err != nil {
		return err
	}
	containerMap := make(map[string]*entity.InstanceContainer, len(containers))
	for _, container := range containers {
		if container == nil {
			continue
		}
		containerMap[container.ContainerName] = container
	}

	for _, state := range states {
		container := containerMap[state.ContainerName]
		if container == nil || container.PodName == nil || *container.PodName == "" {
			continue
		}
		if err := s.k8sSvc.RestoreContainerRuntimeState(ctx, *instance.Namespace, *container.PodName, container.ContainerName, (*RuntimeContainerState)(&state)); err != nil {
			return err
		}
	}
	return nil
}

// resolveResumeSnapshot 获取恢复实例时实际要使用的快照实体。
func (s *instanceService) resolveResumeSnapshot(ctx context.Context, instanceID int64, requested *string) (*entity.InstanceSnapshot, error) {
	snapshots, err := s.snapshotRepo.ListByInstanceID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	snapshotID := resolveSnapshotIDForResume(requested, snapshots)
	if snapshotID == 0 {
		return nil, errcode.ErrSnapshotNotFound
	}
	for _, snapshot := range snapshots {
		if snapshot != nil && snapshot.ID == snapshotID {
			return snapshot, nil
		}
	}
	return nil, errcode.ErrSnapshotNotFound
}

// syncPausedContainerState 将实例容器状态同步为已停止，清理运行态地址信息。
func (s *instanceService) syncPausedContainerState(ctx context.Context, instanceID int64) error {
	containers, err := s.containerRepo.ListByInstanceID(ctx, instanceID)
	if err != nil {
		return err
	}
	for _, container := range containers {
		if container == nil {
			continue
		}
		if err := s.containerRepo.UpdateFields(ctx, container.ID, map[string]interface{}{
			"status":      enum.ContainerStatusStopped,
			"pod_name":    nil,
			"internal_ip": nil,
		}); err != nil {
			return err
		}
	}
	return nil
}

// syncRestoredContainerState 将恢复后的实例容器状态同步为运行中。
func (s *instanceService) syncRestoredContainerState(ctx context.Context, instanceID int64) error {
	containers, err := s.containerRepo.ListByInstanceID(ctx, instanceID)
	if err != nil {
		return err
	}
	for _, container := range containers {
		if container == nil {
			continue
		}
		if err := s.containerRepo.UpdateFields(ctx, container.ID, map[string]interface{}{
			"status": enum.ContainerStatusRunning,
		}); err != nil {
			return err
		}
	}
	return nil
}

// collectVolumeMountPaths 提取容器快照需要采集的挂载路径列表。
func collectVolumeMountPaths(volumes []VolumeSpec) []string {
	if len(volumes) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(volumes))
	paths := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		mountPath := strings.TrimSpace(volume.MountPath)
		if mountPath == "" {
			continue
		}
		if _, ok := seen[mountPath]; ok {
			continue
		}
		seen[mountPath] = struct{}{}
		paths = append(paths, mountPath)
	}
	sort.Strings(paths)
	return paths
}

// decodeRuntimeContainerStates 解析快照归档中的容器运行态列表。
func decodeRuntimeContainerStates(raw json.RawMessage) ([]runtimeContainerState, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var states []runtimeContainerState
	if err := json.Unmarshal(raw, &states); err != nil {
		return nil, fmt.Errorf("解析容器运行态快照失败: %w", err)
	}
	return states, nil
}

// encodeRuntimeContainerStates 序列化容器运行态快照列表。
func encodeRuntimeContainerStates(states []runtimeContainerState) (json.RawMessage, error) {
	if len(states) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(states)
	if err != nil {
		return nil, fmt.Errorf("序列化容器运行态快照失败: %w", err)
	}
	return payload, nil
}

// stringifySnapshotID 将快照实体 ID 转为字符串。
func stringifySnapshotID(snapshot *entity.InstanceSnapshot) string {
	if snapshot == nil {
		return ""
	}
	return strconv.FormatInt(snapshot.ID, 10)
}
