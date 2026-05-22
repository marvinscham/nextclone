//go:build windows

package autostart

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "Nextclone"
)

func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(valueName, fmt.Sprintf("%q --background", exe))
}

func Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(valueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

func IsEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(valueName)
	return err == nil && strings.Contains(value, "--background")
}
