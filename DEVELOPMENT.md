# Development

Nextclone is a Go/Fyne desktop app for independent local-to-Nextcloud copy or sync jobs using `rclone`.

## Local Development

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

## Background Scheduling

Each sync job can be set to run automatically every 1, 2, 3, 7, 14, or 30 days at a selected local time. If the computer or background app was not running at the selected time, Nextclone runs the missed backup the next time the background scheduler checks after that time.

The Settings dialog can install or remove system sign-in autostart. Autostart launches Nextclone with `--background`, which runs scheduled backups without opening the app window.

Headless modes are also available directly:

```bash
nextclone --background
nextclone --run-due
```

## rclone Lookup Order

Nextclone looks for rclone in this order:

1. The configured rclone path in Settings
2. `NEXTCLONE_RCLONE_PATH`
3. A bundled `rclone` next to the executable
4. `rclone` on `PATH`

Release packages bundle rclone where possible.

## Release

Merges to `main` are versioned automatically from Conventional Commit messages since the latest `v*` tag:

- `feat:` bumps the minor version
- `fix:`, `perf:`, and `revert:` bump the patch version
- `!` in the commit header or `BREAKING CHANGE:` in the body bumps the major version

The version workflow updates `VERSION`, generates `CHANGELOG.md`, commits `chore(release): v{new_version}`, and tags that bump commit as `v{new_version}`. The tag release pipeline builds Linux and Windows artifacts, creates a Debian package, and publishes a GitHub Release.
