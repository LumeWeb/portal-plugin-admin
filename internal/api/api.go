package api

import (
	_ "embed"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal-plugin-admin/internal/service"
	"go.lumeweb.com/portal/config"
	"go.lumeweb.com/portal/core"
	"go.lumeweb.com/portal/middleware"
	"go.lumeweb.com/portal/middleware/swagger"
	"net/http"
)

//go:embed swagger.yaml
var swagSpec []byte

var _ core.API = (*API)(nil)

type API struct {
	ctx      core.Context
	cron     *service.AdminCronService
	settings *service.AdminSettingsService
}

func (a API) Name() string {
	return internal.PluginName
}

func (a API) Subdomain() string {
	return internal.PluginName
}

func (a API) Configure(router *mux.Router, accessSvc core.AccessService) error {
	// Swagger routes
	if err := swagger.Swagger(swagSpec, router); err != nil {
		return err
	}

	authMw := middleware.AuthMiddleware(middleware.AuthMiddlewareOptions{
		Context: a.ctx,
		Purpose: core.JWTPurposeLogin,
	})

	accessMw := middleware.AccessMiddleware(a.ctx)

	corsHandler := middleware.CorsMiddleware(&cors.Options{
		ExposedHeaders: []string{"X-Total-Count"},
	})

	router.Use(corsHandler, authMw, accessMw)

	routes := []struct {
		path    string
		method  string
		handler http.HandlerFunc
	}{
		{"/api/cron/jobs", "GET", a.handleListCronJobs},
		{"/api/cron/jobs/{uuid}", "GET", a.handleGetCronJob},
		{"/api/cron/jobs/{uuid}/logs", "GET", a.handleListCronJobLogs},
		{"/api/cron/stats", "GET", a.handleGetCronStats},
		{"/api/settings/schema", "GET", a.handleGetSchema},
		{"/api/settings", "GET", a.handleListSettings},
		{"/api/settings/{id}", "GET", a.handleGetSetting},
		{"/api/settings/{id}", "POST", a.handleUpdateSetting},
	}

	subdomain := a.Subdomain()

	for _, route := range routes {
		router.HandleFunc(route.path, route.handler).Methods(route.method)
		if err := accessSvc.RegisterRoute(subdomain, route.path, route.method, core.ACCESS_ADMIN_ROLE); err != nil {
			return err
		}
	}

	return nil
}

func (a API) AuthTokenName() string {
	return core.AUTH_TOKEN_NAME
}

func (a API) Config() config.APIConfig {
	return nil
}

func NewAPI() (core.API, []core.ContextBuilderOption, error) {
	api := &API{}

	opts := core.ContextOptions(
		core.ContextWithStartupFunc(func(ctx core.Context) error {
			api.ctx = ctx
			api.cron = core.GetService[*service.AdminCronService](ctx, service.ADMIN_CRON_SERVICE)
			api.settings = core.GetService[*service.AdminSettingsService](ctx, service.ADMIN_SETTINGS_SERVICE)
			return nil
		}),
	)

	return api, opts, nil
}
