//go:build darwin

package screenglance

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"structs"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo"
)

type screenPoint struct {
	_    structs.HostLayout
	X, Y float64
}
type screenSize struct {
	_             structs.HostLayout
	Width, Height float64
}
type screenRect struct {
	_      structs.HostLayout
	Origin screenPoint
	Size   screenSize
}

var screenAccess struct {
	once      sync.Once
	preflight func() bool
	err       error
}

func screenRecordingAllowed() (bool, error) {
	screenAccess.once.Do(func() {
		lib, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_NOW|purego.RTLD_LOCAL)
		if err != nil {
			screenAccess.err = err
			return
		}
		address, err := purego.Dlsym(lib, "CGPreflightScreenCaptureAccess")
		if err != nil {
			screenAccess.err = err
			return
		}
		purego.RegisterFunc(&screenAccess.preflight, address)
	})
	if screenAccess.err != nil {
		return false, screenAccess.err
	}
	return screenAccess.preflight(), nil
}

// Capture invokes Apple's installed screenshot utility after the app approval.
// TCC permission is checked, never granted or changed by this code. A private
// temporary image is removed after conversion; the system handles OS consent.
func Capture(ctx context.Context, window uintptr) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if window == 0 {
		return nil, fmt.Errorf("screen glance: missing pet window")
	}
	allowed, err := screenRecordingAllowed()
	if err != nil || !allowed {
		return nil, fmt.Errorf("screen glance: enable Screen Recording for WhaleIsle in macOS System Settings, then retry")
	}
	region := ""
	mygo.RunOnMain(func() {
		if ctx.Err() != nil {
			return
		}
		pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(objc.RegisterName("new"))
		defer pool.Send(objc.RegisterName("drain"))
		// Resolve only a currently registered application window, not a stale pointer.
		app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
		windows := app.Send(objc.RegisterName("windows"))
		count := objc.Send[uintptr](windows, objc.RegisterName("count"))
		var pet objc.ID
		for i := uintptr(0); i < count; i++ {
			candidate := windows.Send(objc.RegisterName("objectAtIndex:"), i)
			if uintptr(candidate) == window {
				pet = candidate
				break
			}
		}
		if pet == 0 {
			return
		}
		screen := pet.Send(objc.RegisterName("screen"))
		screens := objc.ID(objc.GetClass("NSScreen")).Send(objc.RegisterName("screens"))
		if screen == 0 || objc.Send[uintptr](screens, objc.RegisterName("count")) == 0 {
			return
		}
		primary := screens.Send(objc.RegisterName("objectAtIndex:"), uintptr(0))
		bounds := objc.Send[screenRect](screen, objc.RegisterName("frame"))
		mainBounds := objc.Send[screenRect](primary, objc.RegisterName("frame"))
		// AppKit uses bottom-left coordinates; screencapture -R uses top-left.
		x := int(math.Round(bounds.Origin.X))
		y := int(math.Round(mainBounds.Size.Height - bounds.Origin.Y - bounds.Size.Height))
		w, h := int(math.Round(bounds.Size.Width)), int(math.Round(bounds.Size.Height))
		if w > 0 && h > 0 {
			region = fmt.Sprintf("%d,%d,%d,%d", x, y, w, h)
		}
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if region == "" {
		return nil, fmt.Errorf("screen glance: pet display unavailable")
	}
	dir, err := os.MkdirTemp("", "whaleisle-glance-")
	if err != nil {
		return nil, fmt.Errorf("screen glance: temporary screenshot directory unavailable")
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "screen.png")
	if err := exec.CommandContext(ctx, "/usr/sbin/screencapture", "-x", "-t", "png", "-R", region, path).Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("screen glance: macOS denied or could not capture the pet display; check Screen Recording permission")
	}
	return captureFile(ctx, path)
}
