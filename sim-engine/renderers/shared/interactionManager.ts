/**
 * interactionManager.ts — 把场景 InteractionSchema 转换为前端表单结构，
 * 并构造 SimAction 发送回后端。
 *
 * 职责：
 *   • mapInteractionDefinition() 把后端 InteractionDefinition (snake_case) → 前端 InteractionSchema (camelCase)。
 *   • validateInputs() 检查必填字段。
 *   • buildSimAction() 构造发送给 WS 的 action payload。
 *
 * 不写兑底：未知 trigger / category / type → 抛错而非静默接受。
 */

import type {
  ActionDef, FieldDef, FieldType,
  InteractionAction, InteractionDefinition, InteractionField, InteractionInputMap,
  InteractionInputValue, InteractionSchema, InteractionValidationIssue,
  JsonObject, JsonValue, SimAction, UserRole,
} from "./types.js";

const VALID_FIELD_TYPES: ReadonlySet<FieldType> = new Set([
  "string", "number", "boolean", "select", "enum", "range", "json", "multi_select",
]);

/** 将后端 InteractionDefinition 映射为前端 InteractionSchema。 */
export function mapInteractionDefinition(def: InteractionDefinition): InteractionSchema {
  return {
    sceneCode: def.scene_code,
    schemaVersion: def.schema_version,
    actions: def.actions.map(mapAction),
  };
}

function mapAction(a: ActionDef): InteractionAction {
  const action: InteractionAction = {
    actionCode: a.action_code,
    label: a.label,
    category: a.category,
    trigger: a.trigger,
    fields: a.fields.map(mapField),
    roles: [...a.roles],
  };
  if (a.description) action.description = a.description;
  if (typeof a.cooldown_ms === "number") action.cooldownMs = a.cooldown_ms;
  if (a.writes_owned_fields) action.writesOwnedFields = [...a.writes_owned_fields];
  if (a.link_owner_fields) action.linkOwnerFields = [...a.link_owner_fields];
  if (a.hybrid_channel) action.hybridChannel = a.hybrid_channel;
  if (a.container_cmd) action.containerCmd = a.container_cmd;
  if (typeof a.reversible === "boolean") action.reversible = a.reversible;
  if (a.intervene_type) action.interveneType = a.intervene_type;
  return action;
}

function mapField(f: FieldDef): InteractionField {
  if (!VALID_FIELD_TYPES.has(f.type)) {
    throw new Error(`mapField: 未知 FieldType "${f.type}" (字段 "${f.name}")`);
  }
  const field: InteractionField = {
    key: f.name,
    type: f.type,
    label: f.label,
  };
  if (f.required !== undefined) field.required = f.required;
  if (f.default !== undefined) field.defaultValue = f.default;
  if (f.options) field.options = mapOptions(f.options);
  return field;
}

function mapOptions(options: readonly JsonValue[]): { label: string; value: string | number }[] {
  return options.map(opt => {
    if (typeof opt === "string" || typeof opt === "number") return { label: String(opt), value: opt };
    if (opt && typeof opt === "object" && !Array.isArray(opt)) {
      const o = opt as JsonObject;
      const label = typeof o.label === "string" ? o.label : String(o.value);
      const value = o.value;
      if (typeof value !== "string" && typeof value !== "number") {
        throw new Error(`mapOptions: option.value 必须是 string|number，得到 ${typeof value}`);
      }
      return { label, value };
    }
    throw new Error(`mapOptions: 无法识别的 option 形态 ${JSON.stringify(opt)}`);
  });
}

/** 校验输入，返回问题列表（空数组 = 校验通过）。 */
export function validateInputs(
  action: InteractionAction,
  inputs: InteractionInputMap,
): InteractionValidationIssue[] {
  const issues: InteractionValidationIssue[] = [];
  for (const f of action.fields) {
    const v = inputs[f.key];
    if (f.required && (v === undefined || v === null || v === "")) {
      issues.push({ fieldKey: f.key, message: `字段 "${f.label}" 必填` });
      continue;
    }
    if (v === undefined) continue;
    if (f.type === "number" || f.type === "range") {
      if (typeof v !== "number" || !Number.isFinite(v)) {
        issues.push({ fieldKey: f.key, message: `字段 "${f.label}" 必须是数字` });
      }
    } else if (f.type === "boolean") {
      if (typeof v !== "boolean") {
        issues.push({ fieldKey: f.key, message: `字段 "${f.label}" 必须是布尔值` });
      }
    }
  }
  return issues;
}

/** 构造 SimAction（前端发起的标准请求结构）。 */
export interface BuildSimActionInput {
  sceneCode: string;
  action: InteractionAction;
  inputs: InteractionInputMap;
  actorId?: string;
  roleKey?: string;
  userRole?: UserRole;
}

export function buildSimAction(input: BuildSimActionInput): SimAction {
  const issues = validateInputs(input.action, input.inputs);
  if (issues.length > 0) {
    throw new Error(
      `buildSimAction: 输入校验失败 ${issues.map(i => `[${i.fieldKey}] ${i.message}`).join("; ")}`,
    );
  }
  const params: JsonObject = {};
  for (const f of input.action.fields) {
    const v: InteractionInputValue | undefined = input.inputs[f.key];
    if (v === undefined) continue;
    params[f.key] = v as JsonValue;
  }
  const action: SimAction = {
    sceneCode: input.sceneCode,
    actionCode: input.action.actionCode,
    params,
  };
  if (input.actorId) action.actorId = input.actorId;
  if (input.roleKey) action.roleKey = input.roleKey;
  if (input.userRole) action.userRole = input.userRole;
  return action;
}
