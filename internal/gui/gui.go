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
	"github.com/marvinscham/nextclone/internal/i18n"
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
	localizer *i18n.Localizer
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
		localizer: i18n.New(cfg.Settings.Language),
	}

	w.SetContent(s.dashboard())
	if err != nil {
		dialog.ShowError(err, w)
	}
	w.ShowAndRun()
}

func (s *state) dashboard() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(s.t("app.title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	add := widget.NewButtonWithIcon(s.t("dashboard.addSync"), theme.ContentAddIcon(), func() { s.showJobDialog(nil) })
	remote := widget.NewButton(s.t("dashboard.remoteSetup"), s.showRemoteDialog)
	settings := widget.NewButton(s.t("dashboard.settings"), s.showSettingsDialog)
	configFile := widget.NewButton(s.t("dashboard.configFile"), s.revealConfigFile)
	check := widget.NewButton(s.t("dashboard.checkRclone"), s.checkRclone)
	language := widget.NewButtonWithIcon("", assets.GlobeIcon, s.showLanguageDialog)

	header := container.NewBorder(nil, nil, container.NewVBox(title), container.NewHBox(add, remote, settings, configFile, check, language))
	s.refreshJobs()
	return container.NewBorder(header, nil, nil, nil, container.NewVScroll(s.jobsBox))
}

func (s *state) t(key string, args ...any) string {
	return s.localizer.T(key, args...)
}

func (s *state) showLanguageDialog() {
	languages := s.localizer.Languages()
	labels := make([]string, 0, len(languages))
	labelToCode := map[string]string{}
	selected := s.cfg.Settings.Language
	if selected == "" {
		selected = i18n.SystemLanguage
	}
	selectedLabel := ""
	for _, language := range languages {
		label := language.Name
		labels = append(labels, label)
		labelToCode[label] = language.Code
		if language.Code == selected {
			selectedLabel = label
		}
	}

	choice := widget.NewSelect(labels, nil)
	choice.SetSelected(selectedLabel)
	d := dialog.NewForm(s.t("language.title"), s.t("common.save"), s.t("common.cancel"), []*widget.FormItem{
		widget.NewFormItem(s.t("language.title"), choice),
	}, func(save bool) {
		if !save {
			return
		}
		code := labelToCode[choice.Selected]
		if code == "" {
			code = i18n.SystemLanguage
		}
		s.cfg.Settings.Language = code
		s.localizer = i18n.New(code)
		s.runner.Settings = s.cfg.Settings
		if err := config.Save(s.cfg); err != nil {
			dialog.ShowError(err, s.window)
			return
		}
		s.window.SetContent(s.dashboard())
	}, s.window)
	d.Show()
}

func (s *state) refreshJobs() {
	s.jobsBox.Objects = nil
	if len(s.cfg.Jobs) == 0 {
		s.jobsBox.Add(container.NewCenter(widget.NewLabel(s.t("dashboard.emptyJobs"))))
		s.jobsBox.Refresh()
		return
	}
	for i := range s.cfg.Jobs {
		idx := i
		job := s.cfg.Jobs[i]
		status := s.jobStatus(job)
		name := widget.NewLabelWithStyle(job.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		detail := widget.NewLabel(fmt.Sprintf("%s -> %s:%s", job.LocalPath, job.RemoteName, strings.TrimPrefix(job.RemotePath, "/")))
		mode := s.t("job.mode.copy")
		if job.Mode == "sync" {
			mode = s.t("job.mode.sync")
		}
		meta := widget.NewLabel(s.t("job.meta", mode, s.scheduleStatus(job), status))

		start := widget.NewButtonWithIcon(s.t("job.action.start"), theme.MediaPlayIcon(), func() { s.startJob(idx) })
		if _, running := s.cancel[job.ID]; running {
			start.Disable()
		}
		stop := widget.NewButtonWithIcon(s.t("job.action.stop"), theme.MediaStopIcon(), func() { s.stopJob(job.ID) })
		if _, running := s.cancel[job.ID]; !running {
			stop.Disable()
		}
		edit := widget.NewButtonWithIcon(s.t("job.action.edit"), theme.DocumentCreateIcon(), func() { s.showJobDialog(&idx) })
		logs := widget.NewButton(s.t("job.action.logs"), func() { s.showLogs(job) })
		duplicate := widget.NewButton(s.t("job.action.duplicate"), func() { s.duplicateJob(idx) })
		deleteBtn := widget.NewButtonWithIcon(s.t("job.action.delete"), theme.DeleteIcon(), func() { s.deleteJob(idx) })

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
		return s.t("job.status.notRun")
	}
	state := s.t("job.status.failed")
	if job.LastRun.Success {
		state = s.t("job.status.succeeded")
	}
	return s.t("job.lastRun", state, job.LastRun.EndedAt.Format("2006-01-02 15:04"))
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
	remoteName.PlaceHolder = s.t("sync.selectRemote")
	if job.RemoteName != "" {
		remoteName.SetSelected(job.RemoteName)
	}
	remotePath := widget.NewEntry()
	remotePath.SetText(job.RemotePath)
	remotePath.SetPlaceHolder("Backups/Documents")
	copyMode := s.t("sync.mode.copy")
	syncMode := s.t("sync.mode.sync")
	mode := widget.NewSelect([]string{copyMode, syncMode}, nil)
	if job.Mode == "sync" {
		mode.SetSelected(syncMode)
	} else {
		mode.SetSelected(copyMode)
	}
	dryRun := widget.NewCheck(s.t("sync.dryRun"), nil)
	dryRun.SetChecked(job.DryRun)
	scheduleEnabled := widget.NewCheck(s.t("sync.runAutomatically"), nil)
	scheduleEnabled.SetChecked(job.Schedule.Enabled)
	every := widget.NewSelect(s.scheduleOptions(), nil)
	every.SetSelected(s.scheduleLabel(job.Schedule.EveryNDays))
	hour := widget.NewSelect(numberOptions(0, 23), nil)
	hour.SetSelected(fmt.Sprintf("%02d", normalizedHour(job.Schedule.AtHour)))
	minute := widget.NewSelect(numberOptions(0, 59), nil)
	minute.SetSelected(fmt.Sprintf("%02d", normalizedMinute(job.Schedule.AtMinute)))
	excludes := widget.NewMultiLineEntry()
	excludes.SetMinRowsVisible(4)
	excludes.SetText(strings.Join(job.Excludes, "\n"))
	extra := widget.NewEntry()
	extra.SetText(job.ExtraFlags)
	extra.SetPlaceHolder(s.t("sync.advancedFlags"))

	browse := widget.NewButton(s.t("sync.browse"), func() {
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

	form := dialog.NewForm(s.t("sync.title"), s.t("common.save"), s.t("common.cancel"), []*widget.FormItem{
		widget.NewFormItem(s.t("sync.name"), name),
		widget.NewFormItem(s.t("sync.localFolder"), container.NewBorder(nil, nil, nil, browse, local)),
		widget.NewFormItem(s.t("sync.remoteName"), remoteName),
		widget.NewFormItem(s.t("sync.remotePath"), remotePath),
		widget.NewFormItem(s.t("sync.mode"), mode),
		widget.NewFormItem(s.t("sync.options"), dryRun),
		widget.NewFormItem(s.t("sync.schedule"), scheduleEnabled),
		widget.NewFormItem(s.t("schedule.every"), every),
		widget.NewFormItem(s.t("sync.at"), container.NewHBox(hour, widget.NewLabel(":"), minute)),
		widget.NewFormItem(s.t("sync.excludePatterns"), excludes),
		widget.NewFormItem(s.t("sync.extraFlags"), extra),
	}, func(save bool) {
		if !save {
			return
		}
		if strings.TrimSpace(name.Text) == "" || strings.TrimSpace(local.Text) == "" || strings.TrimSpace(remoteName.Selected) == "" {
			dialog.ShowInformation(s.t("dialog.missingInfo.title"), s.t("sync.missingInfo"), s.window)
			return
		}
		job.Name = strings.TrimSpace(name.Text)
		job.LocalPath = strings.TrimSpace(local.Text)
		job.RemoteName = strings.TrimSpace(remoteName.Selected)
		job.RemotePath = strings.TrimSpace(remotePath.Text)
		job.Mode = "copy"
		if mode.Selected == syncMode {
			job.Mode = "sync"
		}
		if job.Mode != "sync" {
			job.Mode = "copy"
		}
		job.DryRun = dryRun.Checked
		job.Schedule.Enabled = scheduleEnabled.Checked
		job.Schedule.EveryNDays = s.scheduleDays(every.Selected)
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
	s.status[job.ID] = s.t("job.status.running")
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
			s.status[job.ID] = s.t("job.status.success")
		} else {
			s.status[job.ID] = s.t("job.status.failedMessage", result.Message)
		}
		s.refreshJobs()
	}()
}

func (s *state) stopJob(id string) {
	if cancel := s.cancel[id]; cancel != nil {
		cancel()
		s.status[id] = s.t("job.status.stopping")
		s.refreshJobs()
	}
}

func (s *state) showLogs(job config.SyncJob) {
	text := s.t("logs.empty")
	if lines := s.liveLogs[job.ID]; len(lines) > 0 {
		text = strings.Join(lines, "\n")
	} else if job.LastRun != nil {
		text = s.t("logs.lastFile", job.LastRun.LogPath, job.LastRun.Message)
	}
	entry := widget.NewMultiLineEntry()
	entry.SetText(text)
	entry.Disable()
	d := dialog.NewCustom(s.t("logs.title", job.Name), s.t("common.close"), container.NewVScroll(entry), s.window)
	d.Resize(fyne.NewSize(760, 500))
	d.Show()
}

func (s *state) duplicateJob(index int) {
	job := s.cfg.Jobs[index]
	job.ID = newID()
	job.Name += s.t("job.duplicateSuffix")
	job.LastRun = nil
	job.LastScheduledRun = nil
	job.CreatedAt = time.Now()
	job.UpdatedAt = job.CreatedAt
	s.cfg.Jobs = append(s.cfg.Jobs, job)
	s.saveAndRefresh()
}

func (s *state) deleteJob(index int) {
	job := s.cfg.Jobs[index]
	dialog.ShowConfirm(s.t("job.delete.title"), s.t("job.delete.confirm", job.Name), func(ok bool) {
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
	appPassword := widget.NewButton(s.t("remote.appPassword"), func() {
		appURL := nextcloudAppPasswordURL(server.Text)
		if appURL == nil {
			return
		}
		if err := s.app.OpenURL(appURL); err != nil {
			dialog.ShowError(err, s.window)
		}
	})
	appPassword.Disable()
	server.OnChanged = func(_ string) {
		if nextcloudAppPasswordURL(server.Text) == nil {
			appPassword.Disable()
			return
		}
		appPassword.Enable()
	}
	username := widget.NewEntry()
	password := widget.NewPasswordEntry()
	info := widget.NewLabel(s.t("remote.info"))

	d := dialog.NewForm(s.t("remote.title"), s.t("remote.submit"), s.t("common.cancel"), []*widget.FormItem{
		widget.NewFormItem(s.t("sync.remoteName"), remoteName),
		widget.NewFormItem(s.t("remote.serverURL"), server),
		widget.NewFormItem("", appPassword),
		widget.NewFormItem(s.t("remote.username"), username),
		widget.NewFormItem(s.t("remote.appPasswordLabel"), password),
		widget.NewFormItem(s.t("remote.note"), info),
	}, func(save bool) {
		if !save {
			return
		}
		remote := strings.TrimSpace(remoteName.Text)
		user := strings.TrimSpace(username.Text)
		if remote == "" || user == "" || strings.TrimSpace(server.Text) == "" || password.Text == "" {
			dialog.ShowInformation(s.t("dialog.missingInfo.title"), s.t("remote.missingInfo"), s.window)
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
			dialog.ShowInformation(s.t("remote.ready.title"), s.t("remote.ready.message"), s.window)
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
	uploadLimit := widget.NewSelect(s.uploadLimitOptions(), nil)
	uploadLimit.SetSelected(s.uploadLimitLabel(s.cfg.Settings.UploadLimit))
	autoStart := widget.NewCheck(s.t("settings.startInBackground"), nil)
	autoStart.SetChecked(s.cfg.Settings.AutoStart || autostart.IsEnabled())

	d := dialog.NewForm(s.t("settings.title"), s.t("common.save"), s.t("common.cancel"), []*widget.FormItem{
		widget.NewFormItem(s.t("settings.rclonePath"), rclonePath),
		widget.NewFormItem(s.t("settings.logRetentionDays"), retention),
		widget.NewFormItem(s.t("settings.uploadLimit"), uploadLimit),
		widget.NewFormItem(s.t("settings.autostart"), autoStart),
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
		s.cfg.Settings.UploadLimit = s.uploadLimitValue(uploadLimit.Selected)
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
		dialog.ShowInformation(s.t("rclone.found"), version, s.window)
	}()
}

func (s *state) revealConfigFile() {
	if err := config.Save(s.cfg); err != nil {
		dialog.ShowError(err, s.window)
		return
	}
	path, err := config.ConfigPath()
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}
	if err := revealFile(path); err != nil {
		dialog.ShowError(err, s.window)
	}
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

func nextcloudAppPasswordURL(server string) *url.URL {
	server = strings.TrimRight(strings.TrimSpace(server), "/")
	if server == "" {
		return nil
	}
	instanceURL, err := url.Parse(server)
	if err != nil || instanceURL.Scheme == "" || instanceURL.Host == "" {
		return nil
	}
	if instanceURL.Scheme != "http" && instanceURL.Scheme != "https" {
		return nil
	}
	instanceURL.Path = strings.TrimRight(instanceURL.Path, "/") + "/settings/user/security"
	instanceURL.RawQuery = ""
	instanceURL.Fragment = ""
	return instanceURL
}

func newID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}

func (s *state) scheduleStatus(job config.SyncJob) string {
	if !job.Schedule.Enabled {
		return s.t("schedule.off")
	}
	return s.t("schedule.status", normalizedDays(job.Schedule.EveryNDays), normalizedHour(job.Schedule.AtHour), normalizedMinute(job.Schedule.AtMinute))
}

func (s *state) scheduleOptions() []string {
	return []string{s.scheduleLabel(1), s.scheduleLabel(2), s.scheduleLabel(3), s.scheduleLabel(7), s.scheduleLabel(14), s.scheduleLabel(30)}
}

func (s *state) scheduleLabel(days int) string {
	switch normalizedDays(days) {
	case 2:
		return s.t("sync.every.2")
	case 3:
		return s.t("sync.every.3")
	case 7:
		return s.t("sync.every.7")
	case 14:
		return s.t("sync.every.14")
	case 30:
		return s.t("sync.every.30")
	default:
		return s.t("sync.every.1")
	}
}

func (s *state) scheduleDays(label string) int {
	for _, days := range []int{1, 2, 3, 7, 14, 30} {
		if label == s.scheduleLabel(days) {
			return days
		}
	}
	return 1
}

func (s *state) uploadLimitOptions() []string {
	return []string{
		s.t("uploadLimit.unlimited"),
		"512 KiB/s",
		"1 MiB/s",
		"2 MiB/s",
		"5 MiB/s",
		"10 MiB/s",
		"25 MiB/s",
		"50 MiB/s",
		"100 MiB/s",
	}
}

func (s *state) uploadLimitLabel(value string) string {
	switch strings.TrimSpace(value) {
	case "512K":
		return "512 KiB/s"
	case "1M":
		return "1 MiB/s"
	case "2M":
		return "2 MiB/s"
	case "5M":
		return "5 MiB/s"
	case "10M":
		return "10 MiB/s"
	case "25M":
		return "25 MiB/s"
	case "50M":
		return "50 MiB/s"
	case "100M":
		return "100 MiB/s"
	default:
		return s.t("uploadLimit.unlimited")
	}
}

func (s *state) uploadLimitValue(label string) string {
	switch label {
	case "512 KiB/s":
		return "512K"
	case "1 MiB/s":
		return "1M"
	case "2 MiB/s":
		return "2M"
	case "5 MiB/s":
		return "5M"
	case "10 MiB/s":
		return "10M"
	case "25 MiB/s":
		return "25M"
	case "50 MiB/s":
		return "50M"
	case "100 MiB/s":
		return "100M"
	default:
		return ""
	}
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
