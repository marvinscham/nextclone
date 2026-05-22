package rclone

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/marvinscham/nextclone/internal/config"
)

type Event struct {
	Line string
}

type Runner struct {
	Settings config.Settings
}

func (r Runner) FindBinary() (string, error) {
	if r.Settings.RclonePath != "" {
		return r.Settings.RclonePath, nil
	}
	if env := os.Getenv("NEXTCLONE_RCLONE_PATH"); env != "" {
		return env, nil
	}
	if exe, err := os.Executable(); err == nil {
		name := "rclone"
		if runtime.GOOS == "windows" {
			name = "rclone.exe"
		}
		for _, candidate := range []string{
			filepath.Join(filepath.Dir(exe), name),
			filepath.Join(filepath.Dir(exe), "bin", name),
			filepath.Join(filepath.Dir(exe), "..", "lib", "nextclone", name),
		} {
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				return candidate, nil
			}
		}
	}
	path, err := exec.LookPath("rclone")
	if err != nil {
		return "", errors.New("rclone was not found; install rclone or place the bundled rclone next to Nextclone")
	}
	return path, nil
}

func (r Runner) Version(ctx context.Context) (string, error) {
	bin, err := r.FindBinary()
	if err != nil {
		return "", err
	}
	out, err := commandContext(ctx, bin, "version").CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (r Runner) CreateNextcloudRemote(ctx context.Context, remoteName, url, username, appPassword string) error {
	bin, err := r.FindBinary()
	if err != nil {
		return err
	}
	args := []string{
		"config", "create", remoteName, "webdav",
		"url", url,
		"vendor", "nextcloud",
		"user", username,
		"pass", appPassword,
		"--obscure",
	}
	out, err := commandContext(ctx, bin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone config failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (r Runner) ListRemotes(ctx context.Context) ([]string, error) {
	bin, err := r.FindBinary()
	if err != nil {
		return nil, err
	}
	out, err := commandContext(ctx, bin, "listremotes").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("rclone listremotes failed: %s", strings.TrimSpace(string(out)))
	}
	var remotes []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSuffix(strings.TrimSpace(line), ":")
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

func (r Runner) TestRemote(ctx context.Context, remoteName string) error {
	bin, err := r.FindBinary()
	if err != nil {
		return err
	}
	out, err := commandContext(ctx, bin, "lsd", remoteName+":", "--max-depth", "1").CombinedOutput()
	if err != nil {
		return fmt.Errorf("remote test failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (r Runner) RunJob(ctx context.Context, job config.SyncJob, logPath string) (<-chan Event, <-chan config.RunResult) {
	events := make(chan Event, 20)
	done := make(chan config.RunResult, 1)

	go func() {
		defer close(events)
		defer close(done)

		started := time.Now()
		result := config.RunResult{StartedAt: started, LogPath: logPath, ExitCode: -1}
		bin, err := r.FindBinary()
		if err != nil {
			result.EndedAt = time.Now()
			result.Message = err.Error()
			done <- result
			return
		}

		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			result.EndedAt = time.Now()
			result.Message = err.Error()
			done <- result
			return
		}
		logFile, err := os.Create(logPath)
		if err != nil {
			result.EndedAt = time.Now()
			result.Message = err.Error()
			done <- result
			return
		}
		defer logFile.Close()

		mode := job.Mode
		if mode != "sync" {
			mode = "copy"
		}
		args := []string{
			mode,
			job.LocalPath,
			job.RemoteName + ":" + strings.TrimPrefix(job.RemotePath, "/"),
			"--progress",
			"--stats", "2s",
			"--transfers", "4",
			"--checkers", "8",
			"--retries", "3",
			"--low-level-retries", "10",
			"--create-empty-src-dirs",
		}
		if strings.TrimSpace(r.Settings.UploadLimit) != "" {
			args = append(args, "--bwlimit", strings.TrimSpace(r.Settings.UploadLimit))
		}
		if job.DryRun {
			args = append(args, "--dry-run")
		}
		for _, pattern := range job.Excludes {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				args = append(args, "--exclude", pattern)
			}
		}
		if strings.TrimSpace(job.ExtraFlags) != "" {
			args = append(args, strings.Fields(job.ExtraFlags)...)
		}

		fmt.Fprintf(logFile, "Started: %s\nCommand: %s %s\n\n", started.Format(time.RFC3339), filepath.Base(bin), strings.Join(redact(args), " "))
		cmd := commandContext(ctx, bin, args...)
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			result.EndedAt = time.Now()
			result.Message = err.Error()
			done <- result
			return
		}

		var wg sync.WaitGroup
		var logMu sync.Mutex
		stream := func(scanner *bufio.Scanner) {
			defer wg.Done()
			for scanner.Scan() {
				line := scanner.Text()
				logMu.Lock()
				fmt.Fprintln(logFile, line)
				logMu.Unlock()
				events <- Event{Line: line}
			}
		}
		wg.Add(2)
		go stream(bufio.NewScanner(stdout))
		go stream(bufio.NewScanner(stderr))

		err = cmd.Wait()
		wg.Wait()
		result.EndedAt = time.Now()
		if err == nil {
			result.Success = true
			result.ExitCode = 0
			result.Message = "Completed successfully"
		} else {
			result.Message = err.Error()
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				result.ExitCode = exitErr.ExitCode()
			}
		}
		fmt.Fprintf(logFile, "\nEnded: %s\nSuccess: %t\nExit code: %d\n", result.EndedAt.Format(time.RFC3339), result.Success, result.ExitCode)
		done <- result
	}()

	return events, done
}

func redact(args []string) []string {
	out := append([]string(nil), args...)
	for i, arg := range out {
		if strings.EqualFold(arg, "pass") && i+1 < len(out) {
			out[i+1] = "<redacted>"
		}
	}
	return out
}
