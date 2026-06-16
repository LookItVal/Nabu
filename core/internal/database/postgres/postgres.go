// Package postgres provides connectivity and management for PostgreSQL databases.
// It wraps the standard library sql package with the lib/pq driver.
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"

	"github.com/lookitval/nabu/core/internal/config"
)

// Connect initializes a new PostgreSQL database connection using the configuration.
// It pings the database to ensure the connection is active before returning.
func Connect() (*sql.DB, error) {
	cfg := config.Load()

	c, err := pq.NewConnectorConfig(pq.Config{
		Host:           cfg.PGHost,
		Port:           cfg.PGPort,
		User:           cfg.PGUser,
		Password:       cfg.PGPass,
		Database:       cfg.PGDB,
		ConnectTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(c)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Ping checks the health of the given PostgreSQL database connection.
// It returns the time taken to ping the database or an error if the connection is unhealthy.
// Returns -1 if the ping fails, otherwise returns the latency in milliseconds.
func Ping(db *sql.DB) int64 {
	if db == nil {
		return -1
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := db.PingContext(ctx)
	if err != nil {
		return -1
	}

	return time.Since(start).Milliseconds()
}
