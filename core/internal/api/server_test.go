package api

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	"github.com/lookitval/nabu/core/internal/database/redisdb"
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

// api.NewServer

func TestNewServer_ReturnsServerInstance(t *testing.T) {
	cfg := config.Load()
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	srv := NewServer(cfg, db, rdb)
	if srv == nil {
		t.Fatal("expected NewServer to return a server instance, got nil")
	}
}

func TestNewServer_ReturnsServerWithExpectedFields(t *testing.T) {
	cfg := config.Load()
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	srv := NewServer(cfg, db, rdb)
	if srv == nil {
		t.Fatal("expected NewServer to return a server instance, got nil")
	}
	if srv.router == nil {
		t.Error("expected server router to be initialized, got nil")
	}
	if srv.db == nil {
		t.Error("expected server database connection to be initialized, got nil")
	}
	if srv.rdb == nil {
		t.Error("expected server Redis client to be initialized, got nil")
	}
	if srv.cfg == nil {
		t.Error("expected server configuration to be initialized, got nil")
	}
}

// api.Server.Run

func TestRun_ReturnsNilOnSuccessfulStart(t *testing.T) {
	cfg := config.Load()
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	srv := NewServer(cfg, db, rdb)
	defer srv.Close()

	runErr := make(chan error, 1)
	go func() {
		runErr <- srv.Run()
	}()

	// Wait briefly to ensure server starts without immediate error
	select {
	case err := <-runErr:
		t.Fatalf("server exited unexpectedly during startup: %v", err)
	case <-time.After(100 * time.Millisecond):
		// server ran properly without error
	}
}

// api.Server.Close

func TestClose_ShutsDownRunningServerAndReturnsNil(t *testing.T) {
	cfg := config.Load()
	cfg.Port = "31338"

	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	srv := NewServer(cfg, db, rdb)

	runErr := make(chan error, 1)
	go func() {
		runErr <- srv.Run()
	}()

	// Give it a moment to boot
	time.Sleep(100 * time.Millisecond)

	// Call close to trigger graceful shutdown
	srv.Close()

	// Wait for Run to return naturally
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("expected Run to return nil after graceful shutdown, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("srv.Run() did not return in a timely manner after srv.Close() was called")
	}
}

func TestClose_DoesNotPanicOnUnstartedServer(t *testing.T) {
	cfg := config.Load()
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	srv := NewServer(cfg, db, rdb)

	// Close before Run is ever called
	// Should do safely what it can (close db connections) without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Close panicked on unstarted server: %v", r)
		}
	}()

	srv.Close()
}

func TestClose_WithNilDependencies_DoesNotPanic(t *testing.T) {
	srv := &Server{}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Close panicked with nil dependencies: %v", r)
		}
	}()

	srv.Close()
}
