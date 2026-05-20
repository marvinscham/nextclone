# Nextclone

Nextclone is a Go/Fyne desktop app for manually starting independent local-to-Nextcloud copy or sync jobs using `rclone`.

## Current scope

- GUI dashboard for multiple manual jobs
- Local-to-Nextcloud only
- `copy` mode for safer uploads
- `sync` mode for mirroring local folders to Nextcloud
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
