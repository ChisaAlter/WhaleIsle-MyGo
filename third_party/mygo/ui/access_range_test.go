package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// A slider, a stepper, a rating, a meter and a progress bar are the roles the
// surface gives the range-value pattern to (surface_access.go:281, through
// AccessRole.Ranged). That pattern is a promise twice over: a client reads the
// range and the value from it, and it moves the value by incrementing, which
// setRangeValue sends as one arrow-key step (surface_access.go:1012-1022).
//
// So a range element must report a range a client can interpret, and one that
// offers the increment must move when it is incremented. A meter and a progress
// bar offer no increment on purpose — that is what makes them read-only.
//
// A spinner is absent: it is a progress bar of unknown length, and it reports
// the toolkit's own marker as its value. See the note on it in
// reports/HOME-79-access-selection-range.md.
func TestEveryRangeControlReportsARangeAndMovesWhenIncremented(t *testing.T) {
	var (
		slider  float64
		stepper float64
		rating  int
	)
	cases := []struct {
		name  string
		reset func()
		build func(c *context)
	}{
		{"Slider", func() { slider = 5 }, func(c *context) {
			coreSliderBase(c, &slider, 0, 10).Label("Volume").Width(200)
		}},
		{"Stepper", func() { stepper = 5 }, func(c *context) {
			coreStepper(c, &stepper, 0, 10, 1).Label("Count")
		}},
		{"Rating", func() { rating = 2 }, func(c *context) {
			coreRating(c, &rating, 5).Label("Stars")
		}},
		{"Meter", func() {}, func(c *context) {
			coreMeter(c, 40, 0, 100, nil).Label("Disk")
		}},
		{"Progress", func() {}, func(c *context) {
			coreProgress(c, 0.4).Label("Upload")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.reset()
			tt := coreNewTester(tc.build, 420, 420)
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			element := rangeElement(t, tt, tc.name)
			if element.Max <= element.Min {
				t.Fatalf("%s reports the range %v..%v", tc.name, element.Min, element.Max)
			}
			if element.Now < element.Min || element.Now > element.Max {
				t.Fatalf("%s reports %v outside %v..%v", tc.name, element.Now, element.Min, element.Max)
			}

			before := element.Now
			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: element.ID, Action: platform.AccessIncrement})
			tt.Frame()
			after := rangeElement(t, tt, tc.name).Now
			incrementable := element.Actions&platform.ActionIncrement != 0
			switch {
			case incrementable && !(after > before):
				t.Fatalf("%s offers the increment and stays at %v", tc.name, after)
			case !incrementable && after != before:
				t.Fatalf("%s refuses the increment but moved from %v to %v", tc.name, before, after)
			}
		})
	}
}

// rangeElement is the one element of the tree whose role reports a range.
func rangeElement(t *testing.T, tt *Tester, name string) platform.AccessNode {
	t.Helper()
	var found []platform.AccessNode
	for _, n := range tt.h.access.Nodes {
		if n.Role.Ranged() {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: %d range elements", name, len(found))
	}
	return found[0]
}
