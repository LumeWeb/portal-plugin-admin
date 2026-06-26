package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lumeweb.com/portal-plugin-admin/internal"
	pluginConfig "go.lumeweb.com/portal-plugin-admin/internal/config"
	coreTesting "go.lumeweb.com/portal/core/testing"
)

func TestMain(m *testing.M) {
	coreTesting.WithOptions(m,
		coreTesting.WithAPI(internal.PLUGIN_NAME, NewAPI),
	)
}

func getAdminAPITestOptions() coreTesting.TestContextBuilderOption {
	return coreTesting.CombineOptions(
		coreTesting.WithAPIConfig(internal.PLUGIN_NAME, &pluginConfig.APIConfig{
			PprofSecret: "test-secret",
		}),
		coreTesting.WithAPIID(internal.PLUGIN_NAME),
	)
}

var pprofGetRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/debug/pprof/cmdline"},
	{http.MethodGet, "/api/debug/pprof/symbol"},
	{http.MethodGet, "/api/debug/pprof/goroutine"},
	{http.MethodGet, "/api/debug/pprof/heap"},
	{http.MethodGet, "/api/debug/pprof/threadcreate"},
	{http.MethodGet, "/api/debug/pprof/block"},
	{http.MethodGet, "/api/debug/pprof/mutex"},
	{http.MethodGet, "/api/debug/pprof/status"},
}

// TestPprofRoutes_SharedSecret_ValidBearer verifies that every registered pprof
// child route returns 200 when the correct bearer token is provided.
// profile and trace use seconds=1 to avoid long-running CPU captures.
func TestPprofRoutes_SharedSecret_ValidBearer(t *testing.T) {
	tests := append(pprofGetRoutes,
		struct{ method, path string }{http.MethodGet, "/api/debug/pprof/profile?seconds=1"},
		struct{ method, path string }{http.MethodGet, "/api/debug/pprof/trace?seconds=1"},
	)

	for _, r := range tests {
		t.Run(r.path, func(t *testing.T) {
			coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
				req := ctx.NewAPIRequest(r.method, r.path, nil)
				req.Header.Set("Authorization", "Bearer test-secret")

				w := httptest.NewRecorder()
				ctx.Router().ServeHTTP(w, req)

				assert.Equal(tb, http.StatusOK, w.Code)
			}, getAdminAPITestOptions())
		})
	}
}

// TestPprofRoutes_SharedSecret_NoBearer verifies that every registered pprof
// child route returns 401 when no bearer token is provided.
func TestPprofRoutes_SharedSecret_NoBearer(t *testing.T) {
	tests := append(pprofGetRoutes,
		struct{ method, path string }{http.MethodGet, "/api/debug/pprof/profile"},
		struct{ method, path string }{http.MethodGet, "/api/debug/pprof/trace"},
	)

	for _, r := range tests {
		t.Run(r.path, func(t *testing.T) {
			coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
				req := ctx.NewAPIRequest(r.method, r.path, nil)

				w := httptest.NewRecorder()
				ctx.Router().ServeHTTP(w, req)

				assert.Equal(tb, http.StatusUnauthorized, w.Code)
			}, getAdminAPITestOptions())
		})
	}
}

// TestPprofRoutes_SharedSecret_InvalidBearer verifies that an incorrect bearer
// token is rejected with 401 on all pprof child routes.
func TestPprofRoutes_SharedSecret_InvalidBearer(t *testing.T) {
	tests := append(pprofGetRoutes,
		struct{ method, path string }{http.MethodGet, "/api/debug/pprof/profile"},
		struct{ method, path string }{http.MethodGet, "/api/debug/pprof/trace"},
	)

	for _, r := range tests {
		t.Run(r.path, func(t *testing.T) {
			coreTesting.RunTestCase(t, func(tb coreTesting.TB, ctx coreTesting.TestContext) {
				req := ctx.NewAPIRequest(r.method, r.path, nil)
				req.Header.Set("Authorization", "Bearer wrong-secret")

				w := httptest.NewRecorder()
				ctx.Router().ServeHTTP(w, req)

				assert.Equal(tb, http.StatusUnauthorized, w.Code)
			}, getAdminAPITestOptions())
		})
	}
}
