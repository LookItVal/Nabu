package api

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lookitval/nabu/core/internal/database/postgres"
	"github.com/lookitval/nabu/core/internal/database/redisdb"
)

// v1.RegisterRoutes

func TestRegisterRoutes_ReturnsRouterInstance(t *testing.T) {
	db, err := postgres.Connect()
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	rdb, err := redisdb.Connect()
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	router := gin.Default()

	RegisterRoutes(router, db, rdb)

	if router == nil {
		t.Fatal("expected RegisterRoutes to return a router instance, got nil")
	}
	if len(router.Routes()) == 0 {
		t.Fatal("expected RegisterRoutes to register routes, but no routes were found")
	}
	if router.Routes()[0].Path != "/v1/status" {
		t.Fatalf("expected first registered route to be /v1/status, got %s", router.Routes()[0].Path)
	}
}
