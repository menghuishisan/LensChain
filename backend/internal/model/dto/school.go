// school.go
// 模块02 — 学校与租户管理：请求/响应 DTO
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md
// 包含入驻申请、学校管理、SSO配置、授权状态等接口的 DTO

package dto

// ========== 入驻申请接口 DTO ==========

// SubmitApplicationReq 提交入驻申请请求
// POST /api/v1/school-applications
type SubmitApplicationReq struct {
	SchoolName    string  `json:"school_name" binding:"required,max=100"`
	SchoolCode    string  `json:"school_code" binding:"required,max=50"`
	SchoolAddress *string `json:"school_address" binding:"omitempty,max=200"`
	SchoolWebsite *string `json:"school_website" binding:"omitempty,max=200,url"`
	SchoolLogoURL *string `json:"school_logo_url" binding:"omitempty,max=500,url"`
	ContactName   string  `json:"contact_name" binding:"required,max=50"`
	ContactPhone  string  `json:"contact_phone" binding:"required,phone"`
	ContactEmail  *string `json:"contact_email" binding:"omitempty,email,max=100"`
	ContactTitle  *string `json:"contact_title" binding:"omitempty,max=100"`
}

// SubmitApplicationResp 提交入驻申请响应
type SubmitApplicationResp struct {
	ApplicationID string `json:"application_id"`
	Status        int16  `json:"status"`
	StatusText    string `json:"status_text"`
	Tip           string `json:"tip"`
}

// SendSMSCodeReq 发送查询验证码请求
// POST /api/v1/school-applications/send-sms-code
type SendSMSCodeReq struct {
	Phone string `json:"phone" binding:"required,phone"`
}

// QueryApplicationReq 查询申请状态请求
// GET /api/v1/school-applications/query
type QueryApplicationReq struct {
	Phone   string `form:"phone" binding:"required,phone"`
	SMSCode string `form:"sms_code" binding:"required,len=6"`
}

// QueryApplicationResp 查询申请状态响应
type QueryApplicationResp struct {
	Applications []ApplicationStatusItem `json:"applications"`
}

// ApplicationStatusItem 申请状态列表项
type ApplicationStatusItem struct {
	ApplicationID string  `json:"application_id"`
	SchoolName    string  `json:"school_name"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
	CreatedAt     string  `json:"created_at"`
	ReviewedAt    *string `json:"reviewed_at"`
	RejectReason  *string `json:"reject_reason"`
}

// ReapplyDetailReq 获取重新申请预填详情请求
// GET /api/v1/school-applications/:id/reapply-detail
type ReapplyDetailReq struct {
	Phone   string `form:"phone" binding:"required,phone"`
	SMSCode string `form:"sms_code" binding:"required,len=6"`
}

// ReapplyDetailResp 重新申请预填详情响应
type ReapplyDetailResp struct {
	ApplicationID string  `json:"application_id"`
	SchoolName    string  `json:"school_name"`
	SchoolCode    string  `json:"school_code"`
	SchoolAddress *string `json:"school_address"`
	SchoolWebsite *string `json:"school_website"`
	SchoolLogoURL *string `json:"school_logo_url"`
	ContactName   string  `json:"contact_name"`
	ContactPhone  string  `json:"contact_phone"`
	ContactEmail  *string `json:"contact_email"`
	ContactTitle  *string `json:"contact_title"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
	RejectReason  *string `json:"reject_reason"`
}

// ReapplyReq 重新申请请求
// POST /api/v1/school-applications/:id/reapply
type ReapplyReq struct {
	SMSCode       string  `json:"sms_code" binding:"required,len=6"`
	SchoolName    string  `json:"school_name" binding:"required,max=100"`
	SchoolCode    string  `json:"school_code" binding:"required,max=50"`
	SchoolAddress *string `json:"school_address" binding:"omitempty,max=200"`
	SchoolWebsite *string `json:"school_website" binding:"omitempty,max=200,url"`
	SchoolLogoURL *string `json:"school_logo_url" binding:"omitempty,max=500,url"`
	ContactName   string  `json:"contact_name" binding:"required,max=50"`
	ContactPhone  string  `json:"contact_phone" binding:"required,phone"`
	ContactEmail  *string `json:"contact_email" binding:"omitempty,email,max=100"`
	ContactTitle  *string `json:"contact_title" binding:"omitempty,max=100"`
}

// ========== 入驻审核接口 DTO ==========

// ApplicationListReq 申请列表请求
// GET /api/v1/admin/school-applications
type ApplicationListReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status    int16  `form:"status" binding:"omitempty,oneof=1 2 3"`
	Keyword   string `form:"keyword"`
	SortBy    string `form:"sort_by" binding:"omitempty"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ApplicationListItem 申请列表项
type ApplicationListItem struct {
	ID           string  `json:"id"`
	SchoolName   string  `json:"school_name"`
	SchoolCode   string  `json:"school_code"`
	ContactName  string  `json:"contact_name"`
	ContactPhone string  `json:"contact_phone"`
	Status       int16   `json:"status"`
	StatusText   string  `json:"status_text"`
	CreatedAt    string  `json:"created_at"`
	ReviewedAt   *string `json:"reviewed_at"`
}

// ApplicationDetailResp 申请详情响应
type ApplicationDetailResp struct {
	ID                    string  `json:"id"`
	SchoolName            string  `json:"school_name"`
	SchoolCode            string  `json:"school_code"`
	SchoolAddress         *string `json:"school_address"`
	SchoolWebsite         *string `json:"school_website"`
	SchoolLogoURL         *string `json:"school_logo_url"`
	ContactName           string  `json:"contact_name"`
	ContactPhone          string  `json:"contact_phone"`
	ContactEmail          *string `json:"contact_email"`
	ContactTitle          *string `json:"contact_title"`
	Status                int16   `json:"status"`
	StatusText            string  `json:"status_text"`
	ReviewerID            *string `json:"reviewer_id"`
	ReviewedAt            *string `json:"reviewed_at"`
	RejectReason          *string `json:"reject_reason"`
	SchoolID              *string `json:"school_id"`
	PreviousApplicationID *string `json:"previous_application_id"`
	CreatedAt             string  `json:"created_at"`
}

// ApproveApplicationReq 审核通过请求
// POST /api/v1/admin/school-applications/:id/approve
type ApproveApplicationReq struct {
	LicenseEndAt string `json:"license_end_at" binding:"required"`
}

// ApproveApplicationResp 审核通过响应
type ApproveApplicationResp struct {
	SchoolID    string `json:"school_id"`
	AdminUserID string `json:"admin_user_id"`
	AdminPhone  string `json:"admin_phone"`
	SMSSent     bool   `json:"sms_sent"`
}

// RejectApplicationReq 审核拒绝请求
// POST /api/v1/admin/school-applications/:id/reject
type RejectApplicationReq struct {
	RejectReason string `json:"reject_reason" binding:"required,max=500"`
}

// ========== 学校管理接口 DTO ==========

// SchoolListReq 学校列表请求
// GET /api/v1/admin/schools
type SchoolListReq struct {
	Page            int    `form:"page" binding:"omitempty,min=1"`
	PageSize        int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword         string `form:"keyword"`
	Status          int16  `form:"status" binding:"omitempty,oneof=1 2 3 4 5 6"`
	LicenseExpiring bool   `form:"license_expiring"`
	SortBy          string `form:"sort_by" binding:"omitempty"`
	SortOrder       string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// SchoolListItem 学校列表项
type SchoolListItem struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Code                 string  `json:"code"`
	LogoURL              *string `json:"logo_url"`
	Status               int16   `json:"status"`
	StatusText           string  `json:"status_text"`
	LicenseStartAt       *string `json:"license_start_at"`
	LicenseEndAt         *string `json:"license_end_at"`
	LicenseRemainingDays *int    `json:"license_remaining_days"`
	ContactName          string  `json:"contact_name"`
	ContactPhone         string  `json:"contact_phone"`
	CreatedAt            string  `json:"created_at"`
}

// SchoolDetailResp 学校详情响应
type SchoolDetailResp struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Code           string  `json:"code"`
	LogoURL        *string `json:"logo_url"`
	Address        *string `json:"address"`
	Website        *string `json:"website"`
	Description    *string `json:"description"`
	Status         int16   `json:"status"`
	StatusText     string  `json:"status_text"`
	LicenseStartAt *string `json:"license_start_at"`
	LicenseEndAt   *string `json:"license_end_at"`
	FrozenAt       *string `json:"frozen_at"`
	FrozenReason   *string `json:"frozen_reason"`
	ContactName    string  `json:"contact_name"`
	ContactPhone   string  `json:"contact_phone"`
	ContactEmail   *string `json:"contact_email"`
	ContactTitle   *string `json:"contact_title"`
	CreatedAt      string  `json:"created_at"`
	CreatedBy      *string `json:"created_by"`
}

// CreateSchoolReq 后台直接创建学校请求
// POST /api/v1/admin/schools
type CreateSchoolReq struct {
	Name           string  `json:"name" binding:"required,max=100"`
	Code           string  `json:"code" binding:"required,max=50"`
	Address        *string `json:"address" binding:"omitempty,max=200"`
	Website        *string `json:"website" binding:"omitempty,max=200,url"`
	LogoURL        *string `json:"logo_url" binding:"omitempty,max=500,url"`
	Description    *string `json:"description"`
	LicenseStartAt string  `json:"license_start_at" binding:"required"`
	LicenseEndAt   string  `json:"license_end_at" binding:"required"`
	ContactName    string  `json:"contact_name" binding:"required,max=50"`
	ContactPhone   string  `json:"contact_phone" binding:"required,phone"`
	ContactEmail   *string `json:"contact_email" binding:"omitempty,email,max=100"`
	ContactTitle   *string `json:"contact_title" binding:"omitempty,max=100"`
}

// CreateSchoolResp 创建学校响应
type CreateSchoolResp struct {
	SchoolID    string `json:"school_id"`
	AdminUserID string `json:"admin_user_id"`
	AdminPhone  string `json:"admin_phone"`
	SMSSent     bool   `json:"sms_sent"`
}

// UpdateSchoolReq 编辑学校信息请求
// PUT /api/v1/admin/schools/:id
type UpdateSchoolReq struct {
	Name         *string `json:"name" binding:"omitempty,max=100"`
	Code         *string `json:"code" binding:"omitempty,max=50"`
	Address      *string `json:"address" binding:"omitempty,max=200"`
	Website      *string `json:"website" binding:"omitempty,max=200,url"`
	LogoURL      *string `json:"logo_url" binding:"omitempty,max=500,url"`
	Description  *string `json:"description"`
	ContactName  *string `json:"contact_name" binding:"omitempty,max=50"`
	ContactPhone *string `json:"contact_phone" binding:"omitempty,phone"`
	ContactEmail *string `json:"contact_email" binding:"omitempty,email,max=100"`
	ContactTitle *string `json:"contact_title" binding:"omitempty,max=100"`
}

// SetLicenseReq 设置有效期请求
// PATCH /api/v1/admin/schools/:id/license
type SetLicenseReq struct {
	LicenseEndAt string `json:"license_end_at" binding:"required"`
}

// FreezeSchoolReq 冻结学校请求
// POST /api/v1/admin/schools/:id/freeze
type FreezeSchoolReq struct {
	Reason string `json:"reason" binding:"required,max=200"`
}

// CancelSchoolReq 注销学校请求
// POST /api/v1/admin/schools/:id/cancel
type CancelSchoolReq struct {
	Confirm *bool `json:"confirm" binding:"required"`
}

// ========== 学校配置接口 DTO（校管） ==========

// SchoolProfileResp 本校信息响应
// GET /api/v1/school/profile
type SchoolProfileResp struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	LogoURL     *string `json:"logo_url"`
	Address     *string `json:"address"`
	Website     *string `json:"website"`
	Description *string `json:"description"`
	Status      int16   `json:"status"`
	StatusText  string  `json:"status_text"`
}

// UpdateSchoolProfileReq 编辑本校信息请求（校管仅可修改部分字段）
// PUT /api/v1/school/profile
type UpdateSchoolProfileReq struct {
	LogoURL     *string `json:"logo_url" binding:"omitempty,max=500,url"`
	Description *string `json:"description"`
	Address     *string `json:"address" binding:"omitempty,max=200"`
	Website     *string `json:"website" binding:"omitempty,max=200,url"`
}

// SSOConfigResp SSO配置响应
// GET /api/v1/school/sso-config
type SSOConfigResp struct {
	Provider  string     `json:"provider"`
	IsEnabled bool       `json:"is_enabled"`
	IsTested  bool       `json:"is_tested"`
	TestedAt  *string    `json:"tested_at"`
	Config    *SSOConfig `json:"config"`
}

// SSOConfig SSO协议配置参数。
// 同一结构承载 CAS 与 OAuth2 的配置字段，具体必填项由 provider 决定并在 service 层按协议校验。
type SSOConfig struct {
	CASServerURL    *string `json:"cas_server_url,omitempty" binding:"omitempty,url,max=500"`
	CASServiceURL   *string `json:"cas_service_url,omitempty" binding:"omitempty,url,max=500"`
	CASVersion      *string `json:"cas_version,omitempty" binding:"omitempty,oneof=2.0 3.0"`
	AuthorizeURL    *string `json:"authorize_url,omitempty" binding:"omitempty,url,max=500"`
	TokenURL        *string `json:"token_url,omitempty" binding:"omitempty,url,max=500"`
	UserinfoURL     *string `json:"userinfo_url,omitempty" binding:"omitempty,url,max=500"`
	ClientID        *string `json:"client_id,omitempty" binding:"omitempty,max=200"`
	ClientSecret    *string `json:"client_secret,omitempty" binding:"omitempty,max=500"`
	RedirectURI     *string `json:"redirect_uri,omitempty" binding:"omitempty,url,max=500"`
	Scope           *string `json:"scope,omitempty" binding:"omitempty,max=200"`
	UserIDAttribute string  `json:"user_id_attribute" binding:"required,max=100"`
}

// UpdateSSOConfigReq 更新SSO配置请求
// PUT /api/v1/school/sso-config
type UpdateSSOConfigReq struct {
	Provider string     `json:"provider" binding:"required,oneof=cas oauth2"`
	Config   *SSOConfig `json:"config" binding:"required"`
}

// ToggleSSOEnableReq 切换SSO启用状态请求
// POST /api/v1/school/sso-config/enable
type ToggleSSOEnableReq struct {
	IsEnabled *bool `json:"is_enabled" binding:"required"`
}

// SSOTestResp SSO连接测试响应
type SSOTestResp struct {
	IsTested    bool    `json:"is_tested"`
	TestedAt    *string `json:"tested_at"`
	TestDetail  *string `json:"test_detail"`
	ErrorDetail *string `json:"error_detail"`
}

// LicenseStatusResp 授权状态响应
// GET /api/v1/school/license
type LicenseStatusResp struct {
	LicenseStartAt *string `json:"license_start_at"`
	LicenseEndAt   *string `json:"license_end_at"`
	RemainingDays  int     `json:"remaining_days"`
	Status         int16   `json:"status"`
	StatusText     string  `json:"status_text"`
	IsExpiringSoon bool    `json:"is_expiring_soon"`
}

// ========== 公开接口 DTO ==========

// SSOSchoolItem SSO学校列表项
// GET /api/v1/schools/sso-list
type SSOSchoolItem struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}
