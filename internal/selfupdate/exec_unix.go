//go:build darwin || linux

package selfupdate

import "syscall"

func replaceProcess(path string, argv, env []string) error { return syscall.Exec(path, argv, env) }
