package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/service"
	"github.com/yockii/dify_tools/pkg/logger"
)

type rateLimiter struct {
	maxRequests int
	duration    time.Duration
	mu          sync.RWMutex
	tokens      map[string]*tokenBucket
}

type tokenBucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(maxRequests int, duration time.Duration) *rateLimiter {
	return &rateLimiter{
		maxRequests: maxRequests,
		duration:    duration,
		tokens:      make(map[string]*tokenBucket),
	}
}

// RateLimit 限流中间件
func RateLimit(maxRequests int, duration time.Duration) fiber.Handler {
	limiter := NewRateLimiter(maxRequests, duration)

	return func(c *fiber.Ctx) error {
		// 获取客户端标识（IP或用户ID）
		clientID := c.IP()
		if user, ok := c.Locals("user").(*model.User); ok && user != nil {
			clientID = fmt.Sprintf("user_%d", user.ID)
		}

		// 检查限流
		if !limiter.allow(clientID) {
			logger.Warn("rate limit exceeded",
				logger.F("clientId", clientID),
				logger.F("path", c.Path()),
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(service.NewResponse(nil, fiber.ErrTooManyRequests))
		}

		return c.Next()
	}
}

// allow 检查是否允许请求
func (rl *rateLimiter) allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.tokens[clientID]

	if !exists {
		// 新客户端，创建令牌桶
		rl.tokens[clientID] = &tokenBucket{
			tokens:    rl.maxRequests - 1, // 减1是因为当前请求
			lastReset: now,
		}
		return true
	}

	// 检查是否需要重置令牌
	if now.Sub(bucket.lastReset) >= rl.duration {
		bucket.tokens = rl.maxRequests - 1
		bucket.lastReset = now
		return true
	}

	// 检查令牌是否足够
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanup 清理过期的令牌桶
func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for clientID, bucket := range rl.tokens {
		if now.Sub(bucket.lastReset) >= rl.duration*2 {
			delete(rl.tokens, clientID)
		}
	}
}

// StartCleanup 启动清理任务
func (rl *rateLimiter) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			rl.cleanup()
		}
	}()
}
