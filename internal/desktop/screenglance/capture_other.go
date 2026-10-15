//go:build !windows

package screenglance

import (
	"context"
	"fmt"
)

func Capture(ctx context.Context, window uintptr) ([]byte, error) {
	return nil, fmt.Errorf("screen glance: desktop capture is unavailable on this platform")
}
