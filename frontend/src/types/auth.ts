// auth.ts
// 模块01：用户与认证类型定义，严格对齐模块01 API 文档。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/**
 * 平台用户角色。
 */
export type UserRole = "student" | "teacher" | "school_admin" | "super_admin";

/**
 * 可用角色列表。
 */
export const USER_ROLES: readonly UserRole[] = ["student", "teacher", "school_admin", "super_admin"];

/**
 * 当前登录用户基础信息。
 */
export interface AuthUser {
  id: ID;
  name: string;
  phone?: string;
  roles: UserRole[];
  school_id?: ID | null;
  school_name?: string | null;
  avatar_url?: string | null;
  is_first_login?: boolean;
}

/**
 * 登录请求体。
 */
export interface LoginRequest {
  phone: string;
  password: string;
}

/**
 * 后端 Token 响应字段。
 */
export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

/**
 * 正常登录响应 data。
 */
export interface LoginSuccessResponse extends TokenResponse {
  user: AuthUser;
}

/**
 * 首次登录需强制改密响应 data。
 */
export interface ForceChangeRequiredResponse {
  force_change_password: true;
  temp_token: string;
  temp_token_expires_in: number;
}

/**
 * 登录响应 data 联合类型。
 */
export type LoginResponse = LoginSuccessResponse | ForceChangeRequiredResponse;

/**
 * 强制改密请求参数。
 */
export interface ForceChangePasswordRequest {
  tempToken: string;
  newPassword: string;
}

/**
 * 修改密码请求体。
 */
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

/**
 * 登录或刷新后返回的 Token 信息。
 */
export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
}

/**
 * 本地认证会话快照。
 */
export interface AuthSession {
  accessToken: string | null;
  refreshToken: string | null;
  tokenType: string;
  expiresAt: number | null;
  user: AuthUser | null;
  tempToken: string | null;
  tempTokenExpiresAt: number | null;
}

/**
 * SSO 学校信息。
 */
export interface SsoSchool {
  id: ID;
  name: string;
  logo_url: string | null;
}

/**
 * SSO 学校列表响应 data。
 */
export interface SsoSchoolListResponse {
  list: SsoSchool[];
}

/**
 * 用户账号状态。
 */
export type UserStatus = 1 | 2 | 3;

/**
 * 用户创建/编辑角色字段。
 */
export type ManageableUserRole = "teacher" | "student";

/**
 * 用户列表查询参数。
 */
export interface UserListParams extends QueryParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: UserStatus;
  role?: "teacher" | "student";
  college?: string;
  education_level?: number;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

/**
 * 用户列表项。
 */
export interface UserListItem {
  id: ID;
  phone: string;
  name: string;
  student_no: string | null;
  status: UserStatus;
  status_text: string;
  roles: UserRole[];
  college: string | null;
  major: string | null;
  class_name: string | null;
  education_level: number | null;
  education_level_text: string | null;
  last_login_at: string | null;
  created_at: string;
}

/**
 * 用户详情。
 */
export interface UserDetail extends UserListItem {
  email?: string | null;
  remark?: string | null;
  school_id?: ID | null;
  school_name?: string | null;
  is_first_login?: boolean;
  is_school_admin?: boolean;
  avatar_url?: string | null;
  nickname?: string | null;
  enrollment_year?: number | null;
  grade?: number | null;
}

/**
 * 创建用户请求体。
 */
export interface CreateUserRequest {
  phone: string;
  name: string;
  password: string;
  role: ManageableUserRole;
  student_no?: string | null;
  college?: string | null;
  major?: string | null;
  class_name?: string | null;
  education_level?: number | null;
  email?: string | null;
  remark?: string | null;
}

/**
 * 创建超级管理员请求体。
 */
export interface CreateSuperAdminRequest {
  phone: string;
  name: string;
  password: string;
  school_id?: ID | null;
  email?: string | null;
  remark?: string | null;
}

/**
 * 更新用户请求体。
 */
export interface UpdateUserRequest {
  name?: string;
  student_no?: string | null;
  college?: string | null;
  major?: string | null;
  class_name?: string | null;
  enrollment_year?: number | null;
  education_level?: number | null;
  grade?: number | null;
  email?: string | null;
  remark?: string | null;
}

/**
 * 变更账号状态请求体。
 */
export interface UpdateUserStatusRequest {
  status: UserStatus;
  reason: string;
}

/**
 * 重置密码请求体。
 */
export interface ResetUserPasswordRequest {
  new_password: string;
}

/**
 * 批量删除用户请求体。
 */
export interface BatchDeleteUsersRequest {
  ids: ID[];
}

/**
 * 用户导入类型。
 */
export type UserImportType = "student" | "teacher";

/**
 * 用户导入预览请求参数。
 */
export interface UserImportPreviewRequest {
  file: File;
  type: UserImportType;
}

/**
 * 导入预览状态。
 */
export type UserImportPreviewStatus = "valid" | "invalid" | "conflict";

/**
 * 用户导入预览行。
 */
export interface UserImportPreviewRow {
  row: number;
  name: string;
  phone: string;
  student_no: string;
  status: UserImportPreviewStatus;
  message: string | null;
}

/**
 * 用户导入预览响应 data。
 */
export interface UserImportPreviewResponse {
  import_id: string;
  total: number;
  valid: number;
  invalid: number;
  conflict: number;
  preview_list: UserImportPreviewRow[];
}

/**
 * 导入冲突处理策略。
 */
export type UserImportConflictStrategy = "skip" | "overwrite";

/**
 * 执行导入请求体。
 */
export interface ExecuteUserImportRequest {
  import_id: string;
  conflict_strategy: UserImportConflictStrategy;
  conflict_overrides?: string[];
}

/**
 * 执行导入响应 data。
 */
export interface ExecuteUserImportResponse {
  import_id: string;
  success_count: number;
  fail_count: number;
  skip_count: number;
  overwrite_count: number;
}

/**
 * 个人中心基础资料。
 */
export interface Profile {
  id: ID;
  phone: string;
  name: string;
  nickname: string | null;
  avatar_url: string | null;
  email: string | null;
  student_no: string | null;
  school_name: string | null;
  college: string | null;
  major: string | null;
  class_name: string | null;
  education_level: number | null;
  education_level_text: string | null;
  roles: UserRole[];
}

/**
 * 更新个人资料请求体。
 */
export interface UpdateProfileRequest {
  nickname: string;
  avatar_url: string;
  email: string;
}

/**
 * 安全策略配置。
 */
export interface SecurityPolicy {
  login_fail_max_count: number;
  login_lock_duration_minutes: number;
  password_min_length: number;
  password_require_uppercase: boolean;
  password_require_lowercase: boolean;
  password_require_digit: boolean;
  password_require_special_char: boolean;
  access_token_expire_minutes: number;
  refresh_token_expire_days: number;
}

/**
 * 登录日志查询参数。
 */
export interface LoginLogParams extends QueryParams {
  page?: number;
  page_size?: number;
  user_id?: ID;
  action?: number;
  created_from?: string;
  created_to?: string;
}

/**
 * 操作日志查询参数。
 */
export interface OperationLogParams extends Omit<LoginLogParams, "action"> {
  operator_id?: ID;
  action?: string;
  target_type?: string;
}

/**
 * 登录日志项。
 */
export interface LoginLogItem {
  id: ID;
  user_id: ID;
  user_name: string;
  action: number;
  action_text: string;
  login_method?: number | null;
  login_method_text?: string | null;
  ip: string;
  user_agent?: string;
  fail_reason?: string | null;
  created_at: string;
}

/**
 * 操作日志项。
 */
export interface OperationLogItem {
  id: ID;
  operator_id: ID;
  operator_name: string;
  action: string;
  target_type: string;
  target_id?: ID | null;
  ip: string;
  detail?: string | null;
  created_at: string;
}

/**
 * 用户分页响应。
 */
export type UserListResponse = PaginatedData<UserListItem>;

/**
 * 登录日志分页响应。
 */
export type LoginLogListResponse = PaginatedData<LoginLogItem>;

/**
 * 操作日志分页响应。
 */
export type OperationLogListResponse = PaginatedData<OperationLogItem>;
