// Package ratelimit implements IP-based rate limiting middleware for the Gin HTTP framework.
// It uses a leaky bucket algorithm backed by Redis, keeping the application layer fully stateless.
package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed leaky_bucket.lua
var leakyBucketScript string

// BucketConfig holds the leaky bucket parameters.
type BucketConfig struct {
	// Capacity is the maximum number of tokens the bucket can hold.
	Capacity int
	// LeakRatePerSec is the number of tokens that leak (are restored) per second.
	LeakRatePerSec float64
}

// DefaultBucketConfig returns the standard rate limit configuration:
// 10-token capacity with a leak rate of 1 token per second.
func DefaultBucketConfig() BucketConfig {
	return BucketConfig{
		Capacity:       10,
		LeakRatePerSec: 0.1,
	}
}

// bucketResult holds the parsed response from the Lua script.
type bucketResult struct {
	allowed      bool
	tokens       int
	capacity     int
	retryAfterMs int64
}

// checkLeakyBucket runs the leaky bucket Lua script against Redis for the given IP key.
// It returns the result of the check and any Redis error encountered.
func checkLeakyBucket(ctx context.Context, rdb *redis.Client, ip string, cfg BucketConfig) (bucketResult, error) {
	key := fmt.Sprintf("ratelimit:ip:%s", ip)
	nowMs := time.Now().UnixMilli()

	result, err := rdb.Eval(ctx, leakyBucketScript,
		[]string{key},
		cfg.Capacity,
		cfg.LeakRatePerSec,
		nowMs,
	).Int64Slice()

	if err != nil {
		return bucketResult{}, err
	}

	retryAfterMs := int64(float64(int(result[1])-int(result[2])) / cfg.LeakRatePerSec * 1000)
	res := bucketResult{
		allowed:      result[0] == 1,
		tokens:       int(result[1]),
		capacity:     int(result[2]),
		retryAfterMs: retryAfterMs,
	}

	return res, nil
}
