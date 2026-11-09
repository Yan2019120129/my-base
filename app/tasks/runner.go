package tasks

import (
	"context"
	"sync"
	"time"

	"my-base/code"

	adminDB "github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/modules/logger"
)

// Job describes a recurring background task.
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(context.Context) error
}

// Runner starts and stops background jobs with the application lifecycle.
type Runner struct {
	jobs []Job
	wg   sync.WaitGroup
}

func NewRunner(jobs ...Job) *Runner {
	return &Runner{jobs: jobs}
}

func (r *Runner) Start(ctx context.Context) {
	for _, job := range r.jobs {
		if job.Interval <= 0 || job.Run == nil {
			continue
		}

		r.wg.Add(1)
		go r.loop(ctx, job)
	}
}

func (r *Runner) Wait() {
	r.wg.Wait()
}

func (r *Runner) loop(ctx context.Context, job Job) {
	defer r.wg.Done()

	r.run(ctx, job)

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Infof("task stopped: %s", job.Name)
			return
		case <-ticker.C:
			r.run(ctx, job)
		}
	}
}

func (r *Runner) run(ctx context.Context, job Job) {
	if err := job.Run(ctx); err != nil {
		logger.Errorf("task %s failed: %v", job.Name, err)
	}
}

func Start(ctx context.Context, conn adminDB.Connection) (*Runner, error) {
	db, err := conn.GetGorm(code.DefaultGoAdminConnectionName)
	if err != nil {
		return nil, err
	}

	notificationJob, err := systemNotificationJob(db)
	if err != nil {
		return nil, err
	}
	runner := NewRunner(
		systemPlanJob(db),
		notificationJob,
		heartbeatJob(),
		testTableMonitorJob(db),
		tusUploadCleanupJob(db),
	)
	runner.Start(ctx)
	return runner, nil
}
