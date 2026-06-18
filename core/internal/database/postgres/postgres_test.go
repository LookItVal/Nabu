package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/config"
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

// postgres.Connect

func TestConnect_ReturnsDBOnSuccess(t *testing.T) {
	db, err := Connect()
	if err != nil {
		t.Fatalf("expected Connect to return nil error, got %v", err)
	}
	if db == nil {
		t.Fatal("expected Connect to return a database connection, got nil")
	}
	defer db.Close()
}

func TestConnect_ReturnsErrorOnFailure(t *testing.T) {
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

func TestConnect_ErrorOnInvalidPort(t *testing.T) {
	originalPort := fmt.Sprintf("%d", config.Load().PGPort)
	os.Setenv("PG_PORT", "1234567890")
	defer func() { os.Setenv("PG_PORT", originalPort) }()

	db, err := Connect()
	if err == nil {
		t.Fatal("expected Connect to return an error for invalid port, got nil")
	}
	if db != nil {
		t.Fatal("expected Connect to return nil database connection for invalid port, got non-nil")
	}
}

// postgres.Ping

func TestPing_ReturnsNegativeOnNilDB(t *testing.T) {
	if got := Ping(nil); got != -1 {
		t.Fatalf("expected Ping(nil) to return -1, got %d", got)
	}
}

func TestPing_ReturnsLatencyOnSuccess(t *testing.T) {
	db, err := Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if got := Ping(db); got < 0 {
		t.Fatalf("expected Ping to return non-negative latency, got %d", got)
	}
}

func TestPing_ReturnsNegativeOnClosedDB(t *testing.T) {
	db, err := Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	_ = db.Close()

	if got := Ping(db); got != -1 {
		t.Fatalf("expected Ping on closed DB to return -1, got %d", got)
	}
}
