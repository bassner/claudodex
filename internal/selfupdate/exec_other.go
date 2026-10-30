//go:build !darwin && !linux

package selfupdate

import "errors"

func replaceProcess(string, []string, []string) error {
	return errors.New("same-launch updates are unsupported on this platform")
}
