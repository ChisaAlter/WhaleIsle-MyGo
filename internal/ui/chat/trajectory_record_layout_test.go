package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestTrajectoryRecordLayoutAndOptionalInspector(t *testing.T) {
	for _, width := range []float32{451, 900} {
		t.Run(fmt.Sprintf("%gDIP", width), func(t *testing.T) {
			selected := ""
			snap := session.ViewState{SelectedID: "layout-fixture", Messages: []session.Message{{ID: "u", Kind: session.MessageKindText, Role: "user", Body: strings.Repeat("long record ", 50)}}}
			tt := ui.NewTester(func(c *ui.Context) { TrajectoryView(c, TrajectoryModel{Snapshot: snap, SelectedID: &selected}) }, int(width), 600)
			before, ok := tt.Find("Trajectory row u")
			if !ok || before.H != 30 || before.W < width-2 {
				t.Fatalf("record did not occupy one full 30DIP row: %+v", before)
			}
			if _, ok := tt.Find("Close trajectory details"); ok {
				t.Fatal("empty inspector consumed ledger space")
			}
			tt.Click("Trajectory row u")
			if selected != "u" {
				t.Fatalf("selected=%q", selected)
			}
			panel, ok := tt.Find("Trajectory event inspector")
			if !ok || panel.W <= 0 {
				t.Fatalf("selected inspector missing: %+v", panel)
			}
			ledger, _ := tt.Find("Trajectory row u")
			if width <= 760 {
				if ledger.W < width-2 {
					t.Fatalf("overlay squeezed ledger: %+v", ledger)
				}
			} else if ledger.W < 280 {
				t.Fatalf("docked inspector squeezed ledger below source minimum: %+v", ledger)
			}
			tt.Click("Close trajectory details")
			if selected != "" {
				t.Fatal("close did not clear inspector selection")
			}
			restored, _ := tt.Find("Trajectory row u")
			if restored.W != before.W {
				t.Fatalf("close did not restore full ledger width: %+v / %+v", before, restored)
			}
		})
	}
}

func TestTrajectoryMissingSelectionDoesNotReserveInspector(t *testing.T) {
	selected := "not-in-this-session"
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{SelectedID: "empty"}, SelectedID: &selected})
	}, 451, 600)
	if _, ok := tt.Find("Close trajectory details"); ok {
		t.Fatal("stale selection opened an empty inspector")
	}
}

func TestTrajectoryInitiallySelectedNestedPaneUsesLayoutWidth(t *testing.T) {
	selected, tab, copied := "u", "overview", ""
	const body = "owned full detail"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(451).FillHeight().Children(func() {
			TrajectoryView(c, TrajectoryModel{PaneWidth: 451, Snapshot: session.ViewState{Messages: []session.Message{{ID: "u", Kind: session.MessageKindText, Role: "user", Body: body}}}, SelectedID: &selected, DetailTab: &tab, OnCopyDetail: func(v string) { copied = v }})
		})
	}, 1200, 700)
	panel, ok := tt.Find("Trajectory event inspector")
	if !ok || panel.W != 171 {
		t.Fatalf("first selected nested inspector width: %+v", panel)
	}
	row, _ := tt.Find("Trajectory row u")
	if row.W < 279 || row.W > 281 {
		t.Fatalf("source ledger minimum lost: %+v", row)
	}
	if err := tt.Click("复制"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(copied, body) {
		t.Fatalf("first-frame copy did not remain actionable: %q", copied)
	}
}

func TestTrajectoryNestedSupplementalTabsRemainReachable(t *testing.T) {
	for _, tc := range []struct {
		name, selected, strip, button, want string
		snapshot                            session.ViewState
	}{
		{name: "header", selected: "result", strip: "Trajectory request tabs", button: "Request header", want: "request-header", snapshot: headerFixture()},
		{name: "events", selected: "request:assistant:1:1", strip: "Trajectory lifecycle tabs", button: "Events", want: "request-events", snapshot: session.ViewState{TrajectoryRequests: []session.TrajectoryRequest{{ID: "assistant:1:1", Purpose: "assistant", StartSeq: 1, Status: "error"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected, tab := tc.selected, "overview"
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(380).FillHeight().Children(func() {
					TrajectoryView(c, TrajectoryModel{PaneWidth: 380, Snapshot: tc.snapshot, SelectedID: &selected, DetailTab: &tab})
				})
			}, 1200, 700)
			strip, ok := tt.Find(tc.strip)
			if !ok || strip.W > 81 || strip.W <= 0 {
				t.Fatalf("nested tab viewport: %+v", strip)
			}
			tt.Scroll(strip.X+strip.W/2, strip.Y+strip.H/2, 1000, 0)
			button, ok := tt.Find(tc.button)
			if !ok || button.X+button.W/2 < strip.X || button.X+button.W/2 > strip.X+strip.W {
				t.Fatalf("tab not reachable after horizontal scroll: strip=%+v button=%+v", strip, button)
			}
			if err := tt.Click(tc.button); err != nil {
				t.Fatal(err)
			}
			if tab != tc.want {
				t.Fatalf("tab did not activate: %q", tab)
			}
		})
	}
}
