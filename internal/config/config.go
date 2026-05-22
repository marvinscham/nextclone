package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const AppName = "Nextclone"

type Config struct {
	Settings Settings  `json:"settings"`
	Jobs     []SyncJob `json:"jobs"`
}

type Settings struct {
	RclonePath       string `json:"rclonePath"`
	LogRetentionDays int    `json:"logRetentionDays"`
	Theme            string `json:"theme"`
	AutoStart        bool   `json:"autoStart"`
	Language         string `json:"language"`
	UploadLimit      string `json:"uploadLimit"`
}

type SyncJob struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	LocalPath        string     `json:"localPath"`
	RemoteName       string     `json:"remoteName"`
	RemotePath       string     `json:"remotePath"`
	Mode             string     `json:"mode"`
	DryRun           bool       `json:"dryRun"`
	Excludes         []string   `json:"excludes"`
	ExtraFlags       string     `json:"extraFlags"`
	Schedule         Schedule   `json:"schedule"`
	LastScheduledRun *time.Time `json:"lastScheduledRun,omitempty"`
	LastRun          *RunResult `json:"lastRun,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type Schedule struct {
	Enabled    bool `json:"enabled"`
	EveryNDays int  `json:"everyNDays"`
	AtHour     int  `json:"atHour"`
	AtMinute   int  `json:"atMinute"`
}

type RunResult struct {
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
	Success   bool      `json:"success"`
	ExitCode  int       `json:"exitCode"`
	LogPath   string    `json:"logPath"`
	Message   string    `json:"message"`
}

func Default() *Config {
	return &Config{Settings: Settings{LogRetentionDays: 30, Theme: "system", Language: "system"}}
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if cfg.Settings.LogRetentionDays == 0 {
		cfg.Settings.LogRetentionDays = 30
	}
	if cfg.Settings.Language == "" {
		cfg.Settings.Language = "system"
	}
	Normalize(cfg)
	return cfg, nil
}

func Normalize(cfg *Config) {
	for i := range cfg.Jobs {
		if cfg.Jobs[i].Schedule.EveryNDays <= 0 {
			cfg.Jobs[i].Schedule.EveryNDays = 1
		}
		if cfg.Jobs[i].Schedule.AtHour < 0 || cfg.Jobs[i].Schedule.AtHour > 23 {
			cfg.Jobs[i].Schedule.AtHour = 2
		}
		if cfg.Jobs[i].Schedule.AtMinute < 0 || cfg.Jobs[i].Schedule.AtMinute > 59 {
			cfg.Jobs[i].Schedule.AtMinute = 0
		}
	}
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	Normalize(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			return "", errors.New("APPDATA is not set")
		}
		return filepath.Join(base, AppName), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "nextclone"), nil
}

func LogDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, AppName, "logs"), nil
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "nextclone", "logs"), nil
}
