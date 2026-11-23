package ui

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// scanKept is keptAlive as it was, walking the pages of every state.
func scanKept(rt *engine, s *state) bool {
	for n := 0; s != nil && n < 64; n++ {
		if rt.kept[s.id] {
			return true
		}
		if s.page == 0 {
			return false
		}
		s = rt.states[s.page]
	}
	return false
}

// checkPruned verifies that the states prune left are those a scan of
// every state would have left, and that the live list is what the frame
// built.
func checkPruned(t *testing.T, rt *engine, step int) {
	t.Helper()
	built := 0
	for id, s := range rt.states {
		if s.id != id {
			t.Fatalf("step %d: state of %x has the ID %x", step, id, s.id)
		}
		if s.seen == rt.frame && s.pass == rt.pass {
			built++
			continue
		}
		if !scanKept(rt, s) {
			t.Fatalf("step %d: state %x outlived its element", step, id)
		}
	}
	live := map[*state]bool{}
	for _, s := range rt.live {
		if s.seen != rt.frame || s.pass != rt.pass {
			t.Fatalf("step %d: live state %x was not built", step, s.id)
		}
		if rt.states[s.id] != s {
			t.Fatalf("step %d: live state %x was forgotten", step, s.id)
		}
		live[s] = true
	}
	if len(live) != built {
		t.Fatalf("step %d: %d live states, %d built", step, len(live), built)
	}
}

func TestPruneTouchedStates(t *testing.T) {
	r := NewRouter("/p/0")
	r.Transition = TransitionNone
	rows := map[string]int{}
	rng := rand.New(rand.NewPCG(1, 2))
	clicked := 0
	tt := coreNewTester(func(c *context) {
		r.coreView(c, func(rt *Route) {
			path := rt.Path()
			coreScroll(c).Grow(1).Children(func() {
				for i := range rows[path] {
					coreRow(c).Key(i).Children(func() {
						coreTextf(c, "%s row %d", path, i)
						if coreButton(c, fmt.Sprintf("%s button %d", path, i)).Clicked() {
							clicked++
						}
					})
				}
			})
		})
	}, 400, 300)
	pages := 0
	for step := range 400 {
		switch rng.IntN(6) {
		case 0:
			pages++
			r.Push(fmt.Sprintf("/p/%d", pages))
		case 1:
			r.Back()
		case 2:
			r.Forward()
		case 3:
			r.Replace(fmt.Sprintf("/p/%d", rng.IntN(pages+1)))
		case 4:
			if n := rows[r.Path()]; n > 0 {
				tt.Click(fmt.Sprintf("%s button %d", r.Path(), rng.IntN(n)))
			}
		}
		rows[r.Path()] = rng.IntN(12)
		tt.Frame()
		checkPruned(t, tt.rt, step)
	}
	if clicked == 0 {
		t.Fatal("no clicks")
	}
}

// A page kept out of sight keeps its states without prune looking at them
// in every frame, and gives them up once the Router no longer keeps it.
func TestPruneDormantPages(t *testing.T) {
	r := NewRouter("/list")
	r.Transition = TransitionNone
	tt := coreNewTester(func(c *context) {
		r.coreView(c, func(rt *Route) {
			if rt.Match("/list") {
				for i := range 30 {
					coreTextf(c, "Row %d", i)
				}
				return
			}
			coreText(c, "Page "+rt.Path())
		})
	}, 400, 300)
	rows := map[uint64]bool{}
	for id := range tt.rt.states {
		rows[id] = true
	}
	r.Push("/other")
	tt.Frame()
	tt.Frame()
	rt := tt.rt
	if len(rt.dormant) < 30 {
		t.Fatalf("%d dormant states", len(rt.dormant))
	}
	for range 10 {
		tt.Frame()
		checkPruned(t, rt, 0)
	}
	if len(rt.live) > 20 {
		t.Fatalf("%d live states, which include the page kept", len(rt.live))
	}
	for i := range keptPages + 1 {
		r.Push(fmt.Sprintf("/p/%d", i))
		tt.Frame()
		checkPruned(t, rt, i)
	}
	for id := range rows {
		if s := rt.states[id]; s != nil && s.seen != rt.frame {
			t.Fatalf("state %x of the page outlived it", id)
		}
	}
}
