package messages

import "time"

type ListCronJobsResponse = []CronJob

type CronJob struct {
	UUID      string     `json:"uuid"`
	Function  string     `json:"function"`
	LastRun   *time.Time `json:"last_run"`
	Failures  uint       `json:"failures"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type GetCronJobResponse struct {
	Job CronJob `json:"job"`
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

type PaginationData struct {
	Offset     int   `json:"offset"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
}

type SettingsItem struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type SettingUpdateRequest struct {
	Value any `json:"value"`
}
