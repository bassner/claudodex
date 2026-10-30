//go:build !darwin && !linux

package selfupdate

import "errors"

func flushTerminalInput() error {
	return errors.New("interactive updates are unsupported on this platform")
}
