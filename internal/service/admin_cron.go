package service

import (
	"fmt"
	"github.com/google/uuid"
	"go.lumeweb.com/portal/core"
	"go.lumeweb.com/portal/db"
	"go.lumeweb.com/portal/db/models"
	"go.lumeweb.com/portal/db/types"
	"gorm.io/gorm"
)

var _ core.Service = (*AdminCronService)(nil)

type CronJobStats struct {
	Total  int64
	Failed int64
}

const ADMIN_CRON_SERVICE = "admin_cron"

type AdminCronService struct {
	ctx  core.Context
	db   *gorm.DB
	cron core.CronService
}

func (a *AdminCronService) ID() string {
	return ADMIN_CRON_SERVICE
}

func NewAdminCronService() (core.Service, []core.ContextBuilderOption, error) {
	adminCronService := &AdminCronService{}

	opts := core.ContextOptions(
		core.ContextWithStartupFunc(func(ctx core.Context) error {
			adminCronService.ctx = ctx
			adminCronService.db = ctx.DB()
			adminCronService.cron = core.GetService[core.CronService](ctx, core.CRON_SERVICE)
			return nil
		}),
	)

	return adminCronService, opts, nil
}

func (a *AdminCronService) ListCronJobs(offset, limit int, sortBy, sortOrder string) ([]models.CronJob, int64, error) {
	var jobs []models.CronJob
	var totalCount int64

	query := a.db.Model(&models.CronJob{})

	// Count total items
	if err := db.RetryOnLock(a.db, func(db *gorm.DB) *gorm.DB {
		return query.Count(&totalCount)
	}); err != nil {
		return nil, 0, err
	}

	// Apply sorting
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	query = query.Offset(offset).Limit(limit)

	// Execute the query
	if err := db.RetryOnLock(a.db, func(db *gorm.DB) *gorm.DB {
		return query.Find(&jobs)
	}); err != nil {
		return nil, 0, err
	}

	return jobs, totalCount, nil
}

func (a *AdminCronService) GetCronJobByUUID(uuid uuid.UUID) (*models.CronJob, error) {
	var job models.CronJob

	job.UUID = types.BinaryUUID(uuid)

	if err := db.RetryOnLock(a.db, func(db *gorm.DB) *gorm.DB {
		return db.Where(&job).First(&job)
	}); err != nil {
		return nil, err
	}

	return &job, nil
}

func (a *AdminCronService) ListCronJobLogs(jobID uint) ([]models.CronJobLog, error) {
	var logs []models.CronJobLog
	var log models.CronJobLog

	log.CronJobID = jobID

	if err := db.RetryOnLock(a.db, func(db *gorm.DB) *gorm.DB {
		return db.Where(&log).Find(&logs)
	}); err != nil {
		return nil, err
	}

	return logs, nil
}

func (a *AdminCronService) GetCronJobStats() (*CronJobStats, error) {
	var totalJobs int64
	var failedJobs int64

	if err := db.RetryOnLock(a.db, func(db *gorm.DB) *gorm.DB {
		return db.Model(&models.CronJob{}).Count(&totalJobs)
	}); err != nil {
		return nil, err
	}

	if err := db.RetryOnLock(a.db, func(db *gorm.DB) *gorm.DB {
		return db.Model(&models.CronJob{}).Where("failures > 0").Count(&failedJobs)
	}); err != nil {
		return nil, err
	}
	return &CronJobStats{
		Total:  totalJobs,
		Failed: failedJobs,
	}, nil
}

func (a *AdminCronService) GetRecentCronJobLogs(limit int) ([]models.CronJobLog, error) {
	var logs []models.CronJobLog
	result := a.db.Order("created_at DESC").Limit(limit).Find(&logs)
	return logs, result.Error
}
