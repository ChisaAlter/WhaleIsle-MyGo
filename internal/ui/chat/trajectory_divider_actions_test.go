package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestParityTrajectoryDividerGesture(t *testing.T) {
	if low, high := trajectoryResizeBounds(451); low != 171 || high != 171 {
		t.Fatal("nested divider can squeeze ledger below 280 DIP")
	}
	selected := "a"
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{PaneWidth: 1100, Snapshot: session.ViewState{Messages: []session.Message{{ID: "a", Seq: 1, Role: "assistant", Kind: session.MessageKindText, Body: "answer"}}}, SelectedID: &selected})
	}, 1100, 640)
	before, _ := tt.Find("Trajectory event inspector")
	rail, ok := tt.Find("Resize trajectory details")
	if !ok {
		t.Fatal("divider unreachable")
	}
	tt.Press(rail.X+rail.W/2, rail.Y+100)
	tt.Move(rail.X-40, rail.Y+100)
	tt.Release(rail.X-40, rail.Y+100)
	after, _ := tt.Find("Trajectory event inspector")
	if after.W <= before.W+30 {
		t.Fatalf("drag did not resize: %+v %+v", before, after)
	}
	tt.Key(0, ui.KeyRight)
	keyed, _ := tt.Find("Trajectory event inspector")
	if keyed.W >= after.W {
		t.Fatal("divider keyboard arrow did not resize")
	}
	tt.Key(0, ui.KeyHome)
	reset, _ := tt.Find("Trajectory event inspector")
	if reset.W != before.W {
		t.Fatalf("divider reset lost default: %+v %+v", before, reset)
	}
}
