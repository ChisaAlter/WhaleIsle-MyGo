package assistant

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestRoutineMissedDismissalKeepsHostSessionAndNewMissesSeparate(t *testing.T) {
	state := dshbotstore.RoutineSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "host-a", BotID: "bot"}, SessionID: "session-a", Loaded: true, MissedKnown: true, MissedIDs: []string{"missed-a"}}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, Routines: RoutineModel{State: state, OnRefresh: func() {}}}
		routineControlBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
	}, 1100, 1800)
	dismiss := func() {
		t.Helper()
		for _, label := range tt.Texts() {
			if strings.Contains(label, "Dismiss missed-run notice") {
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
		t.Fatal("missed-run dismiss action unavailable", tt.Texts())
	}
	if !tt.HasText("Missed scheduled runs") {
		t.Fatal("initial missed-run notice unavailable")
	}
	dismiss()
	if tt.HasText("Missed scheduled runs") {
		t.Fatal("dismissed notice still visible")
	}
	if len(state.MissedIDs) != 1 || state.MissedIDs[0] != "missed-a" {
		t.Fatal("dismissal changed Host missed-run state")
	}
	state.MissedIDs = append(state.MissedIDs, "new-miss")
	tt.Frame()
	if !tt.HasText("Missed scheduled runs") {
		t.Fatal("previous dismissal hid a newly missed routine")
	}
	dismiss()
	state.Scope.HostKey = "host-b"
	tt.Frame()
	if !tt.HasText("Missed scheduled runs") {
		t.Fatal("another Host inherited dismissal")
	}
	dismiss()
	state.SessionID = "session-b"
	tt.Frame()
	if !tt.HasText("Missed scheduled runs") {
		t.Fatal("another canonical session inherited dismissal")
	}
	state.MissedKnown = false
	tt.Frame()
	if tt.HasText("Missed scheduled runs") {
		t.Fatal("unverified missed state presented as authoritative")
	}
}
