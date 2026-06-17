package testenv

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func resetForTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	Stop(ctx)
	cancel()
	instance = nil
	once = sync.Once{}
}

// testenv.Start

func TestStart_ReturnsNilErrorOnSuccess(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := Start(ctx); err != nil {
		t.Fatalf("expected Start to return nil error, got %v", err)
	}
}

func TestStart_ReturnsErrorOnContextDeadlineExceeded(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := Start(ctx); err == nil {
		t.Fatal("expected Start to return error on context deadline exceeded, got nil")
	}
}

// testenv.Get

func TestGet_ReturnsNilWhenNotStarted(t *testing.T) {
	resetForTest()

	if got := Get(); got != nil {
		t.Fatalf("expected nil environment, got %#v", got)
	}
}

func TestGet_ReturnsEnvironmentInstance(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := Start(ctx); err != nil {
		t.Fatalf("failed to start environment: %v", err)
	}

	got := Get()
	if got == nil {
		t.Fatal("expected non-nil environment instance, got nil")
	}
	if got != instance {
		t.Fatalf("expected Get to return the initialized instance, got %#v", got)
	}
}

// testenv.Stop

func TestStop_ReturnsNilWhenNotStarted(t *testing.T) {
	resetForTest()

	if err := Stop(context.Background()); err != nil {
		t.Fatalf("expected nil error from Stop when not started, got %v", err)
	}
}

func TestStop_ReturnsNilOnSuccessfulTermination(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := Start(ctx); err != nil {
		t.Fatalf("failed to start environment: %v", err)
	}

	if err := Stop(ctx); err != nil {
		t.Fatalf("expected nil error from Stop, got %v", err)
	}
	if instance != nil {
		t.Fatal("expected instance to be nil after Stop, got non-nil")
	}
}

func TestStop_ReturnsErrorOnContextDeadlineExceeded(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if err := Start(ctx); err != nil {
		t.Fatalf("failed to start environment: %v", err)
	}
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	if err := Stop(ctx); err == nil {
		t.Fatal("expected error from Stop on context deadline exceeded, got nil")
	}
}

// testenv.SetEnv

func TestSetEnv_RequiresStartedEnvironment(t *testing.T) {
	resetForTest()

	if err := SetEnv(); err == nil {
		t.Fatal("expected error from SetEnv when environment is not started")
	}
}

func TestSetEnv_PopulatesExpectedKeys(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := Start(ctx); err != nil {
		t.Fatalf("failed to start environment: %v", err)
	}

	if err := SetEnv(); err != nil {
		t.Fatalf("failed to set environment variables: %v", err)
	}

	expectedKeys := []string{
		"PG_HOST", "PG_PORT", "PG_DB", "PG_USER", "PG_PASSWORD",
		"REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DB",
	}
	for _, key := range expectedKeys {
		if os.Getenv(key) == "" {
			if key == "REDIS_PASSWORD" {
				continue // Redis password may be empty, skip this check
			}
			t.Errorf("expected environment variable %s to be set, but it was empty", key)
		}
	}
}

func TestSetEnv_PopulatesExpectedValues(t *testing.T) {
	resetForTest()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := Start(ctx); err != nil {
		t.Fatalf("failed to start environment: %v", err)
	}

	if err := SetEnv(); err != nil {
		t.Fatalf("failed to set environment variables: %v", err)
	}

	if os.Getenv("PG_HOST") != instance.PGHost {
		t.Errorf("expected PG_HOST to be %s, got %s", instance.PGHost, os.Getenv("PG_HOST"))
	}
	if os.Getenv("PG_PORT") != instance.PGPort {
		t.Errorf("expected PG_PORT to be %s, got %s", instance.PGPort, os.Getenv("PG_PORT"))
	}
	if os.Getenv("PG_DB") != instance.PGDB {
		t.Errorf("expected PG_DB to be %s, got %s", instance.PGDB, os.Getenv("PG_DB"))
	}
	if os.Getenv("PG_USER") != instance.PGUser {
		t.Errorf("expected PG_USER to be %s, got %s", instance.PGUser, os.Getenv("PG_USER"))
	}
	if os.Getenv("PG_PASSWORD") != instance.PGPass {
		t.Errorf("expected PG_PASSWORD to be %s, got %s", instance.PGPass, os.Getenv("PG_PASSWORD"))
	}
	if os.Getenv("REDIS_ADDRESS") != instance.RedisAddr {
		t.Errorf("expected REDIS_ADDRESS to be %s, got %s", instance.RedisAddr, os.Getenv("REDIS_ADDRESS"))
	}
	if os.Getenv("REDIS_PASSWORD") != instance.RedisPass {
		t.Errorf("expected REDIS_PASSWORD to be %s, got %s", instance.RedisPass, os.Getenv("REDIS_PASSWORD"))
	}
	if os.Getenv("REDIS_DB") != strconv.Itoa(instance.RedisDB) {
		t.Errorf("expected REDIS_DB to be %d, got %s", instance.RedisDB, os.Getenv("REDIS_DB"))
	}
}
