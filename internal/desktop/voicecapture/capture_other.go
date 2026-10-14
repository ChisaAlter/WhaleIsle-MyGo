//go:build !windows

package voicecapture

import "context"

// Start does not acquire hardware on unsupported platforms.
func Start(ctx context.Context, limits Limits) (Capture, error) {
	return nil, ErrUnsupported
}
