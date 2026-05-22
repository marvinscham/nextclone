//go:build darwin

package gui

import "os/exec"

func revealFile(path string) error {
	return exec.Command("open", "-R", path).Start()
}
