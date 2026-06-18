package config

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/testutils"
)

// TestMain sets up the test environment using testcontainers before running the tests.
func TestMain(m *testing.M) {
	// Start the test environment with PostgreSQL and Redis containers.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := testutils.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test environment: %v\n", err)
		cancel()
		os.Exit(1)
	}

	// set environment variables for the application to connect to the test containers
	if err := testutils.SetEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set test environment variables: %v\n", err)
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		_ = testutils.Stop(ctx)
		cancel()
		os.Exit(1)
	}
	cancel()

	// Run the tests
	code := m.Run()

	// Teardown the test environment after tests complete.
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	if err := testutils.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop test environment: %v\n", err)
	}
	cancel()

	os.Exit(code)
}

// config.Load

func TestLoad_ReturnsConfigWithExpectedValues(t *testing.T) {
	cfg := Load()

	if cfg.PGHost == "" {
		t.Error("expected PGHost to be set, got empty string")
	}
	if cfg.PGPort == 0 {
		t.Error("expected PGPort to be set, got 0")
	}
	if cfg.PGUser == "" {
		t.Error("expected PGUser to be set, got empty string")
	}
	if cfg.PGPass == "" {
		t.Error("expected PGPass to be set, got empty string")
	}
	if cfg.PGDB == "" {
		t.Error("expected PGDB to be set, got empty string")
	}
	if cfg.RedisAddr == "" {
		t.Error("expected RedisAddr to be set, got empty string")
	}
	if cfg.RedisDB < 0 {
		t.Errorf("expected RedisDB to be non-negative, got %d", cfg.RedisDB)
	}
}

func TestLoad_HandleNonNumericPgPort(t *testing.T) {
	originalPgPort := os.Getenv("PG_PORT")
	t.Setenv("PG_PORT", "not-a-number")
	defer t.Setenv("PG_PORT", originalPgPort)

	cfg := Load()

	if cfg.PGPort != 5432 {
		t.Errorf("expected PGPort to be 5432 on non-numeric input, got %d", cfg.PGPort)
	}
}

func TestLoad_HandleNonNumericRedisDB(t *testing.T) {
	originalRedisDB := os.Getenv("REDIS_DB")
	t.Setenv("REDIS_DB", "not-a-number")
	defer t.Setenv("REDIS_DB", originalRedisDB)

	cfg := Load()

	if cfg.RedisDB != 0 {
		t.Errorf("expected RedisDB to be 0 on non-numeric input, got %d", cfg.RedisDB)
	}
}
