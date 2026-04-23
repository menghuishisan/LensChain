// school.ts
// 模块02：学校与租户管理类型定义，严格对齐学校入驻、学校管理、SSO与授权接口。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/**
 * 入驻申请状态：1待审核 2已通过 3已拒绝。
 */
export type SchoolApplicationStatus = 1 | 2 | 3;

/**
 * 学校状态：1待审核 2已激活 3缓冲期 4已冻结 5已注销 6已拒绝。
 */
export type SchoolStatus = 1 | 2 | 3 | 4 | 5 | 6;

/**
 * SSO 协议类型。
 */
export type SsoProvider = "cas" | "oauth2";

/**
 * 入驻申请表单请求体。
 */
export interface SchoolApplicationRequest {
  school_name: string;
  school_code: string;
  school_address?: string | null;
  school_website?: string | null;
  school_logo_url?: string | null;
  contact_name: string;
  contact_phone: string;
  contact_email?: string | null;
  contact_title?: string | null;
}

/**
 * 提交入驻申请响应。
 */
export interface SchoolApplicationSubmitResponse {
  application_id: ID;
  status: SchoolApplicationStatus;
  status_text: string;
  tip: string;
}

/**
 * 发送验证码请求体。
 */
export interface SendSchoolApplicationSmsRequest {
  phone: string;
}

/**
 * 查询入驻申请请求参数。
 */
export interface QuerySchoolApplicationsParams extends QueryParams {
  phone: string;
  sms_code: string;
}

/**
 * 入驻申请查询项。
 */
export interface SchoolApplicationQueryItem {
  application_id: ID;
  school_name: string;
  status: SchoolApplicationStatus;
  status_text: string;
  created_at: string;
  reviewed_at: string | null;
  reject_reason: string | null;
}

/**
 * 重新申请预填详情查询参数。
 */
export interface SchoolApplicationReapplyDetailParams extends QueryParams {
  phone: string;
  sms_code: string;
}

/**
 * 重新申请预填详情响应。
 */
export interface SchoolApplicationReapplyDetail {
  application_id: ID;
  school_name: string;
  school_code: string;
  school_address: string | null;
  school_website: string | null;
  school_logo_url: string | null;
  contact_name: string;
  contact_phone: string;
  contact_email: string | null;
  contact_title: string | null;
  status: SchoolApplicationStatus;
  status_text: string;
  reject_reason: string | null;
}

/**
 * 查询入驻申请响应。
 */
export interface QuerySchoolApplicationsResponse {
  applications: SchoolApplicationQueryItem[];
}

/**
 * 重新申请请求体。
 */
export interface ReapplySchoolApplicationRequest extends SchoolApplicationRequest {
  sms_code: string;
}

/**
 * 超管申请列表查询参数。
 */
export interface AdminSchoolApplicationListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: SchoolApplicationStatus;
  keyword?: string;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

/**
 * 入驻申请列表项。
 */
export interface SchoolApplicationListItem {
  id: ID;
  school_name: string;
  school_code: string;
  contact_name: string;
  contact_phone: string;
  status: SchoolApplicationStatus;
  status_text: string;
  created_at: string;
}

/**
 * 入驻申请详情。
 */
export interface SchoolApplicationDetail extends SchoolApplicationListItem {
  school_address: string | null;
  school_website: string | null;
  school_logo_url: string | null;
  contact_email: string | null;
  contact_title: string | null;
  reviewed_at: string | null;
  reject_reason: string | null;
  previous_application_id?: ID | null;
}

/**
 * 入驻申请分页响应。
 */
export type SchoolApplicationListResponse = PaginatedData<SchoolApplicationListItem>;

/**
 * 审核通过请求体。
 */
export interface ApproveSchoolApplicationRequest {
  license_end_at: string;
}

/**
 * 审核通过响应。
 */
export interface ApproveSchoolApplicationResponse {
  school_id: ID;
  admin_user_id: ID;
  admin_phone: string;
  sms_sent: boolean;
}

/**
 * 审核拒绝请求体。
 */
export interface RejectSchoolApplicationRequest {
  reject_reason: string;
}

/**
 * 学校列表查询参数。
 */
export interface SchoolListParams extends QueryParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: SchoolStatus;
  license_expiring?: boolean;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

/**
 * 学校列表项。
 */
export interface SchoolListItem {
  id: ID;
  name: string;
  code: string;
  logo_url: string | null;
  status: SchoolStatus;
  status_text: string;
  license_start_at: string | null;
  license_end_at: string | null;
  license_remaining_days: number | null;
  contact_name: string;
  contact_phone: string;
  created_at: string;
}

/**
 * 学校详情。
 */
export interface SchoolDetail extends SchoolListItem {
  address: string | null;
  website: string | null;
  description: string | null;
  contact_email: string | null;
  contact_title: string | null;
  frozen_at?: string | null;
  frozen_reason?: string | null;
  created_by?: string | null;
}

/**
 * 学校分页响应。
 */
export type SchoolListResponse = PaginatedData<SchoolListItem>;

/**
 * 创建学校请求体。
 */
export interface CreateSchoolRequest {
  name: string;
  code: string;
  address?: string | null;
  website?: string | null;
  logo_url?: string | null;
  description?: string | null;
  license_start_at: string;
  license_end_at: string;
  contact_name: string;
  contact_phone: string;
  contact_email?: string | null;
  contact_title?: string | null;
}

/**
 * 创建学校响应。
 */
export interface CreateSchoolResponse {
  school_id: ID;
  admin_user_id: ID;
  admin_phone: string;
  sms_sent: boolean;
}

/**
 * 更新学校请求体。
 */
export interface UpdateSchoolRequest {
  name?: string;
  code?: string;
  address?: string | null;
  website?: string | null;
  logo_url?: string | null;
  description?: string | null;
  contact_name?: string;
  contact_phone?: string;
  contact_email?: string | null;
  contact_title?: string | null;
}

/**
 * 设置学校授权有效期请求体。
 */
export interface UpdateSchoolLicenseRequest {
  license_end_at: string;
}

/**
 * 冻结学校请求体。
 */
export interface FreezeSchoolRequest {
  reason: string;
}

/**
 * 校管本校资料。
 */
export interface SchoolProfile {
  id: ID;
  name: string;
  code: string;
  logo_url: string | null;
  address: string | null;
  website: string | null;
  description: string | null;
  status: SchoolStatus;
  status_text: string;
}

/**
 * 更新本校资料请求体。
 */
export interface UpdateSchoolProfileRequest {
  logo_url?: string | null;
  description?: string | null;
  address?: string | null;
  website?: string | null;
}

/**
 * CAS SSO 配置。
 */
export interface CasSsoConfig {
  cas_server_url?: string;
  cas_service_url?: string;
  cas_version?: "2.0" | "3.0";
  user_id_attribute?: string;
}

/**
 * OAuth2 SSO 配置。client_secret 返回时可能为 ******，前端不得强制明文回显。
 */
export interface OAuth2SsoConfig {
  authorize_url?: string;
  token_url?: string;
  userinfo_url?: string;
  client_id?: string;
  client_secret?: string;
  redirect_uri?: string;
  scope?: string;
  user_id_attribute?: string;
}

/**
 * SSO 配置响应。
 */
export interface SchoolSsoConfig {
  provider: SsoProvider;
  is_enabled: boolean;
  is_tested: boolean;
  tested_at: string | null;
  config: CasSsoConfig | OAuth2SsoConfig;
}

/**
 * 更新 SSO 配置请求体。
 */
export interface UpdateSchoolSsoConfigRequest {
  provider: SsoProvider;
  config: CasSsoConfig | OAuth2SsoConfig;
}

/**
 * SSO 测试响应。
 */
export interface SchoolSsoTestResponse {
  is_tested: boolean;
  tested_at?: string | null;
  test_detail?: string;
  error_detail?: string;
}

/**
 * 学校授权状态。
 */
export interface SchoolLicense {
  license_start_at: string | null;
  license_end_at: string | null;
  remaining_days: number;
  status: SchoolStatus;
  status_text: string;
  is_expiring_soon: boolean;
}

/**
 * 公开 SSO 学校列表项。
 */
export interface PublicSsoSchool {
  id: ID;
  name: string;
  logo_url: string | null;
}

/**
 * 公开 SSO 学校列表响应。
 */
export interface PublicSsoSchoolListResponse {
  list: PublicSsoSchool[];
}
