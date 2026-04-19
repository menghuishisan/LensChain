// ratelimit.go
// 该文件实现平台级 HTTP 限流中间件，负责按照统一配置执行全局限流和按来源 IP 的基础限流，
// 防止单个客户端或突发流量冲垮接口。它解决的是通用请求保护问题，接口级的业务限流规则
// 仍应由对应模块在自己的 service 或专用能力里实现。

package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/response"
)

// ipEntry IP限流器条目（含最后访问时间）
type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	ipLimiters          = make(map[string]*ipEntry)
	ipMu                sync.Mutex
	rateLimiterInitOnce sync.Once
)

// maxIPEntries IP限流器最大缓存数量
const maxIPEntries = 10000

// ipCleanupInterval IP限流器清理间隔
const ipCleanupInterval = 5 * time.Minute

// ipEntryTTL IP限流器条目过期时间
const ipEntryTTL = 10 * time.Minute

// RateLimit 限流中间件
func RateLimit() gin.HandlerFunc {
	cfg := config.Get().RateLimit
	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 全局限流器
	globalLimiter := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst)

	// 只在首次启用时启动一次后台清理协程，避免重复构建路由时产生协程泄漏。
	rateLimiterInitOnce.Do(func() {
		go cleanupIPLimiters()
	})

	return func(c *gin.Context) {
		// 全局限流检查
		if !globalLimiter.Allow() {
			response.Abort(c, errcode.New(42900, 429, "请求过于频繁，请稍后再试"))
			return
		}

		// 按 IP 限流检查，避免单个来源独占全局配额。
		ip := c.ClientIP()
		limiter := getIPLimiter(ip, cfg)
		if !limiter.Allow() {
			response.Abort(c, errcode.New(42901, 429, "您的请求过于频繁，请稍后再试"))
			return
		}

		c.Next()
	}
}

// getIPLimiter 获取或创建IP限流器
func getIPLimiter(ip string, cfg config.RateLimitConfig) *rate.Limiter {
	ipMu.Lock()
	defer ipMu.Unlock()

	if entry, exists := ipLimiters[ip]; exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// 如果缓存已满，先清理过期条目
	if len(ipLimiters) >= maxIPEntries {
		evictOldestIPEntries()
	}

	// 单 IP 配额按全局限流配置收敛，避免硬编码与配置脱节。
	perIPRPS := cfg.RequestsPerSecond
	if perIPRPS <= 0 || perIPRPS > 10 {
		perIPRPS = 10
	}
	perIPBurst := cfg.Burst
	if perIPBurst <= 0 || perIPBurst > 20 {
		perIPBurst = 20
	}

	limiter := rate.NewLimiter(rate.Limit(perIPRPS), perIPBurst)
	ipLimiters[ip] = &ipEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	return limiter
}

// cleanupIPLimiters 定期清理过期的IP限流器
func cleanupIPLimiters() {
	ticker := time.NewTicker(ipCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ipMu.Lock()
		now := time.Now()
		for ip, entry := range ipLimiters {
			if now.Sub(entry.lastSeen) > ipEntryTTL {
				delete(ipLimiters, ip)
			}
		}
		ipMu.Unlock()
	}
}

// evictOldestIPEntries 淘汰最久未访问的IP限流器条目
// 当缓存达到上限时调用，清理一半过期条目
func evictOldestIPEntries() {
	now := time.Now()
	// 先清理过期的
	for ip, entry := range ipLimiters {
		if now.Sub(entry.lastSeen) > ipEntryTTL {
			delete(ipLimiters, ip)
		}
	}
	// 如果还是超限，清理最旧的一半
	if len(ipLimiters) >= maxIPEntries {
		oldest := now
		for _, entry := range ipLimiters {
			if entry.lastSeen.Before(oldest) {
				oldest = entry.lastSeen
			}
		}
		mid := oldest.Add(now.Sub(oldest) / 2)
		for ip, entry := range ipLimiters {
			if entry.lastSeen.Before(mid) {
				delete(ipLimiters, ip)
			}
		}
	}
}
