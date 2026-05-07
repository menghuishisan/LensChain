// experiment.ts
// 模块04实验环境 service：HTTP 接口仅通过统一 apiClient 调用，WebSocket 另由 hook 管理。

import { apiClient } from "@/lib/api-client";
import type { ID, PaginatedData, QueryParams } from "@/types/api";
import type {
  AssetStatus,
  AutoAssignGroupsResponse,
  ContainerResourceItem,
  CreateExperimentResourceQuotaRequest,
  CreateExperimentGroupRequest,
  CreateExperimentInstanceRequest,
  CreateExperimentInstanceResponse,
  CreateExperimentTemplateResponse,
  CreateImageRequest,
  CreateImageResponse,
  ExperimentFilePurpose,
  ExperimentGroup,
  ExperimentGroupListResponse,
  ExperimentGroupMessageListResponse,
  ExperimentGroupProgress,
  ExperimentHeartbeatResponse,
  ExperimentInstanceDetail,
  ExperimentInstanceListParams,
  ExperimentInstanceListResponse,
  ExperimentMonitor,
  SchoolExperimentMonitor,
  ExperimentOperationLogListResponse,
  ExperimentOverview,
  ExperimentReport,
  ExperimentResourceQuota,
  ExperimentResourceUsage,
  ExperimentSnapshot,
  ExperimentStatistics,
  ExperimentTemplateDetail,
  ExperimentTemplateListParams,
  ExperimentTemplateListResponse,
  ExperimentTemplateRequest,
  ImageCategory,
  ImageConfigTemplate,
  ImageDetail,
  ImageDocumentation,
  ImageListParams,
  ImageListResponse,
  ImagePullStatusResponse,
  ImageVersion,
  JsonObject,
  K8sClusterStatus,
  ReviewAssetRequest,
  ResumeExperimentInstanceRequest,
  SimLinkGroup,
  SimScenarioDetail,
  SimScenarioListResponse,
  SimScenarioRequest,
  StudentTemplateSummary,
  TemplateCheckpoint,
  TemplateCheckpointRequest,
  TemplateContainer,
  TemplateContainerRequest,
  TemplateRoleRequest,
  TemplateSimScene,
  TemplateSimSceneRequest,
  TemplateValidationResponse,
  UpdateImageRequest,
  UpdateExperimentResourceQuotaRequest,
  UploadExperimentFileResponse,
  VerifyCheckpointResponse,
} from "@/types/experiment";

/**
 * listImages 对应 GET /api/v1/images，用于镜像库列表。
 */
export function listImages(params: ImageListParams) {
  return apiClient.get<ImageListResponse>("/images", { query: params });
}

/**
 * createImage 对应 POST /api/v1/images，用于上传或登记镜像。
 */
export function createImage(payload: CreateImageRequest) {
  return apiClient.post<CreateImageResponse>("/images", payload);
}

/**
 * getImage 对应 GET /api/v1/images/:id，用于镜像详情。
 */
export function getImage(imageID: ID) {
  return apiClient.get<ImageDetail>(`/images/${imageID}`);
}

/**
 * updateImage 对应 PUT /api/v1/images/:id，用于编辑镜像元数据。
 */
export function updateImage(imageID: ID, payload: UpdateImageRequest) {
  return apiClient.put<null>(`/images/${imageID}`, payload);
}

/**
 * deleteImage 对应 DELETE /api/v1/images/:id，用于删除未被使用的镜像。
 */
export function deleteImage(imageID: ID) {
  return apiClient.delete<null>(`/images/${imageID}`);
}

/**
 * reviewImage 对应 POST /api/v1/images/:id/review，用于镜像审核。
 */
export function reviewImage(imageID: ID, payload: ReviewAssetRequest) {
  return apiClient.post<null>(`/images/${imageID}/review`, payload);
}

/**
 * listImageVersions 对应 GET /api/v1/images/:id/versions，用于镜像版本列表。
 */
export function listImageVersions(imageID: ID) {
  return apiClient.get<ImageVersion[]>(`/images/${imageID}/versions`);
}

/**
 * createImageVersion 对应 POST /api/v1/images/:id/versions，用于新增镜像版本。
 */
export function createImageVersion(imageID: ID, payload: { version: string; registry_url: string; image_size?: number | null; digest?: string | null; min_cpu?: string | null; min_memory?: string | null; min_disk?: string | null; is_default: boolean }) {
  return apiClient.post<{ id: ID; version: string; is_default: boolean }>(`/images/${imageID}/versions`, payload);
}

/**
 * updateImageVersion 对应 PUT /api/v1/image-versions/:id，用于编辑镜像版本。
 */
export function updateImageVersion(versionID: ID, payload: { registry_url?: string | null; min_cpu?: string | null; min_memory?: string | null; min_disk?: string | null }) {
  return apiClient.put<null>(`/image-versions/${versionID}`, payload);
}

/**
 * deleteImageVersion 对应 DELETE /api/v1/image-versions/:id，用于删除镜像版本。
 */
export function deleteImageVersion(versionID: ID) {
  return apiClient.delete<null>(`/image-versions/${versionID}`);
}

/**
 * setDefaultImageVersion 对应 PATCH /api/v1/image-versions/:id/default，用于设为默认版本。
 */
export function setDefaultImageVersion(versionID: ID) {
  return apiClient.patch<null>(`/image-versions/${versionID}/default`);
}

/**
 * listImageCategories 对应 GET /api/v1/image-categories，用于镜像分类树。
 */
export function listImageCategories() {
  return apiClient.get<ImageCategory[]>("/image-categories");
}

/**
 * getImageConfigTemplate 对应 GET /api/v1/images/:id/config-template，用于模板编辑器自动填充。
 */
export function getImageConfigTemplate(imageID: ID) {
  return apiClient.get<ImageConfigTemplate>(`/images/${imageID}/config-template`);
}

/**
 * getImageDocumentation 对应 GET /api/v1/images/:id/documentation，用于镜像结构化文档。
 */
export function getImageDocumentation(imageID: ID) {
  return apiClient.get<ImageDocumentation>(`/images/${imageID}/documentation`);
}

/**
 * listExperimentTemplates 对应 GET /api/v1/experiment-templates，用于实验模板列表。
 */
export function listExperimentTemplates(params: ExperimentTemplateListParams) {
  return apiClient.get<ExperimentTemplateListResponse>("/experiment-templates", { query: params });
}

/**
 * listSharedExperimentTemplates 对应 GET /api/v1/shared-experiment-templates，用于共享实验模板库。
 */
export function listSharedExperimentTemplates(params: QueryParams = {}) {
  return apiClient.get<ExperimentTemplateListResponse>("/shared-experiment-templates", { query: params });
}

/**
 * listStudentExperimentTemplates 对应 GET /api/v1/student/experiment-templates，学生端已发布模板列表。
 */
export function listStudentExperimentTemplates(params: QueryParams = {}) {
  return apiClient.get<ExperimentTemplateListResponse>("/student/experiment-templates", { query: params });
}

/**
 * getStudentExperimentTemplate 对应 GET /api/v1/student/experiment-templates/:id，学生端模板摘要详情。
 */
export function getStudentExperimentTemplate(templateID: ID) {
  return apiClient.get<StudentTemplateSummary>(`/student/experiment-templates/${templateID}`);
}

/**
 * createExperimentTemplate 对应 POST /api/v1/experiment-templates，用于创建实验模板草稿。
 */
export function createExperimentTemplate(payload: ExperimentTemplateRequest) {
  return apiClient.post<CreateExperimentTemplateResponse>("/experiment-templates", payload);
}

/**
 * getExperimentTemplate 对应 GET /api/v1/experiment-templates/:id，用于实验模板详情。
 */
export function getExperimentTemplate(templateID: ID) {
  return apiClient.get<ExperimentTemplateDetail>(`/experiment-templates/${templateID}`);
}

/**
 * updateExperimentTemplate 对应 PUT /api/v1/experiment-templates/:id，用于编辑实验模板基础信息。
 */
export function updateExperimentTemplate(templateID: ID, payload: Partial<ExperimentTemplateRequest>) {
  return apiClient.put<null>(`/experiment-templates/${templateID}`, payload);
}

/**
 * deleteExperimentTemplate 对应 DELETE /api/v1/experiment-templates/:id，用于删除草稿模板。
 */
export function deleteExperimentTemplate(templateID: ID) {
  return apiClient.delete<null>(`/experiment-templates/${templateID}`);
}

/**
 * publishExperimentTemplate 对应 POST /api/v1/experiment-templates/:id/publish，用于发布模板。
 */
export function publishExperimentTemplate(templateID: ID) {
  return apiClient.post<null>(`/experiment-templates/${templateID}/publish`);
}

/**
 * cloneExperimentTemplate 对应 POST /api/v1/experiment-templates/:id/clone，用于克隆模板。
 */
export function cloneExperimentTemplate(templateID: ID) {
  return apiClient.post<{ id: ID }>(`/experiment-templates/${templateID}/clone`);
}

/**
 * shareExperimentTemplate 对应 PATCH /api/v1/experiment-templates/:id/share，用于设置共享状态。
 */
export function shareExperimentTemplate(templateID: ID, isShared: boolean) {
  return apiClient.patch<null>(`/experiment-templates/${templateID}/share`, { is_shared: isShared });
}

/**
 * getTemplateK8sConfig 对应 GET /api/v1/experiment-templates/:id/k8s-config，用于读取 K8s 编排配置。
 */
export function getTemplateK8sConfig(templateID: ID) {
  return apiClient.get<{ template_id: ID; k8s_config: JsonObject | null }>(`/experiment-templates/${templateID}/k8s-config`);
}

/**
 * setTemplateK8sConfig 对应 POST /api/v1/experiment-templates/:id/k8s-config，用于保存 K8s 编排配置。
 */
export function setTemplateK8sConfig(templateID: ID, k8sConfig: JsonObject) {
  return apiClient.post<null>(`/experiment-templates/${templateID}/k8s-config`, { k8s_config: k8sConfig });
}

/**
 * validateExperimentTemplate 对应 POST /api/v1/experiment-templates/:id/validate，用于五层发布验证。
 */
export function validateExperimentTemplate(templateID: ID, levels: number[] = [1, 2, 3, 4, 5]) {
  return apiClient.post<TemplateValidationResponse>(`/experiment-templates/${templateID}/validate`, { levels });
}

/**
 * createTemplateContainer 对应 POST /api/v1/experiment-templates/:id/containers，用于添加容器。
 */
export function createTemplateContainer(templateID: ID, payload: TemplateContainerRequest) {
  return apiClient.post<TemplateContainer>(`/experiment-templates/${templateID}/containers`, payload);
}

/**
 * updateTemplateContainer 对应 PUT /api/v1/template-containers/:id，用于编辑容器。
 */
export function updateTemplateContainer(containerID: ID, payload: Partial<TemplateContainerRequest>) {
  return apiClient.put<null>(`/template-containers/${containerID}`, payload);
}

/**
 * deleteTemplateContainer 对应 DELETE /api/v1/template-containers/:id，用于删除容器。
 */
export function deleteTemplateContainer(containerID: ID) {
  return apiClient.delete<null>(`/template-containers/${containerID}`);
}

/**
 * sortTemplateContainers 对应 PUT /api/v1/experiment-templates/:id/containers/sort，用于容器启动顺序排序。
 */
export function sortTemplateContainers(templateID: ID, orderedIDs: ID[]) {
  return apiClient.put<null>(`/experiment-templates/${templateID}/containers/sort`, { ordered_ids: orderedIDs });
}

/**
 * createTemplateCheckpoint 对应 POST /api/v1/experiment-templates/:id/checkpoints，用于添加检查点。
 */
export function createTemplateCheckpoint(templateID: ID, payload: TemplateCheckpointRequest) {
  return apiClient.post<TemplateCheckpoint>(`/experiment-templates/${templateID}/checkpoints`, payload);
}

/**
 * updateTemplateCheckpoint 对应 PUT /api/v1/template-checkpoints/:id，用于编辑检查点。
 */
export function updateTemplateCheckpoint(checkpointID: ID, payload: Partial<TemplateCheckpointRequest>) {
  return apiClient.put<null>(`/template-checkpoints/${checkpointID}`, payload);
}

/**
 * deleteTemplateCheckpoint 对应 DELETE /api/v1/template-checkpoints/:id，用于删除检查点。
 */
export function deleteTemplateCheckpoint(checkpointID: ID) {
  return apiClient.delete<null>(`/template-checkpoints/${checkpointID}`);
}

/**
 * sortTemplateCheckpoints 对应 PUT /api/v1/experiment-templates/:id/checkpoints/sort，用于检查点排序。
 */
export function sortTemplateCheckpoints(templateID: ID, orderedIDs: ID[]) {
  return apiClient.put<null>(`/experiment-templates/${templateID}/checkpoints/sort`, { ordered_ids: orderedIDs });
}

/**
 * createTemplateSimScene 对应 POST /api/v1/experiment-templates/:id/sim-scenes，用于绑定仿真场景。
 */
export function createTemplateSimScene(templateID: ID, payload: TemplateSimSceneRequest) {
  return apiClient.post<TemplateSimScene>(`/experiment-templates/${templateID}/sim-scenes`, payload);
}

/**
 * updateTemplateSimScene 对应 PUT /api/v1/template-sim-scenes/:id，用于编辑仿真场景绑定。
 */
export function updateTemplateSimScene(sceneID: ID, payload: Partial<TemplateSimSceneRequest>) {
  return apiClient.put<null>(`/template-sim-scenes/${sceneID}`, payload);
}

/**
 * deleteTemplateSimScene 对应 DELETE /api/v1/template-sim-scenes/:id，用于移除仿真场景绑定。
 */
export function deleteTemplateSimScene(sceneID: ID) {
  return apiClient.delete<null>(`/template-sim-scenes/${sceneID}`);
}

/**
 * updateTemplateSimSceneLayout 对应 PUT /api/v1/experiment-templates/:id/sim-scenes/layout，用于保存 SimEngine 面板布局。
 */
export function updateTemplateSimSceneLayout(templateID: ID, layouts: Array<{ scene_id: ID; layout_position: JsonObject }>) {
  return apiClient.put<null>(`/experiment-templates/${templateID}/sim-scenes/layout`, { layouts });
}

/**
 * listExperimentTags 对应 GET /api/v1/tags，用于实验标签筛选和绑定。
 */
export function listExperimentTags(params: QueryParams = {}) {
  return apiClient.get<PaginatedData<{ id: ID; name: string; category: string }>>("/tags", { query: params });
}

/**
 * setTemplateTags 对应 PUT /api/v1/experiment-templates/:id/tags，用于保存模板标签。
 */
export function setTemplateTags(templateID: ID, tagIDs: ID[]) {
  return apiClient.put<null>(`/experiment-templates/${templateID}/tags`, { tag_ids: tagIDs });
}

/**
 * createTemplateRole 对应 POST /api/v1/experiment-templates/:id/roles，用于创建分组角色。
 */
export function createTemplateRole(templateID: ID, payload: TemplateRoleRequest) {
  return apiClient.post<{ id: ID }>(`/experiment-templates/${templateID}/roles`, payload);
}

/**
 * updateTemplateRole 对应 PUT /api/v1/template-roles/:id，用于编辑分组角色。
 */
export function updateTemplateRole(roleID: ID, payload: Partial<TemplateRoleRequest>) {
  return apiClient.put<null>(`/template-roles/${roleID}`, payload);
}

/**
 * deleteTemplateRole 对应 DELETE /api/v1/template-roles/:id，用于删除分组角色。
 */
export function deleteTemplateRole(roleID: ID) {
  return apiClient.delete<null>(`/template-roles/${roleID}`);
}

/**
 * listSimScenarios 对应 GET /api/v1/sim-scenarios，用于仿真场景库。
 */
export function listSimScenarios(params: QueryParams = {}) {
  return apiClient.get<SimScenarioListResponse>("/sim-scenarios", { query: params });
}

/**
 * createSimScenario 对应 POST /api/v1/sim-scenarios，用于上传自定义仿真场景。
 */
export function createSimScenario(payload: SimScenarioRequest) {
  return apiClient.post<{ id: ID }>(`/sim-scenarios`, payload);
}

/**
 * getSimScenario 对应 GET /api/v1/sim-scenarios/:id，用于场景详情。
 */
export function getSimScenario(scenarioID: ID) {
  return apiClient.get<SimScenarioDetail>(`/sim-scenarios/${scenarioID}`);
}

/**
 * updateSimScenario 对应 PUT /api/v1/sim-scenarios/:id，用于编辑场景。
 */
export function updateSimScenario(scenarioID: ID, payload: Partial<SimScenarioRequest>) {
  return apiClient.put<null>(`/sim-scenarios/${scenarioID}`, payload);
}

/**
 * deleteSimScenario 对应 DELETE /api/v1/sim-scenarios/:id，用于删除场景。
 */
export function deleteSimScenario(scenarioID: ID) {
  return apiClient.delete<null>(`/sim-scenarios/${scenarioID}`);
}

/**
 * reviewSimScenario 对应 POST /api/v1/sim-scenarios/:id/review，用于审核仿真场景。
 */
export function reviewSimScenario(scenarioID: ID, payload: ReviewAssetRequest) {
  return apiClient.post<null>(`/sim-scenarios/${scenarioID}/review`, payload);
}

/**
 * listSimLinkGroups 对应 GET /api/v1/sim-link-groups，用于场景联动组列表。
 */
export function listSimLinkGroups() {
  return apiClient.get<SimLinkGroup[]>("/sim-link-groups");
}

/**
 * getSimLinkGroup 对应 GET /api/v1/sim-link-groups/:id，用于联动组详情。
 */
export function getSimLinkGroup(linkGroupID: ID) {
  return apiClient.get<SimLinkGroup>(`/sim-link-groups/${linkGroupID}`);
}

/**
 * createExperimentInstance 对应 POST /api/v1/experiment-instances，用于创建实验实例。
 */
export function createExperimentInstance(payload: CreateExperimentInstanceRequest) {
  return apiClient.post<CreateExperimentInstanceResponse>("/experiment-instances", payload);
}

/**
 * listExperimentInstances 对应 GET /api/v1/experiment-instances，用于学生、教师和管理端实例列表。
 */
export function listExperimentInstances(params: ExperimentInstanceListParams) {
  return apiClient.get<ExperimentInstanceListResponse>("/experiment-instances", { query: params });
}

/**
 * getExperimentInstance 对应 GET /api/v1/experiment-instances/:id，用于实例详情。
 */
export function getExperimentInstance(instanceID: ID) {
  return apiClient.get<ExperimentInstanceDetail>(`/experiment-instances/${instanceID}`);
}

/**
 * pauseExperimentInstance 对应 POST /api/v1/experiment-instances/:id/pause，用于暂停实例。
 */
export function pauseExperimentInstance(instanceID: ID) {
  return apiClient.post<{ snapshot_id: ID | null }>(`/experiment-instances/${instanceID}/pause`);
}

/**
 * resumeExperimentInstance 对应 POST /api/v1/experiment-instances/:id/resume，用于恢复实例。
 */
export function resumeExperimentInstance(instanceID: ID, payload: ResumeExperimentInstanceRequest = {}) {
  return apiClient.post<null>(`/experiment-instances/${instanceID}/resume`, payload);
}

/**
 * restartExperimentInstance 对应 POST /api/v1/experiment-instances/:id/restart，用于重启实例。
 */
export function restartExperimentInstance(instanceID: ID) {
  return apiClient.post<null>(`/experiment-instances/${instanceID}/restart`);
}

/**
 * submitExperimentInstance 对应 POST /api/v1/experiment-instances/:id/submit，用于提交实验。
 */
export function submitExperimentInstance(instanceID: ID) {
  return apiClient.post<{ total_score: number | null }>(`/experiment-instances/${instanceID}/submit`);
}

/**
 * destroyExperimentInstance 对应 POST /api/v1/experiment-instances/:id/destroy，用于销毁实例。
 */
export function destroyExperimentInstance(instanceID: ID) {
  return apiClient.post<null>(`/experiment-instances/${instanceID}/destroy`);
}

/**
 * sendExperimentHeartbeat 对应 POST /api/v1/experiment-instances/:id/heartbeat，用于续活和超时提醒。
 */
export function sendExperimentHeartbeat(instanceID: ID) {
  return apiClient.post<ExperimentHeartbeatResponse>(`/experiment-instances/${instanceID}/heartbeat`, {});
}

/**
 * verifyCheckpoints 对应 POST /api/v1/experiment-instances/:id/checkpoints/verify，用于自动检查点验证。
 */
export function verifyCheckpoints(instanceID: ID, payload: { checkpoint_id?: ID }) {
  return apiClient.post<VerifyCheckpointResponse>(`/experiment-instances/${instanceID}/checkpoints/verify`, payload);
}

/**
 * listCheckpointResults 对应 GET /api/v1/experiment-instances/:id/checkpoints，用于读取检查点结果。
 */
export function listCheckpointResults(instanceID: ID) {
  return apiClient.get<VerifyCheckpointResponse>(`/experiment-instances/${instanceID}/checkpoints`);
}

/**
 * gradeCheckpoint 对应 POST /api/v1/checkpoint-results/:id/grade，用于手动检查点评分。
 */
export function gradeCheckpoint(resultID: ID, payload: { score: number; comment?: string | null }) {
  return apiClient.post<null>(`/checkpoint-results/${resultID}/grade`, payload);
}

/**
 * manualGradeExperimentInstance 对应 POST /api/v1/experiment-instances/:id/manual-grade，用于整体手动评分。
 */
export function manualGradeExperimentInstance(instanceID: ID, payload: { manual_score: number; comment?: string | null }) {
  return apiClient.post<{ total_score: number | null }>(`/experiment-instances/${instanceID}/manual-grade`, payload);
}

/**
 * createSnapshot 对应 POST /api/v1/experiment-instances/:id/snapshots，用于创建快照。
 */
export function createSnapshot(instanceID: ID, payload: { description?: string | null }) {
  return apiClient.post<ExperimentSnapshot>(`/experiment-instances/${instanceID}/snapshots`, payload);
}

/**
 * listSnapshots 对应 GET /api/v1/experiment-instances/:id/snapshots，用于快照列表。
 */
export function listSnapshots(instanceID: ID) {
  return apiClient.get<ExperimentSnapshot[]>(`/experiment-instances/${instanceID}/snapshots`);
}

/**
 * restoreSnapshot 对应 POST /api/v1/experiment-instances/:id/snapshots/:snapshot_id/restore，用于恢复快照。
 */
export function restoreSnapshot(instanceID: ID, snapshotID: ID) {
  return apiClient.post<null>(`/experiment-instances/${instanceID}/snapshots/${snapshotID}/restore`);
}

/**
 * deleteSnapshot 对应 DELETE /api/v1/experiment-instances/:id/snapshots/:snapshot_id，用于删除快照。
 */
export function deleteSnapshot(instanceID: ID, snapshotID: ID) {
  return apiClient.delete<null>(`/experiment-instances/${instanceID}/snapshots/${snapshotID}`);
}

/**
 * listExperimentOperationLogs 对应 GET /api/v1/experiment-instances/:id/operation-logs，用于操作历史。
 */
export function listExperimentOperationLogs(instanceID: ID, params: QueryParams = {}) {
  return apiClient.get<ExperimentOperationLogListResponse>(`/experiment-instances/${instanceID}/operation-logs`, { query: params });
}

/**
 * getExperimentReport 对应 GET /api/v1/experiment-instances/:id/report，用于读取报告。
 */
export function getExperimentReport(instanceID: ID) {
  return apiClient.get<ExperimentReport | null>(`/experiment-instances/${instanceID}/report`);
}

/**
 * createExperimentReport 对应 POST /api/v1/experiment-instances/:id/report，用于提交实验报告。
 */
export function createExperimentReport(instanceID: ID, payload: { content?: string | null; file_url?: string | null; file_name?: string | null; file_size?: number | null }) {
  return apiClient.post<ExperimentReport>(`/experiment-instances/${instanceID}/report`, payload);
}

/**
 * updateExperimentReport 对应 PUT /api/v1/experiment-instances/:id/report，用于更新实验报告。
 */
export function updateExperimentReport(instanceID: ID, payload: { content?: string | null; file_url?: string | null; file_name?: string | null; file_size?: number | null }) {
  return apiClient.put<ExperimentReport>(`/experiment-instances/${instanceID}/report`, payload);
}

/**
 * uploadExperimentFile 对应 POST /api/v1/experiment-files/upload，用于报告、场景包和镜像文档真实上传。
 */
export function uploadExperimentFile(file: File, purpose: ExperimentFilePurpose, onUploadProgress?: (progress: number) => void) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("purpose", purpose);
  return apiClient.upload<UploadExperimentFileResponse>("/experiment-files/upload", formData, { onUploadProgress });
}

/**
 * createExperimentGroups 对应 POST /api/v1/experiment-groups，用于教师创建实验分组。
 */
export function createExperimentGroups(payload: CreateExperimentGroupRequest) {
  return apiClient.post<{ groups: Array<{ id: ID; group_name: string }> }>("/experiment-groups", payload);
}

/**
 * listExperimentGroups 对应 GET /api/v1/experiment-groups，用于实验分组列表。
 */
export function listExperimentGroups(params: QueryParams) {
  return apiClient.get<ExperimentGroupListResponse>("/experiment-groups", { query: params });
}

/**
 * getExperimentGroup 对应 GET /api/v1/experiment-groups/:id，用于分组详情。
 */
export function getExperimentGroup(groupID: ID) {
  return apiClient.get<ExperimentGroup>(`/experiment-groups/${groupID}`);
}

/**
 * updateExperimentGroup 对应 PUT /api/v1/experiment-groups/:id，用于编辑分组。
 */
export function updateExperimentGroup(groupID: ID, payload: { group_name?: string; max_members?: number; status?: number }) {
  return apiClient.put<null>(`/experiment-groups/${groupID}`, payload);
}

/**
 * deleteExperimentGroup 对应 DELETE /api/v1/experiment-groups/:id，用于删除分组。
 */
export function deleteExperimentGroup(groupID: ID) {
  return apiClient.delete<null>(`/experiment-groups/${groupID}`);
}

/**
 * autoAssignExperimentGroups 对应 POST /api/v1/experiment-groups/auto-assign，用于随机分组。
 */
export function autoAssignExperimentGroups(payload: { template_id: ID; course_id: ID; group_size: number; group_name_prefix: string }) {
  return apiClient.post<AutoAssignGroupsResponse>("/experiment-groups/auto-assign", payload);
}

/**
 * joinExperimentGroup 对应 POST /api/v1/experiment-groups/:id/join，用于学生加入分组。
 */
export function joinExperimentGroup(groupID: ID, roleID?: ID | null) {
  return apiClient.post<null>(`/experiment-groups/${groupID}/join`, { role_id: roleID ?? null });
}

/**
 * listExperimentGroupMembers 对应 GET /api/v1/experiment-groups/:id/members，用于分组成员。
 */
export function listExperimentGroupMembers(groupID: ID) {
  return apiClient.get<{ members: NonNullable<ExperimentGroup["members"]> }>(`/experiment-groups/${groupID}/members`);
}

/**
 * removeExperimentGroupMember 对应 DELETE /api/v1/experiment-groups/:id/members/:student_id，用于移除组员。
 */
export function removeExperimentGroupMember(groupID: ID, studentID: ID) {
  return apiClient.delete<null>(`/experiment-groups/${groupID}/members/${studentID}`);
}

/**
 * getExperimentGroupProgress 对应 GET /api/v1/experiment-groups/:id/progress，用于组内进度同步。
 */
export function getExperimentGroupProgress(groupID: ID) {
  return apiClient.get<ExperimentGroupProgress>(`/experiment-groups/${groupID}/progress`);
}

/**
 * listExperimentGroupMessages 对应 GET /api/v1/experiment-groups/:id/messages，用于组内消息历史。
 */
export function listExperimentGroupMessages(groupID: ID, params: QueryParams = {}) {
  return apiClient.get<ExperimentGroupMessageListResponse>(`/experiment-groups/${groupID}/messages`, { query: params });
}

/**
 * sendExperimentGroupMessage 对应 POST /api/v1/experiment-groups/:id/messages，用于发送组内消息。
 */
export function sendExperimentGroupMessage(groupID: ID, content: string) {
  return apiClient.post<null>(`/experiment-groups/${groupID}/messages`, { content });
}

/**
 * getCourseExperimentMonitor 对应 GET /api/v1/courses/:id/experiment-monitor，用于教师课程实验监控。
 */
export function getCourseExperimentMonitor(courseID: ID, params: QueryParams = {}) {
  return apiClient.get<ExperimentMonitor>(`/courses/${courseID}/experiment-monitor`, { query: params });
}

/**
 * sendExperimentGuidance 对应 POST /api/v1/experiment-instances/:id/message，用于教师发送指导消息。
 */
export function sendExperimentGuidance(instanceID: ID, content: string) {
  return apiClient.post<null>(`/experiment-instances/${instanceID}/message`, { content });
}

/**
 * forceDestroyExperimentInstance 对应 POST /api/v1/experiment-instances/:id/force-destroy，用于教师强制销毁实例。
 */
export function forceDestroyExperimentInstance(instanceID: ID) {
  return apiClient.post<null>(`/experiment-instances/${instanceID}/force-destroy`);
}

/**
 * getCourseExperimentStatistics 对应 GET /api/v1/courses/:id/experiment-statistics，用于课程实验统计。
 */
export function getCourseExperimentStatistics(courseID: ID, params: QueryParams = {}) {
  return apiClient.get<ExperimentStatistics>(`/courses/${courseID}/experiment-statistics`, { query: params });
}

/**
 * listResourceQuotas 对应 GET /api/v1/resource-quotas，用于资源配额列表。
 */
export function listResourceQuotas(params: QueryParams = {}) {
  return apiClient.get<PaginatedData<ExperimentResourceQuota>>("/resource-quotas", { query: params });
}

/**
 * getResourceQuota 对应 GET /api/v1/resource-quotas/:id，用于资源配额详情。
 */
export function getResourceQuota(quotaID: ID) {
  return apiClient.get<ExperimentResourceQuota>(`/resource-quotas/${quotaID}`);
}

/**
 * createResourceQuota 对应 POST /api/v1/resource-quotas，用于创建资源配额。
 */
export function createResourceQuota(payload: CreateExperimentResourceQuotaRequest) {
  return apiClient.post<{ id: ID }>("/resource-quotas", payload);
}

/**
 * updateResourceQuota 对应 PUT /api/v1/resource-quotas/:id，用于编辑资源配额。
 */
export function updateResourceQuota(quotaID: ID, payload: UpdateExperimentResourceQuotaRequest) {
  return apiClient.put<null>(`/resource-quotas/${quotaID}`, payload);
}

/**
 * getSchoolResourceUsage 对应 GET /api/v1/schools/:id/resource-usage，用于学校资源使用。
 */
export function getSchoolResourceUsage(schoolID: ID) {
  return apiClient.get<ExperimentResourceUsage>(`/schools/${schoolID}/resource-usage`);
}

/**
 * listSchoolImages 对应 GET /api/v1/school/images，用于学校管理员查看本校镜像。
 */
export function listSchoolImages(params: QueryParams = {}) {
  return apiClient.get<ImageListResponse>("/school/images", { query: params });
}

/**
 * getSchoolExperimentMonitor 对应 GET /api/v1/school/experiment-monitor，用于学校管理员监控本校实验。
 */
export function getSchoolExperimentMonitor(params: QueryParams = {}) {
  return apiClient.get<SchoolExperimentMonitor>("/school/experiment-monitor", { query: params });
}

/**
 * assignCourseQuota 对应 PUT /api/v1/school/course-quotas/:course_id，用于本校课程配额分配。
 */
export function assignCourseQuota(courseID: ID, payload: { max_concurrency: number; max_per_student: number }) {
  return apiClient.put<null>(`/school/course-quotas/${courseID}`, payload);
}

/**
 * getExperimentOverview 对应 GET /api/v1/admin/experiment-overview，用于平台实验总览。
 */
export function getExperimentOverview() {
  return apiClient.get<ExperimentOverview>("/admin/experiment-overview");
}

/**
 * listContainerResources 对应 GET /api/v1/admin/container-resources，用于容器资源实时面板。
 */
export function listContainerResources(params: QueryParams = {}) {
  return apiClient.get<PaginatedData<ContainerResourceItem>>("/admin/container-resources", { query: params });
}

/**
 * getK8sClusterStatus 对应 GET /api/v1/admin/k8s-cluster-status，用于 K8s 集群状态。
 */
export function getK8sClusterStatus() {
  return apiClient.get<K8sClusterStatus>("/admin/k8s-cluster-status");
}

/**
 * listAdminExperimentInstances 对应 GET /api/v1/admin/experiment-instances，用于平台实验实例管理。
 */
export function listAdminExperimentInstances(params: QueryParams = {}) {
  return apiClient.get<ExperimentInstanceListResponse>("/admin/experiment-instances", { query: params });
}

/**
 * forceDestroyAdminExperimentInstance 对应 POST /api/v1/admin/experiment-instances/:id/force-destroy，用于平台强制销毁。
 */
export function forceDestroyAdminExperimentInstance(instanceID: ID) {
  return apiClient.post<null>(`/admin/experiment-instances/${instanceID}/force-destroy`);
}

/**
 * listImagePullStatus 对应 GET /api/v1/admin/image-pull-status，用于镜像预拉取状态。
 */
export function listImagePullStatus(params: QueryParams = {}) {
  return apiClient.get<ImagePullStatusResponse>("/admin/image-pull-status", { query: params });
}

/**
 * triggerImagePull 对应 POST /api/v1/admin/image-pull，用于触发镜像预拉取。
 */
export function triggerImagePull(payload: { image_version_id?: ID; node_name?: string; scope?: "all" | "node" }) {
  return apiClient.post<{ task_id: ID; queued_nodes: string[] }>("/admin/image-pull", payload);
}
