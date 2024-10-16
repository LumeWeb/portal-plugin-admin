package api

import (
	_ "embed"
	"github.com/gorilla/mux"
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal-plugin-admin/internal/service"
	"go.lumeweb.com/portal/config"
	"go.lumeweb.com/portal/core"
	"go.lumeweb.com/portal/middleware"
	"go.lumeweb.com/portal/middleware/swagger"
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

func (a API) Configure(router *mux.Router) error {

	// Swagger routes
	err := swagger.Swagger(swagSpec, router)
	if err != nil {
		return err
	}

	router.Use(middleware.CorsMiddleware(nil))

	router.HandleFunc("/api/cron/jobs", a.handleListCronJobs).Methods("GET")
	router.HandleFunc("/api/cron/jobs/{uuid}", a.handleGetCronJob).Methods("GET")
	router.HandleFunc("/api/cron/jobs/{uuid}/logs", a.handleListCronJobLogs).Methods("GET")
	router.HandleFunc("/api/cron/stats", a.handleGetCronStats).Methods("GET")

	router.HandleFunc("/api/settings/schema", a.handleGetSchema).Methods("GET")
	router.HandleFunc("/api/settings", a.handleListSettings).Methods("GET")
	router.HandleFunc("/api/settings/:id", a.handleGetSetting).Methods("GET")
	router.HandleFunc("/api/settings/:id", a.handleUpdateSetting).Methods("POST")
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
