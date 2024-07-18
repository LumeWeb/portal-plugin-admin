package api

import "time"

type ListCronJobsResponse struct {
	Jobs []CronJobData `json:"jobs"`
}

type CronJobData struct {
	UUID     string     `json:"uuid"`
	Function string     `json:"function"`
	LastRun  *time.Time `json:"lastRun"`
	Failures uint       `json:"failures"`
}

type GetCronJobResponse struct {
	Job CronJobData `json:"job"`
}

type ListCronJobLogsResponse struct {
	Logs []CronJobLogData `json:"logs"`
}

type CronJobLogData struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type GetCronStatsResponse struct {
	Total  int64 `json:"total"`
	Failed int64 `json:"failed"`
}
