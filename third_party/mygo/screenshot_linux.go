//go:build linux && (amd64 || arm64)

package mygo

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/internal/linux"
)

// CaptureDesktopScreenshot requests a single user-selected screenshot through
// the Linux desktop portal. Call from a worker goroutine, never the UI thread.
// The returned file URI is owned by the portal; callers must not persist it.
func CaptureDesktopScreenshot(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if isMainThread() {
		return "", fmt.Errorf("desktop screenshot must run off the UI thread")
	}
	type response struct {
		uri string
		err error
	}
	ready := make(chan response, 1)
	var cancel func()
	var startErr error
	onMain(func() {
		if ctx.Err() != nil {
			startErr = ctx.Err()
			return
		}
		cancel, startErr = linux.RequestDesktopScreenshot(func(uri string, err error) { ready <- response{uri, err} })
	})
	if startErr != nil {
		return "", startErr
	}
	if cancel == nil {
		return "", fmt.Errorf("desktop application is closed")
	}
	select {
	case result := <-ready:
		return result.uri, result.err
	case <-ctx.Done():
		onMain(cancel)
		select {
		case result := <-ready:
			return result.uri, ctx.Err()
		default:
			return "", ctx.Err()
		}
	}
}
