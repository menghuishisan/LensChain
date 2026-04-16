import type {
  InteractionAction,
  InteractionField,
  InteractionInputMap,
  InteractionInputValue,
  InteractionSchema,
  InteractionValidationIssue,
  JsonObject,
  JsonValue,
  SimAction
} from "./types.js";
import { asNumber, asObject, asString } from "./utils.js";

/**
 * InteractionPanelModel 为场景专属交互面板提供默认值、校验和提交载荷构造能力。
 */
export class InteractionPanelModel {
  /**
   * listActions 返回 schema 中声明的全部交互动作。
   */
  public listActions(schema: InteractionSchema | undefined): InteractionAction[] {
    return schema?.actions ?? [];
  }

  /**
   * createDefaults 根据动作字段定义生成默认输入值。
   */
  public createDefaults(action: InteractionAction): InteractionInputMap {
    const defaults: InteractionInputMap = {};
    for (const field of action.fields) {
      const value = this.resolveDefaultValue(field);
      if (value !== undefined) {
        defaults[field.key] = value;
      }
    }
    return defaults;
  }

  /**
   * validate 校验动作输入是否满足字段定义。
   */
  public validate(action: InteractionAction, input: InteractionInputMap): InteractionValidationIssue[] {
    const issues: InteractionValidationIssue[] = [];
    for (const field of action.fields) {
      const value = input[field.key];
      if (value === undefined || value === "") {
        if (field.required) {
          issues.push({ fieldKey: field.key, message: `${field.label}不能为空` });
        }
        continue;
      }
      issues.push(...this.validateField(field, value));
    }
    return issues;
  }

  /**
   * buildAction 将交互面板输入转换为标准操作请求。
   */
  public buildAction(sceneCode: string, action: InteractionAction, input: InteractionInputMap): SimAction {
    const issues = this.validate(action, input);
    if (issues.length > 0) {
      throw new Error(issues.map((issue) => issue.message).join("；"));
    }
    return {
      sceneCode,
      actionCode: action.actionCode,
      params: this.normalizeInput(input)
    };
  }

  /**
   * resolveDefaultValue 生成单个字段的默认值。
   */
  private resolveDefaultValue(field: InteractionField): InteractionInputValue | undefined {
    if (field.defaultValue !== undefined) {
      return this.normalizeValue(field.type, field.defaultValue);
    }
    if (field.type === "boolean") {
      return false;
    }
    if (field.type === "number" || field.type === "range") {
      return 0;
    }
    return undefined;
  }

  /**
   * validateField 按字段类型和校验规则校验输入值。
   */
  private validateField(field: InteractionField, value: InteractionInputValue): InteractionValidationIssue[] {
    const issues: InteractionValidationIssue[] = [];
    const validation = asObject(field.validation);
    if ((field.type === "number" || field.type === "range") && typeof value !== "number") {
      issues.push({ fieldKey: field.key, message: `${field.label}必须为数值` });
      return issues;
    }
    if (field.type === "boolean" && typeof value !== "boolean") {
      issues.push({ fieldKey: field.key, message: `${field.label}必须为布尔值` });
      return issues;
    }
    if (field.type === "json" && typeof value !== "object") {
      issues.push({ fieldKey: field.key, message: `${field.label}必须为 JSON 对象` });
      return issues;
    }
    if (field.type === "select" && field.options && !field.options.some((item) => item.value === value)) {
      issues.push({ fieldKey: field.key, message: `${field.label}取值不在候选项中` });
    }
    if ((field.type === "number" || field.type === "range") && typeof value === "number") {
      const min = validation.min;
      const max = validation.max;
      if (typeof min === "number" && value < min) {
        issues.push({ fieldKey: field.key, message: `${field.label}不能小于${min}` });
      }
      if (typeof max === "number" && value > max) {
        issues.push({ fieldKey: field.key, message: `${field.label}不能大于${max}` });
      }
    }
    return issues;
  }

  /**
   * normalizeInput 将字段值统一转换为可序列化对象。
   */
  private normalizeInput(input: InteractionInputMap): JsonObject {
    const payload: JsonObject = {};
    for (const [key, value] of Object.entries(input)) {
      payload[key] = this.toJsonValue(value);
    }
    return payload;
  }

  /**
   * normalizeValue 根据字段类型整理默认值。
   */
  private normalizeValue(type: InteractionField["type"], value: JsonValue): InteractionInputValue {
    switch (type) {
      case "number":
      case "range":
        return asNumber(value, 0);
      case "boolean":
        return Boolean(value);
      case "json":
        return asObject(value);
      default:
        return asString(value, "");
    }
  }

  /**
   * toJsonValue 将交互值转换为标准 JSON 值。
   */
  private toJsonValue(value: InteractionInputValue): JsonValue {
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      return value;
    }
    return value;
  }
}
