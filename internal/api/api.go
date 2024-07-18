package api

import (
	_ "embed"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.lumeweb.com/httputil"
	"go.lumeweb.com/portal-plugin-admin/internal"
	"go.lumeweb.com/portal-plugin-admin/internal/service"
	"go.lumeweb.com/portal/config"
	"go.lumeweb.com/portal/core"
	"go.lumeweb.com/portal/middleware/swagger"
	"net/http"
)

//go:embed swagger.yaml
var swagSpec []byte

var _ core.API = (*API)(nil)

type API struct {
	ctx  core.Context
	cron *service.AdminCronService
}

func (a API) Name() string {
	return internal.PluginName
}

func (a API) Subdomain() string {
	return internal.PluginName
}

func (a API) Configure(router *mux.Router) error {

	// CORS configuration
	corsOpts := cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}
	corsHandler := cors.New(corsOpts)

	// Swagger routes
	err := swagger.Swagger(swagSpec, router)
	if err != nil {
		return err
	}

	router.Use(corsHandler.Handler)

	router.HandleFunc("/api/cron/jobs", a.handleListCronJobs).Methods("GET")
	router.HandleFunc("/api/cron/jobs/{uuid}", a.handleGetCronJob).Methods("GET")
	router.HandleFunc("/api/cron/jobs/{uuid}/logs", a.handleListCronJobLogs).Methods("GET")
	router.HandleFunc("/api/cron/stats", a.handleGetCronStats).Methods("GET")
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
			return nil
		}),
	)

	return api, opts, nil
}

func (a *API) handleListCronJobs(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)

	jobs, err := a.cron.ListCronJobs()
	if ctx.Check("Failed to list cron jobs", err) != nil {
		return
	}

	response := &ListCronJobsResponse{
		Jobs: make([]CronJobData, len(jobs)),
	}

	for i, job := range jobs {
		response.Jobs[i] = CronJobData{
			UUID:     job.UUID.String(),
			Function: job.Function,
			LastRun:  job.LastRun,
			Failures: job.Failures,
		}
	}

	ctx.Encode(response)
}

func (a *API) handleGetCronJob(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	vars := mux.Vars(r)
	_uuid, err := uuid.Parse(vars["uuid"])
	if ctx.Check("Invalid UUID", err) != nil {
		return
	}

	job, err := a.cron.GetCronJobByUUID(_uuid)
	if ctx.Check("Failed to get cron job", err) != nil {
		return
	}

	response := &GetCronJobResponse{
		Job: CronJobData{
			UUID:     job.UUID.String(),
			Function: job.Function,
			LastRun:  job.LastRun,
			Failures: job.Failures,
		},
	}

	ctx.Encode(response)
}

func (a *API) handleListCronJobLogs(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)
	vars := mux.Vars(r)
	_uuid, err := uuid.Parse(vars["uuid"])
	if ctx.Check("Invalid UUID", err) != nil {
		return
	}

	job, err := a.cron.GetCronJobByUUID(_uuid)
	if ctx.Check("Failed to get cron job", err) != nil {
		return
	}

	logs, err := a.cron.ListCronJobLogs(job.ID)
	if ctx.Check("Failed to list cron job logs", err) != nil {
		return
	}

	response := &ListCronJobLogsResponse{
		Logs: make([]CronJobLogData, len(logs)),
	}

	for i, log := range logs {
		response.Logs[i] = CronJobLogData{
			ID:        log.ID,
			Type:      string(log.Type),
			Message:   log.Message,
			CreatedAt: log.CreatedAt,
		}
	}

	ctx.Encode(response)
}

func (a *API) handleGetCronStats(w http.ResponseWriter, r *http.Request) {
	ctx := httputil.Context(r, w)

	stats, err := a.cron.GetCronJobStats()
	if ctx.Check("Failed to get cron stats", err) != nil {
		return
	}

	response := &GetCronStatsResponse{
		Total:  stats.Total,
		Failed: stats.Failed,
	}

	ctx.Encode(response)
}
