//go:build windows

package auth

import "os/exec"

func configureNativeProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error { return cmd.Process.Kill() }
}
