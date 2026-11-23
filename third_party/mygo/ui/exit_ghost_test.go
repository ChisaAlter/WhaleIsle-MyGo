package ui

import (
	"fmt"
	"testing"
	"time"
)

// A row leaving with an Exit shows as it did, from a copy of its own,
// whatever the frames after build in the arena; the frame's arena is not
// kept for it, and the copy of a row that stays reuses its storage.
func TestExitGhostIsolated(t *testing.T) {
	items := []string{"One", "Two"}
	extra := 0
	motion := ElementTransition{Ease: Linear, Exit: &Motion{Opacity: 0}}
	tt, now := clockTester(func(c *context) {
		coreColumn(c).Children(func() {
			for _, s := range items {
				coreRow(c).Key(s).Height(30).Transition(motion).Children(func() { coreText(c, s) })
			}
			for i := range extra {
				coreTextf(c, "Extra %d", i)
			}
		})
	}, 300, 2000)
	tt.Frame()
	chunks := len(tt.rt.c.chunks)
	if allocs := testing.AllocsPerRun(20, tt.Frame); allocs > 2 {
		t.Errorf("%v allocations a frame copying the rows that may leave", allocs)
	}
	if n := len(tt.rt.c.chunks); n != chunks {
		t.Errorf("%d chunks in the arena, %d before", n, chunks)
	}

	items = []string{"Two"}
	extra = 200 // new elements take the arena where the row was
	tt.Frame()
	var r *transition
	for _, tr := range tt.rt.trans {
		if tr.ghost != nil {
			r = tr
		}
	}
	if r == nil || r.ghost.first == nil || r.ghost.first.text != "One" {
		t.Fatalf("no copy of the row gone: %+v", r)
	}
	for _, ch := range tt.rt.c.chunks {
		for i := range ch {
			if &ch[i] == r.ghost || &ch[i] == r.ghost.first {
				t.Fatal("the copy is in the arena")
			}
		}
	}
	for i := range 3 {
		*now = now.Add(30 * time.Millisecond)
		extra = 200 + i*50
		tt.Frame()
		if r.ghost == nil || r.ghost.first.text != "One" || r.ghost.first.st == nil || r.ghost.first.st.node != nil {
			t.Fatalf("frame %d: the copy changed: %+v", i, r.ghost)
		}
		if g := r.ghost; g.parent == nil || g.parent.st.node != g.parent {
			t.Fatalf("frame %d: the copy is not under its parent of the frame", i)
		}
	}
	*now = now.Add(time.Second)
	tt.Frame()
	for id, tr := range tt.rt.trans {
		if tr.ghost != nil {
			t.Fatalf("the copy of %x stayed", id)
		}
	}
	if _, ok := tt.Find(fmt.Sprint("Extra ", 299)); !ok {
		t.Fatal("the frame lost its elements")
	}
}
