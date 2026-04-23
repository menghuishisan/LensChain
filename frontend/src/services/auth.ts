// auth.ts
// 模块01：用户与认证 API service，所有请求只通过 lib/api-client 发起。

import { apiClient } from "@/lib/api-client";
import type { ID } from "@/types/api";
import type {
  BatchDeleteUsersRequest,
  ChangePasswordRequest,
  CreateSuperAdminRequest,
  CreateUserRequest,
  ExecuteUserImportRequest,
  ExecuteUserImportResponse,
  ForceChangePasswordRequest,
  LoginLogListResponse,
  LoginLogParams,
  LoginRequest,
  LoginResponse,
  OperationLogListResponse,
  OperationLogParams,
  Profile,
  ResetUserPasswordRequest,
  SecurityPolicy,
  SsoSchoolListResponse,
  UpdateProfileRequest,
  UpdateUserRequest,
  UpdateUserStatusRequest,
  UserDetail,
  UserImportPreviewRequest,
  UserImportPreviewResponse,
  UserImportType,
  UserListParams,
  UserListResponse,
} from "@/types/auth";

/**
 * login 对应 POST /api/v1/auth/login，用于手机号密码登录。
 */
export function login(payload: LoginRequest) {
  return apiClient.post<LoginResponse>("/auth/login", payload, { auth: false });
}

/**
 * logout 对应 POST /api/v1/auth/logout，用于退出当前会话。
 */
export function logout() {
  return apiClient.post<null>("/auth/logout");
}

/**
 * forceChangePassword 对应 POST /api/v1/auth/force-change-password，用于首次登录强制改密。
 */
export function forceChangePassword(payload: ForceChangePasswordRequest) {
  return apiClient.post<LoginResponse>(
    "/auth/force-change-password",
    { new_password: payload.newPassword },
    { auth: false, headers: { Authorization: `Bearer ${payload.tempToken}` } },
  );
}

/**
 * changePassword 对应 POST /api/v1/auth/change-password，用于已登录用户修改密码。
 */
export function changePassword(payload: ChangePasswordRequest) {
  return apiClient.post<null>("/auth/change-password", payload);
}

/**
 * listSsoSchools 对应 GET /api/v1/auth/sso-schools，用于 SSO 学校选择页。
 */
export function listSsoSchools(keyword?: string) {
  return apiClient.get<SsoSchoolListResponse>("/auth/sso-schools", { auth: false, query: { keyword } });
}

/**
 * getSsoLoginUrl 对应 GET /api/v1/auth/sso/:school_id/login 的浏览器跳转地址。
 */
export function getSsoLoginUrl(schoolID: ID) {
  return apiClient.createURL(`/auth/sso/${schoolID}/login`);
}

/**
 * ssoCallback 对应 GET /api/v1/auth/sso/callback，用于 SSO 回调换取登录结果。
 */
export function ssoCallback(query: Record<string, string>) {
  return apiClient.get<LoginResponse>("/auth/sso/callback", { auth: false, query });
}

/**
 * listUsers 对应 GET /api/v1/users，用于用户管理列表分页查询。
 */
export function listUsers(params: UserListParams) {
  return apiClient.get<UserListResponse>("/users", { query: params });
}

/**
 * getUser 对应 GET /api/v1/users/:id，用于用户详情。
 */
export function getUser(id: ID) {
  return apiClient.get<UserDetail>(`/users/${id}`);
}

/**
 * createUser 对应 POST /api/v1/users，用于手动创建用户。
 */
export function createUser(payload: CreateUserRequest) {
  return apiClient.post<{ id: ID }>("/users", payload);
}

/**
 * createSuperAdmin 对应 POST /api/v1/users/super-admins，用于超管创建新的超级管理员。
 */
export function createSuperAdmin(payload: CreateSuperAdminRequest) {
  return apiClient.post<{ id: ID }>("/users/super-admins", payload);
}

/**
 * updateUser 对应 PUT /api/v1/users/:id，用于更新用户资料。
 */
export function updateUser(id: ID, payload: UpdateUserRequest) {
  return apiClient.put<UserDetail>(`/users/${id}`, payload);
}

/**
 * deleteUser 对应 DELETE /api/v1/users/:id，用于软删除用户。
 */
export function deleteUser(id: ID) {
  return apiClient.delete<null>(`/users/${id}`);
}

/**
 * updateUserStatus 对应 PATCH /api/v1/users/:id/status，用于启用、禁用或归档账号。
 */
export function updateUserStatus(id: ID, payload: UpdateUserStatusRequest) {
  return apiClient.patch<null>(`/users/${id}/status`, payload);
}

/**
 * resetUserPassword 对应 POST /api/v1/users/:id/reset-password，用于管理员重置用户密码。
 */
export function resetUserPassword(id: ID, payload: ResetUserPasswordRequest) {
  return apiClient.post<null>(`/users/${id}/reset-password`, payload);
}

/**
 * unlockUser 对应 POST /api/v1/users/:id/unlock，用于管理员解锁账号。
 */
export function unlockUser(id: ID) {
  return apiClient.post<null>(`/users/${id}/unlock`);
}

/**
 * batchDeleteUsers 对应 POST /api/v1/users/batch-delete，用于批量删除账号。
 */
export function batchDeleteUsers(payload: BatchDeleteUsersRequest) {
  return apiClient.post<null>("/users/batch-delete", payload);
}

/**
 * downloadUserImportTemplate 对应 GET /api/v1/user-imports/template，用于下载导入模板。
 */
export function downloadUserImportTemplate(type: UserImportType) {
  return apiClient.download("/user-imports/template", { query: { type } });
}

/**
 * previewUserImport 对应 POST /api/v1/user-imports/preview，用于上传文件并预览校验结果。
 */
export function previewUserImport(payload: UserImportPreviewRequest) {
  const formData = new FormData();
  formData.append("file", payload.file);
  formData.append("type", payload.type);

  return apiClient.post<UserImportPreviewResponse>("/user-imports/preview", formData);
}

/**
 * executeUserImport 对应 POST /api/v1/user-imports/execute，用于确认执行导入。
 */
export function executeUserImport(payload: ExecuteUserImportRequest) {
  return apiClient.post<ExecuteUserImportResponse>("/user-imports/execute", payload);
}

/**
 * downloadUserImportFailures 对应 GET /api/v1/user-imports/:id/failures，用于下载失败明细。
 */
export function downloadUserImportFailures(importID: string) {
  return apiClient.download(`/user-imports/${importID}/failures`);
}

/**
 * getProfile 对应 GET /api/v1/profile，仅返回个人基础资料，不包含 learning_overview。
 */
export function getProfile() {
  return apiClient.get<Profile>("/profile");
}

/**
 * updateProfile 对应 PUT /api/v1/profile，用于更新昵称、头像和邮箱。
 */
export function updateProfile(payload: UpdateProfileRequest) {
  return apiClient.put<Profile>("/profile", payload);
}

/**
 * getSecurityPolicy 对应 GET /api/v1/security-policies，用于读取安全策略。
 */
export function getSecurityPolicy() {
  return apiClient.get<SecurityPolicy>("/security-policies");
}

/**
 * updateSecurityPolicy 对应 PUT /api/v1/security-policies，用于更新安全策略。
 */
export function updateSecurityPolicy(payload: Partial<SecurityPolicy>) {
  return apiClient.put<SecurityPolicy>("/security-policies", payload);
}

/**
 * listLoginLogs 对应 GET /api/v1/login-logs，用于登录日志分页查询。
 */
export function listLoginLogs(params: LoginLogParams) {
  return apiClient.get<LoginLogListResponse>("/login-logs", { query: params });
}

/**
 * listOperationLogs 对应 GET /api/v1/operation-logs，用于操作日志分页查询。
 */
export function listOperationLogs(params: OperationLogParams) {
  return apiClient.get<OperationLogListResponse>("/operation-logs", { query: params });
}
