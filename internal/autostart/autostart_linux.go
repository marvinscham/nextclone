//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const desktopFileName = "nextclone.desktop"

func Enable() error {
	path, err := entryPath()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Nextclone
Comment=Run scheduled Nextclone backups in the background
Exec=%s --background
Icon=nextclone
Terminal=false
X-GNOME-Autostart-enabled=true
`, quoteDesktopArg(exe))
	return os.WriteFile(path, []byte(content), 0o644)
}

func Disable() error {
	path, err := entryPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsEnabled() bool {
	path, err := entryPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func entryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", desktopFileName), nil
}

func quoteDesktopArg(value string) string {
	return strconv.Quote(strings.ReplaceAll(value, `\`, `\\`))
}
