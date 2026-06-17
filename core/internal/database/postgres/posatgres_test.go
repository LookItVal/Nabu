package postgres

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

// postgres.Connect

func TestConnect_ReturnsDBOnSuccess(t *testing.T) {
	db, err := Connect()
	if err != nil {
		t.Fatalf("expected Connect to return nil error, got %v", err)
	}
	if db == nil {
		t.Fatal("expected Connect to return a database connection, got nil")
	}
}

func TestConnect_ReturnsErrorOnFailure(t *testing.T) {
	// Temporarily override the configuration to an invalid address
	originalHost := config.Load().PGHost
	os.Setenv("PG_HOST", "invalid_host")
	defer func() { os.Setenv("PG_HOST", originalHost) }()

	db, err := Connect()
	if err == nil {
		t.Fatal("expected Connect to return an error for invalid host, got nil")
	}
	if db != nil {
		t.Fatal("expected Connect to return nil database connection for invalid host, got non-nil")
	}
}

// postgres.Ping
