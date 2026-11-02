package pluginui

import "testing"

func TestMirrorGeometryRetainsFullTallNegativeRowAndBoundsVisibleFragment(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "A", Epoch: "1"}
	g := mirrorGeometry{DocumentKey: "doc", Revision: 1, Subject: &subject}
	g.Viewport.W, g.Viewport.H, g.Viewport.Scale = 1000, 900, 1
	g.Owners = []MirrorOccurrence{{Handle: "tail", Entry: "original", Turn: 1, Seq: 2, Rect: MirrorRect{X: 100, Y: -1600, W: 700, H: 3200}, VisibleRect: MirrorRect{X: 100, Y: 80, W: 700, H: 720}, Connected: true, Visible: true}}
	if !validMirrorGeometry(g, subject) {
		t.Fatal("negative full-row coordinates rejected a valid clipped original")
	}
	g.Owners[0].VisibleRect.Y = -1
	if validMirrorGeometry(g, subject) {
		t.Fatal("negative visible fragment escaped viewport")
	}
}
