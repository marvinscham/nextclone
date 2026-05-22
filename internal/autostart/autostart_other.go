//go:build !linux && !windows

package autostart

import "errors"

var errUnsupported = errors.New("autostart is not supported on this platform")

func Enable() error {
	return errUnsupported
}

func Disable() error {
	return errUnsupported
}

func IsEnabled() bool {
	return false
}
