// Package main is the entrypoint to the backend core API.
// Its sole responsibility is wiring up configuration, databases, and triggering the HTTP listener.
package main

import (
	"fmt"

	redisdb "github.com/lookitval/nabu/backend/internal/database/redis"
	"github.com/lookitval/nabu/core/internal/api"
	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/database/postgres"
)

// main reads the configuration and initiates connections to external services before starting the server.
func main() {
	cfg := config.Load()

	db, err := postgres.Connect()
	if err != nil {
		fmt.Errorf("Failed to initialize postgres: %v\n", err)
		return
	}

	rdb, err := redisdb.New(cfg.RedisURI)
	if err != nil {
		fmt.Errorf("Failed to initialize redis: %v\n", err)
		return
	}

	server := api.NewServer(cfg, db, rdb)

	if err := server.Run(); err != nil {
		fmt.Errorf("API server failed: %v\n", err)
	}
}
