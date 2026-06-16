// Package config provides application-wide configuration management.
// It is responsible for reading environment variables and providing
// strongly typed configuration values to the rest of the application.
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration values needed by the application.
type Config struct {
	Port      string // Port is the HTTP server listening port (e.g., "8080").
	PGHost    string // PGHost is the hostname of the PostgreSQL database.
	PGPort    uint16 // PGPort is the port number of the PostgreSQL database.
	PGDB      string // PGDB is the name of the PostgreSQL database.
	PGUser    string // PGUser is the username for the PostgreSQL database.
	PGPass    string // PGPass is the password for the PostgreSQL database.
	RedisAddr string // RedisAddr is the address of the Redis server (e.g., "localhost:6379").
	RedisPass string // RedisPass is the password for the Redis server, if any.
	RedisDB   int    // RedisDB is the database number to use on the Redis server.
}

// Load reads configuration values from environment variables and returns a populated Config.
// If certain variables are not set, it provides sensible defaults (e.g., Port defaults to "8080").
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		redisDB = 0
	}
	pgPort, err := strconv.ParseUint(os.Getenv("PG_PORT"), 10, 16)
	if err != nil {
		pgPort = 5432
	}

	return &Config{
		Port:      port,
		PGHost:    os.Getenv("PG_HOST"),
		PGPort:    uint16(pgPort),
		PGDB:      os.Getenv("PG_DB"),
		PGUser:    os.Getenv("PG_USER"),
		PGPass:    os.Getenv("PG_PASS"),
		RedisAddr: os.Getenv("REDIS_ADDR"),
		RedisPass: os.Getenv("REDIS_PASS"),
		RedisDB:   redisDB,
	}
}
