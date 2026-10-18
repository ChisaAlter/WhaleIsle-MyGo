package webview

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestDockedViewportKeepsRightSeat(t *testing.T) {
	parent := mygo.Rectangle{X: 100, Y: 50, Width: 1200, Height: 800}
	seat := mygo.Rectangle{X: 800, Y: 180, Width: 400, Height: 620}
	got := viewportBounds(parent, 144, &seat)
	if got != (mygo.Rectangle{X: 900, Y: 230, Width: 400, Height: 620}) {
		t.Fatal(got)
	}
	parent.X = 300
	parent.Width = 1000
	got = viewportBounds(parent, 144, &seat)
	if got.X != 1100 || got.Width != 200 || got.Y != 230 {
		t.Fatal("move/resize escaped seat", got)
	}
	full := viewportBounds(parent, 144, nil)
	if full.X != 300 || full.Width != 1000 || full.Y != 194 {
		t.Fatal("full page changed", full)
	}
	seat.Width = 0
	seat.Height = 0
	got = viewportBounds(parent, 144, &seat)
	if got.Width != 1 || got.Height != 1 {
		t.Fatal("unbounded empty viewport", got)
	}
}
