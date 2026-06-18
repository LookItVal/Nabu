package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lookitval/nabu/core/internal/database/redisdb"
	"github.com/lookitval/nabu/core/internal/testutils"
	"github.com/redis/go-redis/v9"
)

var testRedisClient *redis.Client

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

	client, err := redisdb.Connect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to Redis: %v\n", err)
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		_ = testutils.Stop(ctx)
		cancel()
		os.Exit(1)
	}
	testRedisClient = client
	cancel()

	code := m.Run()

	if testRedisClient != nil {
		_ = testRedisClient.Close()
	}
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	if err := testutils.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop test environment: %v\n", err)
	}
	cancel()

	os.Exit(code)
}

func TestIPRateLimiter_AllowsFirstRequest(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)
	router := gin.New()
	router.Use(IPRateLimiter(testRedisClient, DefaultBucketConfig()))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Fatal("expected X-RateLimit-Limit header to be set")
	}
}

func TestIPRateLimiter_ReturnsTooManyRequests(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)
	router := gin.New()
	router.Use(IPRateLimiter(testRedisClient, BucketConfig{Capacity: 1, LeakRatePerSec: 1}))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.RemoteAddr = "203.0.113.11:1234"

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be rate limited, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "rate limit exceeded") {
		t.Fatalf("expected rate limit error response, got %s", rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header to be set")
	}
}

func TestIPRateLimiter_ReturnsServiceUnavailableWhenRedisFails(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)
	router := gin.New()
	badClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer badClient.Close()
	router.Use(IPRateLimiter(badClient, DefaultBucketConfig()))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.RemoteAddr = "203.0.113.12:1234"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected Redis failure to return 503, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "rate limiter unavailable") {
		t.Fatalf("expected Redis failure payload, got %s", rec.Body.String())
	}
}
