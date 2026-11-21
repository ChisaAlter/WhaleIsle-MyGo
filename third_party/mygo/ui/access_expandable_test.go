package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// Every control that reports itself expandable is offered the expand pattern
// and, for its role, no invoke one: a disclosure, a combo box and a pop-up
// button are all opened by that action alone. So a control that advertises the
// state must answer the action, or it cannot be opened without a pointer.
//
// This table is the guard. A widget that sets expandable without reading the
// action fails here, which is how the collapsible, the combo box and the date
// field were each found ignoring it (the select was fixed first).
func TestEveryExpandableControlAnswersTheExpandPattern(t *testing.T) {
	var (
		collapsibleOpen bool
		comboText       string
		date            time.Time
		size            string
	)
	day := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		reset func()
		build func(c *context)
	}{
		{"Collapsible", func() { collapsibleOpen = false }, func(c *context) {
			p := coreCollapsibleBase(c, &collapsibleOpen)
			p.Trigger.Children(func() { coreText(c, "Details") })
			p.Panel(func() { coreText(c, "panel") })
		}},
		{"Select", func() { size = "Medium" }, func(c *context) {
			coreSelect(c, &size, []string{"Small", "Medium", "Large"})
		}},
		{"Combobox", func() { comboText = "" }, func(c *context) {
			p := coreComboboxBase(c, &comboText)
			p.Input.Label("Font").Width(180)
			p.Popup(func(panel *node) {
				panel.Children(func() {
					for _, v := range []string{"Alpha", "Beta"} {
						p.Item(v).Children(func() { coreText(c, v) })
					}
				})
			})
		}},
		{"DateInput", func() { date = day }, func(c *context) {
			coreDateInput(c, &date)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.reset()
			tt := coreNewTester(tc.build, 420, 420)
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			trigger := expandableNode(t, tt)
			if trigger.States&platform.AccessExpanded != 0 {
				t.Fatalf("%s starts expanded", tc.name)
			}

			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: trigger.ID, Action: platform.AccessExpand})
			tt.Frame()
			if n := nodeByID(t, tt, trigger.ID); n.States&platform.AccessExpanded == 0 {
				t.Fatalf("%s did not open from the expand pattern", tc.name)
			}

			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: trigger.ID, Action: platform.AccessCollapse})
			tt.Frame()
			if n := nodeByID(t, tt, trigger.ID); n.States&platform.AccessExpanded != 0 {
				t.Fatalf("%s did not close from the collapse pattern", tc.name)
			}
		})
	}
}

// expandableNode is the one element of the tree that reports itself expandable.
func expandableNode(t *testing.T, tt *Tester) platform.AccessNode {
	t.Helper()
	var found []platform.AccessNode
	for _, n := range tt.h.access.Nodes {
		if n.States&platform.AccessExpandable != 0 {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d expandable elements in %+v", len(found), tt.h.access.Nodes)
	}
	return found[0]
}

// nodeByID is the element of the tree with id, which must still be there.
func nodeByID(t *testing.T, tt *Tester, id uint64) platform.AccessNode {
	t.Helper()
	for _, n := range tt.h.access.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("element %d left the tree", id)
	return platform.AccessNode{}
}

// An assistive-technology toggle is input like a click: Changed must deliver
// it. A consumer that applies its open state through a Changed hook — the
// disclosure row's OnToggle — otherwise never learns the row opened: the
// bound local flips, nothing reports it, and the next frame rebuilds the row
// closed from the stale model. That is what kept UIA Expand() a no-op on the
// chat's disclosure members.
func TestCollapsibleExpandPatternReportsChanged(t *testing.T) {
	var open, heard bool
	tt := coreNewTester(func(c *context) {
		p := coreCollapsibleBase(c, &open)
		if p.Trigger.Changed() {
			heard = true
		}
		p.Trigger.Children(func() { coreText(c, "Details") })
		p.Panel(func() { coreText(c, "panel") })
	}, 420, 420)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	trigger := expandableNode(t, tt)

	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: trigger.ID, Action: platform.AccessExpand})
	tt.Frame()
	if !open {
		t.Fatal("collapsible did not open from the expand pattern")
	}
	if !heard {
		t.Fatal("collapsible's expand-pattern toggle was not reported through Changed")
	}
}
