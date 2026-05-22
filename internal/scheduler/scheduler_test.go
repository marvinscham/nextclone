package scheduler

import (
	"testing"
	"time"

	"github.com/marvinscham/nextclone/internal/config"
)

func TestDueRunsMissedScheduleAfterTime(t *testing.T) {
	now := time.Date(2026, 5, 21, 9, 0, 0, 0, time.Local)
	job := config.SyncJob{Schedule: config.Schedule{Enabled: true, EveryNDays: 1, AtHour: 2}}

	if !Due(job, now) {
		t.Fatal("expected never-run job to be due after scheduled time")
	}
}

func TestDueWaitsUntilScheduledTime(t *testing.T) {
	now := time.Date(2026, 5, 21, 1, 59, 0, 0, time.Local)
	job := config.SyncJob{Schedule: config.Schedule{Enabled: true, EveryNDays: 1, AtHour: 2}}

	if Due(job, now) {
		t.Fatal("expected job not to be due before scheduled time")
	}
}

func TestDueUsesEveryNDays(t *testing.T) {
	now := time.Date(2026, 5, 21, 2, 0, 0, 0, time.Local)
	last := time.Date(2026, 5, 20, 2, 0, 0, 0, time.Local)
	job := config.SyncJob{
		Schedule:         config.Schedule{Enabled: true, EveryNDays: 2, AtHour: 2},
		LastScheduledRun: &last,
	}

	if Due(job, now) {
		t.Fatal("expected job not to be due before every-N-days interval")
	}
}
