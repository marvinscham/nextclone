//go:build !darwin && !linux && !windows

package gui

import "fmt"

func revealFile(path string) error {
	return fmt.Errorf("revealing files is not supported on this platform")
}
