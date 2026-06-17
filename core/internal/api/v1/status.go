package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	"github.com/lookitval/nabu/core/internal/database/redisdb"
)

// getStatus is an HTTP handler that checks backend application health.
// It verifies connectivity for both the Postgres and Redis dependencies.
func (h *handler) getStatus(c *gin.Context) {
	postgresStatus := postgres.Ping(h.db)
	redisStatus := redisdb.Ping(h.rdb)
	c.JSON(200, gin.H{
		"status":   "ok",
		"postgres": postgresStatus,
		"redis":    redisStatus,
	})
}
