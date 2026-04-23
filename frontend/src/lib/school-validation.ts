// school-validation.ts
// 模块02学校与租户管理表单校验工具。

import type { SchoolApplicationRequest, UpdateSchoolSsoConfigRequest } from "@/types/school";

const PHONE_REGEXP = /^1\d{10}$/;
const URL_REGEXP = /^https?:\/\/.+/i;

/**
 * 校验结果。
 */
export interface SchoolValidationResult<TField extends string> {
  isValid: boolean;
  errors: Partial<Record<TField, string>>;
}

/**
 * 申请查询表单值。
 */
export interface SchoolQueryFormValues {
  phone: string;
  sms_code: string;
}

/**
 * 校验学校入驻申请表单。
 */
export function validateSchoolApplicationForm(
  values: Partial<SchoolApplicationRequest>,
): SchoolValidationResult<"school_name" | "school_code" | "school_website" | "contact_name" | "contact_phone" | "contact_email"> {
  const errors: SchoolValidationResult<
    "school_name" | "school_code" | "school_website" | "contact_name" | "contact_phone" | "contact_email"
  >["errors"] = {};

  if (!values.school_name?.trim()) {
    errors.school_name = "学校名称不能为空";
  }
  if (!values.school_code?.trim()) {
    errors.school_code = "学校编码不能为空";
  }
  if (values.school_website && !URL_REGEXP.test(values.school_website)) {
    errors.school_website = "学校官网必须以 http:// 或 https:// 开头";
  }
  if (!values.contact_name?.trim()) {
    errors.contact_name = "联系人姓名不能为空";
  }
  if (!PHONE_REGEXP.test(values.contact_phone ?? "")) {
    errors.contact_phone = "请输入11位中国大陆手机号";
  }
  if (values.contact_email && !values.contact_email.includes("@")) {
    errors.contact_email = "联系人邮箱格式不正确";
  }

  return { isValid: Object.keys(errors).length === 0, errors };
}

/**
 * 校验申请状态查询表单。
 */
export function validateSchoolQueryForm(values: SchoolQueryFormValues): SchoolValidationResult<"phone" | "sms_code"> {
  const errors: SchoolValidationResult<"phone" | "sms_code">["errors"] = {};

  if (!PHONE_REGEXP.test(values.phone)) {
    errors.phone = "请输入11位中国大陆手机号";
  }
  if (!values.sms_code.trim()) {
    errors.sms_code = "请输入短信验证码";
  }

  return { isValid: Object.keys(errors).length === 0, errors };
}

/**
 * 校验 SSO 配置表单。
 */
export function validateSsoConfigForm(
  values: UpdateSchoolSsoConfigRequest,
): SchoolValidationResult<
  | "cas_server_url"
  | "cas_service_url"
  | "authorize_url"
  | "token_url"
  | "userinfo_url"
  | "client_id"
  | "client_secret"
  | "redirect_uri"
  | "user_id_attribute"
> {
  const errors: SchoolValidationResult<
    | "cas_server_url"
    | "cas_service_url"
    | "authorize_url"
    | "token_url"
    | "userinfo_url"
    | "client_id"
    | "client_secret"
    | "redirect_uri"
    | "user_id_attribute"
  >["errors"] = {};

  if (values.provider === "cas") {
    const config = values.config;
    if (!("cas_server_url" in config) || !config.cas_server_url?.trim()) {
      errors.cas_server_url = "请输入CAS服务端URL";
    }
    if (!("cas_service_url" in config) || !config.cas_service_url?.trim()) {
      errors.cas_service_url = "请输入CAS回调URL";
    }
    if (!("user_id_attribute" in config) || !config.user_id_attribute?.trim()) {
      errors.user_id_attribute = "请输入用户ID属性";
    }
    return { isValid: Object.keys(errors).length === 0, errors };
  }

  const config = values.config;
  if (!("authorize_url" in config) || !config.authorize_url?.trim()) {
    errors.authorize_url = "请输入授权端点";
  }
  if (!("token_url" in config) || !config.token_url?.trim()) {
    errors.token_url = "请输入Token端点";
  }
  if (!("userinfo_url" in config) || !config.userinfo_url?.trim()) {
    errors.userinfo_url = "请输入用户信息端点";
  }
  if (!("client_id" in config) || !config.client_id?.trim()) {
    errors.client_id = "请输入Client ID";
  }
  // 后端返回 client_secret=****** 时代表已有密钥；保存原配置时不要求明文回填。
  if (!("client_secret" in config) || !config.client_secret?.trim()) {
    errors.client_secret = "请输入Client Secret";
  }
  if (!("redirect_uri" in config) || !config.redirect_uri?.trim()) {
    errors.redirect_uri = "请输入回调URL";
  }
  if (!("user_id_attribute" in config) || !config.user_id_attribute?.trim()) {
    errors.user_id_attribute = "请输入用户ID属性";
  }

  return { isValid: Object.keys(errors).length === 0, errors };
}
