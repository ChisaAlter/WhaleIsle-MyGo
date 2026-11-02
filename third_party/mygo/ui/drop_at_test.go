package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/transfer"
)

func TestDropAtKeepsReleasePositionAndLegacyDrop(t *testing.T) {
	var drops int
	var gotX, gotY float32
	var legacy string
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Padding(10).Gap(20).Children(func() {
			Box(c).Size(100, 32).Label("Source").Drag("value")
			dst := Box(c).Size(180, 80).Label("Target")
			if value, x, y, ok := DropAt[string](dst); ok {
				if value != "value" {
					t.Errorf("value = %q", value)
				}
				drops++
				gotX, gotY = x, y
			}
			if value, ok := Drop[string](dst); ok {
				legacy = value
			}
		})
	}, 260, 240)
	src, _ := tt.Find("Source")
	dst, _ := tt.Find("Target")
	x, y := dst.X+23, dst.Y+dst.H-3
	tt.Press(src.X+5, src.Y+5)
	tt.Move(x, y)
	// Both events arrive before the next build. The row's lower half at
	// release must win even though the live pointer now sits in its upper half.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: float64(x), Y: float64(y)})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(dst.X + 4), Y: float64(dst.Y + 2)})
	tt.settle()
	if drops != 1 || gotX != x || gotY != y || legacy != "value" {
		t.Fatalf("drops=%d at=%v,%v want=%v,%v legacy=%q", drops, gotX, gotY, x, y, legacy)
	}
	tt.Frame()
	if drops != 1 {
		t.Fatal("drop delivered after its frame")
	}
}

func TestDropAtClickEscapeAndDisabledTarget(t *testing.T) {
	for _, mode := range []string{"click", "escape", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			clicks, drops := 0, 0
			tt := NewTester(func(c *Context) {
				Column(c).Fill().Gap(4).Children(func() {
					s := Button(c, "Source").Size(100, 32).Drag(7)
					if s.Clicked() {
						clicks++
					}
					d := Box(c).Size(160, 60).Label("Target").Disabled(mode == "disabled")
					if _, _, _, ok := DropAt[int](d); ok {
						drops++
					}
				})
			}, 240, 180)
			s, _ := tt.Find("Source")
			d, _ := tt.Find("Target")
			tt.Press(s.X+6, s.Y+8)
			if mode == "click" {
				tt.Move(s.X+8, s.Y+9)
				tt.Release(s.X+8, s.Y+9)
			} else {
				tt.Move(d.X+20, d.Y+d.H-3)
				if mode == "escape" {
					tt.Key(0, KeyEscape)
				}
				tt.Release(d.X+20, d.Y+d.H-3)
			}
			if drops != 0 || clicks != map[bool]int{true: 1, false: 0}[mode == "click"] {
				t.Fatalf("clicks=%d drops=%d", clicks, drops)
			}
		})
	}
}

func TestDropAtNativeLocalReleasePosition(t *testing.T) {
	var x, y float32
	var got string
	tt := NewTester(func(c *Context) {
		dst := Box(c).Size(160, 100)
		if v, px, py, ok := DropAt[string](dst); ok {
			got, x, y = v, px, py
		}
	}, 200, 200)
	d := &platform.DataDragEvent{Offer: transfer.Offer{Formats: []transfer.Format{transfer.Text}, Operations: transfer.Copy}, Local: "local"}
	if !tt.send2(platform.SurfaceEvent{Kind: platform.DataDragOver, X: 17, Y: 88, Drag: d}) {
		t.Fatal("typed local hover refused")
	}
	tt.rt.event(platform.SurfaceEvent{Kind: platform.DataDrop, X: 17, Y: 88, Drag: d})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 2, Y: 3})
	tt.settle()
	if got != "local" || x != 17 || y != 88 {
		t.Fatalf("native local drop=%q at %v,%v", got, x, y)
	}
}
