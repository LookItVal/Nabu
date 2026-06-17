// Package main is the entrypoint to the backend core API.
// Its sole responsibility is wiring up configuration, databases, and triggering the HTTP listener.
package main

import (
	"fmt"

	"github.com/lookitval/nabu/core/internal/api"
	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	"github.com/lookitval/nabu/core/internal/database/redisdb"
)

// main reads the configuration and initiates connections to external services before starting the server.
func main() {
	cfg := config.Load()
	fmt.Printf("Configuration loaded: %+v\n", cfg)

	db, err := postgres.Connect()
	if err != nil {
		fmt.Printf("WARNING: Failed to initialize postgres: %v\n", err)
	}

	rdb, err := redisdb.Connect()
	if err != nil {
		fmt.Printf("WARNING: Failed to initialize redis: %v\n", err)
	}

	server := api.NewServer(cfg, db, rdb)

	if err := server.Run(); err != nil {
		fmt.Printf("ERROR: API server failed: %v\n", err)
	}
}
