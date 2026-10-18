package files

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeGIFAnimationAndCacheLifetime(t *testing.T) {
	root := t.TempDir()
	palette := color.Palette{color.Black, color.White}
	first := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	second := image.NewPaletted(first.Rect, palette)
	second.SetColorIndex(0, 0, 1)
	var data bytes.Buffer
	if err := gif.EncodeAll(&data, &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{5, 7}, LoopCount: -1}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "moving.gif"), data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	p := loadImagePreview(root, "moving.gif")
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.animation == nil || len(p.frames) != 2 || len(p.animation.Frames) != 0 {
		t.Fatal("animation frames or staging cleanup missing")
	}
	start := time.Unix(100, 0)
	if wait := p.advance(start); wait != 50*time.Millisecond || p.bitmap != p.frames[0] {
		t.Fatal("first frame", wait)
	}
	if wait := p.advance(start.Add(60 * time.Millisecond)); wait != 60*time.Millisecond || p.bitmap != p.frames[1] {
		t.Fatal("second frame", wait)
	}
	if wait := p.advance(start.Add(time.Second)); wait != 0 || p.bitmap != p.frames[1] {
		t.Fatal("finite animation did not stop", wait)
	}
	a, b := &FileTab{Path: "moving.gif", image: p}, &FileTab{Path: "next.png"}
	a.imageView.scale = 2
	a.imageView.scroll.X = 17
	a.Draft = "unsaved owner draft"
	m := Model{WorkspaceRoot: root, Tabs: &Tabs{}}
	tabsFor(m).Open = []*FileTab{a, b}
	retainImagePreview(m, b, &imagePreview{})
	if a.image != nil || b.image == nil {
		t.Fatal("inactive frame cache retained")
	}
	if a.imageView.scale != 2 || a.imageView.scroll.X != 17 || a.Draft != "unsaved owner draft" {
		t.Fatal("cache cleanup changed owner state")
	}
}
