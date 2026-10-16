package ui

import "testing"

func TestNativeSurfaceClipRespectsModalAndScrollClip(t *testing.T) {
	p := &Painter{clip: Rect{X: 10, Y: 20, W: 300, H: 200}}
	if p.NativeSurfaceClip() != p.clip {
		t.Fatal("lost ancestor scroll clip")
	}
	p.surfaceBlocked = true
	if p.NativeSurfaceClip() != (Rect{}) {
		t.Fatal("native child escaped active modal")
	}
	if p.ClipBounds() != p.clip {
		t.Fatal("ordinary drawing clip changed")
	}
}
