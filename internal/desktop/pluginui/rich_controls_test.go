package pluginui

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestRichSurfacePreservesMultipleSeatsAndClipsEveryGap(t *testing.T) {
	rows := []richPlacement{{handle: "editor", bounds: mygo.Rectangle{X: 10, Y: 20, Width: 100, Height: 100}, clip: mygo.Rectangle{X: 10, Y: 40, Width: 100, Height: 60}}, {handle: "canvas", bounds: mygo.Rectangle{X: 150, Y: 30, Width: 90, Height: 70}, clip: mygo.Rectangle{X: 150, Y: 30, Width: 90, Height: 70}}}
	bounds, holes, seats, ok := richBuildLayout(rows, mygo.Rectangle{})
	if !ok || bounds != (mygo.Rectangle{X: 10, Y: 20, Width: 230, Height: 100}) || len(seats) != 2 || seats[1].X != 140 {
		t.Fatalf("%+v %+v %v", bounds, seats, ok)
	}
	// Every DIP outside both visible rich nodes must be an occlusion. No native
	// button, source toolbar, background or invisible editor edge can leak input.
	for y := bounds.Y; y < bounds.Y+bounds.Height; y++ {
		for x := bounds.X; x < bounds.X+bounds.Width; x++ {
			visible := false
			for _, r := range rows {
				v := richIntersect(r.bounds, r.clip)
				visible = visible || (x >= v.X && x < v.X+v.Width && y >= v.Y && y < v.Y+v.Height)
			}
			hidden := false
			for _, h := range holes {
				hidden = hidden || (x >= h.X && x < h.X+h.Width && y >= h.Y && y < h.Y+h.Height)
			}
			if visible == hidden {
				t.Fatalf("pixel %d,%d visible=%v hidden=%v", x, y, visible, hidden)
			}
		}
	}
	if _, _, _, ok := richBuildLayout([]richPlacement{{bounds: mygo.Rectangle{Width: 8193, Height: 30}}}, mygo.Rectangle{}); ok {
		t.Fatal("oversized surface accepted")
	}
}

// Ambient seats carry authored shell-overlay coordinates, so their layout must
// pin the source viewport to the whole content rect rather than the seats'
// union; seat coordinates stay absolute and gaps remain clipped holes.
func TestRichSurfaceAmbientSeatsUseFullContentBounds(t *testing.T) {
	forced := mygo.Rectangle{Width: 1600, Height: 900}
	rows := []richPlacement{
		{handle: "ball", bounds: mygo.Rectangle{X: 1560, Y: 400, Width: 28, Height: 28}, clip: forced, ambient: true},
		{handle: "dock", bounds: mygo.Rectangle{X: 1200, Y: 200, Width: 380, Height: 620}, clip: forced, ambient: true},
		{handle: "editor", bounds: mygo.Rectangle{X: 10, Y: 20, Width: 100, Height: 100}, clip: mygo.Rectangle{X: 10, Y: 20, Width: 100, Height: 100}},
	}
	bounds, holes, seats, ok := richBuildLayout(rows, forced)
	if !ok || bounds != forced {
		t.Fatalf("%+v %v", bounds, ok)
	}
	if seats[0].X != 1560 || seats[0].Y != 400 || seats[1].X != 1200 || seats[2].X != 10 {
		t.Fatalf("ambient seat coordinates must stay absolute: %+v", seats)
	}
	for y := bounds.Y; y < bounds.Y+bounds.Height; y += 7 {
		for x := bounds.X; x < bounds.X+bounds.Width; x += 7 {
			visible := false
			for _, r := range rows {
				v := richIntersect(r.bounds, r.clip)
				visible = visible || (x >= v.X && x < v.X+v.Width && y >= v.Y && y < v.Y+v.Height)
			}
			hidden := false
			for _, h := range holes {
				hidden = hidden || (x >= h.X && x < h.X+h.Width && y >= h.Y && y < h.Y+h.Height)
			}
			if visible == hidden {
				t.Fatalf("pixel %d,%d visible=%v hidden=%v", x, y, visible, hidden)
			}
		}
	}
}
func TestAmbientAxisPreservesTheNearerEdgeOffset(t *testing.T) {
	if got := ambientAxis(8, 200, 800, 1600); got != 8 {
		t.Fatal("left-anchored box must keep its left offset", got)
	}
	// A ball pinned 12px from the right of an 800px page keeps 12px at 1600.
	if got := ambientAxis(700, 88, 800, 1600); got != 1600-12-88 {
		t.Fatal("right-anchored box must keep its right offset", got)
	}
	if got := ambientAxis(20, 100, 0, 900); got != 20 {
		t.Fatal("missing viewport must pass coordinates through", got)
	}
	if got := ambientAxis(20, 100, 900, 900); got != 20 {
		t.Fatal("matching viewport must be the identity", got)
	}
}
func TestAmbientRichNodeRequiresPopupAndMeasuredViewport(t *testing.T) {
	if !validNativeNodes([]NativeNode{{Kind: "rich", RichType: "fragment", Handle: "seat", Height: 80, Width: 40, Popup: true, Ambient: true, VW: 1600, VH: 900}}) {
		t.Fatal("bounded ambient seat rejected")
	}
	for _, node := range []NativeNode{
		{Kind: "rich", RichType: "fragment", Handle: "seat", Height: 80, Width: 40, Ambient: true, VW: 1600, VH: 900},
		{Kind: "rich", RichType: "fragment", Handle: "seat", Height: 80, Width: 40, Popup: true, Ambient: true},
		{Kind: "rich", RichType: "fragment", Handle: "seat", Height: 80, Width: 40, Popup: true, Ambient: true, VW: -1, VH: 900},
		{Kind: "rich", RichType: "fragment", Handle: "seat", Height: 80, Width: 40, Popup: true, Ambient: true, VW: 1e9, VH: 900},
	} {
		if validNativeNodes([]NativeNode{node}) {
			t.Fatal("unbounded ambient seat accepted", node)
		}
	}
}
func TestRichSurfaceAckRequiresCurrentPageRevisionTokenAndOwner(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "4:5"}
	page := &bridgePage{id: 3}
	layout := &richLayout{page: page, revision: 7, subject: subject}
	fresh := func() *Owner {
		return &Owner{selected: subject, page: page, state: Snapshot{Connected: true, Page: 3, Revision: 7, Subject: &subject}, rich: richState{enabled: true, begun: true, token: 9, pending: layout}}
	}
	tests := []struct {
		name            string
		mutate          func(*Owner)
		page            *bridgePage
		token, revision uint64
	}{
		{"retired page", func(*Owner) {}, &bridgePage{id: 3}, 9, 7}, {"old token", func(*Owner) {}, page, 8, 7}, {"old revision", func(*Owner) {}, page, 9, 6}, {"new native revision", func(o *Owner) { o.state.Revision = 8 }, page, 9, 7}, {"new subject", func(o *Owner) { o.selected.SessionID = "other" }, page, 9, 7}, {"masked", func(o *Owner) { o.masked = true }, page, 9, 7}, {"hidden", func(o *Owner) { o.rich.enabled = false }, page, 9, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := fresh()
			tt.mutate(o)
			o.ackRichLayout(tt.page, tt.token, tt.revision, true)
			if o.rich.applied != nil {
				t.Fatal("stale ack made rich content visible")
			}
		})
	}
	o := fresh()
	o.ackRichLayout(page, 9, 7, true)
	if o.rich.applied != layout || o.rich.pending != nil {
		t.Fatal("current exact layout was not acknowledged")
	}
	o = fresh()
	o.ackRichLayout(page, 9, 7, false)
	if o.rich.applied != nil || o.RichError() == "" {
		t.Fatal("unsupported kernel did not fail closed")
	}
}
func TestRichNodeKindsAreBoundedButRetainBrowserOwnedValues(t *testing.T) {
	for _, kind := range []string{"contenteditable", "file", "color", "date", "datetime-local", "time", "month", "week", "img", "audio", "video", "iframe", "object", "embed", "canvas"} {
		if !validNativeNodes([]NativeNode{{Kind: "rich", RichType: kind, Handle: "original-node", Height: 80}}) {
			t.Fatal(kind)
		}
	}
	for _, node := range []NativeNode{{Kind: "rich", RichType: "script", Handle: "x", Height: 80}, {Kind: "rich", RichType: "file", Height: 80}, {Kind: "rich", RichType: "canvas", Handle: "x", Height: 1201}} {
		if validNativeNodes([]NativeNode{node}) {
			t.Fatal("unbounded rich node", node)
		}
	}
}

func TestAmbientReservedWidthMatchesRightEdgeExpandedDock(t *testing.T) {
	dock := NativeNode{Kind: "popup", Ambient: true, X: 1080, Y: 52, Width: 352, Height: 480, VW: 1440, VH: 863}
	ball := NativeNode{Kind: "popup", Ambient: true, X: 1404, Y: 52, Width: 28, Height: 28, VW: 1440, VH: 863}
	owner := &Owner{
		state: Snapshot{
			Connected: true,
			GlobalDocuments: []NativeDocument{
				{Slot: "shell.overlay", Registrant: "dsh-pad", Nodes: []NativeNode{dock, ball}},
			},
		},
	}
	if got := owner.AmbientReservedWidth(1440); got != 364 {
		t.Fatalf("expected 364px reserved width at 1440px viewport, got %v", got)
	}
	if got := owner.AmbientReservedWidth(1200); got != 0 {
		t.Fatalf("narrow viewport (<1280) must not reserve column width, got %v", got)
	}
	owner.masked = true
	if got := owner.AmbientReservedWidth(1440); got != 0 {
		t.Fatalf("masked owner must not reserve column width, got %v", got)
	}
	owner.masked = false
	owner.state.GlobalDocuments[0].Nodes = []NativeNode{ball}
	if got := owner.AmbientReservedWidth(1440); got != 0 {
		t.Fatalf("collapsed floating ball must not reserve column width, got %v", got)
	}
	detached := dock
	detached.X = 400
	owner.state.GlobalDocuments[0].Nodes = []NativeNode{detached}
	if got := owner.AmbientReservedWidth(1440); got != 0 {
		t.Fatalf("dock moved away from the right edge must not reserve column width, got %v", got)
	}
}
