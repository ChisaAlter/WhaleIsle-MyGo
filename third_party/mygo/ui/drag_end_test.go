package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestDragEndOutsideInsideEscapeAndFocusLoss(t *testing.T) {
	for _, mode := range []string{"outside-list", "inside-target", "escape", "blur", "outside-window", "click"} {
		t.Run(mode, func(t *testing.T) {
			var ends []DragEndResult[string]
			drops := 0
			value := "captured-owner"
			tt := NewTester(func(c *Context) {
				Column(c).Fill().Children(func() {
					source := Button(c, "Source").Width(100).Height(32).Drag(value)
					if end, ok := DragEnd[string](source); ok {
						ends = append(ends, end)
					}
					if _, duplicate := DragEnd[string](source); duplicate {
						t.Fatal("completion delivered twice")
					}
					target := Box(c).Size(120, 40).Label("Target")
					if _, _, _, ok := DropAt[string](target); ok {
						drops++
					}
				})
			}, 300, 200)
			s, _ := tt.Find("Source")
			d, _ := tt.Find("Target")
			tt.Press(s.X+5, s.Y+5)
			if mode == "click" {
				tt.Release(s.X+5, s.Y+5)
				if len(ends) != 0 || drops != 0 {
					t.Fatal("click synthesized drag completion")
				}
				return
			}
			tt.Move(d.X+8, d.Y+8)
			value = "replacement-owner"
			tt.Frame()
			x, y := float32(240), float32(160)
			if mode == "inside-target" {
				x, y = d.X+8, d.Y+8
			}
			if mode == "escape" {
				tt.Key(0, KeyEscape)
			}
			if mode == "blur" {
				tt.SetFocused(false)
			}
			if mode == "outside-window" {
				x = 310
			}
			tt.Release(x, y)
			tt.Frame()
			if len(ends) != 1 || ends[0].Value != "captured-owner" {
				t.Fatalf("completion=%+v", ends)
			}
			canceled := mode == "escape" || mode == "blur" || mode == "outside-window"
			inside := mode == "inside-target"
			if ends[0].Canceled != canceled || ends[0].Dropped != inside || drops != map[bool]int{true: 1, false: 0}[inside] {
				t.Fatalf("completion=%+v drops=%d", ends[0], drops)
			}
			if !canceled && (ends[0].X != x || ends[0].Y != y) {
				t.Fatal("completion lost release coordinates")
			}
		})
	}
}

func TestDragEndKeepsReleasePositionAcrossPointerMove(t *testing.T) {
	var got DragEndResult[int]
	count := 0
	tt := NewTester(func(c *Context) {
		e := Box(c).Size(100, 32).Label("Source").Drag(7)
		if end, ok := DragEnd[int](e); ok {
			got, count = end, count+1
		}
	}, 300, 200)
	tt.Press(10, 10)
	tt.Move(150, 80)
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: 220, Y: 150})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 5, Y: 5})
	tt.settle()
	if count != 1 || got.Value != 7 || got.X != 220 || got.Y != 150 || got.Canceled || got.Dropped {
		t.Fatalf("got=%+v count=%d", got, count)
	}
}

func TestDragEndRemovedOwnerDoesNotCompleteOnRemount(t *testing.T) {
	show, count := true, 0
	tt := NewTester(func(c *Context) {
		if show {
			e := Box(c.Key("source")).Size(100, 32).Label("Source").Drag(7)
			if _, ok := DragEnd[int](e); ok {
				count++
			}
		}
	}, 300, 200)
	tt.Press(10, 10)
	tt.Move(150, 80)
	show = false
	tt.Frame()
	tt.Release(220, 150)
	show = true
	tt.Frame()
	if count != 0 || tt.rt.drag != nil {
		t.Fatalf("removed owner delivered count=%d drag=%+v", count, tt.rt.drag)
	}
}
