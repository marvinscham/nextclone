package gui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/marvinscham/nextclone/assets"
	"github.com/marvinscham/nextclone/internal/autostart"
	"github.com/marvinscham/nextclone/internal/config"
	"github.com/marvinscham/nextclone/internal/jobs"
	"github.com/marvinscham/nextclone/internal/rclone"
)

type state struct {
	app       fyne.App
	window    fyne.Window
	cfg       *config.Config
	runner    rclone.Runner
	jobsBox   *fyne.Container
	status    map[string]string
	cancel    map[string]context.CancelFunc
	liveLogs  map[string][]string
	configErr error
}

func Run() {
	a := app.NewWithID("com.nextclone.app")
	a.SetIcon(assets.AppIcon)
	w := a.NewWindow(config.AppName)
	w.Resize(fyne.NewSize(900, 620))

	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	s := &state{
		app:       a,
		window:    w,
		cfg:       cfg,
		runner:    rclone.Runner{Settings: cfg.Settings},
		jobsBox:   container.NewVBox(),
		status:    map[string]string{},
		cancel:    map[string]context.CancelFunc{},
		liveLogs:  map[string][]string{},
		configErr: err,
	}

	w.SetContent(s.dashboard())
	if err != nil {
		dialog.ShowError(err, w)
	}
	w.ShowAndRun()
}

func (s *state) dashboard() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Nextclone", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	add := widget.NewButtonWithIcon("Add Sync", theme.ContentAddIcon(), func() { s.showJobDialog(nil) })
	remote := widget.NewButton("Remote Setup", s.showRemoteDialog)
	settings := widget.NewButton("Settings", s.showSettingsDialog)
	check := widget.NewButton("Check rclone", s.checkRclone)

	header := container.NewBorder(nil, nil, container.NewVBox(title), container.NewHBox(add, remote, settings, check))
	s.refreshJobs()
	return container.NewBorder(header, nil, nil, nil, container.NewVScroll(s.jobsBox))
}

func (s *state) refreshJobs() {
	s.jobsBox.Objects = nil
	if len(s.cfg.Jobs) == 0 {
		s.jobsBox.Add(container.NewCenter(widget.NewLabel("No sync jobs yet. Use Add Sync to create your first local-to-Nextcloud job.")))
		s.jobsBox.Refresh()
		return
	}
	for i := range s.cfg.Jobs {
		idx := i
		job := s.cfg.Jobs[i]
		status := s.jobStatus(job)
		name := widget.NewLabelWithStyle(job.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		detail := widget.NewLabel(fmt.Sprintf("%s -> %s:%s", job.LocalPath, job.RemoteName, strings.TrimPrefix(job.RemotePath, "/")))
		mode := "Copy"
		if job.Mode == "sync" {
			mode = "Sync (destination may be deleted to match source)"
		}
		meta := widget.NewLabel(fmt.Sprintf("Mode: %s | Schedule: %s | Status: %s", mode, scheduleStatus(job), status))

		start := widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() { s.startJob(idx) })
		if _, running := s.cancel[job.ID]; running {
			start.Disable()
		}
		stop := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() { s.stopJob(job.ID) })
		if _, running := s.cancel[job.ID]; !running {
			stop.Disable()
		}
		edit := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() { s.showJobDialog(&idx) })
		logs := widget.NewButton("Logs", func() { s.showLogs(job) })
		duplicate := widget.NewButton("Duplicate", func() { s.duplicateJob(idx) })
		deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() { s.deleteJob(idx) })

		card := widget.NewCard("", "", container.NewBorder(nil, nil, container.NewVBox(name, detail, meta), container.NewHBox(start, stop, edit, logs, duplicate, deleteBtn)))
		s.jobsBox.Add(card)
	}
	s.jobsBox.Refresh()
}

func (s *state) jobStatus(job config.SyncJob) string {
	if status := s.status[job.ID]; status != "" {
		return status
	}
	if job.LastRun == nil {
		return "Not run yet"
	}
	state := "failed"
	if job.LastRun.Success {
		state = "succeeded"
	}
	return fmt.Sprintf("Last %s at %s", state, job.LastRun.EndedAt.Format("2006-01-02 15:04"))
}

func (s *state) showJobDialog(index *int) {
	var job config.SyncJob
	editing := index != nil
	if editing {
		job = s.cfg.Jobs[*index]
	} else {
		job = config.SyncJob{ID: newID(), Mode: "copy", CreatedAt: time.Now()}
	}

	name := widget.NewEntry()
	name.SetText(job.Name)
	local := widget.NewEntry()
	local.SetText(job.LocalPath)
	remotes, err := s.runner.ListRemotes(context.Background())
	if err != nil {
		dialog.ShowError(err, s.window)
	}
	remoteName := widget.NewSelect(remotes, nil)
	remoteName.PlaceHolder = "Select remote"
	if job.RemoteName != "" {
		remoteName.SetSelected(job.RemoteName)
	}
	remotePath := widget.NewEntry()
	remotePath.SetText(job.RemotePath)
	remotePath.SetPlaceHolder("Backups/Documents")
	mode := widget.NewSelect([]string{"copy", "sync"}, nil)
	mode.SetSelected(job.Mode)
	dryRun := widget.NewCheck("Dry run only", nil)
	dryRun.SetChecked(job.DryRun)
	scheduleEnabled := widget.NewCheck("Run automatically", nil)
	scheduleEnabled.SetChecked(job.Schedule.Enabled)
	every := widget.NewSelect(scheduleOptions(), nil)
	every.SetSelected(scheduleLabel(job.Schedule.EveryNDays))
	hour := widget.NewSelect(numberOptions(0, 23), nil)
	hour.SetSelected(fmt.Sprintf("%02d", normalizedHour(job.Schedule.AtHour)))
	minute := widget.NewSelect(numberOptions(0, 59), nil)
	minute.SetSelected(fmt.Sprintf("%02d", normalizedMinute(job.Schedule.AtMinute)))
	excludes := widget.NewMultiLineEntry()
	excludes.SetMinRowsVisible(4)
	excludes.SetText(strings.Join(job.Excludes, "\n"))
	extra := widget.NewEntry()
	extra.SetText(job.ExtraFlags)
	extra.SetPlaceHolder("Advanced rclone flags, optional")

	browse := widget.NewButton("Browse", func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			if uri != nil {
				local.SetText(uri.Path())
			}
		}, s.window).Show()
	})

	form := dialog.NewForm("Sync Job", "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Local folder", container.NewBorder(nil, nil, nil, browse, local)),
		widget.NewFormItem("Remote name", remoteName),
		widget.NewFormItem("Remote path", remotePath),
		widget.NewFormItem("Mode", mode),
		widget.NewFormItem("Options", dryRun),
		widget.NewFormItem("Schedule", scheduleEnabled),
		widget.NewFormItem("Every", every),
		widget.NewFormItem("At", container.NewHBox(hour, widget.NewLabel(":"), minute)),
		widget.NewFormItem("Exclude patterns", excludes),
		widget.NewFormItem("Extra flags", extra),
	}, func(save bool) {
		if !save {
			return
		}
		if strings.TrimSpace(name.Text) == "" || strings.TrimSpace(local.Text) == "" || strings.TrimSpace(remoteName.Selected) == "" {
			dialog.ShowInformation("Missing information", "Name, local folder, and remote name are required.", s.window)
			return
		}
		job.Name = strings.TrimSpace(name.Text)
		job.LocalPath = strings.TrimSpace(local.Text)
		job.RemoteName = strings.TrimSpace(remoteName.Selected)
		job.RemotePath = strings.TrimSpace(remotePath.Text)
		job.Mode = mode.Selected
		if job.Mode != "sync" {
			job.Mode = "copy"
		}
		job.DryRun = dryRun.Checked
		job.Schedule.Enabled = scheduleEnabled.Checked
		job.Schedule.EveryNDays = scheduleDays(every.Selected)
		_, _ = fmt.Sscanf(hour.Selected, "%d", &job.Schedule.AtHour)
		_, _ = fmt.Sscanf(minute.Selected, "%d", &job.Schedule.AtMinute)
		job.Excludes = splitLines(excludes.Text)
		job.ExtraFlags = strings.TrimSpace(extra.Text)
		job.UpdatedAt = time.Now()
		if editing {
			s.cfg.Jobs[*index] = job
		} else {
			s.cfg.Jobs = append(s.cfg.Jobs, job)
		}
		s.saveAndRefresh()
	}, s.window)
	form.Resize(fyne.NewSize(720, 560))
	form.Show()
}

func (s *state) startJob(index int) {
	job := s.cfg.Jobs[index]
	if _, running := s.cancel[job.ID]; running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	events, done, err := jobs.Start(ctx, s.cfg, s.runner, job, false)
	if err != nil {
		cancel()
		dialog.ShowError(err, s.window)
		return
	}
	s.cancel[job.ID] = cancel
	s.status[job.ID] = "Running"
	s.liveLogs[job.ID] = nil
	s.refreshJobs()

	go func() {
		for event := range events {
			s.liveLogs[job.ID] = append(s.liveLogs[job.ID], event.Line)
		}
	}()
	go func() {
		result := <-done
		delete(s.cancel, job.ID)
		if result.Success {
			s.status[job.ID] = "Completed successfully"
		} else {
			s.status[job.ID] = "Failed: " + result.Message
		}
		s.refreshJobs()
	}()
}

func (s *state) stopJob(id string) {
	if cancel := s.cancel[id]; cancel != nil {
		cancel()
		s.status[id] = "Stopping"
		s.refreshJobs()
	}
}

func (s *state) showLogs(job config.SyncJob) {
	text := "No log output for this job yet."
	if lines := s.liveLogs[job.ID]; len(lines) > 0 {
		text = strings.Join(lines, "\n")
	} else if job.LastRun != nil {
		text = fmt.Sprintf("Last log file:\n%s\n\nStatus: %s", job.LastRun.LogPath, job.LastRun.Message)
	}
	entry := widget.NewMultiLineEntry()
	entry.SetText(text)
	entry.Disable()
	d := dialog.NewCustom("Logs: "+job.Name, "Close", container.NewVScroll(entry), s.window)
	d.Resize(fyne.NewSize(760, 500))
	d.Show()
}

func (s *state) duplicateJob(index int) {
	job := s.cfg.Jobs[index]
	job.ID = newID()
	job.Name += " copy"
	job.LastRun = nil
	job.LastScheduledRun = nil
	job.CreatedAt = time.Now()
	job.UpdatedAt = job.CreatedAt
	s.cfg.Jobs = append(s.cfg.Jobs, job)
	s.saveAndRefresh()
}

func (s *state) deleteJob(index int) {
	job := s.cfg.Jobs[index]
	dialog.ShowConfirm("Delete sync job", "Delete "+job.Name+"?", func(ok bool) {
		if !ok {
			return
		}
		s.cfg.Jobs = append(s.cfg.Jobs[:index], s.cfg.Jobs[index+1:]...)
		s.saveAndRefresh()
	}, s.window)
}

func (s *state) showRemoteDialog() {
	remoteName := widget.NewEntry()
	remoteName.SetText("nextcloud")
	server := widget.NewEntry()
	server.SetPlaceHolder("https://cloud.example.com")
	username := widget.NewEntry()
	password := widget.NewPasswordEntry()
	info := widget.NewLabel("Use a Nextcloud app password. It will be stored by rclone, not in Nextclone's settings file.")

	d := dialog.NewForm("Remote Setup", "Create remote", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Remote name", remoteName),
		widget.NewFormItem("Server URL", server),
		widget.NewFormItem("Username", username),
		widget.NewFormItem("App password", password),
		widget.NewFormItem("Note", info),
	}, func(save bool) {
		if !save {
			return
		}
		remote := strings.TrimSpace(remoteName.Text)
		user := strings.TrimSpace(username.Text)
		if remote == "" || user == "" || strings.TrimSpace(server.Text) == "" || password.Text == "" {
			dialog.ShowInformation("Missing information", "Remote name, server URL, username, and app password are required.", s.window)
			return
		}
		webdavURL := nextcloudWebDAVURL(server.Text, user)
		go func() {
			err := s.runner.CreateNextcloudRemote(context.Background(), remote, webdavURL, user, password.Text)
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			err = s.runner.TestRemote(context.Background(), remote)
			if err != nil {
				dialog.ShowError(err, s.window)
				return
			}
			dialog.ShowInformation("Remote ready", "Nextcloud remote created and tested successfully.", s.window)
		}()
	}, s.window)
	d.Resize(fyne.NewSize(620, 420))
	d.Show()
}

func (s *state) showSettingsDialog() {
	rclonePath := widget.NewEntry()
	rclonePath.SetText(s.cfg.Settings.RclonePath)
	retention := widget.NewEntry()
	retention.SetText(fmt.Sprintf("%d", s.cfg.Settings.LogRetentionDays))
	autoStart := widget.NewCheck("Start Nextclone in the background when I sign in", nil)
	autoStart.SetChecked(s.cfg.Settings.AutoStart || autostart.IsEnabled())

	d := dialog.NewForm("Settings", "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Rclone path", rclonePath),
		widget.NewFormItem("Log retention days", retention),
		widget.NewFormItem("Autostart", autoStart),
	}, func(save bool) {
		if !save {
			return
		}
		s.cfg.Settings.RclonePath = strings.TrimSpace(rclonePath.Text)
		var days int
		_, _ = fmt.Sscanf(retention.Text, "%d", &days)
		if days <= 0 {
			days = 30
		}
		s.cfg.Settings.LogRetentionDays = days
		if autoStart.Checked {
			if err := autostart.Enable(); err != nil {
				dialog.ShowError(err, s.window)
				return
			}
		} else if err := autostart.Disable(); err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		s.cfg.Settings.AutoStart = autoStart.Checked
		s.runner.Settings = s.cfg.Settings
		s.saveAndRefresh()
	}, s.window)
	d.Show()
}

func (s *state) checkRclone() {
	go func() {
		version, err := s.runner.Version(context.Background())
		if err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		dialog.ShowInformation("rclone found", version, s.window)
	}()
}

func (s *state) saveAndRefresh() {
	if err := config.Save(s.cfg); err != nil {
		dialog.ShowError(err, s.window)
	}
	s.runner.Settings = s.cfg.Settings
	s.refreshJobs()
}

func splitLines(value string) []string {
	var out []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func nextcloudWebDAVURL(server, username string) string {
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	return server + "/remote.php/dav/files/" + url.PathEscape(username) + "/"
}

func newID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}

func scheduleStatus(job config.SyncJob) string {
	if !job.Schedule.Enabled {
		return "Off"
	}
	return fmt.Sprintf("every %d day(s) at %02d:%02d", normalizedDays(job.Schedule.EveryNDays), normalizedHour(job.Schedule.AtHour), normalizedMinute(job.Schedule.AtMinute))
}

func scheduleOptions() []string {
	return []string{"Every day", "Every 2 days", "Every 3 days", "Every 7 days", "Every 14 days", "Every 30 days"}
}

func scheduleLabel(days int) string {
	switch normalizedDays(days) {
	case 2:
		return "Every 2 days"
	case 3:
		return "Every 3 days"
	case 7:
		return "Every 7 days"
	case 14:
		return "Every 14 days"
	case 30:
		return "Every 30 days"
	default:
		return "Every day"
	}
}

func scheduleDays(label string) int {
	var days int
	if _, err := fmt.Sscanf(label, "Every %d days", &days); err == nil && days > 0 {
		return days
	}
	return 1
}

func numberOptions(min, max int) []string {
	var values []string
	for i := min; i <= max; i++ {
		values = append(values, fmt.Sprintf("%02d", i))
	}
	return values
}

func normalizedDays(days int) int {
	if days <= 0 {
		return 1
	}
	return days
}

func normalizedHour(hour int) int {
	if hour < 0 || hour > 23 {
		return 2
	}
	return hour
}

func normalizedMinute(minute int) int {
	if minute < 0 || minute > 59 {
		return 0
	}
	return minute
}
