// Package v1 provides version 1 of the application's HTTP handler endpoints.
package v1

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// handler is the container for request handlers dependencies that isolates DB state.
type handler struct {
	db  *sql.DB
	rdb *redis.Client
}

// RegisterRoutes sets up all v1 endpoints onto the provided Gin router.
func RegisterRoutes(router *gin.Engine, db *sql.DB, rdb *redis.Client) {
	h := &handler{
		db:  db,
		rdb: rdb,
	}

	router.GET("/status", h.getStatus)
}

