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
	"strconv"
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
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Cookie"},
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

	// Parse query parameters
	queryParams := r.URL.Query()

	// Pagination
	start, err := strconv.Atoi(queryParams.Get("_start"))
	if err != nil || start < 0 {
		start = 0
	}
	end, err := strconv.Atoi(queryParams.Get("_end"))
	if err != nil || end < start {
		end = start + 10 // Default to 10 items if end is invalid
	}
	limit := end - start

	// Sorting
	sortField := queryParams.Get("_sort")
	sortOrder := queryParams.Get("_order")
	if sortField == "" {
		sortField = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// Fetch jobs with sorting and pagination
	dbJobs, totalCount, err := a.cron.ListCronJobs(start, limit, sortField, sortOrder)
	if ctx.Check("Failed to list cron jobs", err) != nil {
		return
	}

	// Convert db models to API response format
	response := make(ListCronJobsResponse, len(dbJobs))
	for i, job := range dbJobs {
		response[i] = CronJob{
			UUID:     job.UUID.String(),
			Function: job.Function,
			LastRun:  job.LastRun,
			Failures: uint(job.Failures),
		}
	}

	// Set X-Total-Count header
	w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))

	// Set Access-Control-Expose-Headers to make X-Total-Count available to the client
	w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")

	// Return the jobs directly as the response body
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
		Job: CronJob{
			UUID:      job.UUID.String(),
			Function:  job.Function,
			LastRun:   job.LastRun,
			Failures:  job.Failures,
			CreatedAt: job.CreatedAt,
			UpdatedAt: job.UpdatedAt,
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
