package ui

import "testing"

// Exercise cursor resolution independently of text/raster allocations. hitChain
// starts at the last containing hit and walks its parents; the no-slice cursor
// path must retain exactly that ownership for overlapping/disabled controls.
func TestCursorHitWalkPriorityAndAllocs(t *testing.T) {
	for _, tc := range []struct {
		name        string
		underFlags  uint32
		leafFlags   uint32
		parentFlags uint32
		topOutside  bool
		pointerIn   bool
		hoverCursor Cursor
		pressCursor Cursor
		want        Cursor
	}{
		{name: "enabled top hides disabled underneath", underFlags: flagDisabled | flagClickable, pointerIn: true, want: CursorDefault},
		{name: "disabled top", leafFlags: flagDisabled | flagClickable, pointerIn: true, want: CursorNotAllowed},
		{name: "disabled ancestor through nested children", parentFlags: flagDisabled | flagClickable, pointerIn: true, want: CursorNotAllowed},
		{name: "disabled nonclickable ancestor", parentFlags: flagDisabled, pointerIn: true, want: CursorDefault},
		{name: "outside top does not obscure disabled hit", underFlags: flagDisabled | flagClickable, topOutside: true, pointerIn: true, want: CursorNotAllowed},
		{name: "pointer outside", leafFlags: flagDisabled | flagClickable, want: CursorDefault},
		{name: "explicit hover cursor wins", leafFlags: flagDisabled | flagClickable, pointerIn: true, hoverCursor: CursorPointer, want: CursorPointer},
		{name: "pressed cursor wins", leafFlags: flagDisabled | flagClickable, pointerIn: true, hoverCursor: CursorPointer, pressCursor: CursorCrosshair, want: CursorCrosshair},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := &state{id: 1}
			under := &state{id: 2, parent: 1, flags: tc.underFlags}
			parent := &state{id: 3, parent: 1, flags: tc.parentFlags}
			middle := &state{id: 4, parent: 3}
			leaf := &state{id: 5, parent: 4, flags: tc.leafFlags}
			rt := &engine{host: &headless{}, states: map[uint64]*state{1: root, 2: under, 3: parent, 4: middle, 5: leaf}, pointerX: 10, pointerY: 10, pointerIn: tc.pointerIn}
			rt.hits = []hit{{st: under, r: Rect{0, 0, 40, 40}}, {st: leaf, r: Rect{0, 0, 40, 40}}}
			if tc.topOutside {
				rt.hits[1].r = Rect{50, 50, 20, 20}
			}
			if tc.hoverCursor != CursorDefault {
				leaf.cursor = tc.hoverCursor + 1
				rt.hover = []uint64{leaf.id}
			}
			if tc.pressCursor != CursorDefault {
				rt.pressed = &state{cursor: tc.pressCursor + 1}
			}
			rt.updateCursor()
			if rt.cursor != tc.want {
				t.Fatalf("cursor=%v, want %v", rt.cursor, tc.want)
			}
			if got := testing.AllocsPerRun(100, rt.updateCursor); got != 0 {
				t.Fatalf("cursor-only update allocated %v times", got)
			}
		})
	}
}
