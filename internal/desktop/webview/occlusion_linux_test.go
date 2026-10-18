//go:build linux && (amd64 || arm64)

package webview

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo"
)

// No GTK, display server, or browser process is needed for these contract tests.
func TestGTKNativeOcclusionPair(t *testing.T) {
	var holes []occlusionGTKRect
	var calls []string
	var installed []uintptr
	api := occlusionGTKAPI{
		window:      func(uintptr) uintptr { return 11 },
		display:     func(uintptr) uintptr { return 12 },
		shapes:      func(uintptr) int32 { return 1 },
		inputShapes: func(uintptr) int32 { return 1 },
		create: func(r *occlusionGTKRect) uintptr {
			if *r != (occlusionGTKRect{Width: 200, Height: 100}) {
				t.Fatalf("logical bounds changed: %+v", r)
			}
			return 13
		},
		status:   func(uintptr) int32 { return 0 },
		subtract: func(_ uintptr, r *occlusionGTKRect) int32 { holes = append(holes, *r); return 0 },
		destroy:  func(uintptr) { calls = append(calls, "destroy") },
		shape: func(_ uintptr, r uintptr, _, _ int32) {
			calls = append(calls, "visual")
			installed = append(installed, r)
		},
		inputShape: func(_ uintptr, r uintptr, _, _ int32) {
			calls = append(calls, "input")
			installed = append(installed, r)
		},
	}
	if !api.apply(1, 200, 100, []mygo.Rectangle{
		{X: -10, Y: 20, Width: 30, Height: 100},
		{X: 150, Y: 10, Width: 90, Height: 20},
		{X: 210, Width: 20, Height: 20},
		{Width: -1, Height: 20},
	}) {
		t.Fatal("supported backend rejected region")
	}
	if !reflect.DeepEqual(holes, []occlusionGTKRect{{0, 20, 20, 80}, {150, 10, 50, 20}}) {
		t.Fatalf("clipped logical holes: %+v", holes)
	}
	if !reflect.DeepEqual(calls, []string{"visual", "input", "destroy"}) || !reflect.DeepEqual(installed, []uintptr{13, 13}) {
		t.Fatalf("shape pair/ownership: %v %v", calls, installed)
	}
	calls, installed = nil, nil
	if !api.apply(1, 200, 100, nil) || !reflect.DeepEqual(installed, []uintptr{0, 0}) {
		t.Fatalf("restoration did not clear both shapes: %v", installed)
	}
	calls, installed = nil, nil
	api.subtract = func(uintptr, *occlusionGTKRect) int32 { return 1 }
	if api.apply(1, 200, 100, []mygo.Rectangle{{Width: 10, Height: 10}}) || !reflect.DeepEqual(calls, []string{"destroy"}) {
		t.Fatalf("failed region changed active shapes or leaked: %v", calls)
	}
	calls = nil
	api.shapes = func(uintptr) int32 { return 0 }
	if api.apply(1, 200, 100, []mygo.Rectangle{{Width: 10, Height: 10}}) || len(calls) != 0 {
		t.Fatal("visual-shape unsupported backend installed partial clipping")
	}
	api.shapes = func(uintptr) int32 { return 1 }
	api.inputShapes = func(uintptr) int32 { return 0 }
	if api.apply(1, 200, 100, []mygo.Rectangle{{Width: 10, Height: 10}}) || len(calls) != 0 {
		t.Fatal("input-shape unsupported backend installed partial clipping")
	}
	api.window = func(uintptr) uintptr { return 0 }
	if api.apply(1, 200, 100, nil) {
		t.Fatal("unrealized widget reported applied clipping")
	}
}
