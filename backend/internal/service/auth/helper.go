// helper.go
// 模块01 — 用户与认证：服务层公共辅助函数
// 仅保留 auth 模块专属的登录日志记录函数
// 通用脱敏（mask）、操作日志（audit）已迁移至 internal/pkg/ 公共包

package auth

import (
	"context"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/repository/auth"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// asyncRecordLoginLog 异步记录登录日志（auth 模块专属）
// 登录日志仅由认证流程（登录、登出、SSO）写入，不跨模块使用
func asyncRecordLoginLog(loginLogRepo authrepo.LoginLogRepository, userID int64, action, loginMethod int, ip, userAgent, failReason string) {
	go func() {
		log := &entity.LoginLog{
			ID:     snowflake.Generate(),
			UserID: userID,
			Action: action,
			IP:     ip,
		}
		if loginMethod > 0 {
			log.LoginMethod = &loginMethod
		}
		if userAgent != "" {
			log.UserAgent = &userAgent
		}
		if failReason != "" {
			log.FailReason = &failReason
		}
		if err := loginLogRepo.Create(context.Background(), log); err != nil {
			logger.L.Error("记录登录日志失败", zap.Error(err))
		}
	}()
}
