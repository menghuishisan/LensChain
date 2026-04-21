// security_service.go
// 模块01 — 用户与认证：安全策略与日志业务逻辑
// 负责安全策略配置的读取/更新、登录日志和操作日志的查询
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
)

// SecurityService 安全策略服务接口
type SecurityService interface {
	GetSecurityPolicy(ctx context.Context) (*dto.SecurityPolicyResp, error)
	UpdateSecurityPolicy(ctx context.Context, req *dto.UpdateSecurityPolicyReq) error
	ListLoginLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.LoginLogListReq) ([]*dto.LoginLogItem, int64, error)
	ListOperationLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.OperationLogListReq) ([]*dto.OperationLogItem, int64, error)
}

// securityService 安全策略服务实现
type securityService struct {
	loginLogRepo authrepo.LoginLogRepository
	opLogRepo    authrepo.OperationLogRepository
	userRepo     authrepo.UserRepository
}

// NewSecurityService 创建安全策略服务实例
func NewSecurityService(
	loginLogRepo authrepo.LoginLogRepository,
	opLogRepo authrepo.OperationLogRepository,
	userRepo authrepo.UserRepository,
) SecurityService {
	return &securityService{
		loginLogRepo: loginLogRepo,
		opLogRepo:    opLogRepo,
		userRepo:     userRepo,
	}
}

// defaultSecurityPolicy 默认安全策略
var defaultSecurityPolicy = &dto.SecurityPolicyResp{
	LoginFailMaxCount:          5,
	LoginLockDurationMinutes:   15,
	PasswordMinLength:          8,
	PasswordRequireUppercase:   true,
	PasswordRequireLowercase:   true,
	PasswordRequireDigit:       true,
	PasswordRequireSpecialChar: false,
	AccessTokenExpireMinutes:   30,
	RefreshTokenExpireDays:     7,
}

// GetSecurityPolicy 获取安全策略配置
func (s *securityService) GetSecurityPolicy(ctx context.Context) (*dto.SecurityPolicyResp, error) {
	data, err := cache.GetString(ctx, cache.KeySecurityPolicy)
	if err != nil {
		// Redis 中没有，返回默认值
		policy := *defaultSecurityPolicy
		return &policy, nil
	}

	var policy dto.SecurityPolicyResp
	if err := json.Unmarshal([]byte(data), &policy); err != nil {
		logger.L.Error("解析安全策略缓存失败", zap.Error(err))
		fallback := *defaultSecurityPolicy
		return &fallback, nil
	}

	return &policy, nil
}

// UpdateSecurityPolicy 更新安全策略配置（支持部分更新）
func (s *securityService) UpdateSecurityPolicy(ctx context.Context, req *dto.UpdateSecurityPolicyReq) error {
	// 获取当前策略
	current, _ := s.GetSecurityPolicy(ctx)

	// 部分更新
	if req.LoginFailMaxCount != nil {
		current.LoginFailMaxCount = *req.LoginFailMaxCount
	}
	if req.LoginLockDurationMinutes != nil {
		current.LoginLockDurationMinutes = *req.LoginLockDurationMinutes
	}
	if req.PasswordMinLength != nil {
		current.PasswordMinLength = *req.PasswordMinLength
	}
	if req.PasswordRequireUppercase != nil {
		current.PasswordRequireUppercase = *req.PasswordRequireUppercase
	}
	if req.PasswordRequireLowercase != nil {
		current.PasswordRequireLowercase = *req.PasswordRequireLowercase
	}
	if req.PasswordRequireDigit != nil {
		current.PasswordRequireDigit = *req.PasswordRequireDigit
	}
	if req.PasswordRequireSpecialChar != nil {
		current.PasswordRequireSpecialChar = *req.PasswordRequireSpecialChar
	}
	if req.AccessTokenExpireMinutes != nil {
		current.AccessTokenExpireMinutes = *req.AccessTokenExpireMinutes
	}
	if req.RefreshTokenExpireDays != nil {
		current.RefreshTokenExpireDays = *req.RefreshTokenExpireDays
	}

	// 序列化并存储到 Redis（无过期时间，配置变更时主动刷新）
	data, err := json.Marshal(current)
	if err != nil {
		return errcode.ErrInternal.WithMessage("序列化安全策略失败")
	}

	if err := cache.Set(ctx, cache.KeySecurityPolicy, string(data), 0); err != nil {
		return errcode.ErrInternal.WithMessage("保存安全策略失败")
	}

	return nil
}

// ListLoginLogs 登录日志列表
func (s *securityService) ListLoginLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.LoginLogListReq) ([]*dto.LoginLogItem, int64, error) {
	// 解析 user_id
	var userID int64
	if req.UserID != "" {
		id, err := snowflake.ParseString(req.UserID)
		if err != nil || id <= 0 {
			return nil, 0, errcode.ErrInvalidParams.WithMessage("user_id 格式不正确")
		}
		userID = id
	}

	params := &authrepo.LoginLogListParams{
		SchoolID:    sc.SchoolID,
		UserID:      userID,
		Action:      req.Action,
		CreatedFrom: req.CreatedFrom,
		CreatedTo:   req.CreatedTo,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}

	logs, total, err := s.loginLogRepo.List(ctx, params)
	if err != nil {
		return nil, 0, errcode.ErrInternal.WithMessage("查询登录日志失败")
	}

	// 收集用户ID，批量查询用户名
	userIDs := make([]int64, 0, len(logs))
	for _, log := range logs {
		userIDs = append(userIDs, log.UserID)
	}
	userNameMap := s.getUserNameMap(ctx, userIDs)

	// 转换为 DTO
	items := make([]*dto.LoginLogItem, 0, len(logs))
	for _, log := range logs {
		item := &dto.LoginLogItem{
			ID:          strconv.FormatInt(log.ID, 10),
			UserID:      strconv.FormatInt(log.UserID, 10),
			UserName:    userNameMap[log.UserID],
			Action:      log.Action,
			ActionText:  enum.GetLoginActionText(log.Action),
			LoginMethod: log.LoginMethod,
			IP:          log.IP,
			UserAgent:   log.UserAgent,
			FailReason:  log.FailReason,
			CreatedAt:   log.CreatedAt.Format(time.RFC3339),
		}
		if log.LoginMethod != nil {
			text := enum.GetLoginMethodText(*log.LoginMethod)
			item.LoginMethodText = &text
		}
		items = append(items, item)
	}

	return items, total, nil
}

// ListOperationLogs 操作日志列表
func (s *securityService) ListOperationLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.OperationLogListReq) ([]*dto.OperationLogItem, int64, error) {
	// 解析 operator_id
	var operatorID int64
	if req.OperatorID != "" {
		id, err := snowflake.ParseString(req.OperatorID)
		if err != nil || id <= 0 {
			return nil, 0, errcode.ErrInvalidParams.WithMessage("operator_id 格式不正确")
		}
		operatorID = id
	}

	params := &authrepo.OperationLogListParams{
		SchoolID:    sc.SchoolID,
		OperatorID:  operatorID,
		Action:      req.Action,
		TargetType:  req.TargetType,
		CreatedFrom: req.CreatedFrom,
		CreatedTo:   req.CreatedTo,
		Page:        req.Page,
		PageSize:    req.PageSize,
	}

	logs, total, err := s.opLogRepo.List(ctx, params)
	if err != nil {
		return nil, 0, errcode.ErrInternal.WithMessage("查询操作日志失败")
	}

	// 收集操作人ID，批量查询用户名
	operatorIDs := make([]int64, 0, len(logs))
	for _, log := range logs {
		operatorIDs = append(operatorIDs, log.OperatorID)
	}
	userNameMap := s.getUserNameMap(ctx, operatorIDs)

	// 转换为 DTO
	items := make([]*dto.OperationLogItem, 0, len(logs))
	for _, log := range logs {
		var detail *string
		if len(log.Detail) > 0 {
			text := string(log.Detail)
			detail = &text
		}
		item := &dto.OperationLogItem{
			ID:           strconv.FormatInt(log.ID, 10),
			OperatorID:   strconv.FormatInt(log.OperatorID, 10),
			OperatorName: userNameMap[log.OperatorID],
			Action:       log.Action,
			TargetType:   log.TargetType,
			Detail:       detail,
			IP:           log.IP,
			CreatedAt:    log.CreatedAt.Format(time.RFC3339),
		}
		if log.TargetID != nil {
			tid := strconv.FormatInt(*log.TargetID, 10)
			item.TargetID = &tid
		}
		items = append(items, item)
	}

	return items, total, nil
}

// getUserNameMap 批量获取用户名映射
func (s *securityService) getUserNameMap(ctx context.Context, userIDs []int64) map[int64]string {
	nameMap := make(map[int64]string)
	if len(userIDs) == 0 {
		return nameMap
	}

	// 去重
	uniqueIDs := make(map[int64]bool)
	deduped := make([]int64, 0)
	for _, id := range userIDs {
		if !uniqueIDs[id] && id > 0 {
			uniqueIDs[id] = true
			deduped = append(deduped, id)
		}
	}

	users, err := s.userRepo.GetByIDsIncludingDeleted(ctx, deduped)
	if err != nil {
		return nameMap
	}

	for _, user := range users {
		nameMap[user.ID] = user.Name
	}

	return nameMap
}
