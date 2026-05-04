// experiment.ts
// 模块04实验环境类型定义：镜像、模板、实例、检查点、快照、报告、分组、监控、WebSocket、SimEngine。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/** JSON 对象类型，用于后端 JSONB/RawMessage 字段。 */
export type JsonObject = Record<string, unknown>;

/** 镜像来源类型：1官方 2自定义。 */
export type ImageSourceType = 1 | 2;

/** 镜像/场景状态：1正常 2待审核 3审核拒绝 4已下架。 */
export type AssetStatus = 1 | 2 | 3 | 4;

/** 实验类型：1纯仿真 2真实环境 3混合。 */
export type ExperimentType = 1 | 2 | 3;

/** 拓扑模式：1单人单节点 2单人多节点 3多人协作组网 4共享基础设施。 */
export type TopologyMode = 1 | 2 | 3 | 4;

/** 判题模式：1自动 2手动 3混合。 */
export type JudgeMode = 1 | 2 | 3;

/** 实验模板状态：1草稿 2已发布 3已下架。 */
export type ExperimentTemplateStatus = 1 | 2 | 3;

/** 实验实例状态。 */
export type ExperimentInstanceStatus = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10;

/** SimEngine 时间控制模式。 */
export type SimTimeControlMode = "process" | "reactive" | "continuous";

/** SimEngine 消息类型。 */
export type SimEngineMessageType = "state_diff" | "state_full" | "event" | "link_update" | "control_ack" | "snapshot" | "action" | "control" | "rewind_to";

/** 实验实例 WebSocket 消息类型。 */
export type ExperimentInstanceWSMessageType = "status_change" | "checkpoint_result" | "guidance_message" | "idle_warning" | "duration_warning" | "container_status";

/** 组内聊天 WebSocket 消息类型。 */
export type ExperimentGroupWSMessageType = "chat_message" | "system_notification";

/** 教师监控 WebSocket 消息类型。 */
export type ExperimentMonitorWSMessageType = "student_status_change" | "checkpoint_completed" | "experiment_submitted" | "instance_error";

/** 终端 WebSocket 消息类型。 */
export type ExperimentTerminalWSMessageType = "terminal_output" | "terminal_init" | "guidance_message";

/** 镜像端口配置。 */
export interface ImagePortItem {
  port: number;
  protocol: string;
  name: string;
}

/** 镜像环境变量条件。 */
export interface ImageEnvVarCondition {
  when: string;
  value: string;
  inject_vars: Array<{ key: string; value: string }>;
}

/** 镜像环境变量。 */
export interface ImageEnvVarItem {
  key: string;
  value: string;
  desc?: string | null;
  conditions?: ImageEnvVarCondition[] | null;
}

/** 镜像卷配置。 */
export interface ImageVolumeItem {
  path: string;
  desc?: string | null;
}

/** 镜像依赖/搭配项。 */
export interface ImageDependencyItem {
  image: string;
  reason: string;
}

/** 镜像典型搭配三级提示。 */
export interface ImageTypicalCompanions {
  required: ImageDependencyItem[];
  recommended: ImageDependencyItem[];
  optional: ImageDependencyItem[];
}

/** 镜像资源建议。 */
export interface ImageResourceRecommendation {
  cpu: string;
  memory: string;
  disk: string;
}

/** 镜像列表查询。 */
export interface ImageListParams extends QueryParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  category_id?: ID;
  ecosystem?: string;
  source_type?: ImageSourceType;
  status?: AssetStatus;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

/** 镜像列表项。 */
export interface ImageListItem {
  id: ID;
  name: string;
  display_name: string;
  icon_url: string | null;
  ecosystem: string | null;
  category_name: string;
  source_type: ImageSourceType;
  source_type_text: string;
  status: AssetStatus;
  status_text: string;
  version_count: number;
  usage_count: number;
  created_at: string;
}

/** 镜像版本。 */
export interface ImageVersion {
  id: ID;
  image_id: ID;
  version: string;
  registry_url: string;
  image_size: number | null;
  digest: string | null;
  min_cpu: string | null;
  min_memory: string | null;
  min_disk: string | null;
  is_default: boolean;
  status: number;
  status_text: string;
  scan_result?: JsonObject | null;
  scanned_at: string | null;
  created_at: string;
}

/** 镜像详情。 */
export interface ImageDetail extends ImageListItem {
  category_id: ID;
  description: string | null;
  uploaded_by: ID | null;
  uploader_name: string | null;
  review_comment: string | null;
  default_ports: ImagePortItem[];
  default_env_vars: ImageEnvVarItem[];
  default_volumes: ImageVolumeItem[];
  typical_companions: ImageTypicalCompanions;
  required_dependencies: ImageDependencyItem[];
  resource_recommendation: ImageResourceRecommendation;
  documentation_url: string | null;
  versions: ImageVersion[];
  updated_at: string;
}

/** 创建镜像请求。 */
export interface CreateImageRequest {
  category_id: ID;
  name: string;
  display_name: string;
  description?: string | null;
  icon_url?: string | null;
  ecosystem?: string | null;
  default_ports: ImagePortItem[];
  default_env_vars: ImageEnvVarItem[];
  default_volumes: ImageVolumeItem[];
  typical_companions: ImageTypicalCompanions;
  required_dependencies: ImageDependencyItem[];
  resource_recommendation: ImageResourceRecommendation;
  documentation_url?: string | null;
  versions: Array<{ version: string; registry_url: string; min_cpu?: string | null; min_memory?: string | null; min_disk?: string | null; is_default: boolean }>;
}

/** 编辑镜像请求。 */
export type UpdateImageRequest = Partial<Omit<CreateImageRequest, "category_id" | "name" | "versions">>;

/** 创建镜像响应。 */
export interface CreateImageResponse {
  id: ID;
  name: string;
  display_name: string;
  status: AssetStatus;
  status_text: string;
  versions: Array<{ id: ID; version: string; is_default: boolean }>;
}

/** 审核镜像或仿真场景请求。 */
export interface ReviewAssetRequest {
  action: "approve" | "reject";
  comment?: string | null;
}

/** 镜像列表响应。 */
export type ImageListResponse = PaginatedData<ImageListItem>;

/** 镜像配置模板。 */
export interface ImageConfigTemplate {
  image_id: ID;
  name: string;
  display_name: string;
  ecosystem: string | null;
  default_ports: ImagePortItem[];
  default_env_vars: ImageEnvVarItem[];
  default_volumes: ImageVolumeItem[];
  typical_companions: ImageTypicalCompanions;
  required_dependencies: ImageDependencyItem[];
  resource_recommendation: ImageResourceRecommendation;
  conditional_env_vars_example: Array<{
    key: string;
    default_value: string;
    conditions: ImageEnvVarCondition[];
    description: string;
  }>;
}

/** 镜像结构化文档。 */
export interface ImageDocumentation {
  image_id: ID;
  name: string;
  display_name: string;
  sections: Record<string, string>;
}

/** 镜像分类。 */
export interface ImageCategory {
  id: ID;
  name: string;
  display_name: string;
  parent_id: ID | null;
  sort_order: number;
}

/** 实验模板列表查询。 */
export interface ExperimentTemplateListParams extends QueryParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  experiment_type?: ExperimentType;
  status?: ExperimentTemplateStatus;
  tag_id?: ID;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

/** 实验模板创建/编辑请求。 */
export interface ExperimentTemplateRequest {
  title: string;
  description?: string | null;
  objectives?: string | null;
  instructions?: string | null;
  reference_materials?: string | null;
  experiment_type: ExperimentType;
  topology_mode: TopologyMode;
  judge_mode: JudgeMode;
  auto_weight?: number | null;
  manual_weight?: number | null;
  total_score: number;
  max_duration: number;
  idle_timeout?: number | null;
  cpu_limit?: string | null;
  memory_limit?: string | null;
  disk_limit?: string | null;
  score_strategy: 1 | 2;
}

/** 实验模板列表项。 */
export interface ExperimentTemplateListItem {
  id: ID;
  title: string;
  experiment_type: ExperimentType;
  experiment_type_text: string;
  topology_mode: TopologyMode;
  topology_mode_text: string;
  judge_mode: JudgeMode;
  judge_mode_text: string;
  total_score: number;
  max_duration: number;
  is_shared: boolean;
  status: ExperimentTemplateStatus;
  status_text: string;
  container_count: number;
  checkpoint_count: number;
  tags: ExperimentTag[];
  created_at: string;
  updated_at: string;
}

/** 实验标签。 */
export interface ExperimentTag {
  id: ID;
  name: string;
  category: "ecosystem" | "type" | "difficulty" | "custom";
}

/** 模板容器配置。 */
export interface TemplateContainer {
  id: ID;
  template_id: ID;
  image_version_id: ID;
  container_name: string;
  role_id: ID | null;
  env_vars: Array<{ key: string; value: string }>;
  ports: Array<{ container: number; protocol: string }>;
  volumes: Array<{ host_path: string; container_path: string }>;
  cpu_limit: string | null;
  memory_limit: string | null;
  depends_on: string[];
  startup_order: number;
  is_primary: boolean;
  image_version?: {
    id: ID;
    image_name: string;
    image_display_name: string;
    version: string;
    icon_url: string | null;
  };
}

/** 检查点。 */
export interface TemplateCheckpoint {
  id: ID;
  template_id: ID;
  title: string;
  description: string | null;
  check_type: 1 | 2 | 3;
  check_type_text: string;
  script_content: string | null;
  script_language: string | null;
  target_container: string | null;
  assertion_config: JsonObject | null;
  score: number;
  scope: 1 | 2;
  scope_text: string;
  sort_order: number;
}

/** 模板仿真场景。 */
export interface TemplateSimScene {
  id: ID;
  template_id: ID;
  scenario: SimScenarioBrief | null;
  link_group_id: ID | null;
  link_group_name: string | null;
  scene_params: JsonObject | null;
  initial_state: JsonObject | null;
  data_source_mode: 1 | 2 | 3;
  data_source_mode_text: string;
  data_source_config: JsonObject | null;
  layout_position: JsonObject | null;
}

/** 仿真场景简要信息。 */
export interface SimScenarioBrief {
  id: ID;
  name: string;
  code: string;
  category: string;
  category_text: string;
  time_control_mode: SimTimeControlMode;
  container_image_url: string;
  container_image_size: string | null;
}

/** 实验模板详情。 */
export interface ExperimentTemplateDetail extends ExperimentTemplateListItem {
  description: string | null;
  objectives: string | null;
  instructions: string | null;
  reference_materials: string | null;
  auto_weight: number | null;
  manual_weight: number | null;
  idle_timeout: number | null;
  cpu_limit: string | null;
  memory_limit: string | null;
  disk_limit: string | null;
  score_strategy: 1 | 2;
  teacher: { id: ID; name: string } | null;
  containers: TemplateContainer[];
  checkpoints: TemplateCheckpoint[];
  init_scripts: Array<{ id: ID; target_container: string; script_content?: string; script_language: string; execution_order: number; timeout: number }>;
  sim_scenes: TemplateSimScene[];
  roles: Array<{ id: ID; role_name: string; description: string | null; max_members: number; permissions: JsonObject | null }>;
  k8s_config?: JsonObject | null;
}

/** 创建实验模板响应。 */
export interface CreateExperimentTemplateResponse {
  id: ID;
  title: string;
  experiment_type: ExperimentType;
  experiment_type_text: string;
  topology_mode: TopologyMode;
  topology_mode_text: string;
  judge_mode: JudgeMode;
  judge_mode_text: string;
  status: ExperimentTemplateStatus;
  status_text: string;
}

/** 实验模板列表响应。 */
export type ExperimentTemplateListResponse = PaginatedData<ExperimentTemplateListItem>;

/** 模板五层验证响应。 */
export interface TemplateValidationResponse {
  template_id: ID;
  is_publishable: boolean;
  summary: { errors: number; warnings: number; hints: number; infos: number };
  results: Array<{
    level: number;
    level_name: string;
    severity: "error" | "warning" | "hint" | "info";
    passed: boolean;
    issues: Array<Record<string, unknown>>;
  }>;
}

/** 模板容器请求。 */
export interface TemplateContainerRequest {
  image_version_id: ID;
  container_name: string;
  role_id?: ID | null;
  env_vars: Array<{ key: string; value: string }>;
  ports: Array<{ container: number; protocol: string }>;
  volumes: Array<{ host_path: string; container_path: string }>;
  cpu_limit?: string | null;
  memory_limit?: string | null;
  depends_on: string[];
  startup_order: number;
  is_primary: boolean;
}

/** 模板检查点请求。 */
export interface TemplateCheckpointRequest {
  title: string;
  description?: string | null;
  check_type: 1 | 2 | 3;
  script_content?: string | null;
  script_language?: string | null;
  target_container?: string | null;
  assertion_config?: JsonObject | null;
  score: number;
  scope: 1 | 2;
  sort_order: number;
}

/** 模板仿真场景绑定请求。 */
export interface TemplateSimSceneRequest {
  scenario_id: ID;
  link_group_id?: ID | null;
  scene_params?: JsonObject | null;
  initial_state?: JsonObject | null;
  data_source_mode: 1 | 2 | 3;
  data_source_config?: JsonObject | null;
  layout_position?: JsonObject | null;
}

/** 模板角色请求。 */
export interface TemplateRoleRequest {
  role_name: string;
  description?: string | null;
  max_members: number;
  permissions?: JsonObject | null;
}

/** 仿真场景创建/编辑请求。 */
export interface SimScenarioRequest {
  name: string;
  code: string;
  description?: string | null;
  category: string;
  algorithm_type: string;
  time_control_mode: SimTimeControlMode;
  container_image_url: string;
  container_image_size?: string | null;
  default_params?: JsonObject | null;
  interaction_schema?: JsonObject | null;
  data_source_mode: 1 | 2 | 3;
  default_size?: JsonObject | null;
}

/** 仿真场景列表响应。 */
export type SimScenarioListResponse = PaginatedData<SimScenarioBrief & { status: AssetStatus; status_text: string; source_type: ImageSourceType; source_type_text: string; data_source_mode: 1 | 2 | 3; created_at: string }>;

/** 仿真场景详情。 */
export interface SimScenarioDetail extends SimScenarioBrief {
  description: string | null;
  algorithm_type: string;
  source_type: ImageSourceType;
  source_type_text: string;
  status: AssetStatus;
  status_text: string;
  default_params: JsonObject | null;
  interaction_schema: JsonObject | null;
  data_source_mode: 1 | 2 | 3;
  default_size: JsonObject | null;
  review_comment: string | null;
  created_at: string;
  updated_at: string;
}

/** 联动组。 */
export interface SimLinkGroup {
  id: ID;
  name: string;
  description: string | null;
  scene_count?: number;
  scenes?: Array<{ id: ID; scenario_id: ID; scene_name: string; scene_code: string; link_role: string; sort_order: number }>;
}

/** 实验实例创建请求。 */
export interface CreateExperimentInstanceRequest {
  template_id: ID;
  course_id?: ID | null;
  lesson_id?: ID | null;
  assignment_id?: ID | null;
  snapshot_id?: ID | null;
  group_id?: ID | null;
}

/** 实验实例创建响应。 */
export interface CreateExperimentInstanceResponse {
  instance_id: ID | null;
  sim_session_id: string | null;
  status: ExperimentInstanceStatus;
  status_text: string;
  attempt_no?: number;
  estimated_ready_seconds?: number | null;
  queue_position?: number | null;
  estimated_wait_seconds?: number | null;
}

/** 实验实例详情。 */
export interface ExperimentInstanceDetail {
  id: ID;
  template: { id: ID; title: string; topology_mode: TopologyMode; judge_mode: JudgeMode; instructions: string | null; max_duration: number; idle_timeout: number; total_score: number };
  student: { id: ID; name: string; student_no: string };
  status: ExperimentInstanceStatus;
  status_text: string;
  attempt_no: number;
  sim_session_id: string | null;
  tools: Array<{ kind: string; container_id: ID; container_name: string; proxy_url: string; status: number; status_text: string }>;
  containers: Array<{ id: ID; container_name: string; image_name: string; image_version: string; status: number; status_text: string; internal_ip: string | null; cpu_usage: string | null; memory_usage: string | null; tool_kind: string | null }>;
  checkpoints: Array<{ checkpoint_id: ID; title: string; check_type: number; score: number; result: { id: ID; is_passed: boolean; score: number; checked_at: string } | null }>;
  scores: { auto_score: number | null; manual_score: number | null; total_score: number | null };
  started_at: string | null;
  last_active_at: string | null;
  created_at: string;
}

/** 实验实例列表项。 */
export interface ExperimentInstanceListItem {
  id: ID;
  template_id: ID;
  template_title: string;
  student_id?: ID | null;
  student_name?: string | null;
  school_id?: ID | null;
  school_name?: string | null;
  course_id: ID | null;
  course_title: string | null;
  status: ExperimentInstanceStatus;
  status_text: string;
  attempt_no: number;
  total_score: number | null;
  error_message?: string | null;
  started_at: string | null;
  submitted_at: string | null;
  created_at: string;
  updated_at?: string | null;
}

/** 实验实例列表响应。 */
export type ExperimentInstanceListResponse = PaginatedData<ExperimentInstanceListItem>;

/** 实验实例列表查询参数。 */
export interface ExperimentInstanceListParams extends QueryParams {
  page?: number;
  page_size?: number;
  template_id?: ID;
  course_id?: ID;
  student_id?: ID;
  school_id?: ID;
  status?: ExperimentInstanceStatus;
}

/** 实例恢复请求。 */
export interface ResumeExperimentInstanceRequest {
  snapshot_id?: ID | null;
}

/** 实例心跳响应。 */
export interface ExperimentHeartbeatResponse {
  server_time: string;
  should_continue: boolean;
  warnings: Array<{ type: string; message: string }>;
}

/** 检查点验证响应。 */
export interface VerifyCheckpointResponse {
  results: Array<{ checkpoint_id: ID; title: string; is_passed: boolean; score: number; check_output: string; checked_at: string }>;
}

/** 快照。 */
export interface ExperimentSnapshot {
  id: ID;
  instance_id: ID;
  snapshot_type: number;
  snapshot_type_text: string;
  snapshot_data_url: string;
  container_states: JsonObject | null;
  sim_engine_state: JsonObject | null;
  description: string | null;
  created_at: string;
}

/** 实验报告。 */
export interface ExperimentReport {
  id: ID;
  instance_id: ID;
  content: string | null;
  file_url: string | null;
  file_name: string | null;
  file_size?: number | null;
  created_at: string;
  updated_at: string;
}

/** 实验操作日志。 */
export interface ExperimentOperationLog {
  id: ID;
  instance_id: ID;
  operation_type: string;
  operator_name: string;
  detail: string | null;
  created_at: string;
}

/** 实验操作日志列表。 */
export type ExperimentOperationLogListResponse = PaginatedData<ExperimentOperationLog>;

/** 实验文件用途。 */
export type ExperimentFilePurpose = "experiment_report" | "scenario_package" | "image_document";

/** 实验文件上传响应。 */
export interface UploadExperimentFileResponse {
  file_name: string;
  file_url: string;
  download_url: string;
  file_size: number;
  file_type: string;
}

/** 实验分组。 */
export interface ExperimentGroup {
  id: ID;
  template_id: ID;
  course_id: ID;
  group_name: string;
  group_method: number;
  max_members: number;
  status: number;
  status_text: string;
  members?: Array<{ id: ID; student_id: ID; student_name: string; student_no: string; role_id: ID | null; role_name: string | null; instance_id: ID | null; joined_at: string }>;
  created_at: string;
  updated_at: string;
}

/** 实验分组列表项。 */
export interface ExperimentGroupListItem {
  id: ID;
  group_name: string;
  member_count: number;
  max_members: number;
  status: number;
  status_text: string;
}

/** 实验分组列表响应。 */
export type ExperimentGroupListResponse = PaginatedData<ExperimentGroupListItem>;

/** 实验分组创建请求。 */
export interface CreateExperimentGroupRequest {
  template_id: ID;
  course_id: ID;
  group_method: 1 | 2 | 3;
  groups: Array<{ group_name: string; max_members: number; members: Array<{ student_id: ID; role_id?: ID | null }> }>;
}

/** 实验分组消息。 */
export interface ExperimentGroupMessage {
  id: ID;
  sender_id: ID;
  sender_name: string;
  content: string;
  message_type: number;
  created_at: string;
}

/** 实验分组消息列表响应。 */
export type ExperimentGroupMessageListResponse = PaginatedData<ExperimentGroupMessage>;

/** 实验分组进度。 */
export interface ExperimentGroupProgress {
  group_id: ID;
  group_name: string;
  group_status: number;
  group_status_text: string;
  members: Array<{ student_id: ID; student_name: string; role_name: string; instance_id: ID | null; instance_status: number | null; instance_status_text: string | null; checkpoints_passed: number; checkpoints_total: number; personal_score: number | null }>;
  group_checkpoints: Array<{ checkpoint_id: ID; title: string; scope: number; is_passed: boolean; checked_at: string | null }>;
}

/** 自动分组响应。 */
export interface AutoAssignGroupsResponse {
  total_groups: number;
  total_students: number;
  groups: Array<{ id: ID; group_name: string; members: Array<{ student_id: ID; student_name: string; role_name: string }> }>;
}

/** 监控面板响应。 */
export interface ExperimentMonitor {
  summary: { total_students: number; running: number; paused: number; completed: number; not_started: number; avg_progress: number; resource_usage: { cpu_used: string; cpu_total: string; memory_used: string; memory_total: string } };
  students: Array<{ student_id: ID; student_name: string; student_no: string; instance_id: ID | null; status: number | null; status_text: string | null; checkpoints_passed: number; checkpoints_total: number; progress_percent: number; cpu_usage: string | null; memory_usage: string | null; started_at: string | null; last_active_at: string | null }>;
}

/** 实验统计响应。 */
export interface ExperimentStatistics {
  templates: Array<{ template_id: ID; template_title: string; total_instances: number; submitted_instances: number; avg_score: number | null; avg_duration_minutes: number | null; checkpoint_pass_rates: Array<{ checkpoint_id: ID; title: string; pass_rate: number }> }>;
}

/** 资源配额。 */
export interface ExperimentResourceQuota {
  id: ID;
  quota_level: 1 | 2;
  quota_level_text: string;
  school_id: ID;
  school_name: string;
  course_id: ID | null;
  course_title: string | null;
  max_cpu: string;
  max_memory: string;
  max_storage: string;
  max_concurrency: number;
  max_per_student: number;
  used_cpu: string;
  used_memory: string;
  used_storage: string;
  used_concurrency: number;
  expires_at?: string | null;
  created_at: string;
  updated_at: string;
}

/** 创建资源配额请求。 */
export interface CreateExperimentResourceQuotaRequest {
  quota_level: 1 | 2;
  school_id: ID;
  course_id?: ID | null;
  max_cpu?: string | null;
  max_memory?: string | null;
  max_storage?: string | null;
  max_concurrency?: number | null;
  max_per_student?: number | null;
}

/** 编辑资源配额请求。 */
export interface UpdateExperimentResourceQuotaRequest {
  max_cpu?: string | null;
  max_memory?: string | null;
  max_storage?: string | null;
  max_concurrency?: number | null;
  max_per_student?: number | null;
}

/** 资源使用概览。 */
export interface ExperimentResourceUsage {
  school_id: ID;
  school_name?: string;
  quota?: { max_cpu: string; max_memory: string; max_storage: string; max_concurrency: number };
  usage?: { used_cpu: string; used_memory: string; used_storage: string; used_concurrency: number };
  course_breakdown?: Array<{ course_id: ID; course_title: string; used_cpu: string; used_memory: string; used_storage: string; running_instances: number }>;
}

/** 管理端实验总览。 */
export interface ExperimentOverview {
  total_instances: number;
  running_instances: number;
  queued_instances: number;
  failed_instances: number;
  total_images: number;
  pending_images: number;
  total_scenarios: number;
  pending_scenarios: number;
}

/** 容器资源监控项。 */
export interface ContainerResourceItem {
  instance_id: ID;
  container_name: string;
  school_name: string;
  student_name: string;
  cpu_usage: string;
  memory_usage: string;
  network_rx: string;
  network_tx: string;
  status_text: string;
}

/** K8s 集群状态。 */
export interface K8sClusterStatus {
  nodes: Array<{ name: string; status: string; cpu_allocatable: string; memory_allocatable: string; cpu_used: string; memory_used: string }>;
  namespaces: Array<{ name: string; pod_count: number; running_pods: number; failed_pods: number }>;
}

/** 镜像预拉取状态。 */
export interface ImagePullStatusItem {
  id: ID;
  image_version_id: ID;
  image_name: string;
  version: string;
  node_name: string;
  status: number;
  status_text: string;
  message: string | null;
  updated_at: string;
}

/** 镜像预拉取状态响应。 */
export type ImagePullStatusResponse = PaginatedData<ImagePullStatusItem>;

/** SimEngine 消息。 */
export interface SimEngineMessage {
  type: SimEngineMessageType;
  scene_code?: string;
  tick?: number;
  timestamp?: number;
  payload: JsonObject;
}

/** 通用实时连接状态。 */
export type RealtimeStatus = "idle" | "connecting" | "open" | "closed" | "reconnecting" | "error";
