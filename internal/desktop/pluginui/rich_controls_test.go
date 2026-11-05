package pluginui

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestRichSurfacePreservesMultipleSeatsAndClipsEveryGap(t *testing.T) {
	rows := []richPlacement{{handle: "editor", bounds: mygo.Rectangle{X: 10, Y: 20, Width: 100, Height: 100}, clip: mygo.Rectangle{X: 10, Y: 40, Width: 100, Height: 60}}, {handle: "canvas", bounds: mygo.Rectangle{X: 150, Y: 30, Width: 90, Height: 70}, clip: mygo.Rectangle{X: 150, Y: 30, Width: 90, Height: 70}}}
	bounds, holes, seats, ok := richBuildLayout(rows)
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
	if _, _, _, ok := richBuildLayout([]richPlacement{{bounds: mygo.Rectangle{Width: 8193, Height: 30}}}); ok {
		t.Fatal("oversized surface accepted")
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
