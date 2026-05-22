package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/marvinscham/nextclone/internal/config"
	"github.com/marvinscham/nextclone/internal/rclone"
)

func Start(ctx context.Context, cfg *config.Config, runner rclone.Runner, job config.SyncJob, scheduled bool) (<-chan rclone.Event, <-chan config.RunResult, error) {
	logDir, err := config.LogDir()
	if err != nil {
		return nil, nil, err
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", SafeName(job.Name), time.Now().Format("20060102-150405")))
	events, runnerDone := runner.RunJob(ctx, job, logPath)
	done := make(chan config.RunResult, 1)

	go func() {
		result := <-runnerDone
		for i := range cfg.Jobs {
			if cfg.Jobs[i].ID == job.ID {
				cfg.Jobs[i].LastRun = &result
				if scheduled {
					runAt := result.StartedAt
					cfg.Jobs[i].LastScheduledRun = &runAt
				}
				break
			}
		}
		_ = config.Save(cfg)
		done <- result
		close(done)
	}()

	return events, done, nil
}

func SafeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "job"
	}
	return b.String()
}
