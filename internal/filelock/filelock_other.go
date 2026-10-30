//go:build !unix

package filelock

import (
	"context"
	"fmt"
)

func Acquire(ctx context.Context, path string) (func(), error) {
	return nil, fmt.Errorf("process file locks are unsupported on this platform")
}
