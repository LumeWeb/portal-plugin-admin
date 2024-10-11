package api

import (
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.lumeweb.com/httputil"
	"net/http"
	"strconv"
)

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
			Failures:  uint(job.Failures),
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
