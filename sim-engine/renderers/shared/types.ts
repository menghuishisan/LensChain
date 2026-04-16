/**
 * JsonPrimitive 定义标准 JSON 原子值。
 */
export type JsonPrimitive = string | number | boolean | null;

/**
 * JsonValue 定义可序列化的 JSON 值。
 */
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[];

/**
 * JsonObject 定义 JSON 对象结构。
 */
export interface JsonObject {
  [key: string]: JsonValue;
}

/**
 * BuiltInSceneCategory 定义平台内置的 8 个领域编码。
 */
export type BuiltInSceneCategory =
  | "node_network"
  | "consensus"
  | "cryptography"
  | "data_structure"
  | "transaction"
  | "smart_contract"
  | "attack_security"
  | "economic";

/**
 * SceneCategory 定义渲染层可识别的领域编码。
 * 除内置 8 个领域外，也允许教师上传自定义领域渲染器。
 */
export type SceneCategory = BuiltInSceneCategory | (string & {});

/**
 * TimeControlMode 定义三种时间控制模式。
 */
export type TimeControlMode = "process" | "reactive" | "continuous";

/**
 * WebSocketMessageType 定义 SimEngine 标准数据通道消息类型。
 */
export type WebSocketMessageType =
  | "state_diff"
  | "state_full"
  | "event"
  | "link_update"
  | "control_ack"
  | "snapshot"
  | "action"
  | "control"
  | "rewind_to"
  | "teacher_summary";

/**
 * TimeControlCommand 定义时间控制命令集合。
 */
export type TimeControlCommand =
  | "play"
  | "pause"
  | "step"
  | "set_speed"
  | "reset"
  | "resume";

/**
 * InteractionFieldType 定义动态交互字段类型。
 */
export type InteractionFieldType =
  | "string"
  | "number"
  | "boolean"
  | "select"
  | "node_ref"
  | "range"
  | "json";

/**
 * InteractionTrigger 定义动态交互触发方式。
 */
export type InteractionTrigger =
  | "click"
  | "form_submit"
  | "drag"
  | "canvas_select";

/**
 * RenderMetric 表示内嵌指标卡片。
 */
export interface RenderMetric {
  key: string;
  label: string;
  value: string;
  tone?: "neutral" | "info" | "success" | "warning" | "danger";
}

/**
 * TooltipEntry 表示悬停详情中的单行信息。
 */
export interface TooltipEntry {
  label: string;
  value: string;
}

/**
 * TimelineEvent 表示仿真事件时间线项。
 */
export interface TimelineEvent {
  id: string;
  tick: number;
  title: string;
  description?: string;
  tone?: "neutral" | "info" | "success" | "warning" | "danger";
}

/**
 * Annotation 表示用户在画布上的标注。
 */
export interface Annotation {
  id: string;
  text: string;
  x: number;
  y: number;
  color?: string;
  createdAt: number;
}

/**
 * InteractionFieldOption 表示选择型字段候选项。
 */
export interface InteractionFieldOption {
  value: string;
  label: string;
}

/**
 * InteractionField 表示场景交互面板字段定义。
 */
export interface InteractionField {
  key: string;
  label: string;
  type: InteractionFieldType;
  required: boolean;
  defaultValue?: JsonValue;
  options?: InteractionFieldOption[];
  validation?: JsonObject;
}

/**
 * InteractionAction 定义一个可执行的交互动作。
 */
export interface InteractionAction {
  actionCode: string;
  label: string;
  description?: string;
  trigger: InteractionTrigger;
  fields: InteractionField[];
  uiSchema?: JsonObject;
}

/**
 * InteractionSchema 定义场景操作面板结构。
 */
export interface InteractionSchema {
  sceneCode: string;
  actions: InteractionAction[];
}

/**
 * InteractionEvent 表示渲染层采集到的用户交互。
 */
export interface InteractionEvent {
  sceneCode: string;
  algorithmType: string;
  type: "click" | "drag" | "hover" | "select" | "submit";
  x?: number;
  y?: number;
  deltaX?: number;
  deltaY?: number;
  targetId?: string;
  payload?: JsonObject;
}

/**
 * SimAction 表示发送给 SimEngine Core 的操作请求。
 */
export interface SimAction {
  sceneCode: string;
  actionCode: string;
  params: JsonObject;
  actorId?: string;
  roleKey?: string;
}

/**
 * InteractionInputValue 定义交互面板字段可提交的值类型。
 */
export type InteractionInputValue = string | number | boolean | JsonObject;

/**
 * InteractionInputMap 定义交互面板字段值映射。
 */
export interface InteractionInputMap {
  [key: string]: InteractionInputValue;
}

/**
 * InteractionValidationIssue 定义交互输入校验结果。
 */
export interface InteractionValidationIssue {
  fieldKey: string;
  message: string;
}

/**
 * PanelLayoutItem 定义单个场景面板在 12 列栅格中的布局。
 */
export interface PanelLayoutItem {
  sceneCode: string;
  x: number;
  y: number;
  w: number;
  h: number;
  fullscreen?: boolean;
}

/**
 * RendererTheme 定义渲染器主题色板。
 */
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

/**
 * RenderConfig 定义算法类型对应的渲染配置。
 */
export interface RenderConfig {
  algorithmType: string;
  title: string;
  subtitle?: string;
  auxiliaryDomains?: SceneCategory[];
  theme: RendererTheme;
}

/**
 * RenderState 定义渲染器消费的标准状态包。
 */
export interface RenderState {
  sceneCode: string;
  title: string;
  category: SceneCategory;
  algorithmType: string;
  timeControlMode: TimeControlMode;
  tick: number;
  totalTicks?: number;
  linked?: boolean;
  linkGroupName?: string;
  renderData: JsonObject;
  metrics?: RenderMetric[];
  tooltip?: TooltipEntry[];
  timeline?: TimelineEvent[];
  changedKeys?: string[];
  annotations?: Annotation[];
  schema?: InteractionSchema;
}

/**
 * SceneSnapshot 定义状态缓存中记录的单次快照。
 */
export interface SceneSnapshot {
  tick: number;
  state: RenderState;
}

/**
 * SceneSummary 定义教师监控缩略视角使用的场景摘要。
 */
export interface SceneSummary {
  sceneCode: string;
  title: string;
  algorithmType: string;
  timeControlMode: TimeControlMode;
  tick: number;
  linked: boolean;
  linkGroupName?: string;
  metricCount: number;
  eventCount: number;
  changedKeyCount: number;
  thumbnail?: string;
}

/**
 * RenderSurface 定义单个场景画布挂载点。
 */
export interface RenderSurface {
  canvas: HTMLCanvasElement;
  overlay?: HTMLElement;
}

/**
 * ViewportState 定义场景视口状态。
 */
export interface ViewportState {
  zoom: number;
  offsetX: number;
  offsetY: number;
}

/**
 * RenderContext 定义渲染执行上下文。
 */
export interface RenderContext {
  surface: RenderSurface;
  width: number;
  height: number;
  viewport: ViewportState;
  now: number;
}

/**
 * ControlDescriptor 定义时间控制按钮描述。
 */
export interface ControlDescriptor {
  command: TimeControlCommand;
  label: string;
  enabled: boolean;
  valueOptions?: number[];
}

/**
 * TeacherSummaryPayload 定义教师监控摘要消息载荷。
 */
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

/**
 * WebSocketMessage 定义前后端数据通道消息结构。
 */
export interface WebSocketMessage {
  type: WebSocketMessageType;
  scene_code?: string;
  tick: number;
  timestamp: number;
  payload: JsonObject;
}

/**
 * CollectorEvent 定义混合实验采集模式的标准化事件载荷。
 */
export interface CollectorEvent {
  source: string;
  timestamp_ms: number;
  data_type: string;
  payload: JsonObject;
}

/**
 * SimPanelOptions 定义仿真面板控制器的启动配置。
 */
export interface SimPanelOptions {
  sessionId: string;
  token: string;
  endpoint: string;
  initialLayout?: PanelLayoutItem[];
  layoutStorageKey?: string;
  actorId?: string;
  roleKey?: string;
}
