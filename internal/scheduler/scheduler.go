package scheduler

import (
	"context"
	"time"

	"github.com/marvinscham/nextclone/internal/config"
	"github.com/marvinscham/nextclone/internal/jobs"
	"github.com/marvinscham/nextclone/internal/rclone"
)

const CheckInterval = time.Minute

func RunDue(ctx context.Context) (int, error) {
	cfg, err := config.Load()
	if err != nil {
		return 0, err
	}
	runner := rclone.Runner{Settings: cfg.Settings}
	now := time.Now()
	runCount := 0

	for _, job := range cfg.Jobs {
		if !Due(job, now) {
			continue
		}
		events, done, err := jobs.Start(ctx, cfg, runner, job, true)
		if err != nil {
			return runCount, err
		}
		go func() {
			for range events {
			}
		}()
		select {
		case <-ctx.Done():
			return runCount, ctx.Err()
		case <-done:
			runCount++
		}
	}

	return runCount, nil
}

func Loop(ctx context.Context) error {
	for {
		if _, err := RunDue(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(CheckInterval):
		}
	}
}

func Due(job config.SyncJob, now time.Time) bool {
	if !job.Schedule.Enabled {
		return false
	}
	every := job.Schedule.EveryNDays
	if every <= 0 {
		every = 1
	}
	hour := job.Schedule.AtHour
	if hour < 0 || hour > 23 {
		hour = 2
	}
	minute := job.Schedule.AtMinute
	if minute < 0 || minute > 59 {
		minute = 0
	}
	scheduledToday := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(scheduledToday) {
		return false
	}
	if job.LastScheduledRun == nil {
		return true
	}
	last := job.LastScheduledRun.In(now.Location())
	lastScheduledDay := time.Date(last.Year(), last.Month(), last.Day(), hour, minute, 0, 0, now.Location())
	days := int(scheduledToday.Sub(lastScheduledDay).Hours() / 24)
	return days >= every
}
