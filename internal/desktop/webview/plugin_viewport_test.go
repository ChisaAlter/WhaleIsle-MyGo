package webview

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
)

func TestPluginClippingPreservesFullViewportAndSubtractsNativeMenu(t *testing.T) {
	bounds := mygo.Rectangle{X: -40, Y: -70, Width: 600, Height: 560}
	clip := mygo.Rectangle{X: 20, Y: 30, Width: 400, Height: 300}
	visible := intersectMediaRect(bounds, clip)
	holes := clippedSurfaceHoles(bounds, visible, []mygo.Rectangle{{X: 100, Y: 100, Width: 80, Height: 90}})
	want := []mygo.Rectangle{{Width: 600, Height: 100}, {Y: 400, Width: 600, Height: 160}, {Y: 100, Width: 60, Height: 300}, {X: 460, Y: 100, Width: 140, Height: 300}, {X: 140, Y: 170, Width: 80, Height: 90}}
	if !slices.Equal(holes, want) {
		t.Fatalf("scrolled/offset viewport clipping %v want %v", holes, want)
	}
	if bounds.Width != 600 || bounds.Height != 560 {
		t.Fatal("clip resized/reflowed original viewport")
	}
}
