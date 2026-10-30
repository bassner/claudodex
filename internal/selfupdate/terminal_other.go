//go:build !darwin && !linux

package selfupdate

import (
	"context"
	"errors"
)

func readTerminalLine(context.Context) (string, error) {
	return "", errors.New("interactive updates are unsupported on this platform")
}
