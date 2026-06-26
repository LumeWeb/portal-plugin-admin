package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	pluginConfig "go.lumeweb.com/portal-plugin-admin/internal/config"
	pluginMw "go.lumeweb.com/portal-plugin-admin/internal/api/middleware"
)

// TestPprofRoutes_SharedSecret_AllChildRoutes verifies that every registered
// pprof child route accepts the shared secret bearer token and rejects
// requests without it.
func TestPprofRoutes_SharedSecret_AllChildRoutes(t *testing.T) {
	const secret = "test-secret"
	cfg := &pluginConfig.APIConfig{PprofSecret: secret}

	e := echo.New()
	authMw := pluginMw.PprofSharedSecretAuth(cfg)

	// Register every pprof route the API exposes, with shared secret middleware.
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/debug/pprof/"},
		{http.MethodGet, "/api/debug/pprof/cmdline"},
		{http.MethodGet, "/api/debug/pprof/symbol"},
		{http.MethodGet, "/api/debug/pprof/goroutine"},
		{http.MethodGet, "/api/debug/pprof/heap"},
		{http.MethodGet, "/api/debug/pprof/threadcreate"},
		{http.MethodGet, "/api/debug/pprof/block"},
		{http.MethodGet, "/api/debug/pprof/mutex"},
		{http.MethodGet, "/api/debug/pprof/profile"},
		{http.MethodGet, "/api/debug/pprof/trace"},
		{http.MethodGet, "/api/debug/pprof/status"},
	}

	// Use a dummy handler that returns 200 so we can verify middleware pass-through.
	dummyHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	for _, r := range routes {
		e.Add(r.method, r.path, dummyHandler, authMw)
	}

	for _, tc := range routes {
		t.Run(tc.path, func(t *testing.T) {
			// With bearer token -> should pass auth
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+secret)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("with bearer token: expected 200, got %d", rec.Code)
			}

			// Without bearer token -> should be 401
			req2 := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec2 := httptest.NewRecorder()
			e.ServeHTTP(rec2, req2)

			if rec2.Code != http.StatusUnauthorized {
				t.Errorf("without bearer token: expected 401, got %d", rec2.Code)
			}
		})
	}
}
