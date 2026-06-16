package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	RegisterRoutes(router, nil, nil) // passing nil for db and rdb to simulate disconnected state

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/status", nil)
	require.NoError(t, err)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.JSONEq(t, "{\"postgres\":\"disconnected\",\"redis\":\"disconnected\",\"status\":\"ok\"}", w.Body.String())
}