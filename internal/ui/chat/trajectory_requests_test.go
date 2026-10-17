package chat

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTrajectoryRequestRowsAndCompleteEventsCopy(t *testing.T) {
	evidence := `{"type":"assistant/attempt","seq":2,"data":{"stream":[{"type":"text-delta","text":"` + strings.Repeat("long ", 1000) + `UNIQUE-END"}]}}`
	snapshot := session.ViewState{TrajectoryRequests: []session.TrajectoryRequest{{ID: "assistant:1:1", Purpose: "assistant", StartSeq: 1, Status: "error", Events: []json.RawMessage{json.RawMessage(evidence)}}}}
	rows := buildTrajectoryRows(trajectoryMessages(snapshot), "UNIQUE-END", false, false)
	if len(rows) != 1 {
		t.Fatal("full request event not searchable")
	}
	selected, tab, query := "request:assistant:1:1", "overview", ""
	copied := ""
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: snapshot, SelectedID: &selected, DetailTab: &tab, SearchQuery: &query, OnCopyDetail: func(s string) { copied = s }})
	}, 1000, 700)
	if err := tt.Click("Events"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Structured JSON"); !ok {
		t.Fatal("request events missing JSON inspector")
	}
	if err := tt.Click(i18n.T(i18n.KeyTrajCopy)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(copied, "UNIQUE-END") || len(copied) < len(evidence) {
		t.Fatal("request events truncated")
	}
	if err := tt.Click("Request"); err != nil {
		t.Fatal(err)
	}
	if tab != "request" {
		t.Fatal("request lifecycle tab unreachable")
	}
}
