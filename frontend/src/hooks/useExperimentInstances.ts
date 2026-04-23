"use client";

// useExperimentInstances.ts
// 模块04实验实例 hook，统一管理实例生命周期、检查点、快照、报告、监控和资源面板缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  assignCourseQuota,
  createExperimentInstance,
  createExperimentReport,
  createResourceQuota,
  createSnapshot,
  deleteSnapshot,
  destroyExperimentInstance,
  forceDestroyAdminExperimentInstance,
  forceDestroyExperimentInstance,
  getCourseExperimentMonitor,
  getCourseExperimentStatistics,
  getExperimentInstance,
  getExperimentOverview,
  getExperimentReport,
  getK8sClusterStatus,
  getResourceQuota,
  getSchoolExperimentMonitor,
  getSchoolResourceUsage,
  gradeCheckpoint,
  listAdminExperimentInstances,
  listCheckpointResults,
  listContainerResources,
  listExperimentInstances,
  listExperimentOperationLogs,
  listResourceQuotas,
  listSchoolImages,
  listSnapshots,
  manualGradeExperimentInstance,
  pauseExperimentInstance,
  restartExperimentInstance,
  restoreSnapshot,
  resumeExperimentInstance,
  sendExperimentGuidance,
  sendExperimentHeartbeat,
  submitExperimentInstance,
  updateExperimentReport,
  updateResourceQuota,
  uploadExperimentFile,
  verifyCheckpoints,
} from "@/services/experiment";
import type { ID, QueryParams } from "@/types/api";
import type { ExperimentFilePurpose, ExperimentInstanceListParams } from "@/types/experiment";

/**
 * experimentInstanceQueryKey 实验实例详情 Query key。
 */
export function experimentInstanceQueryKey(instanceID: ID) {
  return ["experiment", "instance", instanceID] as const;
}

/**
 * useExperimentInstances 查询实验实例列表。
 */
export function useExperimentInstances(params: ExperimentInstanceListParams) {
  return useQuery({ queryKey: ["experiment", "instances", params], queryFn: () => listExperimentInstances(params) });
}

/**
 * useAdminExperimentInstances 查询平台管理端实验实例列表。
 */
export function useAdminExperimentInstances(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "admin-instances", params], queryFn: () => listAdminExperimentInstances(params) });
}

/**
 * useExperimentInstance 查询实验实例详情。
 */
export function useExperimentInstance(instanceID: ID) {
  return useQuery({ queryKey: experimentInstanceQueryKey(instanceID), queryFn: () => getExperimentInstance(instanceID), enabled: instanceID.length > 0 });
}

/**
 * useExperimentInstanceLifecycleMutations 实例创建、暂停、恢复、重启、提交和销毁操作。
 */
export function useExperimentInstanceLifecycleMutations(instanceID?: ID) {
  const queryClient = useQueryClient();
  const refreshInstances = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "instances"] });
    void queryClient.invalidateQueries({ queryKey: ["experiment", "admin-instances"] });
    void queryClient.invalidateQueries({ queryKey: ["experiment", "monitor"] });
    if (instanceID) {
      void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
      void queryClient.invalidateQueries({ queryKey: ["experiment", "checkpoints", instanceID] });
    }
  };

  return {
    create: useMutation({ mutationFn: createExperimentInstance, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["experiment", "instances"] }) }),
    pause: useMutation({ mutationFn: () => pauseExperimentInstance(instanceID ?? ""), onSuccess: refreshInstances }),
    resume: useMutation({ mutationFn: (payload: Parameters<typeof resumeExperimentInstance>[1]) => resumeExperimentInstance(instanceID ?? "", payload), onSuccess: refreshInstances }),
    restart: useMutation({ mutationFn: () => restartExperimentInstance(instanceID ?? ""), onSuccess: refreshInstances }),
    submit: useMutation({ mutationFn: () => submitExperimentInstance(instanceID ?? ""), onSuccess: refreshInstances }),
    destroy: useMutation({ mutationFn: () => destroyExperimentInstance(instanceID ?? ""), onSuccess: refreshInstances }),
    forceDestroy: useMutation({ mutationFn: () => forceDestroyExperimentInstance(instanceID ?? ""), onSuccess: refreshInstances }),
    adminForceDestroy: useMutation({ mutationFn: () => forceDestroyAdminExperimentInstance(instanceID ?? ""), onSuccess: refreshInstances }),
    heartbeat: useMutation({ mutationFn: () => sendExperimentHeartbeat(instanceID ?? ""), onSuccess: refreshInstances }),
  };
}

/**
 * useCheckpointResults 查询实例检查点结果。
 */
export function useCheckpointResults(instanceID: ID) {
  return useQuery({ queryKey: ["experiment", "checkpoints", instanceID], queryFn: () => listCheckpointResults(instanceID), enabled: instanceID.length > 0 });
}

/**
 * useCheckpointMutations 检查点自动验证、单项手动评分和实例整体手动评分。
 */
export function useCheckpointMutations(instanceID: ID) {
  const queryClient = useQueryClient();
  const refreshCheckpointState = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "checkpoints", instanceID] });
    void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
    void queryClient.invalidateQueries({ queryKey: ["experiment", "instances"] });
  };

  return {
    verify: useMutation({ mutationFn: (checkpointID?: ID) => verifyCheckpoints(instanceID, checkpointID ? { checkpoint_id: checkpointID } : {}), onSuccess: refreshCheckpointState }),
    gradeCheckpoint: useMutation({ mutationFn: ({ resultID, score, comment }: { resultID: ID; score: number; comment?: string | null }) => gradeCheckpoint(resultID, { score, comment }), onSuccess: refreshCheckpointState }),
    manualGrade: useMutation({ mutationFn: (payload: Parameters<typeof manualGradeExperimentInstance>[1]) => manualGradeExperimentInstance(instanceID, payload), onSuccess: refreshCheckpointState }),
  };
}

/**
 * useSnapshots 查询实例快照列表。
 */
export function useSnapshots(instanceID: ID) {
  return useQuery({ queryKey: ["experiment", "snapshots", instanceID], queryFn: () => listSnapshots(instanceID), enabled: instanceID.length > 0 });
}

/**
 * useSnapshotMutations 快照创建、恢复和删除 mutation。
 */
export function useSnapshotMutations(instanceID: ID) {
  const queryClient = useQueryClient();
  const refreshSnapshots = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "snapshots", instanceID] });
    void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
  };

  return {
    create: useMutation({ mutationFn: (description?: string) => createSnapshot(instanceID, { description: description ?? null }), onSuccess: refreshSnapshots }),
    restore: useMutation({ mutationFn: (snapshotID: ID) => restoreSnapshot(instanceID, snapshotID), onSuccess: refreshSnapshots }),
    remove: useMutation({ mutationFn: (snapshotID: ID) => deleteSnapshot(instanceID, snapshotID), onSuccess: refreshSnapshots }),
  };
}

/**
 * useExperimentOperationLogs 查询实验操作历史。
 */
export function useExperimentOperationLogs(instanceID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "operation-logs", instanceID, params], queryFn: () => listExperimentOperationLogs(instanceID, params), enabled: instanceID.length > 0 });
}

/**
 * useExperimentReport 查询实验报告。
 */
export function useExperimentReport(instanceID: ID) {
  return useQuery({ queryKey: ["experiment", "report", instanceID], queryFn: () => getExperimentReport(instanceID), enabled: instanceID.length > 0 });
}

/**
 * useExperimentReportMutations 报告上传、创建和更新 mutation；文件先上传，报告只保存对象存储 key。
 */
export function useExperimentReportMutations(instanceID: ID) {
  const queryClient = useQueryClient();
  const refreshReport = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "report", instanceID] });
    void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
  };

  return {
    upload: useMutation({ mutationFn: ({ file, purpose, onUploadProgress }: { file: File; purpose: ExperimentFilePurpose; onUploadProgress?: (progress: number) => void }) => uploadExperimentFile(file, purpose, onUploadProgress) }),
    create: useMutation({ mutationFn: (payload: Parameters<typeof createExperimentReport>[1]) => createExperimentReport(instanceID, payload), onSuccess: refreshReport }),
    update: useMutation({ mutationFn: (payload: Parameters<typeof updateExperimentReport>[1]) => updateExperimentReport(instanceID, payload), onSuccess: refreshReport }),
  };
}

/**
 * useCourseExperimentMonitor 查询教师课程实验监控面板。
 */
export function useCourseExperimentMonitor(courseID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "monitor", "course", courseID, params], queryFn: () => getCourseExperimentMonitor(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useCourseExperimentStatistics 查询课程实验统计。
 */
export function useCourseExperimentStatistics(courseID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "statistics", courseID, params], queryFn: () => getCourseExperimentStatistics(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useExperimentMonitorMutations 教师监控中的指导消息和强制销毁操作。
 */
export function useExperimentMonitorMutations() {
  const queryClient = useQueryClient();
  return {
    guidance: useMutation({ mutationFn: ({ instanceID, content }: { instanceID: ID; content: string }) => sendExperimentGuidance(instanceID, content), onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["experiment", "monitor"] }) }),
    forceDestroy: useMutation({ mutationFn: forceDestroyExperimentInstance, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["experiment"] }) }),
  };
}

/**
 * useSchoolImages 查询学校管理员本校镜像。
 */
export function useSchoolImages(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "school-images", params], queryFn: () => listSchoolImages(params) });
}

/**
 * useSchoolExperimentMonitor 查询学校管理员本校实验监控。
 */
export function useSchoolExperimentMonitor(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "school-monitor", params], queryFn: () => getSchoolExperimentMonitor(params) });
}

/**
 * useSchoolResourceUsage 查询学校资源使用。
 */
export function useSchoolResourceUsage(schoolID: ID) {
  return useQuery({ queryKey: ["experiment", "resource-usage", schoolID], queryFn: () => getSchoolResourceUsage(schoolID), enabled: schoolID.length > 0 });
}

/**
 * useResourceQuotas 查询资源配额列表。
 */
export function useResourceQuotas(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "resource-quotas", params], queryFn: () => listResourceQuotas(params) });
}

/**
 * useResourceQuota 查询单条资源配额详情。
 */
export function useResourceQuota(quotaID: ID) {
  return useQuery({ queryKey: ["experiment", "resource-quota", quotaID], queryFn: () => getResourceQuota(quotaID), enabled: quotaID.length > 0 });
}

/**
 * useResourceQuotaMutations 资源配额创建、编辑和课程分配 mutation。
 */
export function useResourceQuotaMutations() {
  const queryClient = useQueryClient();
  const refreshQuotas = () => void queryClient.invalidateQueries({ queryKey: ["experiment", "resource-quotas"] });

  return {
    create: useMutation({ mutationFn: createResourceQuota, onSuccess: refreshQuotas }),
    update: useMutation({ mutationFn: ({ quotaID, payload }: { quotaID: ID; payload: Parameters<typeof updateResourceQuota>[1] }) => updateResourceQuota(quotaID, payload), onSuccess: refreshQuotas }),
    assignCourse: useMutation({ mutationFn: ({ courseID, payload }: { courseID: ID; payload: Parameters<typeof assignCourseQuota>[1] }) => assignCourseQuota(courseID, payload), onSuccess: refreshQuotas }),
  };
}

/**
 * useExperimentAdminDashboard 查询平台实验总览、容器资源和 K8s 集群状态。
 */
export function useExperimentAdminDashboard(params: QueryParams = {}) {
  const overview = useQuery({ queryKey: ["experiment", "overview"], queryFn: getExperimentOverview });
  const containers = useQuery({ queryKey: ["experiment", "container-resources", params], queryFn: () => listContainerResources(params) });
  const k8s = useQuery({ queryKey: ["experiment", "k8s-cluster"], queryFn: getK8sClusterStatus });
  return { overview, containers, k8s };
}
