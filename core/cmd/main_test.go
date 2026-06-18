package main

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

func TestMainDoesNotCrash(t *testing.T) {
	t.Setenv("PORT", "58081")
	testutils.CaptureAndWaitForOutput(t, "Starting server on port 58081", 15*time.Second, func() {
		main()
	})
}

func TestMainHandlesPostgresFailure(t *testing.T) {
	t.Setenv("PORT", "58082")
	t.Setenv("PG_HOST", "invalid_connection_string")
	testutils.CaptureAndWaitForOutput(t, "WARNING: Failed to initialize postgres:", 15*time.Second, func() {
		main()
	})
}

func TestMainHandlesRedisFailure(t *testing.T) {
	t.Setenv("PORT", "58083")
	t.Setenv("REDIS_ADDRESS", "invalid_connection_string")
	testutils.CaptureAndWaitForOutput(t, "WARNING: Failed to initialize redis:", 15*time.Second, func() {
		main()
	})
}

func TestMainHandlesRunFailure(t *testing.T) {
	t.Setenv("PORT", "invalid_port")
	testutils.CaptureAndWaitForOutput(t, "ERROR: API server failed:", 15*time.Second, func() {
		main()
	})
}
