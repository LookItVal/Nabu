// Package redisdb provides connectivity and management for Redis data stores.
package redisdb

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/lookitval/nabu/core/internal/config"
)

// Connect initializes a new Redis client connection using configuration settings.
// It pings the Redis server to ensure the connection is active before returning.
func Connect() (*redis.Client, error) {
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}

// Ping checks the health of the given Redis client connection.
// It returns the time taken to ping the database or an error if the connection is unhealthy.
// Returns -1 if the ping fails, otherwise returns the latency in milliseconds.
func Ping(rdb *redis.Client) int64 {
	if rdb == nil {
		return -1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return -1
	}
	return time.Since(start).Milliseconds()
}
