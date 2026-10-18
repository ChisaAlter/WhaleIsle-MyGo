//go:build linux && (amd64 || arm64)

package webview

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo"
)

// GTK uses logical window coordinates for both shapes; GDK applies the monitor
// scale itself. Multiplying by the scale here would cut the wrong area at HiDPI.
type occlusionGTKRect struct{ X, Y, Width, Height int32 }

type occlusionGTKAPI struct {
	window      func(uintptr) uintptr
	display     func(uintptr) uintptr
	shapes      func(uintptr) int32
	inputShapes func(uintptr) int32
	create      func(*occlusionGTKRect) uintptr
	status      func(uintptr) int32
	subtract    func(uintptr, *occlusionGTKRect) int32
	destroy     func(uintptr)
	shape       func(uintptr, uintptr, int32, int32)
	inputShape  func(uintptr, uintptr, int32, int32)
}

var occlusionGTK struct {
	sync.Once
	api occlusionGTKAPI
	ok  bool
}

func loadOcclusionGTK() bool {
	occlusionGTK.Do(func() {
		// MyGo has already loaded GTK/GDK/Cairo globally before making a
		// window. Reuse those libraries rather than loading another backend.
		a := &occlusionGTK.api
		for symbol, dst := range map[string]any{
			"gtk_widget_get_window":                 &a.window,
			"gdk_window_get_display":                &a.display,
			"gdk_display_supports_shapes":           &a.shapes,
			"gdk_display_supports_input_shapes":     &a.inputShapes,
			"cairo_region_create_rectangle":         &a.create,
			"cairo_region_status":                   &a.status,
			"cairo_region_subtract_rectangle":       &a.subtract,
			"cairo_region_destroy":                  &a.destroy,
			"gdk_window_shape_combine_region":       &a.shape,
			"gdk_window_input_shape_combine_region": &a.inputShape,
		} {
			address, err := purego.Dlsym(purego.RTLD_DEFAULT, symbol)
			if err != nil || address == 0 {
				return
			}
			purego.RegisterFunc(dst, address)
		}
		occlusionGTK.ok = true
	})
	return occlusionGTK.ok
}

func setNativeOcclusions(win *mygo.Window, width, height int, holes []mygo.Rectangle) {
	mygo.RunOnMain(func() {
		if win.IsDestroyed() || !loadOcclusionGTK() {
			return
		}
		occlusionGTK.api.apply(win.NativeHandle(), width, height, holes)
	})
}

// apply returns false when the backend cannot guarantee BOTH visible clipping
// and pointer exclusion. In particular Wayland does not support bounding
// shapes; installing an input-only hole there would leave misleading pixels.
// This is not parity on Wayland, and does not hide or replace its live view.
func (a *occlusionGTKAPI) apply(widget uintptr, width, height int, holes []mygo.Rectangle) bool {
	window := a.window(widget)
	if window == 0 { // Not realized yet; the next layout can apply the shape.
		return false
	}
	display := a.display(window)
	if display == 0 || a.shapes(display) == 0 || a.inputShapes(display) == 0 {
		return false
	}
	if len(holes) == 0 {
		a.shape(window, 0, 0, 0)
		a.inputShape(window, 0, 0, 0)
		return true
	}
	if width <= 0 || height <= 0 || width > 1<<31-1 || height > 1<<31-1 {
		return false
	}
	region := a.create(&occlusionGTKRect{Width: int32(width), Height: int32(height)})
	if region == 0 {
		return false
	}
	defer a.destroy(region) // GDK copies the region; ownership stays with us.
	if a.status(region) != 0 {
		return false
	}
	for _, r := range holes {
		if r.Width <= 0 || r.Height <= 0 {
			continue
		}
		left, top := max(0, r.X), max(0, r.Y)
		right, bottom := min(width, r.X+r.Width), min(height, r.Y+r.Height)
		if left >= right || top >= bottom {
			continue
		}
		hole := occlusionGTKRect{int32(left), int32(top), int32(right - left), int32(bottom - top)}
		if a.subtract(region, &hole) != 0 {
			return false // Keep the previous pair intact on allocation failure.
		}
	}
	a.shape(window, region, 0, 0)
	a.inputShape(window, region, 0, 0)
	return true
}
