package v1

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

// v1.RegisterRoutes

func TestRegisterRoutes_ReturnsRouterInstance(t *testing.T) {
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	router := gin.Default()

	RegisterRoutes(router, db, rdb)

	if router == nil {
		t.Fatal("expected RegisterRoutes to return a router instance, got nil")
	}
	if len(router.Routes()) == 0 {
		t.Fatal("expected RegisterRoutes to register routes, but no routes were found")
	}
	if router.Routes()[0].Path != "/v1/status" {
		t.Fatalf("expected first registered route to be /v1/status, got %s", router.Routes()[0].Path)
	}
}
