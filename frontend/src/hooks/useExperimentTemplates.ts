"use client";

// useExperimentTemplates.ts
// 模块04实验模板、镜像和仿真场景 hook，集中管理 TanStack Query 缓存和五层验证刷新。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  cloneExperimentTemplate,
  createImage,
  createImageVersion,
  createSimScenario,
  createTemplateCheckpoint,
  createTemplateContainer,
  createTemplateRole,
  createTemplateSimScene,
  createExperimentTemplate,
  deleteImage,
  deleteImageVersion,
  deleteSimScenario,
  deleteTemplateCheckpoint,
  deleteTemplateContainer,
  deleteTemplateRole,
  deleteTemplateSimScene,
  deleteExperimentTemplate,
  getImage,
  getImageConfigTemplate,
  getImageDocumentation,
  getSimScenario,
  getExperimentTemplate,
  getTemplateK8sConfig,
  listImageCategories,
  listImagePullStatus,
  listImageVersions,
  listImages,
  listSimLinkGroups,
  listSimScenarios,
  listSharedExperimentTemplates,
  listExperimentTags,
  listExperimentTemplates,
  publishExperimentTemplate,
  reviewImage,
  reviewSimScenario,
  setDefaultImageVersion,
  setTemplateK8sConfig,
  setTemplateTags,
  shareExperimentTemplate,
  sortTemplateCheckpoints,
  sortTemplateContainers,
  triggerImagePull,
  updateImage,
  updateImageVersion,
  updateSimScenario,
  updateTemplateCheckpoint,
  updateTemplateContainer,
  updateTemplateRole,
  updateTemplateSimScene,
  updateTemplateSimSceneLayout,
  updateExperimentTemplate,
  uploadExperimentFile,
  validateExperimentTemplate,
} from "@/services/experiment";
import type { ID, QueryParams } from "@/types/api";
import type { ExperimentFilePurpose, ExperimentTemplateListParams, ImageListParams, SimScenarioRequest } from "@/types/experiment";

/**
 * experimentTemplateQueryKey 实验模板详情 Query key。
 */
export function experimentTemplateQueryKey(templateID: ID) {
  return ["experiment", "template", templateID] as const;
}

/**
 * imageQueryKey 镜像详情 Query key。
 */
export function imageQueryKey(imageID: ID) {
  return ["experiment", "image", imageID] as const;
}

/**
 * useImages 查询镜像库列表。
 */
export function useImages(params: ImageListParams) {
  return useQuery({ queryKey: ["experiment", "images", params], queryFn: () => listImages(params) });
}

/**
 * useImage 查询镜像详情。
 */
export function useImage(imageID: ID) {
  return useQuery({ queryKey: imageQueryKey(imageID), queryFn: () => getImage(imageID), enabled: imageID.length > 0 });
}

/**
 * useImageVersions 查询镜像版本列表。
 */
export function useImageVersions(imageID: ID) {
  return useQuery({ queryKey: ["experiment", "image-versions", imageID], queryFn: () => listImageVersions(imageID), enabled: imageID.length > 0 });
}

/**
 * useImageCategories 查询镜像分类。
 */
export function useImageCategories() {
  return useQuery({ queryKey: ["experiment", "image-categories"], queryFn: listImageCategories });
}

/**
 * useImageConfigTemplate 查询镜像配置模板，用于自动填充端口、环境变量、依赖和资源建议。
 */
export function useImageConfigTemplate(imageID: ID) {
  return useQuery({ queryKey: ["experiment", "image-config-template", imageID], queryFn: () => getImageConfigTemplate(imageID), enabled: imageID.length > 0 });
}

/**
 * useImageDocumentation 查询镜像结构化文档，敏感字段由后端过滤后展示。
 */
export function useImageDocumentation(imageID: ID) {
  return useQuery({ queryKey: ["experiment", "image-documentation", imageID], queryFn: () => getImageDocumentation(imageID), enabled: imageID.length > 0 });
}

/**
 * useImageMutations 镜像创建、编辑、审核、删除和版本管理 mutation。
 */
export function useImageMutations(imageID?: ID) {
  const queryClient = useQueryClient();
  const refreshImages = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "images"] });
    void queryClient.invalidateQueries({ queryKey: ["experiment", "school-images"] });
    if (imageID) {
      void queryClient.invalidateQueries({ queryKey: imageQueryKey(imageID) });
      void queryClient.invalidateQueries({ queryKey: ["experiment", "image-versions", imageID] });
    }
  };

  return {
    create: useMutation({ mutationFn: createImage, onSuccess: refreshImages }),
    update: useMutation({ mutationFn: (payload: Parameters<typeof updateImage>[1]) => updateImage(imageID ?? "", payload), onSuccess: refreshImages }),
    remove: useMutation({ mutationFn: () => deleteImage(imageID ?? ""), onSuccess: refreshImages }),
    review: useMutation({ mutationFn: (payload: Parameters<typeof reviewImage>[1]) => reviewImage(imageID ?? "", payload), onSuccess: refreshImages }),
    createVersion: useMutation({ mutationFn: (payload: Parameters<typeof createImageVersion>[1]) => createImageVersion(imageID ?? "", payload), onSuccess: refreshImages }),
    updateVersion: useMutation({ mutationFn: ({ versionID, payload }: { versionID: ID; payload: Parameters<typeof updateImageVersion>[1] }) => updateImageVersion(versionID, payload), onSuccess: refreshImages }),
    deleteVersion: useMutation({ mutationFn: deleteImageVersion, onSuccess: refreshImages }),
    setDefaultVersion: useMutation({ mutationFn: setDefaultImageVersion, onSuccess: refreshImages }),
  };
}

/**
 * useExperimentTemplates 查询实验模板列表。
 */
export function useExperimentTemplates(params: ExperimentTemplateListParams) {
  return useQuery({ queryKey: ["experiment", "templates", params], queryFn: () => listExperimentTemplates(params) });
}

/**
 * useSharedExperimentTemplates 查询共享实验模板库。
 */
export function useSharedExperimentTemplates(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "shared-templates", params], queryFn: () => listSharedExperimentTemplates(params) });
}

/**
 * useExperimentTemplate 查询实验模板详情。
 */
export function useExperimentTemplate(templateID: ID) {
  return useQuery({ queryKey: experimentTemplateQueryKey(templateID), queryFn: () => getExperimentTemplate(templateID), enabled: templateID.length > 0 });
}

/**
 * useTemplateK8sConfig 查询模板 K8s 编排配置。
 */
export function useTemplateK8sConfig(templateID: ID) {
  return useQuery({ queryKey: ["experiment", "template-k8s", templateID], queryFn: () => getTemplateK8sConfig(templateID), enabled: templateID.length > 0 });
}

/**
 * useTemplateValidation 查询模板五层验证结果，发布前必须依赖后端验证结果。
 */
export function useTemplateValidation(templateID: ID, enabled = false) {
  return useQuery({ queryKey: ["experiment", "template-validation", templateID], queryFn: () => validateExperimentTemplate(templateID), enabled: enabled && templateID.length > 0 });
}

/**
 * useExperimentTemplateMutations 实验模板基础信息、发布、克隆、共享和标签 mutation。
 */
export function useExperimentTemplateMutations(templateID?: ID) {
  const queryClient = useQueryClient();
  const refreshTemplate = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "templates"] });
    if (templateID) {
      void queryClient.invalidateQueries({ queryKey: experimentTemplateQueryKey(templateID) });
      void queryClient.invalidateQueries({ queryKey: ["experiment", "template-validation", templateID] });
    }
  };

  return {
    create: useMutation({ mutationFn: createExperimentTemplate, onSuccess: refreshTemplate }),
    update: useMutation({ mutationFn: (payload: Parameters<typeof updateExperimentTemplate>[1]) => updateExperimentTemplate(templateID ?? "", payload), onSuccess: refreshTemplate }),
    remove: useMutation({ mutationFn: () => deleteExperimentTemplate(templateID ?? ""), onSuccess: refreshTemplate }),
    publish: useMutation({ mutationFn: () => publishExperimentTemplate(templateID ?? ""), onSuccess: refreshTemplate }),
    clone: useMutation({ mutationFn: () => cloneExperimentTemplate(templateID ?? ""), onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["experiment", "templates"] }) }),
    share: useMutation({ mutationFn: (isShared: boolean) => shareExperimentTemplate(templateID ?? "", isShared), onSuccess: refreshTemplate }),
    setTags: useMutation({ mutationFn: (tagIDs: ID[]) => setTemplateTags(templateID ?? "", tagIDs), onSuccess: refreshTemplate }),
    setK8s: useMutation({ mutationFn: (config: Parameters<typeof setTemplateK8sConfig>[1]) => setTemplateK8sConfig(templateID ?? "", config), onSuccess: refreshTemplate }),
    validate: useMutation({ mutationFn: () => validateExperimentTemplate(templateID ?? ""), onSuccess: refreshTemplate }),
  };
}

/**
 * useTemplateConfigMutations 容器、检查点、仿真场景、角色和排序 mutation。
 */
export function useTemplateConfigMutations(templateID: ID) {
  const queryClient = useQueryClient();
  const refreshTemplate = () => {
    void queryClient.invalidateQueries({ queryKey: experimentTemplateQueryKey(templateID) });
    void queryClient.invalidateQueries({ queryKey: ["experiment", "template-validation", templateID] });
  };

  return {
    createContainer: useMutation({ mutationFn: (payload: Parameters<typeof createTemplateContainer>[1]) => createTemplateContainer(templateID, payload), onSuccess: refreshTemplate }),
    updateContainer: useMutation({ mutationFn: ({ containerID, payload }: { containerID: ID; payload: Parameters<typeof updateTemplateContainer>[1] }) => updateTemplateContainer(containerID, payload), onSuccess: refreshTemplate }),
    deleteContainer: useMutation({ mutationFn: deleteTemplateContainer, onSuccess: refreshTemplate }),
    sortContainers: useMutation({ mutationFn: (orderedIDs: ID[]) => sortTemplateContainers(templateID, orderedIDs), onSuccess: refreshTemplate }),
    createCheckpoint: useMutation({ mutationFn: (payload: Parameters<typeof createTemplateCheckpoint>[1]) => createTemplateCheckpoint(templateID, payload), onSuccess: refreshTemplate }),
    updateCheckpoint: useMutation({ mutationFn: ({ checkpointID, payload }: { checkpointID: ID; payload: Parameters<typeof updateTemplateCheckpoint>[1] }) => updateTemplateCheckpoint(checkpointID, payload), onSuccess: refreshTemplate }),
    deleteCheckpoint: useMutation({ mutationFn: deleteTemplateCheckpoint, onSuccess: refreshTemplate }),
    sortCheckpoints: useMutation({ mutationFn: (orderedIDs: ID[]) => sortTemplateCheckpoints(templateID, orderedIDs), onSuccess: refreshTemplate }),
    createScene: useMutation({ mutationFn: (payload: Parameters<typeof createTemplateSimScene>[1]) => createTemplateSimScene(templateID, payload), onSuccess: refreshTemplate }),
    updateScene: useMutation({ mutationFn: ({ sceneID, payload }: { sceneID: ID; payload: Parameters<typeof updateTemplateSimScene>[1] }) => updateTemplateSimScene(sceneID, payload), onSuccess: refreshTemplate }),
    deleteScene: useMutation({ mutationFn: deleteTemplateSimScene, onSuccess: refreshTemplate }),
    updateSceneLayout: useMutation({ mutationFn: (layouts: Parameters<typeof updateTemplateSimSceneLayout>[1]) => updateTemplateSimSceneLayout(templateID, layouts), onSuccess: refreshTemplate }),
    createRole: useMutation({ mutationFn: (payload: Parameters<typeof createTemplateRole>[1]) => createTemplateRole(templateID, payload), onSuccess: refreshTemplate }),
    updateRole: useMutation({ mutationFn: ({ roleID, payload }: { roleID: ID; payload: Parameters<typeof updateTemplateRole>[1] }) => updateTemplateRole(roleID, payload), onSuccess: refreshTemplate }),
    deleteRole: useMutation({ mutationFn: deleteTemplateRole, onSuccess: refreshTemplate }),
  };
}

/**
 * useExperimentTags 查询实验标签。
 */
export function useExperimentTags(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "tags", params], queryFn: () => listExperimentTags(params) });
}

/**
 * useSimScenarios 查询仿真场景库。
 */
export function useSimScenarios(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "sim-scenarios", params], queryFn: () => listSimScenarios(params) });
}

/**
 * useSimScenario 查询仿真场景详情。
 */
export function useSimScenario(scenarioID: ID) {
  return useQuery({ queryKey: ["experiment", "sim-scenario", scenarioID], queryFn: () => getSimScenario(scenarioID), enabled: scenarioID.length > 0 });
}

/**
 * useSimScenarioMutations 仿真场景创建、编辑、删除和审核 mutation。
 */
export function useSimScenarioMutations(scenarioID?: ID) {
  const queryClient = useQueryClient();
  const refreshScenarios = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "sim-scenarios"] });
    if (scenarioID) {
      void queryClient.invalidateQueries({ queryKey: ["experiment", "sim-scenario", scenarioID] });
    }
  };

  return {
    create: useMutation({ mutationFn: (payload: SimScenarioRequest) => createSimScenario(payload), onSuccess: refreshScenarios }),
    update: useMutation({ mutationFn: (payload: Partial<SimScenarioRequest>) => updateSimScenario(scenarioID ?? "", payload), onSuccess: refreshScenarios }),
    remove: useMutation({ mutationFn: () => deleteSimScenario(scenarioID ?? ""), onSuccess: refreshScenarios }),
    review: useMutation({ mutationFn: (payload: Parameters<typeof reviewSimScenario>[1]) => reviewSimScenario(scenarioID ?? "", payload), onSuccess: refreshScenarios }),
  };
}

/**
 * useSimLinkGroups 查询仿真场景联动组。
 */
export function useSimLinkGroups() {
  return useQuery({ queryKey: ["experiment", "sim-link-groups"], queryFn: listSimLinkGroups });
}

/**
 * useExperimentFileUploadMutation 上传实验报告、场景包或镜像文档，并提供上传进度。
 */
export function useExperimentFileUploadMutation() {
  return useMutation({ mutationFn: ({ file, purpose, onUploadProgress }: { file: File; purpose: ExperimentFilePurpose; onUploadProgress?: (progress: number) => void }) => uploadExperimentFile(file, purpose, onUploadProgress) });
}

/**
 * useImagePullStatus 查询管理端镜像预拉取状态。
 */
export function useImagePullStatus(params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "image-pull-status", params], queryFn: () => listImagePullStatus(params) });
}

/**
 * useTriggerImagePullMutation 触发镜像预拉取，成功后刷新预拉取状态。
 */
export function useTriggerImagePullMutation() {
  const queryClient = useQueryClient();
  return useMutation({ mutationFn: triggerImagePull, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["experiment", "image-pull-status"] }) });
}
