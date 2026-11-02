package frame

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestDisplayWorkAreaLandscape(t *testing.T) {
	landscape := mygo.Display{
		Primary:  true,
		Bounds:   mygo.Rectangle{Width: 1920, Height: 1080},
		WorkArea: mygo.Rectangle{Y: 24, Width: 1920, Height: 1008},
	}
	portrait := mygo.Display{
		Bounds:   mygo.Rectangle{X: -1080, Y: -400, Width: 1080, Height: 1920},
		WorkArea: mygo.Rectangle{X: -1080, Y: -376, Width: 1080, Height: 1848},
	}
	for _, tc := range []struct {
		name      string
		window    mygo.Rectangle
		displays  []mygo.Display
		landscape bool
		known     bool
	}{
		{name: "no displays", window: mygo.Rectangle{Width: 600, Height: 900}},
		{name: "invalid window", displays: []mygo.Display{landscape}},
		{name: "portrait viewport on landscape display", window: mygo.Rectangle{X: 40, Y: 40, Width: 600, Height: 900}, displays: []mygo.Display{landscape}, landscape: true, known: true},
		{name: "secondary portrait not primary", window: mygo.Rectangle{X: -1000, Y: -200, Width: 800, Height: 600}, displays: []mygo.Display{landscape, portrait}, known: true},
		{name: "crossed to landscape", window: mygo.Rectangle{X: -100, Y: 100, Width: 600, Height: 600}, displays: []mygo.Display{landscape, portrait}, landscape: true, known: true},
		{name: "crossed to portrait", window: mygo.Rectangle{X: -500, Y: 100, Width: 600, Height: 600}, displays: []mygo.Display{landscape, portrait}, known: true},
		{name: "equal overlap is unknown", window: mygo.Rectangle{X: -300, Y: 100, Width: 600, Height: 600}, displays: []mygo.Display{landscape, portrait}},
		{name: "offscreen has no primary fallback", window: mygo.Rectangle{X: 3000, Y: 3000, Width: 600, Height: 900}, displays: []mygo.Display{landscape, portrait}},
		{name: "edge contact is not overlap", window: mygo.Rectangle{X: 1920, Width: 600, Height: 900}, displays: []mygo.Display{landscape}},
		{name: "work area determines direction", window: mygo.Rectangle{X: 100, Y: 100, Width: 600, Height: 600}, displays: []mygo.Display{{Bounds: landscape.Bounds, WorkArea: mygo.Rectangle{Width: 900, Height: 1000}}}, known: true},
		{name: "square work area is unknown", window: mygo.Rectangle{Width: 600, Height: 900}, displays: []mygo.Display{{Bounds: landscape.Bounds, WorkArea: mygo.Rectangle{Width: 1000, Height: 1000}}}},
		{name: "missing work area is unknown", window: mygo.Rectangle{Width: 600, Height: 900}, displays: []mygo.Display{{Bounds: landscape.Bounds}}},
		{name: "invalid display bounds", window: mygo.Rectangle{Width: 600, Height: 900}, displays: []mygo.Display{{WorkArea: landscape.WorkArea}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotLandscape, gotKnown := displayWorkAreaLandscape(tc.window, tc.displays)
			if gotLandscape != tc.landscape || gotKnown != tc.known {
				t.Fatalf("got landscape=%v known=%v, want %v %v", gotLandscape, gotKnown, tc.landscape, tc.known)
			}
		})
	}
}

func TestDisplayWindowSourceLifecycleHeadless(t *testing.T) {
	// Zero windows are ownership tokens only. The injected reader below never
	// calls a native method or starts MyGo's application/event loop.
	var source displayWindowSource
	first, second := new(mygo.Window), new(mygo.Window)
	closedCallbacks := make(map[*mygo.Window][]func())
	bind := func(window *mygo.Window) {
		source.bindOnClosed(window, func(owner *mygo.Window, closed func()) {
			closedCallbacks[owner] = append(closedCallbacks[owner], closed)
		})
	}
	readCalls := 0
	bounds := mygo.Rectangle{Width: 600, Height: 900}
	displays := []mygo.Display{{Bounds: mygo.Rectangle{Width: 1920, Height: 1080}, WorkArea: mygo.Rectangle{Width: 1920, Height: 1040}}}
	read := func(window *mygo.Window) (mygo.Rectangle, []mygo.Display, bool) {
		readCalls++
		if window != first && window != second {
			t.Fatal("source invented an owner")
		}
		return bounds, displays, true
	}
	if _, known := source.landscape(read); known || readCalls != 0 {
		t.Fatal("headless source called the native reader without an explicit owner")
	}
	bind(first)
	defer bind(nil)
	oldBinding, owner := source.current()
	if owner != first {
		t.Fatal("explicit owner was not bound")
	}
	bind(first)
	if repeated, _ := source.current(); repeated != oldBinding || len(closedCallbacks[first]) != 1 {
		t.Fatal("rebinding the same window added another close listener")
	}
	if landscape, known := source.landscape(read); !landscape || !known {
		t.Fatal("bound owner did not read its work area")
	}
	bounds.X = -800
	displays = append(displays, mygo.Display{Bounds: mygo.Rectangle{X: -1080, Width: 1080, Height: 1920}, WorkArea: mygo.Rectangle{X: -1080, Width: 1080, Height: 1880}})
	if landscape, known := source.landscape(read); landscape || !known {
		t.Fatal("moving the bound window did not refresh its display")
	}
	displays[1].WorkArea.Width, displays[1].WorkArea.Height = 1500, 1000
	if landscape, known := source.landscape(read); !landscape || !known {
		t.Fatal("changing the work area left stale orientation")
	}
	if _, known := source.landscape(func(*mygo.Window) (mygo.Rectangle, []mygo.Display, bool) { return bounds, displays, false }); known {
		t.Fatal("unavailable/hidden window produced screen orientation")
	}
	bind(second)
	closedCallbacks[first][0]() // An already-dispatched OnClosed of the old window.
	if _, owner := source.current(); owner != second {
		t.Fatal("closing the previous window cleared its replacement")
	}
	if _, known := source.landscape(func(*mygo.Window) (mygo.Rectangle, []mygo.Display, bool) {
		bind(first)
		return bounds, displays, true
	}); known {
		t.Fatal("snapshot from a replaced owner escaped the ownership fence")
	}
	closedCallbacks[first][1]() // The new binding's OnClosed retires exactly it.
	callsBefore := readCalls
	if _, known := source.landscape(read); known || readCalls != callsBefore {
		t.Fatal("retired source queried the native reader")
	}
	bind(second)
	bind(nil)
	if _, owner := source.current(); owner != nil {
		t.Fatal("explicit unbind left a window owner")
	}
}
