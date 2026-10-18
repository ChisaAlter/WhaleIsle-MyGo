package files

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestB23ImageViewZoomAndTabState(t *testing.T) {
	if got := imageFitScale(524, 1000); got != .5 {
		t.Fatalf("fit width = %v", got)
	}
	if got := imageZoomStep(imageFitScale(624, 1000), 1); got != .75 {
		t.Fatalf("fit-to-fixed = %v", got)
	}
	if got := imageFitScale(1024, 100); got != 1 {
		t.Fatalf("fit enlarged a small image: %v", got)
	}
	for _, tc := range []struct {
		scale     float32
		direction int
		want      float32
	}{{.25, -1, .25}, {4, 1, 4}, {.6, 1, .75}, {.6, -1, .5}, {1, 1, 1.25}, {1, -1, .75}} {
		if got := imageZoomStep(tc.scale, tc.direction); got != tc.want {
			t.Fatalf("step %+v = %v", tc, got)
		}
	}
	a, b := &FileTab{}, &FileTab{}
	a.imageView.viewport = ui.Rect{W: 400, H: 300}
	a.imageView.scroll.X, a.imageView.scroll.Y = 100, 200
	a.imageView.zoom(1, 2)
	a.image = nil // decoding is released when another file is selected
	if a.imageView.scale != 2 || a.imageView.scroll.X != 388 || a.imageView.scroll.Y != 538 {
		t.Fatalf("lost image view: %+v", a.imageView)
	}
	if b.imageView.scale != 0 || b.imageView.scroll.X != 0 || b.imageView.scroll.Y != 0 {
		t.Fatal("image tabs share view state")
	}
}

func TestB23ImageWheelZoom(t *testing.T) {
	v := imageViewState{viewport: ui.Rect{W: 624, H: 400}}
	if v.wheel(ui.InputEvent{Kind: ui.InputScroll, DY: 20}, 1000) {
		t.Fatal("ordinary scrolling was consumed")
	}
	if !v.wheel(ui.InputEvent{Kind: ui.InputScroll, Mods: ui.Ctrl, DY: -20, X: 100, Y: 100}, 1000) {
		t.Fatal("Ctrl+wheel was not handled")
	}
	if v.scale <= .6 || v.scale >= 1 {
		t.Fatalf("pinch did not start from actual fit scale: %v", v.scale)
	}
	v.scale = 4
	v.wheel(ui.InputEvent{Kind: ui.InputScroll, Mods: ui.Ctrl, DY: -10000}, 1000)
	if v.scale != 4 {
		t.Fatalf("upper gesture limit: %v", v.scale)
	}
	v.scale = .25
	v.wheel(ui.InputEvent{Kind: ui.InputScroll, Mods: ui.Ctrl, DY: 10000}, 1000)
	if v.scale != .25 {
		t.Fatalf("lower gesture limit: %v", v.scale)
	}
}
