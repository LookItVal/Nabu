package redisdb

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/testenv"
)

// TestMain sets up the test environment using testcontainers before running the tests.
func TestMain(m *testing.M) {
	// Start the test environment with PostgreSQL and Redis containers.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := testenv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test environment: %v\n", err)
		cancel()
		os.Exit(1)
	}

	// set environment variables for the application to connect to the test containers
	if err := testenv.SetEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set test environment variables: %v\n", err)
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		_ = testenv.Stop(ctx)
		cancel()
		os.Exit(1)
	}
	cancel()

	// Run the tests
	code := m.Run()

	// Teardown the test environment after tests complete.
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	if err := testenv.Stop(ctx); err != nil {
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
	// Temporarily override the configuration to an invalid address
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
