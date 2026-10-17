//go:build (!windows && !darwin) || (darwin && !cgo)

package voicecapture

import "context"

// Start does not acquire hardware on unsupported platforms.
func Start(ctx context.Context, limits Limits) (Capture, error) {
	return nil, ErrUnsupported
}
