// redis.go
// 该文件封装后端使用的 Redis 连接、通用读写方法和全项目 Redis Key 前缀规范。它的职责
// 是给上层提供稳定一致的缓存入口，避免各模块直接散落地创建 Redis 客户端或随意定义键名。
// 模块里的验证码、令牌黑名单、限流计数、排行榜、系统健康状态等缓存能力都应复用这里的
// 连接与键前缀约定，而不是重新实现一套缓存访问逻辑。

package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// 全局 Redis 客户端
var rdb *redis.Client

var errRedisNotInitialized = errors.New("Redis客户端未初始化")

// Init 初始化 Redis 连接
func Init(cfg *config.RedisConfig) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis连接失败: %w", err)
	}

	logger.L.Info("Redis连接成功",
		zap.String("addr", cfg.Addr()),
		zap.Int("db", cfg.DB),
	)

	return nil
}

// Get 获取全局 Redis 客户端
func Get() *redis.Client {
	return rdb
}

// Close 关闭 Redis 连接
func Close() error {
	if rdb != nil {
		return rdb.Close()
	}
	return nil
}

// ---- 通用操作封装 ----

// Set 设置键值对
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if rdb == nil {
		return errRedisNotInitialized
	}
	return rdb.Set(ctx, key, value, expiration).Err()
}

// GetString 获取字符串值
func GetString(ctx context.Context, key string) (string, error) {
	if rdb == nil {
		return "", errRedisNotInitialized
	}
	return rdb.Get(ctx, key).Result()
}

// Del 删除键
func Del(ctx context.Context, keys ...string) error {
	if rdb == nil {
		return errRedisNotInitialized
	}
	return rdb.Del(ctx, keys...).Err()
}

// DeleteByPatterns 按通配模式批量删除键。
// 该方法用于归档、租户回收等需要按业务前缀清理一组 Redis 键的场景，统一封装 SCAN + DEL，
// 避免上层业务直接依赖 Redis 游标遍历细节。
func DeleteByPatterns(ctx context.Context, patterns ...string) error {
	if rdb == nil {
		return errRedisNotInitialized
	}
	uniquePatterns := make(map[string]struct{}, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		uniquePatterns[pattern] = struct{}{}
	}
	for pattern := range uniquePatterns {
		var cursor uint64
		for {
			keys, nextCursor, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return err
			}
			if len(keys) > 0 {
				if err := rdb.Del(ctx, keys...).Err(); err != nil {
					return err
				}
			}
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}
	return nil
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	if rdb == nil {
		return false, errRedisNotInitialized
	}
	n, err := rdb.Exists(ctx, key).Result()
	return n > 0, err
}

// TTL 获取键的剩余有效期。
// 用于向上层返回锁定剩余时长、临时状态剩余有效期等业务提示信息。
func TTL(ctx context.Context, key string) (time.Duration, error) {
	if rdb == nil {
		return 0, errRedisNotInitialized
	}
	return rdb.TTL(ctx, key).Result()
}

// Incr 自增计数器
func Incr(ctx context.Context, key string) (int64, error) {
	if rdb == nil {
		return 0, errRedisNotInitialized
	}
	return rdb.Incr(ctx, key).Result()
}

// IncrWithExpire 自增计数器并设置过期时间（原子操作）
// 使用 Lua 脚本保证 INCR + EXPIRE 的原子性，避免竞态条件
func IncrWithExpire(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	if rdb == nil {
		return 0, errRedisNotInitialized
	}
	script := redis.NewScript(`
		local val = redis.call('INCR', KEYS[1])
		if val == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		return val
	`)
	result, err := script.Run(ctx, rdb, []string{key}, int(expiration.Seconds())).Int64()
	if err != nil {
		return 0, fmt.Errorf("IncrWithExpire 执行失败: %w", err)
	}
	return result, nil
}

// SetNX 仅在键不存在时设置（用于分布式锁等场景）
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if rdb == nil {
		return false, errRedisNotInitialized
	}
	return rdb.SetNX(ctx, key, value, expiration).Result()
}

// ---- Redis Key 前缀常量 ----
// 对照 docs/数据库表总览.md 中的 Redis 数据结构定义

const (
	// 模块01 — 用户与认证
	KeySession         = "session:"               // session:{user_id} — Refresh Token
	KeyTokenBlacklist  = "token_blacklist:"       // token_blacklist:{jti} — Access Token 黑名单
	KeyTokenValidAfter = "token_valid_after:"     // token_valid_after:{user_id} — 用户Token生效时间基线
	KeyLoginFail       = "login_fail:"            // login_fail:{phone} — 登录失败计数
	KeyAccountLocked   = "account_locked:"        // account_locked:{phone} — 账号锁定标记
	KeySSOState        = "sso:state:"             // sso:state:{state} — OAuth2 状态校验
	KeySecurityPolicy  = "config:security_policy" // 安全策略配置

	// 模块02 — 学校与租户管理
	KeySchoolStatus            = "school:status:"             // school:status:{school_id}
	KeySchoolExpiring          = "school:expiring_soon"       // Sorted Set
	KeySMSVerification         = "sms_verification:"          // sms_verification:{phone}
	KeySMSVerificationCooldown = "sms_verification:cooldown:" // sms_verification:cooldown:{phone}

	// 模块03 — 课程与教学
	KeyCourseProgressRateLimit = "course:progress:ratelimit:" // course:progress:ratelimit:{user_id}

	// 模块04 — 实验环境
	KeyExpInstanceStatus = "exp:instance:"             // exp:instance:{id}:status
	KeyExpStudentRunning = "exp:student:"              // exp:student:{id}:running_count
	KeyExpCourseConcur   = "exp:course:"               // exp:course:{id}:concurrency
	KeyExpQueue          = "exp:queue:"                // exp:queue:{course_id}
	KeyExpQueueSnapshot  = "exp:queue:snapshot:"       // exp:queue:snapshot:{instance_id}
	KeyExpHeartbeat      = "exp:heartbeat:"            // exp:heartbeat:{instance_id}
	KeyExpMonitor        = "exp:monitor:"              // exp:monitor:{course_id}:{template_id}
	KeyExpSimState       = "exp:sim:"                  // exp:sim:{instance_id}:state
	KeyExpCourseEndWarn  = "exp:course-end-warning:"   // exp:course-end-warning:{instance_id}:{end_unix}
	KeyExpHeartbeatRate  = "exp:heartbeat:ratelimit:"  // exp:heartbeat:ratelimit:{user_id}
	KeyExpCheckpointRate = "exp:checkpoint:ratelimit:" // exp:checkpoint:ratelimit:{user_id}

	// 模块05 — CTF竞赛
	KeyCTFLeaderboard   = "ctf:leaderboard:"    // ctf:leaderboard:{competition_id}
	KeyCTFADLeaderboard = "ctf:ad:leaderboard:" // ctf:ad:leaderboard:{comp_id}:{group_id}
	KeyCTFFrozen        = "ctf:leaderboard:"    // ctf:leaderboard:{comp_id}:frozen
	KeyCTFRateLimit     = "ctf:ratelimit:"      // ctf:ratelimit:{comp}:{team}:{challenge}
	KeyCTFFailCount     = "ctf:failcount:"      // ctf:failcount:{comp}:{team}:{challenge}
	KeyCTFCooldown      = "ctf:cooldown:"       // ctf:cooldown:{comp}:{team}:{challenge}
	KeyCTFScore         = "ctf:score:"          // ctf:score:{comp_id}:{challenge_id}
	KeyCTFADRound       = "ctf:ad:round:"       // ctf:ad:round:{comp_id}:{group_id}
	KeyCTFADToken       = "ctf:ad:token:"       // ctf:ad:token:{comp_id}:{team_id}
	KeyCTFADExploit     = "ctf:ad:exploit:"     // ctf:ad:exploit:{comp}:{group}:{challenge}
	KeyCTFADAttackLock  = "ctf:ad:attack_lock:" // ctf:ad:attack_lock:{round}:{target_team}:{challenge}
	KeyCTFCompStatus    = "ctf:competition:"    // ctf:competition:{comp_id}:status
	KeyCTFSolved        = "ctf:solved:"         // ctf:solved:{comp}:{team}:{challenge}

	// 模块06 — 评测与成绩
	KeyGradeLevelConfig = "grade_level_config:" // grade_level_config:{school_id}
	KeySemesterCurrent  = "semester:current:"   // semester:current:{school_id}
	KeyStudentGPA       = "grade:student_gpa:"  // grade:student_gpa:{student_id}:{semester_id}
	KeyReviewLock       = "grade:review_lock:"  // grade:review_lock:{course_id}:{semester_id}

	// 模块07 — 通知与消息
	KeyNotifyUnread     = "notify:unread:"     // notify:unread:{user_id}
	KeyNotifyPreference = "notify:preference:" // notify:preference:{user_id}
	KeyNotifyTemplate   = "notify:template:"   // notify:template:{event_code}
	KeyNotifyDedup      = "notify:dedup:"      // notify:dedup:{event_code}:{receiver}:{hash}

	// 模块08 — 系统管理与监控
	KeySystemConfig     = "system_config:"           // system_config:{group}:{key}
	KeyAlertLastTrigger = "alert_rule:last_trigger:" // alert_rule:last_trigger:{rule_id}
	KeyPlatformStats    = "platform_stats:today"     // 今日统计缓存
	KeyServiceHealth    = "service_health:"          // service_health:{service_name}
)
