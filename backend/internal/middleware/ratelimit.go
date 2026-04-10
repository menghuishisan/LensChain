// ratelimit.go
// 限流中间件
// 基于令牌桶算法（golang.org/x/time/rate）
// 全局限流 + 按IP限流，防止接口被恶意刷取
// IP限流器带 LRU 淘汰机制，防止内存泄漏

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

// ipLimiters 按IP限流器缓存
var (
	ipLimiters = make(map[string]*ipEntry)
	ipMu       sync.Mutex
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

	// 启动后台清理协程
	go cleanupIPLimiters()

	return func(c *gin.Context) {
		// 全局限流检查
		if !globalLimiter.Allow() {
			response.Abort(c, errcode.New(42900, 429, "请求过于频繁，请稍后再试"))
			return
		}

		// 按IP限流检查（每个IP每秒10个请求）
		ip := c.ClientIP()
		limiter := getIPLimiter(ip)
		if !limiter.Allow() {
			response.Abort(c, errcode.New(42901, 429, "您的请求过于频繁，请稍后再试"))
			return
		}

		c.Next()
	}
}

// getIPLimiter 获取或创建IP限流器
func getIPLimiter(ip string) *rate.Limiter {
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

	// 每个IP每秒10个请求，突发20个
	limiter := rate.NewLimiter(10, 20)
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
