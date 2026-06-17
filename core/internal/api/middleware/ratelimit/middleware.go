package ratelimit

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// IPRateLimiter returns a Gin middleware that applies leaky bucket rate limiting per client IP.
// When Redis is unavailable, requests are allowed through to avoid a hard dependency on Redis
func IPRateLimiter(rdb *redis.Client, cfg BucketConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		ctx, cancel := context.WithTimeout(c.Request.Context(), 100*time.Millisecond)
		defer cancel()

		res, err := checkLeakyBucket(ctx, rdb, ip, cfg)
		if err != nil {
			// Redis unavailable — fail close and log
			fmt.Printf("Rate limiter error for IP %s: %v\n", ip, err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "rate limiter unavailable",
			})
			return
		}

		// Always set informational headers so clients can self-throttle
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", res.capacity))
		c.Header("X-RateLimit-Overflow", fmt.Sprintf("%d", res.tokens-res.capacity))

		if !res.allowed {
			retryAfterSecs := int(math.Ceil(float64(res.retryAfterMs) / 1000))
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfterSecs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfterSecs,
			})
			return
		}

		c.Next()
	}
}
