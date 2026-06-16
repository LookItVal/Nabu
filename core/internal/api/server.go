// Package api acts as the application delivery layer containing HTTP routing and configuration.
package api

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	v1 "github.com/lookitval/nabu/backend/internal/api/v1"
	"github.com/lookitval/nabu/core/internal/config"
	"github.com/redis/go-redis/v9"
)

// Server aggregates all the dependencies and routes necessary to run the web application.
type Server struct {
	router *gin.Engine
	db     *sql.DB
	rdb    *redis.Client
	cfg    *config.Config
}

// NewServer initializes a Server instance with configured routing, database connections, and application settings.
func NewServer(cfg *config.Config, db *sql.DB, rdb *redis.Client) *Server {
	s := &Server{
		router: gin.Default(),
		db:     db,
		rdb:    rdb,
		cfg:    cfg,
	}

	v1.RegisterRoutes(s.router, db, rdb)

	return s
}

// Run attaches the router to the configured network port and starts listening for HTTP requests.
func (s *Server) Run() error {
	fmt.Printf("Starting server on port %s", s.cfg.Port)
	return s.router.Run(":" + s.cfg.Port)
}
