//go:build windows && (amd64 || arm64)

package windows

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

func TestCaptionOverlayGeometryDoesNotExpandTitleStrip(t *testing.T) {
	w := &window{opts: &platform.WindowOptions{TitleBarHeight: 12}}
	if a, b, c, d := w.captionGeometry(); a != 46 || b != 12 || c != 0 || d != 0 {
		t.Fatalf("SDK defaults changed: %d %d %d %d", a, b, c, d)
	}
	w.opts.WindowsCaptionButtons = &platform.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 8, Right: 8, CornerRadius: 8, GlyphSize: 12}
	if a, b, c, d := w.captionGeometry(); a != 32 || b != 32 || c != 8 || d != 8 || w.titleBarHeight() != 12 {
		t.Fatal("caption overlay displaced the native title strip")
	}
	if buttonMaximize.hitTest() != htMaxButton || buttonMinimize.hitTest() != htMinButton || buttonClose.hitTest() != htClose {
		t.Fatal("native caption hit codes lost Snap/close semantics")
	}
}

func TestCaptionRoundedCoverageGeometryAndResize(t *testing.T) {
	var raster roundedCaptionRaster
	first := raster.coverage(32, 32, 12, 8, false)
	if len(first) != 3*32*32 || first[0]&255 != 0 || first[16*32+16]&255 != 255 {
		t.Fatal("rounded plate has incorrect extent/corner/center")
	}
	if &first[0] != &raster.coverage(32, 32, 12, 8, false)[0] {
		t.Fatal("unchanged hover geometry reallocated glyph coverage")
	}
	var normal, restored uint64
	for _, v := range first[32*32 : 2*32*32] {
		normal = normal*31 + uint64(v>>8)
	}
	second := raster.coverage(32, 32, 12, 8, true)
	for _, v := range second[32*32 : 2*32*32] {
		restored = restored*31 + uint64(v>>8)
	}
	if normal == restored {
		t.Fatal("maximize did not replace its SVG glyph with restore")
	}
	if got := raster.coverage(48, 48, 18, 12, true); len(got) != 3*48*48 {
		t.Fatal("DPI resize retained stale coverage")
	}
}
