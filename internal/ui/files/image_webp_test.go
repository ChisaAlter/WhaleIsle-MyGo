package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestNativeWebPFilesPlaybackAndInactiveRelease(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/animated-native.webp")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "moving.webp"), data, 0600); err != nil {
		t.Fatal(err)
	}
	p := loadImagePreview(root, "moving.webp")
	if p.err != nil || p.animation == nil || len(p.frames) != 4 || len(p.animation.Frames) != 0 {
		t.Fatalf("native animation/staging: %+v", p)
	}
	start := time.Unix(100, 0)
	p.advance(start)
	if wait := p.advance(start.Add(110 * time.Millisecond)); wait != 90*time.Millisecond || p.bitmap != p.frames[2] {
		t.Fatal("WebP not advanced", wait)
	}
	// Paint the selected bitmap through the real native image view.
	tt := ui.NewTester(func(c *ui.Context) { paintImageView(c, p, &imageViewState{}) }, 200, 120)
	green := false
	img := tt.Image()
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			pixel := img.RGBAAt(x, y)
			if pixel.G > 240 && pixel.R < 20 && pixel.B < 20 {
				green = true
			}
		}
	}
	if !green {
		t.Fatal("current WebP bitmap was not painted green")
	}
	if wait := p.advance(start.Add(time.Second)); wait != 0 || p.bitmap != p.frames[3] {
		t.Fatal("finite WebP did not stop")
	}
	a, b := &FileTab{Path: "moving.webp", image: p}, &FileTab{Path: "other.png"}
	m := Model{WorkspaceRoot: root, Tabs: &Tabs{}}
	tabsFor(m).Open = []*FileTab{a, b}
	retainImagePreview(m, b, &imagePreview{})
	if a.image != nil {
		t.Fatal("inactive WebP tab retains animation")
	}
}
