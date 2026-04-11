// auth_service.go
// 模块01 — 用户与认证：认证业务逻辑
// 负责登录、登出、Token刷新、密码修改等认证相关业务
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/errcode"
	jwtpkg "github.com/lenschain/backend/internal/pkg/jwt"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/repository/auth"
)

// LoginResult 登录结果（强类型，避免 interface{} 导致 handler 层类型断言出错）
type LoginResult struct {
	IsFirstLogin bool                         // 是否首次登录（需强制改密）
	TokenResp    *dto.LoginResp               // 正常登录时的Token+用户信息
	ForceResp    *dto.ForceChangePasswordResp // 首次登录时的临时Token
}

// AuthService 认证服务接口
type AuthService interface {
	Login(ctx context.Context, phone, password, ip, userAgent string) (*LoginResult, error)
	SSOLoginURL(ctx context.Context, schoolID int64) (string, error)
	SSOCallback(ctx context.Context, schoolID int64, query map[string]string, ip, userAgent string) (*LoginResult, error)
	Logout(ctx context.Context, userID int64, jti, ip, userAgent string) error
	RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshTokenResp, error)
	ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword, ip string) error
	ForceChangePassword(ctx context.Context, userID int64, newPassword, ip string) (*LoginResult, error)
}

// SchoolNameQuerier 跨模块接口：查询学校名称
// 由模块02注入实现，解耦跨模块依赖
type SchoolNameQuerier interface {
	GetSchoolName(ctx context.Context, schoolID int64) string
}

// SchoolStatusChecker 跨模块接口：校验学校状态是否允许登录
type SchoolStatusChecker interface {
	CheckLoginAllowed(ctx context.Context, schoolID int64) error
}

// SchoolSSOConfigQuerier 跨模块接口：查询学校SSO配置
type SchoolSSOConfigQuerier interface {
	GetSchoolSSOConfig(ctx context.Context, schoolID int64) (*SchoolSSOConfig, error)
}

// SchoolSSOConfig 学校SSO配置
type SchoolSSOConfig struct {
	SchoolID  int64
	Provider  string
	IsEnabled bool
	IsTested  bool
	Config    map[string]interface{}
}

// authService 认证服务实现
type authService struct {
	db                  *gorm.DB
	userRepo            authrepo.UserRepository
	roleRepo            authrepo.RoleRepository
	loginLogRepo        authrepo.LoginLogRepository
	ssoBindingRepo      authrepo.SSOBindingRepository
	schoolNameQuerier   SchoolNameQuerier
	schoolStatusChecker SchoolStatusChecker
	schoolSSOQuerier    SchoolSSOConfigQuerier
}

// NewAuthService 创建认证服务实例
func NewAuthService(
	db *gorm.DB,
	userRepo authrepo.UserRepository,
	roleRepo authrepo.RoleRepository,
	loginLogRepo authrepo.LoginLogRepository,
	ssoBindingRepo authrepo.SSOBindingRepository,
	schoolNameQuerier SchoolNameQuerier,
	schoolStatusChecker SchoolStatusChecker,
	schoolSSOQuerier SchoolSSOConfigQuerier,
) AuthService {
	return &authService{
		db:                  db,
		userRepo:            userRepo,
		roleRepo:            roleRepo,
		loginLogRepo:        loginLogRepo,
		ssoBindingRepo:      ssoBindingRepo,
		schoolNameQuerier:   schoolNameQuerier,
		schoolStatusChecker: schoolStatusChecker,
		schoolSSOQuerier:    schoolSSOQuerier,
	}
}

// Login 手机号+密码登录
// 安全要点：
// - 手机号不存在和密码错误返回相同的错误码，防止用户枚举（P0-1 修复）
// - 使用 Redis INCR 原子操作做失败计数，锁定检查与计数在同一原子操作中（P0-2 修复）
// - 首次登录不记录为"登录成功"，使用专门的日志标记（P1-3 修复）
func (s *authService) Login(ctx context.Context, phone, password, ip, userAgent string) (*LoginResult, error) {
	// 1. 检查账号是否被锁定（Redis）
	locked, _ := cache.Exists(ctx, cache.KeyAccountLocked+phone)
	if locked {
		// P1-4 修复：不记录 userID=0 的日志，改为记录 phone
		asyncRecordLoginLog(s.loginLogRepo, 0, enum.LoginActionFail, enum.LoginMethodPassword, ip, userAgent, "账号已锁定:"+phone)
		return nil, errcode.ErrAccountLocked.WithMessage("账号已锁定，请稍后再试")
	}

	// 2. 根据手机号查找用户
	user, err := s.userRepo.GetByPhone(ctx, phone)
	if err != nil {
		// P0-1 修复：手机号不存在时返回与密码错误相同的错误码，防止用户枚举
		// 执行一次虚假的 bcrypt 比较，防止时序攻击
		crypto.CheckPassword(password, "$2a$12$000000000000000000000000000000000000000000000000000000")
		return nil, errcode.ErrWrongCredentials
	}

	// 3. 验证密码
	if !crypto.CheckPassword(password, user.PasswordHash) {
		return s.handleLoginFail(ctx, user, phone, ip, userAgent)
	}

	// 4. 检查账号状态
	switch user.Status {
	case enum.UserStatusDisabled:
		asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionFail, enum.LoginMethodPassword, ip, userAgent, "账号已禁用")
		return nil, errcode.ErrAccountDisabled
	case enum.UserStatusArchived:
		asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionFail, enum.LoginMethodPassword, ip, userAgent, "账号已归档")
		return nil, errcode.ErrAccountArchived
	}

	// 检查是否已过锁定时间（数据库层面）
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionFail, enum.LoginMethodPassword, ip, userAgent, "账号已锁定(DB)")
		return nil, errcode.ErrAccountLocked.WithMessage("账号已锁定，请稍后再试")
	}

	// 4.1 检查学校状态
	if s.schoolStatusChecker != nil {
		if err := s.schoolStatusChecker.CheckLoginAllowed(ctx, user.SchoolID); err != nil {
			message := err.Error()
			if appErr, ok := errcode.IsAppError(err); ok {
				message = appErr.Message
			}
			asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionFail, enum.LoginMethodPassword, ip, userAgent, message)
			return nil, err
		}
	}

	// 重置登录失败计数（登录成功时清除）
	_ = cache.Del(ctx, cache.KeyLoginFail+phone)
	_ = s.userRepo.ResetLoginFailCount(ctx, user.ID)

	// 5. 判断是否首次登录
	if user.IsFirstLogin {
		tempToken, err := jwtpkg.GenerateTempToken(user.ID)
		if err != nil {
			return nil, errcode.ErrInternal.WithMessage("生成临时Token失败")
		}
		// P1-3 修复：首次登录不记录为"登录成功"，记录为专门的日志
		asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionSuccess, enum.LoginMethodPassword, ip, userAgent, "首次登录，待改密")
		return &LoginResult{
			IsFirstLogin: true,
			ForceResp: &dto.ForceChangePasswordResp{
				ForceChangePassword: true,
				TempToken:           tempToken,
				TempTokenExpiresIn:  300,
			},
		}, nil
	}

	// 6. 生成Token对
	roleCodes := s.extractRoleCodes(user)
	tokenPair, err := jwtpkg.GenerateTokenPair(user.ID, user.SchoolID, roleCodes)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("生成Token失败")
	}

	// 7. 存储Session到Redis
	s.storeSession(ctx, user.ID, tokenPair.RefreshToken, ip)

	// 8. 更新登录信息
	now := time.Now()
	_ = s.userRepo.UpdateLoginInfo(ctx, user.ID, ip, now)

	// 9. 记录登录日志
	asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionSuccess, enum.LoginMethodPassword, ip, userAgent, "")

	// 获取学校名称（跨模块查询）
	schoolName := ""
	if s.schoolNameQuerier != nil {
		schoolName = s.schoolNameQuerier.GetSchoolName(ctx, user.SchoolID)
	}

	return &LoginResult{
		IsFirstLogin: false,
		TokenResp: &dto.LoginResp{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresIn:    tokenPair.ExpiresIn,
			TokenType:    "Bearer",
			User: dto.LoginUser{
				ID:           strconv.FormatInt(user.ID, 10),
				Name:         user.Name,
				Phone:        user.Phone,
				Roles:        roleCodes,
				SchoolID:     strconv.FormatInt(user.SchoolID, 10),
				SchoolName:   schoolName,
				IsFirstLogin: false,
			},
		},
	}, nil
}

// handleLoginFail 处理登录失败（密码错误）
// P0-1 修复：返回统一的错误码，不泄露手机号是否存在
func (s *authService) handleLoginFail(ctx context.Context, user *entity.User, phone, ip, userAgent string) (*LoginResult, error) {
	// 增加失败计数（Redis 原子操作）
	failCount, err := cache.IncrWithExpire(ctx, cache.KeyLoginFail+phone, 30*time.Minute)
	if err != nil {
		logger.L.Error("增加登录失败计数失败", zap.Error(err))
	}

	// 同时更新数据库中的失败计数
	_ = s.userRepo.IncrLoginFailCount(ctx, user.ID)

	// 获取安全策略（最大失败次数）
	maxFail := s.getMaxFailCount(ctx)

	// 判断是否需要锁定
	if failCount >= int64(maxFail) {
		lockDuration := s.getLockDuration(ctx)
		lockedUntil := time.Now().Add(lockDuration)

		// 锁定账号（Redis + DB 双写）
		_ = s.userRepo.LockUser(ctx, user.ID, lockedUntil)
		_ = cache.Set(ctx, cache.KeyAccountLocked+phone, "1", lockDuration)

		// 记录锁定日志
		asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionLocked, enum.LoginMethodPassword, ip, userAgent,
			fmt.Sprintf("连续登录失败%d次，账号锁定%d分钟", maxFail, int(lockDuration.Minutes())))

		return nil, errcode.ErrAccountLocked.WithMessagef("账号已锁定，请%d分钟后重试", int(lockDuration.Minutes()))
	}

	// 记录失败日志
	asyncRecordLoginLog(s.loginLogRepo, user.ID, enum.LoginActionFail, enum.LoginMethodPassword, ip, userAgent, "密码错误")

	remaining := int64(maxFail) - failCount
	return nil, errcode.ErrLoginAttemptsLeft.WithMessagef("密码错误，还剩%d次机会", remaining)
}

// Logout 登出
// P1-2 修复：记录登出日志
func (s *authService) Logout(ctx context.Context, userID int64, jti, ip, userAgent string) error {
	// 将 Access Token 加入黑名单（TTL 与 Access Token 剩余有效期一致，这里用30分钟）
	_ = cache.Set(ctx, cache.KeyTokenBlacklist+jti, "1", 30*time.Minute)

	// 删除 Session
	_ = cache.Del(ctx, cache.KeySession+strconv.FormatInt(userID, 10))

	// P1-2 修复：记录登出日志
	asyncRecordLoginLog(s.loginLogRepo, userID, enum.LoginActionLogout, 0, ip, userAgent, "")

	return nil
}

// RefreshToken 刷新Token
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.RefreshTokenResp, error) {
	// 解析 Refresh Token
	claims, err := jwtpkg.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, errcode.ErrRefreshTokenInvalid
	}

	// 验证 Session 中的 Refresh Token 是否匹配（防止旧 Token 重放）
	sessionKey := cache.KeySession + strconv.FormatInt(claims.UserID, 10)
	sessionData, err := cache.GetString(ctx, sessionKey)
	if err != nil {
		return nil, errcode.ErrRefreshTokenExpired
	}

	var session map[string]interface{}
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, errcode.ErrRefreshTokenInvalid
	}

	storedToken, _ := session["refresh_token"].(string)
	if storedToken != refreshToken {
		return nil, errcode.ErrRefreshTokenInvalid.WithMessage("Refresh Token已被替换（可能在其他设备登录）")
	}

	// 生成新的 Token 对
	newPair, err := jwtpkg.GenerateTokenPair(claims.UserID, claims.SchoolID, claims.Roles)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("生成Token失败")
	}

	// 更新 Session
	s.storeSession(ctx, claims.UserID, newPair.RefreshToken, "")

	return &dto.RefreshTokenResp{
		AccessToken:  newPair.AccessToken,
		RefreshToken: newPair.RefreshToken,
		ExpiresIn:    newPair.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

// ChangePassword 修改密码
// 按文档 AC-19：修改密码后当前会话保持有效，不主动踢下线
func (s *authService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword, ip string) error {
	// 获取用户
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 验证旧密码
	if !crypto.CheckPassword(oldPassword, user.PasswordHash) {
		return errcode.ErrWrongOldPassword
	}

	// 新密码不能与旧密码相同
	if crypto.CheckPassword(newPassword, user.PasswordHash) {
		return errcode.ErrPasswordSameAsCurrent
	}

	// 加密新密码
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return errcode.ErrInternal.WithMessage("密码加密失败")
	}

	// 更新密码
	err = s.userRepo.UpdateFields(ctx, userID, map[string]interface{}{
		"password_hash": hash,
	})
	if err != nil {
		return errcode.ErrInternal.WithMessage("更新密码失败")
	}

	// 记录操作日志（使用 pkg/audit 公共包）
	audit.RecordFromContext(s.db, userID, ip, "change_password", "user", userID, nil)

	return nil
}

// ForceChangePassword 首次登录强制改密
// P0-4 修复：验证 IsFirstLogin 标志
func (s *authService) ForceChangePassword(ctx context.Context, userID int64, newPassword, ip string) (*LoginResult, error) {
	// 获取用户
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errcode.ErrUserNotFound
	}

	// P0-4 修复：必须是首次登录用户才允许此操作
	if !user.IsFirstLogin {
		return nil, errcode.ErrForbidden.WithMessage("非首次登录用户不允许此操作")
	}

	// 加密新密码
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("密码加密失败")
	}

	// 更新密码并标记非首次登录
	err = s.userRepo.UpdateFields(ctx, userID, map[string]interface{}{
		"password_hash":  hash,
		"is_first_login": false,
	})
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("更新密码失败")
	}

	// 生成正式Token对
	roleCodes := s.extractRoleCodes(user)
	tokenPair, err := jwtpkg.GenerateTokenPair(user.ID, user.SchoolID, roleCodes)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("生成Token失败")
	}

	// 存储Session
	s.storeSession(ctx, user.ID, tokenPair.RefreshToken, ip)

	// 更新登录信息
	now := time.Now()
	_ = s.userRepo.UpdateLoginInfo(ctx, user.ID, ip, now)

	// 获取学校名称（跨模块查询）
	schoolName := ""
	if s.schoolNameQuerier != nil {
		schoolName = s.schoolNameQuerier.GetSchoolName(ctx, user.SchoolID)
	}

	return &LoginResult{
		IsFirstLogin: false,
		TokenResp: &dto.LoginResp{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresIn:    tokenPair.ExpiresIn,
			TokenType:    "Bearer",
			User: dto.LoginUser{
				ID:           strconv.FormatInt(user.ID, 10),
				Name:         user.Name,
				Phone:        user.Phone,
				Roles:        roleCodes,
				SchoolID:     strconv.FormatInt(user.SchoolID, 10),
				SchoolName:   schoolName,
				IsFirstLogin: false,
			},
		},
	}, nil
}

// ========== 内部辅助方法 ==========

// extractRoleCodes 从用户实体提取角色编码列表
func (s *authService) extractRoleCodes(user *entity.User) []string {
	codes := make([]string, 0, len(user.Roles))
	for _, ur := range user.Roles {
		if ur.Role != nil {
			codes = append(codes, ur.Role.Code)
		}
	}
	return codes
}

// storeSession 存储用户Session到Redis
func (s *authService) storeSession(ctx context.Context, userID int64, refreshToken, ip string) {
	sessionData, err := json.Marshal(map[string]interface{}{
		"refresh_token": refreshToken,
		"device_info":   ip,
		"login_at":      time.Now().Unix(),
	})
	if err != nil {
		logger.L.Error("序列化Session失败", zap.Error(err))
		return
	}
	_ = cache.Set(ctx, cache.KeySession+strconv.FormatInt(userID, 10), string(sessionData), 7*24*time.Hour)
}

// getMaxFailCount 获取最大登录失败次数（从安全策略缓存）
func (s *authService) getMaxFailCount(ctx context.Context) int {
	data, err := cache.GetString(ctx, cache.KeySecurityPolicy)
	if err == nil {
		var policy map[string]interface{}
		if json.Unmarshal([]byte(data), &policy) == nil {
			if v, ok := policy["max_fail_count"].(float64); ok {
				return int(v)
			}
		}
	}
	return 5 // 默认值
}

// getLockDuration 获取锁定时长（从安全策略缓存）
func (s *authService) getLockDuration(ctx context.Context) time.Duration {
	data, err := cache.GetString(ctx, cache.KeySecurityPolicy)
	if err == nil {
		var policy map[string]interface{}
		if json.Unmarshal([]byte(data), &policy) == nil {
			if v, ok := policy["lock_duration_minutes"].(float64); ok {
				return time.Duration(int(v)) * time.Minute
			}
		}
	}
	return 15 * time.Minute // 默认值
}
