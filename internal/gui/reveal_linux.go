//go:build linux

package gui

import (
	"os/exec"
	"path/filepath"
)

func revealFile(path string) error {
	return exec.Command("xdg-open", filepath.Dir(path)).Start()
}
