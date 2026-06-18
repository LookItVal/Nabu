package redisdb

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/testutils"
	"github.com/redis/go-redis/v9"
)

// TestMain sets up the test environment using testcontainers before running the tests.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := testutils.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test environment: %v\n", err)
		cancel()
		os.Exit(1)
	}

	if err := testutils.SetEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set test environment variables: %v\n", err)
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		_ = testutils.Stop(ctx)
		cancel()
		os.Exit(1)
	}
	cancel()

	code := m.Run()

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	if err := testutils.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop test environment: %v\n", err)
	}
	cancel()

	os.Exit(code)
}

// redisdb.Connect

func TestConnect_ReturnsClientOnSuccess(t *testing.T) {
	rdb, err := Connect()
	if err != nil {
		t.Fatalf("expected Connect to return nil error, got %v", err)
	}
	if rdb == nil {
		t.Fatal("expected Connect to return a Redis client, got nil")
	}
}

func TestConnect_ReturnsErrorOnFailure(t *testing.T) {
	originalAddr := config.Load().RedisAddr
	os.Setenv("REDIS_ADDRESS", "invalid:6379")
	defer os.Setenv("REDIS_ADDRESS", originalAddr)

	rdb, err := Connect()
	if err == nil {
		t.Fatal("expected Connect to return an error for invalid address, got nil")
	}
	if rdb != nil {
		t.Fatal("expected Connect to return nil Redis client for invalid address, got non-nil")
	}
}

// redisdb.Ping

func TestPing_ReturnsLatencyOnSuccess(t *testing.T) {
	rdb, err := Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}

	latency := Ping(rdb)
	if latency < 0 {
		t.Fatalf("expected Ping to return non-negative latency, got %d", latency)
	}
}

func TestPing_ReturnsNegativeOnFailure(t *testing.T) {
	latency := Ping(nil)
	if latency >= 0 {
		t.Fatalf("expected Ping to return negative latency for nil client, got %d", latency)
	}
}

func TestPing_ReturnsNegativeOnNilClient(t *testing.T) {
	if got := Ping(nil); got != -1 {
		t.Fatalf("expected Ping(nil) to return -1, got %d", got)
	}
}

func TestPing_ReturnsNegativeOnClosedClient(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	_ = rdb.Close()

	if got := Ping(rdb); got != -1 {
		t.Fatalf("expected Ping on closed client to return -1, got %d", got)
	}
}
