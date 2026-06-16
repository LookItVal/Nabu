// Package main is the entrypoint to the backend core API.
// Its sole responsibility is wiring up configuration, databases, and triggering the HTTP listener.
package main

import (
	"fmt"

	"github.com/lookitval/nabu/core/internal/api"
	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	redisdb "github.com/lookitval/nabu/core/internal/database/redis"
)

// main reads the configuration and initiates connections to external services before starting the server.
func main() {
	cfg := config.Load()

	db, err := postgres.Connect()
	if err != nil {
		fmt.Printf("Failed to initialize postgres: %v\n", err)
		return
	}

	rdb, err := redisdb.Connect()
	if err != nil {
		fmt.Printf("Failed to initialize redis: %v\n", err)
		return
	}

	server := api.NewServer(cfg, db, rdb)

	if err := server.Run(); err != nil {
		fmt.Printf("API server failed: %v\n", err)
	}
}
