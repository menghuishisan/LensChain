// auth-validation.ts
// 模块01表单校验工具，集中处理登录、改密、用户导入等前端基础校验。

const PHONE_REGEXP = /^1\d{10}$/;
const EMAIL_REGEXP = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const MAX_IMPORT_FILE_SIZE = 50 * 1024 * 1024;
const IMPORT_EXTENSIONS = [".xlsx", ".csv"] as const;

/**
 * 校验结果。
 */
export interface ValidationResult<TFields extends string> {
  isValid: boolean;
  errors: Partial<Record<TFields, string>>;
}

/**
 * 登录表单值。
 */
export interface LoginFormValues {
  phone: string;
  password: string;
}

/**
 * 改密表单值。
 */
export interface ChangePasswordFormValues {
  oldPassword?: string;
  newPassword: string;
  confirmPassword: string;
}

/**
 * 密码规则展示项。
 */
export interface PasswordRule {
  key: string;
  label: string;
  isPassed: boolean;
}

/**
 * 获取密码复杂度规则。
 */
export function getPasswordRules(password: string): PasswordRule[] {
  return [
    { key: "length", label: "至少8位字符", isPassed: password.length >= 8 },
    { key: "uppercase", label: "包含大写字母", isPassed: /[A-Z]/.test(password) },
    { key: "lowercase", label: "包含小写字母", isPassed: /[a-z]/.test(password) },
    { key: "digit", label: "包含数字", isPassed: /\d/.test(password) },
  ];
}

/**
 * 判断密码是否满足模块01默认复杂度。
 */
export function isPasswordStrong(password: string) {
  return getPasswordRules(password).every((rule) => rule.isPassed);
}

/**
 * 校验登录表单。
 */
export function validateLoginForm(values: LoginFormValues): ValidationResult<"phone" | "password"> {
  const errors: ValidationResult<"phone" | "password">["errors"] = {};

  if (!PHONE_REGEXP.test(values.phone.trim())) {
    errors.phone = "请输入11位中国大陆手机号";
  }

  if (values.password.length === 0) {
    errors.password = "请输入密码";
  }

  return { isValid: Object.keys(errors).length === 0, errors };
}

/**
 * 校验改密表单；传入 oldPassword 时会校验新旧密码不能相同。
 */
export function validateChangePasswordForm(
  values: ChangePasswordFormValues,
): ValidationResult<"oldPassword" | "newPassword" | "confirmPassword"> {
  const errors: ValidationResult<"oldPassword" | "newPassword" | "confirmPassword">["errors"] = {};

  if (values.oldPassword !== undefined && values.oldPassword.length === 0) {
    errors.oldPassword = "请输入旧密码";
  }

  if (!isPasswordStrong(values.newPassword)) {
    errors.newPassword = "新密码不符合复杂度要求";
  }

  if (values.oldPassword !== undefined && values.oldPassword === values.newPassword) {
    errors.newPassword = "新密码不能与当前密码相同";
  }

  if (values.confirmPassword !== values.newPassword) {
    errors.confirmPassword = "两次输入的新密码不一致";
  }

  return { isValid: Object.keys(errors).length === 0, errors };
}

/**
 * 校验邮箱格式。
 */
export function validateOptionalEmail(email: string) {
  const trimmedEmail = email.trim();
  return trimmedEmail.length === 0 || EMAIL_REGEXP.test(trimmedEmail);
}

/**
 * 校验用户导入文件。
 */
export function validateUserImportFile(file: File | null): ValidationResult<"file"> {
  const errors: ValidationResult<"file">["errors"] = {};

  if (file === null) {
    errors.file = "请选择导入文件";
    return { isValid: false, errors };
  }

  const lowerName = file.name.toLowerCase();
  const hasValidExtension = IMPORT_EXTENSIONS.some((extension) => lowerName.endsWith(extension));
  if (!hasValidExtension) {
    errors.file = "仅支持.xlsx和.csv格式";
  } else if (file.size === 0) {
    errors.file = "文件内容为空";
  } else if (file.size > MAX_IMPORT_FILE_SIZE) {
    errors.file = "文件大小不能超过50MB";
  }

  return { isValid: Object.keys(errors).length === 0, errors };
}
