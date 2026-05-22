//go:build windows

package gui

import "os/exec"

func revealFile(path string) error {
	return exec.Command("explorer.exe", "/select,", path).Start()
}
