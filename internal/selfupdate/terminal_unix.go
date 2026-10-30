//go:build darwin || linux

package selfupdate

import (
	"context"
	"errors"
	"io"
	"time"

	"golang.org/x/sys/unix"
)

func readTerminalLine(ctx context.Context) (string, error) {
	fd, err := unix.Open("/dev/tty", unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", err
	}
	defer unix.Close(fd)
	var answer []byte
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		var b [1]byte
		n, err := unix.Read(fd, b[:])
		if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
			// Darwin poll(2) can report POLLNVAL for /dev/tty. A short,
			// cancelable retry of the nonblocking read works on both platforms.
			timer := time.NewTimer(25 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", io.EOF
		}
		if b[0] == '\n' {
			return string(answer), nil
		}
		if len(answer) >= 256 {
			return "", errors.New("update answer too long")
		}
		answer = append(answer, b[0])
	}
}
