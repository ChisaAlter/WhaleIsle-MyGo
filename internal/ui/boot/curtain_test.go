package boot

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestBootCurtainSplitAndExit(t *testing.T) {
	r := ui.Rect{X: 0, Y: 48, W: 1000, H: 800}
	sky, sea := curtainRects(r, 0)
	if sky.H != 496 || sea.H != 304 || sea.Y != 544 {
		t.Fatalf("not 62/38 split: %v %v", sky, sea)
	}
	sky, sea = curtainRects(r, 1)
	if sky.Y+sky.H != r.Y || sea.Y != r.Y+r.H {
		t.Fatalf("ready curtains cover content: %v %v", sky, sea)
	}
}
