package selfupdate

import "golang.org/x/sys/unix"

func flushTerminalInput() error {
	fd, err := unix.Open("/dev/tty", unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH)
}
