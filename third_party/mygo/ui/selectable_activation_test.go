package ui

import (
	"testing"
	"time"
)

// All gestures below go to Tester's in-memory engine. No OS input, window,
// clipboard or platform focus API is used; activation is an integer counter.
func TestSelectableActivationBodyPaddingAndNestedControl(t *testing.T) {
	parent, child := 0, 0
	tt := coreNewTester(func(c *context) {
		card := coreButtonBase(c).Label("card").Selectable().Column().Width(260).Padding(16).Gap(8)
		card.Children(func() {
			coreText(c, "Alpha body")
			if coreButton(c, "Nested action").Clicked() {
				child++
			}
			coreText(c, "Omega body")
		})
		if card.Clicked() {
			parent++
		}
	}, 340, 240)
	x, y := selectableAt(t, tt, "Alpha body", 3)
	tt.ClickAt(x, y)
	if parent != 1 || child != 0 {
		t.Fatalf("body click parent=%d child=%d, want 1/0", parent, child)
	}
	r, _ := tt.Find("card")
	tt.rt.lastPress.at = time.Time{}
	tt.ClickAt(r.X+3, r.Y+3)
	if parent != 2 {
		t.Fatalf("padding click activated parent %d times, want total 2", parent)
	}
	if err := tt.Click("Nested action"); err != nil {
		t.Fatal(err)
	}
	if parent != 2 || child != 1 {
		t.Fatalf("nested control bubbled: parent=%d child=%d", parent, child)
	}
}

func TestSelectableActivationDragKeepsRangeWithoutParentClick(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		calls := 0
		tt := coreNewTester(func(c *context) {
			card := coreButtonBase(c).Label("card").Selectable().Column().Width(260).Padding(16).Gap(8)
			card.Children(func() { coreText(c, "Alpha body"); coreText(c, "Omega body") })
			if card.Clicked() {
				calls++
			}
		}, 340, 220)
		first, last, from, to := "Alpha body", "Omega body", 1, 5
		if reverse {
			first, last, from, to = last, first, to, from
		}
		selectBetween(t, tt, first, from, last, to)
		a, b, selected := tt.rt.textSelectionBounds()
		if !selected || a.id == b.id || calls != 0 {
			t.Fatalf("reverse=%v selected=%v calls=%d endpoints=%+v/%+v", reverse, selected, calls, a, b)
		}
	}
}

func TestSelectableActivationKeyboardAndScopedShortcut(t *testing.T) {
	calls := 0
	tt := coreNewTester(func(c *context) {
		card := coreButtonBase(c).Label("card").Selectable().Column().Width(260).Padding(16)
		card.Children(func() { coreText(c, "Alpha body") })
		activate := card.Clicked()
		enter := card.Shortcut(0, KeyEnter)
		space := card.Shortcut(0, KeySpace)
		if activate || enter || space {
			calls++
		}
	}, 340, 180)
	tt.Key(0, KeyTab)
	tt.Key(0, KeyEnter)
	tt.Key(0, KeySpace)
	if calls != 2 {
		t.Fatalf("Tab-focused keyboard activated %d times, want 2", calls)
	}
	selectBetween(t, tt, "Alpha body", 1, "Alpha body", 6)
	if _, _, selected := tt.rt.textSelectionBounds(); !selected {
		t.Fatal("fixture has no selected text")
	}
	tt.Key(0, KeyEnter)
	tt.Key(0, KeySpace)
	if calls != 4 {
		t.Fatalf("paragraph-focused shortcuts with selected text activated %d times, want 4", calls)
	}
}
