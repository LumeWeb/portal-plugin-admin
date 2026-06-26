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

var pprofInternalRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/internal/pprof/cmdline"},
	{http.MethodGet, "/api/internal/pprof/symbol"},
	{http.MethodGet, "/api/internal/pprof/goroutine"},
	{http.MethodGet, "/api/internal/pprof/heap"},
	{http.MethodGet, "/api/internal/pprof/threadcreate"},
	{http.MethodGet, "/api/internal/pprof/block"},
	{http.MethodGet, "/api/internal/pprof/mutex"},
}

// TestInternalPprofRoutes_ValidBearer verifies that every internal pprof
// route returns 200 when the correct bearer token is provided.
// profile and trace use seconds=1 to avoid long-running CPU captures.
func TestInternalPprofRoutes_ValidBearer(t *testing.T) {
	tests := append(pprofInternalRoutes,
		struct{ method, path string }{http.MethodGet, "/api/internal/pprof/profile?seconds=1"},
		struct{ method, path string }{http.MethodGet, "/api/internal/pprof/trace?seconds=1"},
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

// TestInternalPprofRoutes_NoBearer verifies that every internal pprof
// route returns 401 when no bearer token is provided.
func TestInternalPprofRoutes_NoBearer(t *testing.T) {
	tests := append(pprofInternalRoutes,
		struct{ method, path string }{http.MethodGet, "/api/internal/pprof/profile"},
		struct{ method, path string }{http.MethodGet, "/api/internal/pprof/trace"},
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

// TestInternalPprofRoutes_InvalidBearer verifies that an incorrect bearer
// token is rejected with 401 on all internal pprof routes.
func TestInternalPprofRoutes_InvalidBearer(t *testing.T) {
	tests := append(pprofInternalRoutes,
		struct{ method, path string }{http.MethodGet, "/api/internal/pprof/profile"},
		struct{ method, path string }{http.MethodGet, "/api/internal/pprof/trace"},
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
