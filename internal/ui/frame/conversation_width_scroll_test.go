package frame

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestTranscriptWidthRailsShareCommittedChatScrollAndFollowPolicy(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Center = 1200
	list := &ui.ListState{FollowEnd: true}
	scroll := &ui.ScrollState{}
	tt := ui.NewTester(func(c *ui.Context) {
		body := ui.Box(c).Fill()
		body.Children(func() {
			ui.List(c, list, 40).Fill().TrackScroll(scroll).Rows(func(row ui.ListRow) { ui.Box(row.Context).Height(60) })
			paintConversationWidthControls(c, Model{Sizes: &sizes, FileBindingID: "s", ChatList: list}, body, 768, true, scroll)
		})
	}, 1200, 400)
	tt.SetTime(time.Unix(7000, 0))
	list.ScrollToOffset(100)
	tt.Frame()
	rail, ok := tt.Find("Resize transcript right")
	if !ok {
		t.Fatal("rail absent")
	}
	tt.Scroll(rail.X+rail.W/2, 100, 0, 30)
	if scroll.Y != 130 || list.ScrollOffset() != 130 || !list.FollowEnd {
		t.Fatalf("rail lost tracked scroll or configured follow: %+v list%v", scroll, list.ScrollOffset())
	}
	for range 3 {
		tt.Frame()
		if scroll.Y != 130 {
			t.Fatal("later build adopted stale width tracker")
		}
	}
	list.ScrollToAnimated(20, ui.Center, ui.ScrollAnimation{Duration: func(float32) time.Duration { return time.Second }})
	tt.Frame()
	tt.Scroll(rail.X+rail.W/2, 100, 0, 20)
	if list.ScrollAnimating() || scroll.Y != 150 {
		t.Fatalf("rail wheel did not stop and scroll: %+v", scroll)
	}
	list.ScrollToEnd()
	tt.Frame()
	if !list.AtEnd() || !list.FollowEnd {
		t.Fatal("tail no longer follows")
	}
}
