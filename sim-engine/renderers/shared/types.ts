/**
 * 模块：sim-engine/renderers/shared
 * 文件职责：SimEngine 渲染层协议契约 TypeScript 类型定义。
 * 协议依据：docs/modules/04-实验环境/06-可视化仿真引擎设计.md §3.2 / §3.3 / §6.2 / §6.3 / §七。
 *
 * 设计要点：
 * 1. 本文件类型与 scenarios/internal/framework/types.go、proto/lenschain/sim_scenario/v1/sim_scenario.proto
 *    三方 1:1 对齐；任意一方变更必须三端同步。
 * 2. JSON 字段统一 snake_case；雪花 ID 一律 string；时间统一 Unix 毫秒时间戳。
 * 3. 渲染层仅消费 RenderEnvelope；不持有业务状态，不发起 HTTP。
 */

// ============================================================
// 通用 JSON 类型
// ============================================================

/** JSON 原子值。 */
export type JsonPrimitive = string | number | boolean | null;

/** 可序列化 JSON 值。 */
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[];

/** JSON 对象。 */
export interface JsonObject {
  [key: string]: JsonValue;
}

// ============================================================
// 47 原语类型 / Layer / 时间控制 / 类目 / 角色 等枚举
// ============================================================

/** 47 个原语 type 值（详 06.md §3.2）。 */
export type PrimitiveType =
  // 几何类（8）
  | "node" | "edge" | "bar" | "curve" | "polygon" | "area" | "grid_cell" | "ring"
  // 动效类（7）
  | "particle_stream" | "burst" | "pulse" | "trail" | "glow" | "shake" | "shift_animation"
  // 布局类（6）
  | "horizontal_lane" | "stack" | "ring_layout" | "tree_layout" | "graph_layout" | "matrix_layout"
  // 数据展示类（7）
  | "label" | "tooltip" | "annotation" | "register_row" | "math_pipeline" | "code_block" | "math_formula"
  // 状态指示类（8）
  | "phase_progress" | "progress_bar" | "target_zone" | "link_indicator"
  | "external_event_marker" | "error_overlay" | "verify_path_highlight" | "risk_gauge"
  // 领域复合类（11）
  | "vote_matrix" | "dual_track" | "time_wheel" | "pie_chart" | "sankey_flow" | "heat_map"
  | "mempool_slot" | "bridge_track" | "code_marker" | "partition_zone" | "curve_point";

/** 原语分层（详 06.md §3.3）。 */
export type PrimitiveLayer = "background" | "content" | "effect" | "overlay";

/** 三种时间控制模式（详 06.md §五）。 */
export type TimeControlMode = "process" | "reactive" | "continuous";

/** 场景数据源模式。 */
export type DataSourceMode = "simulation" | "collection" | "dual";

/** 内置 8 类目编码（详 06.md §3.5）。 */
export type BuiltInSceneCategory =
  | "node_network" | "consensus" | "cryptography" | "data_structure"
  | "transaction" | "smart_contract" | "attack_security" | "economic";

/** 场景类目；包括内置 8 类与教师 fallback 的 generic。 */
export type SceneCategory = BuiltInSceneCategory | "generic";

/** ActionDef 业务语义分类（详 06.md §6.3）。 */
export type ActionCategory = "param_tune" | "attack_inject" | "primary" | "observe";

/** ActionDef 触发方式（详 06.md §6.3）。 */
export type ActionTrigger = "submit" | "immediate" | "hold";

/** FieldDef 字段类型。 */
export type FieldType =
  | "string" | "number" | "boolean" | "select"
  | "enum" | "range" | "json" | "multi_select";

/** 混合实验下 ActionDef 执行通道。 */
export type HybridChannel = "sim" | "container";

/** 调用 ActionDef 时的用户角色。 */
export type UserRole = "student" | "teacher";

/** 速率倍率四档（详 06.md §3.10.2）。 */
export type SpeedMultiplier = 0.5 | 1 | 1.5 | 2;

// ============================================================
// 协议数据结构（与 06.md §6.2 / §6.3 1:1 对齐）
// ============================================================

/**
 * Primitive 单个原语描述。
 * params 字段需按 §3.2 各原语 schema 提供，渲染基类按字段名取值。
 */
export interface Primitive {
  id: string;
  type: PrimitiveType;
  layer: PrimitiveLayer;
  params: JsonObject;
  clickable?: boolean;
  hover_tooltip?: string;
  click_action?: string; // 绑定 ActionDef.action_code
}

/**
 * MicroStep 微动画步骤（详 06.md §6.2 / §5.3）。
 * duration_ms 必须 ≥ 本步骤内 fire_primitives 最长动画时长 + 200ms buffer。
 */
export interface MicroStep {
  id: string;
  label: string;
  duration_ms: number;
  highlight_ids?: string[];
  fire_primitives?: string[];
  is_link_trigger?: boolean;
  link_source?: string;
  parent_phase?: string;
}

/**
 * LinkTrigger 跨场景联动事件（详 06.md §6.2 / §8.5.2 / §9.4）。
 * source_anchor_id / target_anchor_id 用于 M8 跨画布弧线起终点定位。
 */
export interface LinkTrigger {
  id: string;
  source_scene: string;
  source_action: string;
  link_group: string;
  changed_fields: string[];
  payload: JsonObject;
  ts: number; // Unix 毫秒时间戳
  source_anchor_id?: string;
  target_anchor_id?: string;
}

/**
 * ContainerMetric 混合实验容器采集数据（详 06.md §6.2 / §十）。
 */
export interface ContainerMetric {
  source_container: string;
  metric_key: string;
  value: JsonValue;
  ts: number;
  target_primitive?: string;
  target_param?: string;
}

/**
 * RenderEnvelope 场景输出给前端的标准渲染载荷。
 * 替代旧 {nodes, messages, data} 模型。
 */
export interface RenderEnvelope {
  primitives: Primitive[];
  micro_steps?: MicroStep[];
  link_triggers?: LinkTrigger[];
  container_data?: ContainerMetric[];
  changed_keys?: string[];
  is_full_snapshot?: boolean;
  /** Data 仅承载侧栏指标 / 文字面板用数据，非原语参数。 */
  data?: JsonObject;
}

// ============================================================
// 交互定义（详 06.md §6.3）
// ============================================================

/** FieldDef 单个输入字段。 */
export interface FieldDef {
  name: string;
  type: FieldType;
  label: string;
  required?: boolean;
  default?: JsonValue;
  min?: JsonValue;
  max?: JsonValue;
  step?: JsonValue;
  options?: JsonValue[];
  options_from?: string; // 动态候选项来源路径（如 "state.nodes"）
}

/** ActionDef 单个学生 / 教师可执行交互动作。 */
export interface ActionDef {
  action_code: string;
  label: string;
  description?: string;
  category: ActionCategory;
  trigger: ActionTrigger;
  fields: FieldDef[];
  roles: UserRole[];
  cooldown_ms?: number;
  link_owner_fields?: string[];
  hybrid_channel?: HybridChannel;
  container_cmd?: string;
  writes_owned_fields?: string[];
  reversible?: boolean;
  intervene_type?: string;
}

/** InteractionDefinition 场景对外暴露的全部 ActionDef。 */
export interface InteractionDefinition {
  scene_code: string;
  schema_version: string;
  actions: ActionDef[];
}

// ============================================================
// WebSocket 协议（详 06.md §七 与 03-API §四）
// ============================================================

/** 后端 → 前端消息类型。 */
export type ServerMessageType =
  | "render"              // RenderEnvelope 推送
  | "event"               // 仿真事件通知
  | "control_ack"         // 控制指令确认
  | "schema_invalidated"; // ActionDef schema 失效通知

/** 前端 → 后端消息类型。 */
export type ClientMessageType =
  | "action"     // 用户交互操作（ActionDef 调用）
  | "control"    // 仿真控制（play/pause/step/set_speed/reset/resume）
  | "step_back"; // 单步回退（仅单场景 process 模式有效）

/** 全部 WebSocket 消息类型。 */
export type WebSocketMessageType = ServerMessageType | ClientMessageType;

/** 时间控制命令（详 06.md §7.4）。 */
export type TimeControlCommand =
  | "play" | "pause" | "step" | "set_speed" | "reset" | "resume";

/** WebSocket 消息通用包络。 */
export interface WebSocketMessage<P = JsonObject> {
  type: WebSocketMessageType;
  scene_code?: string;
  tick: number;
  timestamp: number;
  payload: P;
}

/** action 消息载荷。 */
export interface ActionMessagePayload {
  action_code: string;
  params: JsonObject;
  actor_id?: string;
  user_role?: UserRole;
}

/** control 消息载荷。 */
export interface ControlMessagePayload {
  command: TimeControlCommand;
  value?: number; // set_speed 时的速率值（0.5 / 1 / 1.5 / 2）
}

/** event 消息载荷。 */
export interface EventMessagePayload {
  event: string;
  data: JsonObject;
}

/** control_ack 消息载荷。 */
export interface ControlAckPayload {
  command: string;
  success: boolean;
  error?: string;
}

/** schema_invalidated 消息载荷。 */
export interface SchemaInvalidatedPayload {
  reason: "teacher_published" | string;
}

// ============================================================
// 渲染层运行时类型（仅前端使用，不上协议）
// ============================================================

/** 单场景画布挂载点。 */
export interface RenderSurface {
  canvas: HTMLCanvasElement;
  overlay?: HTMLElement;
}

/** 场景视口状态。 */
export interface ViewportState {
  zoom: number;
  offsetX: number;
  offsetY: number;
}

/** 渲染器主题色板。 */
export interface RendererTheme {
  background: string;
  foreground: string;
  accent: string;
  success: string;
  warning: string;
  danger: string;
  muted: string;
  grid: string;
}

/** 渲染执行上下文。 */
export interface RenderContext {
  surface: RenderSurface;
  width: number;
  height: number;
  viewport: ViewportState;
  /** 当前帧时间（高精度 ms）。 */
  now: number;
  /** 当前速率倍率（仅作用于 tick 推进与 MicroStep 间隔，详 §3.10.2）。 */
  speed: SpeedMultiplier;
  /** 主题色（按场景 category 选择）。 */
  theme: RendererTheme;
}

/** 渲染器配置（领域渲染器返回给基类用于 beginFrame 等）。 */
export interface RenderConfig {
  title: string;
  subtitle?: string;
  theme: RendererTheme;
}

/** 4 种布局模式（详 06.2 §三）。 */
export type LayoutMode = "grid" | "focus" | "carousel" | "multi_compare";

/** 布局角色（详 06.md §9.2 / 02 §2.11）。 */
export type LayoutRole = "primary" | "secondary" | "auxiliary";

/** 单个场景面板在 12 列栅格中的布局描述。 */
export interface PanelLayoutItem {
  sceneCode: string;
  x: number;
  y: number;
  w: number;
  h: number;
  layoutRole?: LayoutRole;
  fullscreen?: boolean;
}

// ============================================================
// 渲染状态（RenderState）— 前端全量场景状态
// ============================================================

/** 指标卡片数据。 */
export interface RenderMetric {
  label: string;
  value: string;
  tone?: "info" | "success" | "warning" | "danger";
}

/** 时间线事件条目。 */
export interface TimelineEvent {
  id: string;
  tick: number;
  title: string;
  description?: string;
  tone?: "neutral" | "info" | "success" | "warning" | "danger";
}

/** 画布文本标注。 */
export interface Annotation {
  id: string;
  x: number;
  y: number;
  text: string;
  color?: string;
  createdAt?: number;
}

/**
 * RenderState 单个场景在前端的完整渲染状态。
 * 由 StateCache 通过累积 WebSocket 消息构造。
 */
export interface RenderState {
  sceneCode: string;
  title: string;
  category: SceneCategory;
  timeControlMode: TimeControlMode;
  tick: number;
  totalTicks?: number;

  /** 核心渲染载荷 — 原语 + 微步骤。 */
  envelope: RenderEnvelope;
  /** 当前活跃的微步骤 ID（由 MicroStepScheduler 驱动）。 */
  activeMicroStepId?: string;

  /** 侧栏指标卡片。 */
  metrics?: RenderMetric[];
  /** 悬停提示。 */
  tooltip?: Array<{ label: string; value: string }>;
  /** 时间线事件。 */
  timeline?: TimelineEvent[];
  /** 本帧变化字段路径列表。 */
  changedKeys?: string[];
  /** 画布标注。 */
  annotations?: Annotation[];

  /** 是否处于联动状态。 */
  linked?: boolean;
  /** 联动组名称。 */
  linkGroupName?: string;

  /** 场景交互定义（schema）。 */
  schema?: InteractionSchema;
}

/** 状态历史快照。 */
export interface SceneSnapshot {
  tick: number;
  state: RenderState;
}

// ============================================================
// 交互面板运行时类型
// ============================================================

/** 交互字段选项。 */
export interface InteractionFieldOption {
  label: string;
  value: string | number;
}

/** 交互字段定义（前端面板用，从 ActionDef.fields 映射）。 */
export interface InteractionField {
  key: string;
  type: FieldType;
  label: string;
  required?: boolean;
  defaultValue?: JsonValue;
  options?: InteractionFieldOption[];
  validation?: JsonObject;
}

/** 交互动作定义（前端面板用，从 ActionDef 映射）。 */
export interface InteractionAction {
  actionCode: string;
  label: string;
  description?: string;
  category: ActionCategory;
  trigger: ActionTrigger;
  fields: InteractionField[];
  roles: UserRole[];
  cooldownMs?: number;
}

/** 场景交互 schema（前端缓存用，从 InteractionDefinition 映射）。 */
export interface InteractionSchema {
  sceneCode: string;
  schemaVersion: string;
  actions: InteractionAction[];
}

/** 交互面板输入值。 */
export type InteractionInputValue = string | number | boolean | JsonObject;

/** 交互面板输入表。 */
export type InteractionInputMap = Record<string, InteractionInputValue>;

/** 交互校验问题。 */
export interface InteractionValidationIssue {
  fieldKey: string;
  message: string;
}

// ============================================================
// 操作与事件
// ============================================================

/** 前端构造的标准操作请求。 */
export interface SimAction {
  sceneCode: string;
  actionCode: string;
  params: JsonObject;
  actorId?: string;
  roleKey?: string;
  userRole?: UserRole;
}

/** 交互事件（浏览器事件标准化后）。 */
export interface InteractionEvent {
  sceneCode: string;
  type: "click" | "hover" | "drag";
  x?: number;
  y?: number;
  deltaX?: number;
  deltaY?: number;
  targetId?: string;
}

/** 时间线控件描述。 */
export interface ControlDescriptor {
  command: TimeControlCommand;
  label: string;
  enabled: boolean;
  valueOptions?: readonly number[];
}

/** 场景摘要（用于面板列表 / 教师概览）。 */
export interface SceneSummary {
  sceneCode: string;
  title: string;
  timeControlMode: TimeControlMode;
  tick: number;
  linked: boolean;
  metricCount: number;
  eventCount: number;
  changedKeyCount: number;
  linkGroupName?: string;
  thumbnail?: string;
}

// ============================================================
// 教师监控 & SimPanel 配置
// ============================================================

/** 教师监控摘要消息载荷（内部 camelCase）。 */
export interface TeacherSummaryPayload {
  sessionId: string;
  instanceId: string;
  tick: number;
  speed: number;
  activeSceneCodes: string[];
  linkGroupCodes: string[];
  collectionRunning: boolean;
  previewSceneCode?: string;
  previewState?: JsonObject;
  lastAction?: string;
  updatedAt: number;
}

/** SimPanel 启动配置。 */
export interface SimPanelOptions {
  sessionId: string;
  /**
   * urlProvider 在每次 WS 拨号 / 重连前被调用，应返回带未过期 token 的完整 ws:// URL。
   * 由上层（useSimPanel）实现：通常先走 ApiClient.ensureFreshAccessToken() 拿活 token，
   * 再拼 `${endpoint}/${sessionId}?token=${token}`。
   *
   * 这样 access_token 过期不会导致 WS 死循环 401——重连自带 refresh。
   */
  urlProvider: () => string | Promise<string>;
  initialLayout?: PanelLayoutItem[];
  layoutStorageKey?: string;
  actorId?: string;
  roleKey?: string;
  userRole?: UserRole;
}

/** 动效原语默认时长（详 06.md §3.10.1）。 */
export const DEFAULT_ANIMATION_DURATIONS = {
  burst_duration_ms: 600,
  pulse_period_ms: 1200,
  trail_duration_ms: 1500,
  trail_fade_ms: 500,
  shake_duration_ms: 400,
  shift_duration_ms: 500,
  particle_lifetime_ms: 1200,
  particle_rate: 20,
  link_trigger_micro_step_min_ms: 600,
  link_trigger_micro_step_max_ms: 1000,
  cross_canvas_overlay_fade_ms: 1500,
} as const;
