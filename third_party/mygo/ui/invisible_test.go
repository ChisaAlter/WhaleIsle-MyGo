package ui

import (
	"testing"
	"time"
)

// An invisible element keeps the box its content gives it, but its content
// is not laid out, takes no input and has no focus; shown again, it is laid
// out where it belongs.
func TestInvisibleSkipsContent(t *testing.T) {
	hidden := false
	var box, inner *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Children(func() {
			coreText(c, "Above")
			box = coreColumn(c).Padding(5)
			if hidden {
				box.Invisible()
			}
			box.Children(func() {
				inner = coreBox(c).Size(80, 30)
				if coreButton(c, "Inside").Clicked() {
					clicks++
				}
			})
			coreText(c, "Below")
		})
	}, 300, 300)
	shownBox := Rect{box.x, box.y, box.w, box.h}
	shownInner := Rect{inner.x, inner.y, inner.w, inner.h}
	below, _ := tt.Find("Below")
	if shownInner.W != 80 || shownInner.X != shownBox.X+5 {
		t.Fatalf("inner at %v in %v", shownInner, shownBox)
	}
	tt.Click("Inside")
	tt.Key(0, KeyTab)
	if clicks != 1 || !tt.Focused("Inside") {
		t.Fatalf("clicks %d, focus %v", clicks, tt.Focused("Inside"))
	}

	hidden = true
	tt.Frame()
	if got := (Rect{box.x, box.y, box.w, box.h}); got != shownBox {
		t.Fatalf("invisible box %v, shown %v", got, shownBox)
	}
	if b, _ := tt.Find("Below"); b != below {
		t.Fatalf("below at %v, was %v", b, below)
	}
	if inner.w != 0 || inner.x != 0 {
		t.Fatalf("the content of an invisible box was laid out: %v", Rect{inner.x, inner.y, inner.w, inner.h})
	}
	if tt.Focused("Inside") || tt.rt.focused != 0 {
		t.Fatal("an invisible button kept the focus")
	}
	if _, ok := tt.Find("Inside"); ok {
		t.Fatal("found an invisible button")
	}
	for _, h := range tt.rt.hits {
		if h.st == inner.st {
			t.Fatal("an invisible element takes the pointer")
		}
	}

	hidden = false
	tt.Frame()
	if got := (Rect{inner.x, inner.y, inner.w, inner.h}); got != shownInner {
		t.Fatalf("shown again, inner at %v, was %v", got, shownInner)
	}
	tt.Click("Inside")
	if clicks != 2 {
		t.Fatalf("clicks %d after showing again", clicks)
	}
}

// An element with a Transition inside an invisible one does not move from
// where it was not laid out: shown again, it is where it belongs at once.
func TestInvisibleTransitionShowsAtOnce(t *testing.T) {
	hidden, wide := false, false
	var moved *node
	now := time.Unix(1000, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Children(func() {
			b := coreColumn(c)
			if hidden {
				b.Invisible()
			}
			b.Children(func() {
				if wide {
					coreBox(c).Size(100, 10)
				}
				moved = coreBox(c).Key("moved").Size(40, 40).Transition(ElementTransition{Position: true})
			})
		})
	}, 300, 300)
	tt.SetTime(now)
	y0 := moved.y
	hidden = true
	now = now.Add(time.Second)
	tt.SetTime(now)
	hidden, wide = false, true
	now = now.Add(time.Second)
	tt.SetTime(now)
	if moved.y != y0+10 {
		t.Fatalf("shown at %v, want %v at once", moved.y, y0+10)
	}
	if _, ok := tt.rt.trans[moved.id]; !ok {
		t.Fatal("the transition was forgotten while hidden")
	}
}

// What is inert shows and keeps its layout, but takes no input.
func TestInertCommitsWithoutInput(t *testing.T) {
	var b *node
	clicks := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Inert(true).Children(func() {
			b = coreButton(c, "Inert")
			if b.Clicked() {
				clicks++
			}
		})
	}, 300, 300)
	if b.w == 0 || b.st.w != b.w || b.st.x != b.x {
		t.Fatalf("inert button laid out %v, state %v", b.w, b.st.w)
	}
	tt.Press(b.x+2, b.y+2)
	tt.Release(b.x+2, b.y+2)
	if clicks != 0 {
		t.Fatal("an inert button was clicked")
	}
	if len(tt.rt.focusOrder) != 0 {
		t.Fatal("an inert button takes the focus")
	}
}
