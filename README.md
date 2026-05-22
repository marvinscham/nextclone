# Nextclone

Nextclone is a Go/Fyne desktop app for independent local-to-Nextcloud copy or sync jobs using `rclone`.

## Current scope

- GUI dashboard for multiple jobs
- Local-to-Nextcloud only
- `copy` mode for safer uploads
- `sync` mode for mirroring local folders to Nextcloud
- Per-job schedules using simple "every N days at HH:MM" presets
- Optional system sign-in autostart for background scheduled backups
- Per-job logs
- Persistent JSON settings in the user config directory
- Nextcloud WebDAV remote setup through bundled or installed `rclone`
- GitHub Actions tag release pipeline with Linux, Windows, and `.deb` artifacts

## Development

Install Go and the Fyne system dependencies for your OS, then run:

```bash
make run
```

Run tests:

```bash
make test
```

Build executables:

```bash
make build-all
```

This writes Linux and Windows executables to `dist/`.

The repository includes a root-user devcontainer with the Linux GUI dependencies needed for Fyne development. On Linux hosts, allow X11 access before running the GUI from the container if needed, for example `xhost +local:root`.

## Background scheduling

Each sync job can be set to run automatically every 1, 2, 3, 7, 14, or 30 days at a selected local time. If the computer or background app was not running at the selected time, Nextclone runs the missed backup the next time the background scheduler checks after that time.

The Settings dialog can install or remove system sign-in autostart. Autostart launches Nextclone with `--background`, which runs scheduled backups without opening the app window.

Headless modes are also available directly:

```bash
nextclone --background
nextclone --run-due
```

## rclone

Nextclone looks for rclone in this order:

1. The configured rclone path in Settings
2. `NEXTCLONE_RCLONE_PATH`
3. A bundled `rclone` next to the executable
4. `rclone` on `PATH`

Release packages bundle rclone where possible.

## Release

Push a Git tag to GitHub:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The GitHub Actions pipeline builds tagged Linux and Windows artifacts, creates a Debian package, and publishes a GitHub Release.
