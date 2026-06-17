package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/lookitval/nabu/core/internal/api/middleware/ratelimit"
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

	// Apply IP-based leaky bucket rate limiting to all v1 routes.
	router.Use(ratelimit.IPRateLimiter(rdb, ratelimit.DefaultBucketConfig()))

	router.GET("/v1/status", h.getStatus)
}
