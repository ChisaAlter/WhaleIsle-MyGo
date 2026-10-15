//go:build !windows && !darwin && !(linux && (amd64 || arm64))

package screenglance

import (
	"context"
	"fmt"
)

func Capture(ctx context.Context, window uintptr) ([]byte, error) {
	return nil, fmt.Errorf("screen glance: desktop capture is unavailable on this platform")
}
