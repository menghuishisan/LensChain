// school.ts
// 模块02：学校与租户管理 API service，所有请求只通过 lib/api-client 发起。

import { apiClient } from "@/lib/api-client";
import type { ID } from "@/types/api";
import type {
  AdminSchoolApplicationListParams,
  ApproveSchoolApplicationRequest,
  ApproveSchoolApplicationResponse,
  CreateSchoolRequest,
  CreateSchoolResponse,
  FreezeSchoolRequest,
  PublicSsoSchoolListResponse,
  QuerySchoolApplicationsParams,
  QuerySchoolApplicationsResponse,
  SchoolApplicationReapplyDetail,
  SchoolApplicationReapplyDetailParams,
  ReapplySchoolApplicationRequest,
  RejectSchoolApplicationRequest,
  SchoolApplicationDetail,
  SchoolApplicationListResponse,
  SchoolApplicationRequest,
  SchoolApplicationSubmitResponse,
  SchoolDetail,
  SchoolLicense,
  SchoolListParams,
  SchoolListResponse,
  SchoolProfile,
  SchoolSsoConfig,
  SchoolSsoTestResponse,
  SendSchoolApplicationSmsRequest,
  UpdateSchoolLicenseRequest,
  UpdateSchoolProfileRequest,
  UpdateSchoolRequest,
  UpdateSchoolSsoConfigRequest,
} from "@/types/school";

/**
 * submitSchoolApplication 对应 POST /api/v1/school-applications，用于公开入驻申请。
 */
export function submitSchoolApplication(payload: Partial<SchoolApplicationRequest>) {
  return apiClient.post<SchoolApplicationSubmitResponse>("/school-applications", payload, { auth: false });
}

/**
 * sendSchoolApplicationSmsCode 对应 POST /api/v1/school-applications/send-sms-code，用于发送查询验证码。
 */
export function sendSchoolApplicationSmsCode(payload: SendSchoolApplicationSmsRequest) {
  return apiClient.post<null>("/school-applications/send-sms-code", payload, { auth: false });
}

/**
 * querySchoolApplications 对应 GET /api/v1/school-applications/query，用于申请人查询状态。
 */
export function querySchoolApplications(params: QuerySchoolApplicationsParams) {
  return apiClient.get<QuerySchoolApplicationsResponse>("/school-applications/query", { auth: false, query: params });
}

/**
 * getSchoolApplicationReapplyDetail 对应 GET /api/v1/school-applications/:id/reapply-detail，用于重新申请预填。
 */
export function getSchoolApplicationReapplyDetail(id: ID, params: SchoolApplicationReapplyDetailParams) {
  return apiClient.get<SchoolApplicationReapplyDetail>(`/school-applications/${id}/reapply-detail`, { auth: false, query: params });
}

/**
 * reapplySchoolApplication 对应 POST /api/v1/school-applications/:id/reapply，用于拒绝后重新申请。
 */
export function reapplySchoolApplication(id: ID, payload: ReapplySchoolApplicationRequest) {
  return apiClient.post<SchoolApplicationSubmitResponse>(`/school-applications/${id}/reapply`, payload, { auth: false });
}

/**
 * listSchoolApplications 对应 GET /api/v1/admin/school-applications，用于超管审核列表。
 */
export function listSchoolApplications(params: AdminSchoolApplicationListParams) {
  return apiClient.get<SchoolApplicationListResponse>("/admin/school-applications", { query: params });
}

/**
 * getSchoolApplication 对应 GET /api/v1/admin/school-applications/:id，用于申请详情。
 */
export function getSchoolApplication(id: ID) {
  return apiClient.get<SchoolApplicationDetail>(`/admin/school-applications/${id}`);
}

/**
 * approveSchoolApplication 对应 POST /api/v1/admin/school-applications/:id/approve，用于审核通过。
 */
export function approveSchoolApplication(id: ID, payload: ApproveSchoolApplicationRequest) {
  return apiClient.post<ApproveSchoolApplicationResponse>(`/admin/school-applications/${id}/approve`, payload);
}

/**
 * rejectSchoolApplication 对应 POST /api/v1/admin/school-applications/:id/reject，用于审核拒绝。
 */
export function rejectSchoolApplication(id: ID, payload: RejectSchoolApplicationRequest) {
  return apiClient.post<null>(`/admin/school-applications/${id}/reject`, payload);
}

/**
 * listSchools 对应 GET /api/v1/admin/schools，用于超管学校列表。
 */
export function listSchools(params: SchoolListParams) {
  return apiClient.get<SchoolListResponse>("/admin/schools", { query: params });
}

/**
 * createSchool 对应 POST /api/v1/admin/schools，用于超管后台直接创建学校。
 */
export function createSchool(payload: CreateSchoolRequest) {
  return apiClient.post<CreateSchoolResponse>("/admin/schools", payload);
}

/**
 * getSchool 对应 GET /api/v1/admin/schools/:id，用于学校详情。
 */
export function getSchool(id: ID) {
  return apiClient.get<SchoolDetail>(`/admin/schools/${id}`);
}

/**
 * updateSchool 对应 PUT /api/v1/admin/schools/:id，用于超管编辑学校信息。
 */
export function updateSchool(id: ID, payload: UpdateSchoolRequest) {
  return apiClient.put<SchoolDetail>(`/admin/schools/${id}`, payload);
}

/**
 * updateSchoolLicense 对应 PATCH /api/v1/admin/schools/:id/license，用于设置授权有效期。
 */
export function updateSchoolLicense(id: ID, payload: UpdateSchoolLicenseRequest) {
  return apiClient.patch<null>(`/admin/schools/${id}/license`, payload);
}

/**
 * freezeSchool 对应 POST /api/v1/admin/schools/:id/freeze，用于冻结学校。
 */
export function freezeSchool(id: ID, payload: FreezeSchoolRequest) {
  return apiClient.post<null>(`/admin/schools/${id}/freeze`, payload);
}

/**
 * unfreezeSchool 对应 POST /api/v1/admin/schools/:id/unfreeze，用于解冻学校。
 */
export function unfreezeSchool(id: ID) {
  return apiClient.post<null>(`/admin/schools/${id}/unfreeze`);
}

/**
 * cancelSchool 对应 POST /api/v1/admin/schools/:id/cancel，用于注销学校，必须传 confirm=true。
 */
export function cancelSchool(id: ID) {
  return apiClient.post<null>(`/admin/schools/${id}/cancel`, { confirm: true });
}

/**
 * restoreSchool 对应 POST /api/v1/admin/schools/:id/restore，用于恢复已注销学校。
 */
export function restoreSchool(id: ID) {
  return apiClient.post<null>(`/admin/schools/${id}/restore`);
}

/**
 * getSchoolProfile 对应 GET /api/v1/school/profile，用于校管查看本校资料。
 */
export function getSchoolProfile() {
  return apiClient.get<SchoolProfile>("/school/profile");
}

/**
 * updateSchoolProfile 对应 PUT /api/v1/school/profile，用于校管编辑本校展示资料。
 */
export function updateSchoolProfile(payload: UpdateSchoolProfileRequest) {
  return apiClient.put<SchoolProfile>("/school/profile", payload);
}

/**
 * getSchoolSsoConfig 对应 GET /api/v1/school/sso-config，用于校管读取SSO配置。
 */
export function getSchoolSsoConfig() {
  return apiClient.get<SchoolSsoConfig>("/school/sso-config");
}

/**
 * updateSchoolSsoConfig 对应 PUT /api/v1/school/sso-config，用于校管更新SSO配置。
 */
export function updateSchoolSsoConfig(payload: UpdateSchoolSsoConfigRequest) {
  return apiClient.put<SchoolSsoConfig>("/school/sso-config", payload);
}

/**
 * testSchoolSsoConfig 对应 POST /api/v1/school/sso-config/test，用于测试SSO连接。
 */
export function testSchoolSsoConfig() {
  return apiClient.post<SchoolSsoTestResponse>("/school/sso-config/test");
}

/**
 * enableSchoolSso 对应 POST /api/v1/school/sso-config/enable，用于启用或禁用SSO。
 */
export function enableSchoolSso(isEnabled: boolean) {
  return apiClient.post<null>("/school/sso-config/enable", { is_enabled: isEnabled });
}

/**
 * getSchoolLicense 对应 GET /api/v1/school/license，用于校管查看授权状态。
 */
export function getSchoolLicense() {
  return apiClient.get<SchoolLicense>("/school/license");
}

/**
 * listPublicSsoSchools 对应 GET /api/v1/schools/sso-list，用于公开SSO学校列表。
 */
export function listPublicSsoSchools() {
  return apiClient.get<PublicSsoSchoolListResponse>("/schools/sso-list", { auth: false });
}
