package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	"github.com/lookitval/nabu/core/internal/database/redisdb"
)

// v1.StatusHandler

func TestStatusHandler_ReturnsOK(t *testing.T) {
	cfg := config.Load()
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	svr := NewServer(cfg, db, rdb)
	go svr.Run()
	defer svr.Close()
	time.Sleep(5 * time.Second) // Give the server a moment to start

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/status", cfg.Port))
	if err != nil {
		t.Fatalf("failed to send GET request to /v1/status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 OK, got %d", resp.StatusCode)
	}
	parsedJson := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&parsedJson); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if parsedJson["status"] != "ok" {
		t.Fatalf("expected status to be 'ok', got '%s'", parsedJson["status"])
	}
	if parsedJson["postgres"] == float64(-1) {
		t.Fatal("expected postgres status to be healthy, got -1")
	}
	if parsedJson["redis"] == float64(-1) {
		t.Fatal("expected redis status to be healthy, got -1")
	}
}

func TestStatusHandler_ShowsDBFailure(t *testing.T) {
	cfg := config.Load()
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	svr := NewServer(cfg, nil, rdb)
	go svr.Run()
	defer svr.Close()
	time.Sleep(5 * time.Second) // Give the server a moment to start

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/status", cfg.Port))
	if err != nil {
		t.Fatalf("failed to send GET request to /v1/status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200 OK, got %d", resp.StatusCode)
	}
	parsedJson := make(map[string]any)
	if err := json.NewDecoder(resp.Body).Decode(&parsedJson); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if parsedJson["postgres"] != float64(-1) {
		t.Fatal("expected postgres status to be unhealthy, got healthy")
	}
	if parsedJson["redis"] == float64(-1) {
		t.Fatal("expected redis status to be healthy, got -1")
	}
}

func TestStatusHandler_ShowsRedisFailure(t *testing.T) {
	cfg := config.Load()
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	svr := NewServer(cfg, db, nil)
	go svr.Run()
	defer svr.Close()
	time.Sleep(5 * time.Second) // Give the server a moment to start

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/status", cfg.Port))
	if err != nil {
		t.Fatalf("failed to send GET request to /v1/status: %v", err)
	}
	defer resp.Body.Close()
	// Without redis there is no rate limiting. without rate limiting there is no response.
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code 500 Internal Server Error, got %d", resp.StatusCode)
	}
}
