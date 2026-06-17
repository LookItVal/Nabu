package ratelimit

import (
	"context"
	"testing"
)

func TestDefaultBucketConfig_ReturnsExpectedValues(t *testing.T) {
	cfg := DefaultBucketConfig()

	if cfg.Capacity != 10 {
		t.Fatalf("expected default capacity 10, got %d", cfg.Capacity)
	}
	if cfg.LeakRatePerSec != 0.5 {
		t.Fatalf("expected default leak rate 0.5, got %v", cfg.LeakRatePerSec)
	}
}

func TestCheckLeakyBucket_AllowsFirstRequest(t *testing.T) {
	if testRedisClient == nil {
		t.Fatal("expected Redis client to be initialized")
	}

	res, err := checkLeakyBucket(context.Background(), testRedisClient, "203.0.113.20", BucketConfig{Capacity: 1, LeakRatePerSec: 1})
	if err != nil {
		t.Fatalf("expected checkLeakyBucket to succeed, got %v", err)
	}
	if !res.allowed {
		t.Fatal("expected first request to be allowed")
	}
	if res.tokens != 1 {
		t.Fatalf("expected token count 1, got %d", res.tokens)
	}
	if res.capacity != 1 {
		t.Fatalf("expected capacity 1, got %d", res.capacity)
	}
}

func TestCheckLeakyBucket_ReturnsOverflowOnRepeatedRequest(t *testing.T) {
	if testRedisClient == nil {
		t.Fatal("expected Redis client to be initialized")
	}

	ip := "203.0.113.21"
	cfg := BucketConfig{Capacity: 1, LeakRatePerSec: 1}

	first, err := checkLeakyBucket(context.Background(), testRedisClient, ip, cfg)
	if err != nil {
		t.Fatalf("expected first bucket check to succeed, got %v", err)
	}
	if !first.allowed {
		t.Fatal("expected first request to be allowed")
	}

	second, err := checkLeakyBucket(context.Background(), testRedisClient, ip, cfg)
	if err != nil {
		t.Fatalf("expected second bucket check to succeed, got %v", err)
	}
	if second.allowed {
		t.Fatal("expected repeated request to be rejected")
	}
	if second.tokens <= second.capacity {
		t.Fatalf("expected tokens to exceed capacity, got tokens=%d capacity=%d", second.tokens, second.capacity)
	}
	if second.retryAfterMs <= 0 {
		t.Fatalf("expected retryAfterMs to be positive, got %d", second.retryAfterMs)
	}
}
