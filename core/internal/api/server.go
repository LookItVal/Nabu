// Package api acts as the application delivery layer containing HTTP routing and configuration.
package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lookitval/nabu/core/internal/config"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	"github.com/redis/go-redis/v9"
)

var listenAndServe = func(s *http.Server) error {
	return s.ListenAndServe()
}

var shutdownServer = func(s *http.Server, ctx context.Context) error {
	return s.Shutdown(ctx)
}

// Server aggregates all the dependencies and routes necessary to run the web application.
type Server struct {
	router  *gin.Engine
	httpSrv *http.Server
	db      *sql.DB
	rdb     *redis.Client
	cfg     *config.Config
}

// NewServer initializes a Server instance with configured routing, database connections, and application settings.
func NewServer(cfg *config.Config, db *sql.DB, rdb *redis.Client) *Server {
	s := &Server{
		router: gin.Default(),
		db:     db,
		rdb:    rdb,
		cfg:    cfg,
	}

	RegisterRoutes(s.router, db, rdb)

	if db == nil {
		return s
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := postgres.ApplyMigrations(ctx, db); err != nil {
		fmt.Printf("WARNING: Failed to apply database migrations: %v\n", err)
		s.db = nil
	}

	return s
}

// Run attaches the router to the configured network port and starts listening for HTTP requests.
func (s *Server) Run() error {
	s.httpSrv = &http.Server{
		Addr:    ":" + s.cfg.Port,
		Handler: s.router,
	}

	fmt.Printf("Starting server on port %s\n", s.cfg.Port)
	if err := listenAndServe(s.httpSrv); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close gracefully shuts down the server and releases any open resources such as database connections.
func (s *Server) Close() {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownServer(s.httpSrv, ctx); err != nil {
			fmt.Printf("WARNING: HTTP server shutdown error: %v\n", err)
		}
	}

	if s.db != nil {
		s.db.Close()
	}
	if s.rdb != nil {
		s.rdb.Close()
	}
}
