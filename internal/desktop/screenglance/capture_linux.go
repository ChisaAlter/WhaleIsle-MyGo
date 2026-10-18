//go:build linux && (amd64 || arm64)

package screenglance

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// Capture uses the desktop's interactive Screenshot portal on X11 and Wayland.
// Portal selection is explicit: it is not an automatic pet-monitor capture.
func Capture(ctx context.Context, window uintptr) ([]byte, error) {
	if window == 0 {
		return nil, fmt.Errorf("screen glance: missing pet window")
	}
	uri, captureErr := mygo.CaptureDesktopScreenshot(ctx)
	if uri == "" {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("screen glance: screenshot cancelled, denied, or interactive XDG Screenshot portal unavailable")
	}
	location, err := url.Parse(uri)
	if err != nil || location.Scheme != "file" || (location.Host != "" && location.Host != "localhost") || !filepath.IsAbs(location.Path) {
		return nil, fmt.Errorf("screen glance: portal did not return a local image")
	}
	// Some portals expose a read-only document grant: cleanup then remains with
	// the desktop. The approval text explicitly discloses that possibility.
	defer os.Remove(location.Path)
	if captureErr != nil {
		return nil, captureErr
	}
	return captureFile(ctx, location.Path)
}
